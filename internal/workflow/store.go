package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var ErrNoActiveChange = errors.New("no active change")

type Store struct {
	Root    string
	Version string
	Now     func() time.Time

	skillSources []skillSource
}

func FindRepositoryRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", errors.New("not inside a Git repository; initialize Git before running elgordo init")
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

func NewStore(root, version string) *Store {
	return &Store{Root: root, Version: version, Now: func() time.Time { return time.Now().UTC() }}
}

func (s *Store) Init() error {
	if s.Root == "" {
		return errors.New("repository root is required")
	}
	dir := s.baseDir()
	if err := os.MkdirAll(filepath.Join(dir, "changes"), 0o755); err != nil {
		return err
	}
	projectPath := filepath.Join(dir, "project.json")
	if _, err := os.Stat(projectPath); errors.Is(err, os.ErrNotExist) {
		project := Project{
			SchemaVersion:  SchemaVersion,
			ProjectName:    filepath.Base(s.Root),
			RepositoryRoot: s.Root,
			CreatedAt:      s.Now(),
			ElGordoVersion: s.Version,
		}
		if err := writeJSONAtomic(projectPath, project); err != nil {
			return err
		}
	}
	if err := addLocalExclude(s.Root, "/.elgordo/"); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(s.Root, ".engram"), 0o755); err != nil {
		return err
	}
	engramConfig := filepath.Join(s.Root, ".engram", "config.json")
	if _, err := os.Stat(engramConfig); errors.Is(err, os.ErrNotExist) {
		if err := writeJSONAtomic(engramConfig, map[string]string{"project_name": filepath.Base(s.Root)}); err != nil {
			return err
		}
	}
	if err := addLocalExclude(s.Root, "/.engram/"); err != nil {
		return err
	}
	if err := addLocalExclude(s.Root, "/.atl/"); err != nil {
		return err
	}
	_, err := s.RefreshSkillRegistry(false)
	return err
}

func (s *Store) StartChange(slug, title string) (*State, error) {
	if !slugPattern.MatchString(slug) {
		return nil, errors.New("change slug must use lowercase letters, numbers, and single hyphens")
	}
	if strings.TrimSpace(title) == "" {
		title = strings.ReplaceAll(slug, "-", " ")
	}
	return s.withLockState(func() (*State, error) {
		active, err := s.activeSlug()
		if err != nil {
			return nil, err
		}
		if active != "" {
			return nil, fmt.Errorf("change %q is already active", active)
		}
		dir := s.changeDir(slug)
		if _, err := os.Stat(dir); err == nil {
			return nil, fmt.Errorf("change %q already exists", slug)
		}
		if err := os.MkdirAll(filepath.Join(dir, "plans"), 0o755); err != nil {
			return nil, err
		}
		for path, content := range initialArtifacts(title) {
			if err := writeFileAtomic(filepath.Join(dir, path), []byte(content), 0o644); err != nil {
				os.RemoveAll(dir)
				return nil, err
			}
		}
		now := s.Now()
		state := State{
			SchemaVersion: SchemaVersion,
			ChangeID:      slug,
			Title:         title,
			Phase:         PhasePlanning,
			Plan:          Plan{Revision: 1, Path: "plans/0001.md"},
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		event := Event{At: now, Command: "change start", To: PhasePlanning}
		if err := s.commitState(&state, event); err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
		if err := writeFileAtomic(s.activePath(), []byte(slug+"\n"), 0o644); err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
		return &state, nil
	})
}

func (s *Store) ActiveState() (*State, error) {
	slug, err := s.activeSlug()
	if err != nil {
		return nil, err
	}
	if slug == "" {
		return nil, fmt.Errorf("%w; run elgordo change start <slug>", ErrNoActiveChange)
	}
	return s.loadState(slug)
}

func (s *Store) PlanHash() (string, error) {
	state, err := s.ActiveState()
	if err != nil {
		return "", err
	}
	return s.hashPlan(state)
}

func (s *Store) PlanReady() (*State, error) {
	return s.transition("plan ready", []Phase{PhasePlanning}, PhasePlanReview, "", "", "", false)
}

func (s *Store) SealPlan(expected string) (*State, error) {
	return s.withLockState(func() (*State, error) {
		state, err := s.ActiveState()
		if err != nil {
			return nil, err
		}
		if state.Phase != PhasePlanReview {
			return nil, invalidPhase(state.Phase, PhasePlanReview)
		}
		actual, err := s.hashPlan(state)
		if err != nil {
			return nil, err
		}
		expected = normalizeHash(expected)
		if expected == "" || expected != actual {
			return nil, fmt.Errorf("plan hash mismatch: expected %q, current plan is %q", expected, actual)
		}
		planContent, err := os.ReadFile(filepath.Join(s.changeDir(state.ChangeID), filepath.FromSlash(state.Plan.Path)))
		if err != nil {
			return nil, err
		}
		state.Plan.Snapshot = fmt.Sprintf("plans/%04d.sealed.md", state.Plan.Revision)
		if err := writeFileAtomic(filepath.Join(s.changeDir(state.ChangeID), filepath.FromSlash(state.Plan.Snapshot)), planContent, 0o444); err != nil {
			return nil, err
		}
		now := s.Now()
		from := state.Phase
		state.Plan.SHA256 = actual
		state.Plan.SealedAt = &now
		state.Phase = PhaseExecuting
		state.ExecutionRound++
		state.UpdatedAt = now
		if err := s.commitState(state, Event{At: now, Command: "plan seal", From: from, To: state.Phase, PlanSHA256: actual}); err != nil {
			return nil, err
		}
		return state, nil
	})
}

func (s *Store) VerifyPlan() error {
	state, err := s.ActiveState()
	if err != nil {
		return err
	}
	return s.ensurePlanUnchanged(state)
}

func (s *Store) ExecutionReady() (*State, error) {
	return s.transition("execution ready", []Phase{PhaseExecuting}, PhaseCodeReview, "", "", "", true)
}

func (s *Store) CodeApprove() (*State, error) {
	return s.transition("code approve", []Phase{PhaseCodeReview}, PhaseQA, "", "", "", true)
}

func (s *Store) CodeReject(route, reason string) (*State, error) {
	return s.routeBack("code reject", []Phase{PhaseCodeReview}, route, reason, "")
}

func (s *Store) QASubmit(verdict, route, reason string) (*State, error) {
	verdict = strings.ToLower(verdict)
	if verdict == "pass" {
		return s.transition("qa submit", []Phase{PhaseQA}, PhaseFinalReview, "", "pass", "", true)
	}
	if verdict != "fail" {
		return nil, errors.New("QA verdict must be pass or fail")
	}
	return s.routeBack("qa submit", []Phase{PhaseQA}, route, reason, "fail")
}

func (s *Store) FinalApprove() (*State, error) {
	return s.withLockState(func() (*State, error) {
		state, err := s.ActiveState()
		if err != nil {
			return nil, err
		}
		if state.Phase == PhaseDone {
			if err := os.Remove(s.activePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			return state, nil
		}
		if state.Phase != PhaseFinalReview {
			return nil, invalidPhase(state.Phase, PhaseFinalReview)
		}
		if err := s.ensurePlanUnchanged(state); err != nil {
			return nil, err
		}
		now := s.Now()
		from := state.Phase
		state.Phase = PhaseDone
		state.UpdatedAt = now
		if err := s.commitState(state, Event{At: now, Command: "final approve", From: from, To: PhaseDone, PlanSHA256: state.Plan.SHA256}); err != nil {
			return nil, err
		}
		if err := os.Remove(s.activePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return state, nil
	})
}

func (s *Store) FinalReject(route, reason string) (*State, error) {
	return s.routeBack("final reject", []Phase{PhaseFinalReview}, route, reason, "")
}

func (s *Store) ForceReplan(reason string) (*State, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("a replanning reason is required")
	}
	return s.replan("plan replan", []Phase{PhaseExecuting, PhaseCodeReview, PhaseQA, PhaseFinalReview}, reason, "")
}

func (s *Store) transition(command string, allowed []Phase, target Phase, route, verdict, reason string, verify bool) (*State, error) {
	return s.withLockState(func() (*State, error) {
		state, err := s.ActiveState()
		if err != nil {
			return nil, err
		}
		if !containsPhase(allowed, state.Phase) {
			return nil, invalidPhase(state.Phase, allowed...)
		}
		if verify {
			if err := s.ensurePlanUnchanged(state); err != nil {
				return nil, err
			}
		}
		now := s.Now()
		from := state.Phase
		state.Phase = target
		if target == PhaseExecuting {
			state.ExecutionRound++
		}
		if target == PhaseQA {
			state.QARound++
		}
		state.UpdatedAt = now
		if err := s.commitState(state, Event{At: now, Command: command, From: from, To: target, PlanSHA256: state.Plan.SHA256, Route: route, Verdict: verdict, Reason: reason}); err != nil {
			return nil, err
		}
		return state, nil
	})
}

func (s *Store) routeBack(command string, allowed []Phase, route, reason, verdict string) (*State, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("a rejection reason is required")
	}
	switch route {
	case "execution":
		return s.transition(command, allowed, PhaseExecuting, route, verdict, reason, true)
	case "planning":
		return s.replan(command, allowed, reason, verdict)
	default:
		return nil, errors.New("route must be execution or planning")
	}
}

func (s *Store) replan(command string, allowed []Phase, reason, verdict string) (*State, error) {
	return s.withLockState(func() (*State, error) {
		state, err := s.ActiveState()
		if err != nil {
			return nil, err
		}
		if !containsPhase(allowed, state.Phase) {
			return nil, invalidPhase(state.Phase, allowed...)
		}
		if state.Plan.Snapshot == "" {
			return nil, errors.New("sealed plan snapshot is missing")
		}
		oldPlan := filepath.Join(s.changeDir(state.ChangeID), filepath.FromSlash(state.Plan.Snapshot))
		content, err := os.ReadFile(oldPlan)
		if err != nil {
			return nil, err
		}
		snapshotSum := sha256.Sum256(content)
		if "sha256:"+hex.EncodeToString(snapshotSum[:]) != state.Plan.SHA256 {
			return nil, errors.New("sealed plan snapshot was modified; manual recovery is required")
		}
		state.Plan.Revision++
		state.Plan.Path = fmt.Sprintf("plans/%04d.md", state.Plan.Revision)
		state.Plan.Snapshot = ""
		state.Plan.SHA256 = ""
		state.Plan.SealedAt = nil
		if err := writeFileAtomic(filepath.Join(s.changeDir(state.ChangeID), filepath.FromSlash(state.Plan.Path)), content, 0o644); err != nil {
			return nil, err
		}
		now := s.Now()
		from := state.Phase
		state.Phase = PhasePlanning
		state.UpdatedAt = now
		if err := s.commitState(state, Event{At: now, Command: command, From: from, To: PhasePlanning, Route: "planning", Verdict: verdict, Reason: reason}); err != nil {
			return nil, err
		}
		return state, nil
	})
}

func (s *Store) ensurePlanUnchanged(state *State) error {
	if state.Plan.SHA256 == "" {
		return errors.New("plan is not sealed")
	}
	actual, err := s.hashPlan(state)
	if err != nil {
		return err
	}
	if actual != state.Plan.SHA256 {
		return fmt.Errorf("sealed plan changed: expected %s, found %s; run elgordo plan replan --reason <reason>", state.Plan.SHA256, actual)
	}
	if state.Plan.Snapshot == "" {
		return errors.New("sealed plan snapshot is missing")
	}
	snapshot, err := os.ReadFile(filepath.Join(s.changeDir(state.ChangeID), filepath.FromSlash(state.Plan.Snapshot)))
	if err != nil {
		return err
	}
	snapshotSum := sha256.Sum256(snapshot)
	if "sha256:"+hex.EncodeToString(snapshotSum[:]) != state.Plan.SHA256 {
		return errors.New("sealed plan snapshot was modified")
	}
	return nil
}

func (s *Store) hashPlan(state *State) (string, error) {
	data, err := os.ReadFile(filepath.Join(s.changeDir(state.ChangeID), filepath.FromSlash(state.Plan.Path)))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (s *Store) withLockState(fn func() (*State, error)) (*State, error) {
	lock := filepath.Join(s.baseDir(), ".lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			if !s.reclaimStaleLock(lock) {
				return nil, errors.New("another elgordo operation is in progress")
			}
			if err := os.Mkdir(lock, 0o700); err != nil {
				return nil, fmt.Errorf("acquire recovered workflow lock: %w", err)
			}
		} else {
			return nil, err
		}
	}
	metadata := struct {
		PID        int       `json:"pid"`
		AcquiredAt time.Time `json:"acquired_at"`
	}{PID: os.Getpid(), AcquiredAt: s.Now()}
	if err := writeJSONAtomic(filepath.Join(lock, "owner.json"), metadata); err != nil {
		os.RemoveAll(lock)
		return nil, err
	}
	defer os.RemoveAll(lock)
	return fn()
}

func (s *Store) reclaimStaleLock(lock string) bool {
	data, err := os.ReadFile(filepath.Join(lock, "owner.json"))
	if err != nil {
		info, statErr := os.Stat(lock)
		if statErr != nil || s.Now().Sub(info.ModTime()) < 10*time.Minute {
			return false
		}
		return os.RemoveAll(lock) == nil
	}
	var owner struct {
		PID        int       `json:"pid"`
		AcquiredAt time.Time `json:"acquired_at"`
	}
	if json.Unmarshal(data, &owner) != nil {
		return false
	}
	dead := owner.PID <= 0 || errors.Is(syscall.Kill(owner.PID, 0), syscall.ESRCH)
	if !dead {
		return false
	}
	return os.RemoveAll(lock) == nil
}

func (s *Store) persistState(state *State) error {
	return writeJSONAtomic(filepath.Join(s.changeDir(state.ChangeID), "state.json"), state)
}

func (s *Store) loadState(slug string) (*State, error) {
	if !slugPattern.MatchString(slug) {
		return nil, errors.New("active change contains an invalid slug")
	}
	data, err := os.ReadFile(filepath.Join(s.changeDir(slug), "state.json"))
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("invalid state.json: %w", err)
	}
	if state.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported state schema %d", state.SchemaVersion)
	}
	if state.ChangeID != slug || !slugPattern.MatchString(state.ChangeID) {
		return nil, errors.New("state change_id does not match the active change")
	}
	expectedPlan := fmt.Sprintf("plans/%04d.md", state.Plan.Revision)
	if state.Plan.Revision < 1 || state.Plan.Path != expectedPlan {
		return nil, errors.New("state contains an invalid plan path")
	}
	if state.Plan.Snapshot != "" {
		expectedSnapshot := fmt.Sprintf("plans/%04d.sealed.md", state.Plan.Revision)
		if state.Plan.Snapshot != expectedSnapshot {
			return nil, errors.New("state contains an invalid sealed plan snapshot path")
		}
	}
	return &state, nil
}

func (s *Store) commitState(state *State, event Event) error {
	event.Sequence = len(state.Events) + 1
	state.Events = append(state.Events, event)
	if err := s.persistState(state); err != nil {
		state.Events = state.Events[:len(state.Events)-1]
		return err
	}
	// events.jsonl is a recoverable projection; state.json is authoritative.
	_ = s.syncEvents(state)
	return nil
}

func (s *Store) syncEvents(state *State) error {
	path := filepath.Join(s.changeDir(state.ChangeID), "events.jsonl")
	var content strings.Builder
	for _, event := range state.Events {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		content.Write(data)
		content.WriteByte('\n')
	}
	return writeFileAtomic(path, []byte(content.String()), 0o644)
}

func (s *Store) activeSlug() (string, error) {
	data, err := os.ReadFile(s.activePath())
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	slug := strings.TrimSpace(string(data))
	if slug != "" && !slugPattern.MatchString(slug) {
		return "", errors.New("active change contains an invalid slug")
	}
	return slug, nil
}

func (s *Store) baseDir() string              { return filepath.Join(s.Root, ".elgordo") }
func (s *Store) activePath() string           { return filepath.Join(s.baseDir(), "active") }
func (s *Store) changeDir(slug string) string { return filepath.Join(s.baseDir(), "changes", slug) }

func initialArtifacts(title string) map[string]string {
	return map[string]string{
		"intent.md":     "# Intent\n\n" + title + "\n\n# User Outcome\n\n# Constraints\n\n# Open Questions\n",
		"plans/0001.md": "# Goal\n\n# Scope\n\n# Non-Goals\n\n# Assumptions\n\n# Work Units\n\n# Acceptance Criteria\n\n# Verification\n\n# Risks and Rollback\n",
		"execution.md":  "# Summary\n\n# Files Changed\n\n# Commands Run\n\n# Tests\n\n# Deviations\n\n# Remaining Concerns\n",
		"qa.md":         "# Verdict\n\n# Plan Conformance\n\n# Automated Checks\n\n# Manual Checks\n\n# Findings\n\n# Recommended Route\n",
	}
}

func addLocalExclude(root, rule string) error {
	cmd := exec.Command("git", "rev-parse", "--git-path", "info/exclude")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("resolve Git exclude path: %w", err)
	}
	path := strings.TrimSpace(string(out))
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == rule {
			return nil
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(rule + "\n")
	return err
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o644)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".elgordo-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func containsPhase(phases []Phase, phase Phase) bool {
	for _, candidate := range phases {
		if candidate == phase {
			return true
		}
	}
	return false
}

func invalidPhase(actual Phase, expected ...Phase) error {
	values := make([]string, len(expected))
	for i, phase := range expected {
		values[i] = string(phase)
	}
	return fmt.Errorf("invalid phase %s; expected %s", actual, strings.Join(values, " or "))
}

func normalizeHash(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if len(value) == 64 {
		return "sha256:" + value
	}
	return value
}
