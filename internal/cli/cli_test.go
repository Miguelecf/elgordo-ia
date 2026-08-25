package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Miguelecf/elgordo-ia/internal/installer"
)

func TestProjectLifecycleCommands(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	commands := [][]string{
		{"init"},
		{"change", "start", "cli-flow", "--title", "CLI flow"},
		{"plan", "ready"},
	}
	for _, args := range commands {
		if code, _, stderr := runTestCLI(root, args...); code != 0 {
			t.Fatalf("%v failed: %s", args, stderr)
		}
	}
	code, stdout, stderr := runTestCLI(root, "plan", "hash")
	if code != 0 {
		t.Fatalf("plan hash failed: %s", stderr)
	}
	hash := strings.TrimSpace(stdout)
	if code, _, stderr = runTestCLI(root, "plan", "seal", "--expect", hash); code != 0 {
		t.Fatalf("plan seal failed: %s", stderr)
	}
	if code, stdout, stderr = runTestCLI(root, "status", "--json"); code != 0 || !strings.Contains(stdout, `"phase": "EXECUTING"`) {
		t.Fatalf("status failed: code=%d out=%s err=%s", code, stdout, stderr)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	code, _, stderr := runTestCLI(t.TempDir(), "wat")
	if code != 1 || !strings.Contains(stderr, "unknown command") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestSkillRegistryRefreshCommand(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if code, _, stderr := runTestCLI(root, "init"); code != 0 {
		t.Fatal(stderr)
	}
	if code, stdout, stderr := runTestCLI(root, "skill-registry", "refresh", "--force"); code != 0 || !strings.Contains(stdout, "Skill registry updated") {
		t.Fatalf("refresh: code=%d out=%q err=%q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".atl", "skill-registry.md")); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runTestCLI(root, "skill-registry", "refresh", "unexpected"); code == 0 {
		t.Fatal("skill registry refresh accepted trailing argument")
	}
}

func TestInitStopsWhenOpenSpecInitializationFails(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"init"}, Dependencies{
		Version: "test",
		Stdin:   strings.NewReader(""),
		Stdout:  &stdout,
		Stderr:  &stderr,
		Getwd:   func() (string, error) { return root, nil },
		EnsureWorkflowDependencies: func(context.Context, installer.Options) error {
			return nil
		},
		InitOpenSpec: func(string) error {
			return errors.New("OpenSpec unavailable")
		},
	})
	if code != 1 || !strings.Contains(stderr.String(), "initialize OpenSpec") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".elgordo", "project.json")); !os.IsNotExist(err) {
		t.Fatalf("workflow state created despite OpenSpec failure: %v", err)
	}
}

func TestInitBootstrapsApprovedDependenciesBeforeOpenSpec(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}

	var steps []string
	code := Run([]string{"init", "--accept-engram-install", "--accept-openspec-install"}, Dependencies{
		Version: "test",
		Stdin:   strings.NewReader(""),
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Getwd:   func() (string, error) { return root, nil },
		EnsureWorkflowDependencies: func(_ context.Context, opts installer.Options) error {
			if !opts.AcceptEngramInstall || !opts.AcceptOpenSpecInstall {
				t.Fatal("init did not forward approved dependency installation")
			}
			steps = append(steps, "dependencies")
			return nil
		},
		InitOpenSpec: func(root string) error {
			steps = append(steps, "openspec")
			return os.MkdirAll(filepath.Join(root, "openspec", "changes"), 0o755)
		},
	})
	if code != 0 {
		t.Fatal("init failed")
	}
	if got, want := strings.Join(steps, ","), "dependencies,openspec"; got != want {
		t.Fatalf("steps = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, ".elgordo", "project.json")); err != nil {
		t.Fatalf("project state missing: %v", err)
	}
}

func TestInitLeavesProjectUninitializedWhenDependencyBootstrapFails(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}

	code := Run([]string{"init"}, Dependencies{
		Version: "test",
		Stdin:   strings.NewReader(""),
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Getwd:   func() (string, error) { return root, nil },
		EnsureWorkflowDependencies: func(context.Context, installer.Options) error {
			return errors.New("OpenSpec installation declined")
		},
		InitOpenSpec: func(string) error {
			t.Fatal("OpenSpec initialization ran after dependency failure")
			return nil
		},
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if _, err := os.Stat(filepath.Join(root, ".elgordo", "project.json")); !os.IsNotExist(err) {
		t.Fatalf("workflow state created despite dependency failure: %v", err)
	}
}

func TestHumanGatesRejectTrailingArguments(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	for _, args := range [][]string{{"init"}, {"change", "start", "strict"}, {"plan", "ready"}} {
		if code, _, stderr := runTestCLI(root, args...); code != 0 {
			t.Fatalf("setup %v: %s", args, stderr)
		}
	}
	hashCode, hash, stderr := runTestCLI(root, "plan", "hash")
	if hashCode != 0 {
		t.Fatal(stderr)
	}
	if code, _, _ := runTestCLI(root, "plan", "seal", "--expect", strings.TrimSpace(hash), "unexpected"); code == 0 {
		t.Fatal("plan seal accepted trailing argument")
	}
	if code, _, stderr := runTestCLI(root, "plan", "seal", "--expect", strings.TrimSpace(hash)); code != 0 {
		t.Fatal(stderr)
	}
	if code, _, _ := runTestCLI(root, "execution", "ready", "unexpected"); code == 0 {
		t.Fatal("execution ready accepted trailing argument")
	}
}

func runTestCLI(root string, args ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, Dependencies{
		Version: "test",
		Stdin:   strings.NewReader(""),
		Stdout:  &stdout,
		Stderr:  &stderr,
		Getwd:   func() (string, error) { return root, nil },
		EnsureWorkflowDependencies: func(context.Context, installer.Options) error {
			return nil
		},
		InitOpenSpec: func(root string) error {
			return os.MkdirAll(filepath.Join(root, "openspec", "changes"), 0o755)
		},
	})
	return code, stdout.String(), stderr.String()
}
