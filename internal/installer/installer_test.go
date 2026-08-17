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
	paths        map[string]string
	runs         []string
	fail         map[string]error
	outputs      map[string][]byte
	installPaths map[string]map[string]string
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
	if err := f.fail[command]; err != nil {
		return err
	}
	for name, path := range f.installPaths[command] {
		f.paths[name] = path
	}
	return nil
}

func (f *fakeRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	command := strings.Join(append([]string{name}, args...), " ")
	f.runs = append(f.runs, command)
	return f.outputs[command], f.fail[command]
}

func testOptions(t *testing.T) Options {
	t.Helper()
	return Options{
		Home:                  t.TempDir(),
		Version:               "0.1.0-test",
		Stdin:                 strings.NewReader("yes\n"),
		Stdout:                &bytes.Buffer{},
		Stderr:                &bytes.Buffer{},
		AcceptEngramInstall:   true,
		AcceptOpenSpecInstall: true,
		OverwriteUserAssets:   true,
		Runner:                &fakeRunner{paths: map[string]string{"opencode": "/bin/opencode", "engram": "/bin/engram", "node": "/bin/node", "openspec": "/bin/openspec"}, fail: map[string]error{}, outputs: map[string][]byte{"node --version": []byte("v20.19.0\n")}},
		Now:                   func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) },
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
	if want := embeddedAssetCount(t); len(manifest.Files) != want {
		t.Fatalf("files = %d, want %d", len(manifest.Files), want)
	}
	if err := Uninstall(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opts.Home, ".config", "opencode", "commands", "eg.md")); !os.IsNotExist(err) {
		t.Fatalf("managed command remains: %v", err)
	}
}

func embeddedAssetCount(t *testing.T) int {
	t.Helper()
	count := 0
	if err := fs.WalkDir(assets, "assets/opencode", func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			count++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return count
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
	if !strings.Contains(joined, "/bin/engram setup opencode") || !strings.Contains(joined, "/bin/engram doctor --json") || !strings.Contains(joined, "node --version") || !strings.Contains(joined, "/bin/openspec --version") {
		t.Fatalf("missing dependency checks: %s", joined)
	}
}

func TestInstallDeclinesOpenSpecInstall(t *testing.T) {
	opts := testOptions(t)
	opts.AcceptOpenSpecInstall = false
	opts.Stdin = strings.NewReader("no\n")
	runner := opts.Runner.(*fakeRunner)
	delete(runner.paths, "openspec")
	if err := Install(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "OpenSpec installation declined") {
		t.Fatalf("expected declined installation error, got %v", err)
	}
	if strings.Contains(strings.Join(runner.runs, "\n"), "npm install -g") {
		t.Fatalf("npm ran after declined installation: %s", runner.runs)
	}
}

func TestInstallAcceptsOpenSpecInstall(t *testing.T) {
	opts := testOptions(t)
	opts.AcceptOpenSpecInstall = false
	opts.Stdin = strings.NewReader("yes\n")
	runner := opts.Runner.(*fakeRunner)
	delete(runner.paths, "openspec")
	runner.paths["npm"] = "/bin/npm"
	runner.installPaths = map[string]map[string]string{"npm install -g @fission-ai/openspec@1.5.0": {"openspec": "/bin/openspec"}}
	if err := Install(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(runner.runs, "\n"), "npm install -g @fission-ai/openspec@1.5.0") {
		t.Fatalf("missing exact npm install command: %s", runner.runs)
	}
}

func TestInstallRejectsMissingOrUnsupportedNode(t *testing.T) {
	for name, version := range map[string]string{"missing": "", "unsupported": "v20.18.0\n", "malformed": "version 20\n"} {
		t.Run(name, func(t *testing.T) {
			opts := testOptions(t)
			runner := opts.Runner.(*fakeRunner)
			if name == "missing" {
				delete(runner.paths, "node")
			} else {
				runner.outputs["node --version"] = []byte(version)
			}
			if err := Install(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "Node.js >=20.19.0") {
				t.Fatalf("expected Node version error, got %v", err)
			}
		})
	}
}

func TestNodeVersionSupported(t *testing.T) {
	for value, want := range map[string]bool{
		"v20.19.0": true,
		"20.19.0":  true,
		"v20.18.9": false,
		"v19.99.0": false,
		"v21.0.0":  true,
		"v20.19":   false,
		"+20.19.0": false,
		"node 20":  false,
	} {
		if got := nodeVersionSupported(value); got != want {
			t.Errorf("nodeVersionSupported(%q) = %t, want %t", value, got, want)
		}
	}
}

func TestDoctorReportsMissingOpenSpec(t *testing.T) {
	opts := testOptions(t)
	delete(opts.Runner.(*fakeRunner).paths, "openspec")
	checks, ok := Doctor(context.Background(), opts)
	if ok {
		t.Fatalf("doctor unexpectedly passed: %+v", checks)
	}
	for _, check := range checks {
		if check.Name == "openspec" && check.Status == "error" {
			return
		}
	}
	t.Fatalf("OpenSpec error missing: %+v", checks)
}

func TestDoctorReportsBrokenOpenSpec(t *testing.T) {
	opts := testOptions(t)
	runner := opts.Runner.(*fakeRunner)
	runner.fail["/bin/openspec --version"] = errors.New("broken")
	checks, ok := Doctor(context.Background(), opts)
	if ok {
		t.Fatalf("doctor unexpectedly passed: %+v", checks)
	}
	for _, check := range checks {
		if check.Name == "openspec-health" && check.Status == "error" {
			return
		}
	}
	t.Fatalf("OpenSpec health error missing: %+v", checks)
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

func TestEmbeddedAgentGraphAndSkillsStayRoleIsolated(t *testing.T) {
	agents := map[string]string{}
	if err := fs.WalkDir(assets, "assets/opencode/agents", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := assets.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		agents[strings.TrimSuffix(filepath.Base(path), ".md")] = string(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(agents) != 5 {
		t.Fatalf("agents = %d, want 5", len(agents))
	}
	conductor := agents["eg"]
	for _, allowed := range []string{"eg-questioner: allow", "eg-planner: allow", "eg-executor: allow", "eg-qa: allow"} {
		if !strings.Contains(conductor, allowed) {
			t.Errorf("conductor missing delegation %q", allowed)
		}
	}
	if !strings.Contains(conductor, "question: deny") || !strings.Contains(conductor, "edit: deny") {
		t.Error("conductor must not edit or ask questions")
	}
	if !strings.Contains(agents["eg-questioner"], "question: allow") || !strings.Contains(agents["eg-questioner"], "edit: deny") {
		t.Error("questioner must be the non-editing question owner")
	}
	for _, name := range []string{"eg-planner", "eg-executor", "eg-qa"} {
		if !strings.Contains(agents[name], "question: deny") || !strings.Contains(agents[name], "task: deny") {
			t.Errorf("%s must return blockers instead of questioning or delegating", name)
		}
	}
	if !strings.Contains(agents["eg-executor"], `"git push*": deny`) || !strings.Contains(agents["eg-qa"], `"**/.elgordo/changes/*/qa.md": allow`) {
		t.Error("executor push and QA product edit boundaries must remain denied")
	}

	legacy := []string{"eg-intake", "eg-plan", "eg-execute", "eg-qa"}
	skills := map[string]bool{}
	if err := fs.WalkDir(assets, "assets/opencode/skills", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Base(path) != "SKILL.md" {
			return nil
		}
		data, readErr := assets.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		name := filepath.Base(filepath.Dir(path))
		skills[name] = true
		for _, expected := range []string{"name: " + name, "description:", "license: MIT", "author: Miguelecf"} {
			if !bytes.Contains(data, []byte(expected)) {
				t.Errorf("skill %s is missing %q", name, expected)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(skills) != 19 {
		t.Fatalf("skills = %d, want 19", len(skills))
	}
	for _, name := range legacy {
		if skills[name] {
			t.Errorf("legacy phase skill %q remains", name)
		}
	}
	for _, name := range []string{"eg-sdd-init", "eg-tdd-cycle", "eg-clean-code", "eg-atomic-commits", "eg-gherkin-verification", "eg-bounded-autopilot"} {
		if !skills[name] {
			t.Errorf("required atomic skill %q is missing", name)
		}
	}
	command, err := assets.ReadFile("assets/opencode/commands/eg.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(command, []byte("agent: eg")) || bytes.Contains(command, []byte("/opsx-")) {
		t.Error("/eg must remain the sole entrypoint and target the conductor")
	}
}

func TestEveryAgentReferenceResolvesAndEverySkillHasAConsumer(t *testing.T) {
	agents := map[string]string{}
	if err := fs.WalkDir(assets, "assets/opencode/agents", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := assets.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		agents[strings.TrimSuffix(filepath.Base(path), ".md")] = string(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	skills := map[string]bool{}
	if err := fs.WalkDir(assets, "assets/opencode/skills", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Base(path) != "SKILL.md" {
			return nil
		}
		skills[filepath.Base(filepath.Dir(path))] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	consumers := map[string][]string{}
	for agent, content := range agents {
		for _, reference := range permissionReferences(content, "task") {
			if _, ok := agents[reference]; !ok {
				t.Errorf("agent %s delegates to missing agent %s", agent, reference)
			}
		}
		for _, reference := range permissionReferences(content, "skill") {
			if !skills[reference] {
				t.Errorf("agent %s allows missing skill %s", agent, reference)
			}
			consumers[reference] = append(consumers[reference], agent)
		}
	}
	for skill := range skills {
		if len(consumers[skill]) == 0 {
			t.Errorf("skill %s has no agent consumer", skill)
		}
	}
}

func permissionReferences(content, section string) []string {
	var references []string
	inSection := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == section+":" {
			inSection = true
			continue
		}
		if inSection && len(line) > 0 && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			inSection = false
		}
		if !inSection || !strings.HasSuffix(trimmed, ": allow") {
			continue
		}
		name := strings.TrimSuffix(trimmed, ": allow")
		if name != "*" && !strings.ContainsAny(name, "*?") {
			references = append(references, name)
		}
	}
	return references
}
