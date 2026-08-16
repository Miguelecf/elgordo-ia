package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	paths map[string]string
	runs  []string
	fail  map[string]error
}

func (f *fakeRunner) LookPath(file string) (string, error) {
	if path, ok := f.paths[file]; ok {
		return path, nil
	}
	return "", errors.New("not found")
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) error {
	command := strings.Join(append([]string{name}, args...), " ")
	f.runs = append(f.runs, command)
	return f.fail[command]
}

func testOptions(t *testing.T) Options {
	t.Helper()
	return Options{
		Home:                t.TempDir(),
		Version:             "0.1.0-test",
		Stdin:               strings.NewReader("yes\n"),
		Stdout:              &bytes.Buffer{},
		Stderr:              &bytes.Buffer{},
		AcceptEngramInstall: true,
		OverwriteUserAssets: true,
		Runner:              &fakeRunner{paths: map[string]string{"opencode": "/bin/opencode", "engram": "/bin/engram"}, fail: map[string]error{}},
		Now:                 func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) },
	}
}

func TestInstallAssetsAndUninstall(t *testing.T) {
	opts := testOptions(t)
	if err := InstallAssets(opts); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(opts.Home, ".config", "opencode", "elgordo", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Files) != 9 {
		t.Fatalf("files = %d, want 9", len(manifest.Files))
	}
	if err := Uninstall(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opts.Home, ".config", "opencode", "commands", "eg.md")); !os.IsNotExist(err) {
		t.Fatalf("managed command remains: %v", err)
	}
}

func TestUninstallPreservesModifiedAsset(t *testing.T) {
	opts := testOptions(t)
	if err := InstallAssets(opts); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(opts.Home, ".config", "opencode", "commands", "eg.md")
	if err := os.WriteFile(path, []byte("user edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("modified asset removed: %v", err)
	}
}

func TestConfigureContext7PreservesConfig(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"share":"disabled","mcp":{"custom":{"enabled":false}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := testOptions(t)
	opts.Home = home
	if err := ConfigureContext7(opts); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !bytes.Contains(data, []byte(`"context7"`)) || !bytes.Contains(data, []byte(`"custom"`)) || !bytes.Contains(data, []byte(`"share": "disabled"`)) {
		t.Fatalf("config was not merged: %s", data)
	}
}

func TestConfigureContext7PreservesPrivateMode(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"token":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := testOptions(t)
	opts.Home = home
	if err := ConfigureContext7(opts); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}

func TestUninstallRejectsManifestTraversal(t *testing.T) {
	opts := testOptions(t)
	configRoot := filepath.Join(opts.Home, ".config", "opencode")
	manifest := Manifest{Version: "bad", Files: map[string]string{"../../victim": "sha256:bad"}}
	if err := writeJSONAtomic(filepath.Join(configRoot, "elgordo", "manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(opts); err == nil || !strings.Contains(err.Error(), "unsafe manifest path") {
		t.Fatalf("expected unsafe path error, got %v", err)
	}
}

func TestInstallRejectsSymlinkedManagedDirectory(t *testing.T) {
	opts := testOptions(t)
	configRoot := filepath.Join(opts.Home, ".config", "opencode")
	outside := filepath.Join(opts.Home, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(configRoot, "agents")); err != nil {
		t.Fatal(err)
	}
	if err := InstallAssets(opts); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestInstallRunsEngramSetup(t *testing.T) {
	opts := testOptions(t)
	runner := opts.Runner.(*fakeRunner)
	if err := Install(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(runner.runs, "\n")
	if !strings.Contains(joined, "/bin/engram setup opencode") || !strings.Contains(joined, "/bin/engram doctor --json") {
		t.Fatalf("missing Engram setup calls: %s", joined)
	}
}

func TestDoctorDetectsModifiedAsset(t *testing.T) {
	opts := testOptions(t)
	if err := InstallAssets(opts); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(opts.Home, ".config", "opencode", "commands", "eg.md")
	if err := os.WriteFile(path, []byte("modified\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checks, ok := Doctor(context.Background(), opts)
	if ok {
		t.Fatalf("doctor unexpectedly passed: %+v", checks)
	}
	found := false
	for _, check := range checks {
		if check.Name == "elgordo-assets" && check.Status == "error" {
			found = true
		}
	}
	if !found {
		t.Fatalf("asset error missing: %+v", checks)
	}
}

func TestAgentsKeepModelSelectionDynamic(t *testing.T) {
	err := fs.WalkDir(assets, "assets/opencode/agents", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := assets.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		frontmatterEnd := bytes.Index(data[4:], []byte("\n---"))
		if frontmatterEnd < 0 {
			t.Errorf("%s has no closing frontmatter", path)
			return nil
		}
		frontmatter := data[:frontmatterEnd+4]
		if bytes.Contains(frontmatter, []byte("\nmodel:")) {
			t.Errorf("%s hardcodes a model", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
