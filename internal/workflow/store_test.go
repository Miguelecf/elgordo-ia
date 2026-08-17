package workflow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	store := NewStore(root, "test")
	store.Now = func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) }
	if err := store.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return store
}

func TestHappyPath(t *testing.T) {
	store := newTestStore(t)
	state, err := store.StartChange("add-search", "Add search")
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhasePlanning {
		t.Fatalf("phase = %s", state.Phase)
	}
	if _, err := store.PlanReady(); err != nil {
		t.Fatal(err)
	}
	hash, err := store.PlanHash()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SealPlan(hash); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ExecutionReady(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CodeApprove(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.QASubmit("pass", "", ""); err != nil {
		t.Fatal(err)
	}
	state, err = store.FinalApprove()
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseDone {
		t.Fatalf("phase = %s", state.Phase)
	}
	if _, err := os.Stat(filepath.Join(store.Root, ".elgordo", "active")); !os.IsNotExist(err) {
		t.Fatalf("active file still exists: %v", err)
	}
}

func TestSealedPlanTamperingBlocksTransition(t *testing.T) {
	store := newTestStore(t)
	state, _ := store.StartChange("safe-plan", "Safe plan")
	_, _ = store.PlanReady()
	hash, _ := store.PlanHash()
	_, _ = store.SealPlan(hash)
	planPath := filepath.Join(store.changeDir(state.ChangeID), filepath.FromSlash(state.Plan.Path))
	if err := os.WriteFile(planPath, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := store.ExecutionReady()
	if err == nil || !strings.Contains(err.Error(), "sealed plan changed") {
		t.Fatalf("expected tamper error, got %v", err)
	}
}

func TestTamperedPlanCanRecoverThroughExplicitReplan(t *testing.T) {
	store := newTestStore(t)
	state, _ := store.StartChange("recover-plan", "Recover plan")
	_, _ = store.PlanReady()
	hash, _ := store.PlanHash()
	_, _ = store.SealPlan(hash)
	planPath := filepath.Join(store.changeDir(state.ChangeID), filepath.FromSlash(state.Plan.Path))
	if err := os.WriteFile(planPath, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := store.ForceReplan("plan file changed accidentally")
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhasePlanning || state.Plan.Revision != 2 || state.Plan.Snapshot != "" {
		t.Fatalf("unexpected recovered state: %+v", state)
	}
}

func TestPlanningRouteCreatesRevision(t *testing.T) {
	store := newTestStore(t)
	_, _ = store.StartChange("revise-plan", "Revise plan")
	_, _ = store.PlanReady()
	hash, _ := store.PlanHash()
	_, _ = store.SealPlan(hash)
	_, _ = store.ExecutionReady()
	state, err := store.CodeReject("planning", "API contract is wrong")
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhasePlanning || state.Plan.Revision != 2 || state.Plan.SHA256 != "" {
		t.Fatalf("unexpected replanning state: %+v", state)
	}
	if _, err := os.Stat(filepath.Join(store.changeDir(state.ChangeID), "plans", "0002.md")); err != nil {
		t.Fatal(err)
	}
}

func TestInitIsIdempotentAndUsesLocalExcludes(t *testing.T) {
	store := newTestStore(t)
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(store.Root, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{"/.elgordo/", "/.engram/", "/.atl/"} {
		if strings.Count(string(data), rule) != 1 {
			t.Fatalf("rule %q count != 1 in %q", rule, data)
		}
	}
}

func TestRejectRequiresReason(t *testing.T) {
	store := newTestStore(t)
	_, _ = store.StartChange("reason", "Reason")
	_, _ = store.PlanReady()
	hash, _ := store.PlanHash()
	_, _ = store.SealPlan(hash)
	_, _ = store.ExecutionReady()
	if _, err := store.CodeReject("execution", ""); err == nil {
		t.Fatal("expected missing reason error")
	}
}

func TestExecutionRetryIncrementsRound(t *testing.T) {
	store := newTestStore(t)
	_, _ = store.StartChange("retry", "Retry")
	_, _ = store.PlanReady()
	hash, _ := store.PlanHash()
	state, _ := store.SealPlan(hash)
	if state.ExecutionRound != 1 {
		t.Fatalf("initial round = %d", state.ExecutionRound)
	}
	_, _ = store.ExecutionReady()
	state, err := store.CodeReject("execution", "fix behavior")
	if err != nil {
		t.Fatal(err)
	}
	if state.ExecutionRound != 2 {
		t.Fatalf("retry round = %d, want 2", state.ExecutionRound)
	}
}

func TestRejectsTamperedStatePaths(t *testing.T) {
	store := newTestStore(t)
	state, _ := store.StartChange("tamper", "Tamper")
	state.Plan.Path = "../../victim"
	data, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(store.changeDir(state.ChangeID), "state.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ActiveState(); err == nil || !strings.Contains(err.Error(), "invalid plan path") {
		t.Fatalf("expected invalid plan path, got %v", err)
	}
	if err := os.WriteFile(store.activePath(), []byte("../../victim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ActiveState(); err == nil || !strings.Contains(err.Error(), "invalid slug") {
		t.Fatalf("expected invalid slug, got %v", err)
	}
}

func TestStartChangeFailsClosedOnCorruptActiveMarker(t *testing.T) {
	store := newTestStore(t)
	if err := os.WriteFile(store.activePath(), []byte("../../victim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartChange("second", "Second"); err == nil || !strings.Contains(err.Error(), "invalid slug") {
		t.Fatalf("expected corrupt active marker error, got %v", err)
	}
}

func TestFinalApproveRetryCleansDoneMarker(t *testing.T) {
	store := newTestStore(t)
	_, _ = store.StartChange("done-retry", "Done retry")
	_, _ = store.PlanReady()
	hash, _ := store.PlanHash()
	_, _ = store.SealPlan(hash)
	_, _ = store.ExecutionReady()
	_, _ = store.CodeApprove()
	_, _ = store.QASubmit("pass", "", "")
	state, err := store.FinalApprove()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.activePath(), []byte(state.ChangeID+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FinalApprove(); err != nil {
		t.Fatalf("retry final approve: %v", err)
	}
	if _, err := os.Stat(store.activePath()); !os.IsNotExist(err) {
		t.Fatalf("active marker remains: %v", err)
	}
}

func TestReclaimsStaleLock(t *testing.T) {
	store := newTestStore(t)
	lock := filepath.Join(store.baseDir(), ".lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		t.Fatal(err)
	}
	owner := map[string]any{"pid": 99999999, "acquired_at": store.Now().Add(-time.Hour)}
	if err := writeJSONAtomic(filepath.Join(lock, "owner.json"), owner); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartChange("recovered", "Recovered"); err != nil {
		t.Fatalf("stale lock was not reclaimed: %v", err)
	}
}

func TestInitSupportsLinkedWorktree(t *testing.T) {
	mainRoot := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-qm", "init"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = mainRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	worktree := filepath.Join(t.TempDir(), "linked")
	cmd := exec.Command("git", "worktree", "add", "-q", "-b", "linked-test", worktree)
	cmd.Dir = mainRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v: %s", err, out)
	}
	store := NewStore(worktree, "test")
	if err := store.Init(); err != nil {
		t.Fatalf("Init linked worktree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".atl", "skill-registry.md")); err != nil {
		t.Fatalf("linked worktree registry: %v", err)
	}
}
