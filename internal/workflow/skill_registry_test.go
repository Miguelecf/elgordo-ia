package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitCreatesDeterministicSkillRegistry(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	projectSkills := filepath.Join(root, "project-skills")
	userSkills := filepath.Join(root, "user-skills")
	writeTestSkill(t, projectSkills, "alpha", "'alpha'", "'Project description | preferred'")
	writeTestSkill(t, userSkills, "alpha", "alpha", "User duplicate")
	writeTestSkill(t, userSkills, "beta", "beta", ">\n  Folded description\n  on two lines")
	writeTestSkill(t, userSkills, "sdd-plan", "sdd-plan", "Excluded SDD skill")
	writeTestSkill(t, userSkills, "registry", "skill-registry", "Excluded registry")

	store := NewStore(root, "test")
	store.Now = func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) }
	store.skillSources = []skillSource{{projectSkills, "project"}, {userSkills, "user"}, {filepath.Join(root, "missing"), "user"}}
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, ".atl", "skill-registry.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	registry := string(data)
	for _, expected := range []string{
		"# Skill Registry — " + filepath.Base(root),
		"Last updated: 2026-08-16",
		"| `alpha` | Project description \\| preferred | project |",
		"| `beta` | Folded description on two lines | user |",
		"`elgordo skill-registry refresh --force`",
	} {
		if !strings.Contains(registry, expected) {
			t.Fatalf("registry missing %q:\n%s", expected, registry)
		}
	}
	if strings.Contains(registry, "User duplicate") || strings.Contains(registry, "sdd-plan") || strings.Contains(registry, "skill-registry` |") {
		t.Fatalf("registry contains excluded or duplicate skill:\n%s", registry)
	}
	if strings.Index(registry, "`alpha`") > strings.Index(registry, "`beta`") {
		t.Fatalf("registry is not sorted:\n%s", registry)
	}

	result, err := store.RefreshSkillRegistry(false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated || result.Count != 2 {
		t.Fatalf("unchanged refresh = %+v", result)
	}
	result, err = store.RefreshSkillRegistry(true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Updated || result.Count != 2 {
		t.Fatalf("forced refresh = %+v", result)
	}
}

func writeTestSkill(t *testing.T, root, dir, name, description string) {
	t.Helper()
	path := filepath.Join(root, dir, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n# Skill\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
