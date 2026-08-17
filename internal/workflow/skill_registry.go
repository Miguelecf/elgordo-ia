package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type skillSource struct {
	path  string
	scope string
}

type skillRegistryEntry struct {
	name        string
	description string
	scope       string
	path        string
}

type SkillRegistryResult struct {
	Count   int
	Updated bool
}

func (s *Store) RefreshSkillRegistry(force bool) (SkillRegistryResult, error) {
	sources, err := s.registrySources()
	if err != nil {
		return SkillRegistryResult{}, err
	}
	entries, scanned, err := discoverSkills(sources)
	if err != nil {
		return SkillRegistryResult{}, err
	}
	content := renderSkillRegistry(filepath.Base(s.Root), s.Now().Local(), scanned, entries)
	path := filepath.Join(s.Root, ".atl", "skill-registry.md")
	if !force {
		current, err := os.ReadFile(path)
		if err == nil && string(current) == content {
			return SkillRegistryResult{Count: len(entries)}, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return SkillRegistryResult{}, err
		}
	}
	if err := writeFileAtomic(path, []byte(content), 0o644); err != nil {
		return SkillRegistryResult{}, err
	}
	return SkillRegistryResult{Count: len(entries), Updated: true}, nil
}

func (s *Store) registrySources() ([]skillSource, error) {
	if s.skillSources != nil {
		return s.skillSources, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory for skill registry: %w", err)
	}
	return []skillSource{
		{filepath.Join(s.Root, ".agents", "skills"), "project"},
		{filepath.Join(s.Root, ".opencode", "skills"), "project"},
		{filepath.Join(s.Root, ".codex", "skills"), "project"},
		{filepath.Join(s.Root, ".kiro", "skills"), "project"},
		{filepath.Join(home, ".agents", "skills"), "user"},
		{filepath.Join(home, ".config", "opencode", "skills"), "user"},
		{filepath.Join(home, ".codex", "skills"), "user"},
		{filepath.Join(home, ".kiro", "skills"), "user"},
	}, nil
}

func discoverSkills(sources []skillSource) ([]skillRegistryEntry, []string, error) {
	seen := make(map[string]bool)
	var entries []skillRegistryEntry
	var scanned []string
	for _, source := range sources {
		dirs, err := os.ReadDir(source.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("scan skills in %s: %w", source.path, err)
		}
		scanned = append(scanned, source.path)
		for _, dir := range dirs {
			path := filepath.Join(source.path, dir.Name(), "SKILL.md")
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			name, description, err := readSkillMetadata(path)
			if err != nil || name == "" || description == "" || name == "_shared" || name == "skill-registry" || strings.HasPrefix(name, "sdd-") || seen[name] {
				continue
			}
			absolute, err := filepath.Abs(path)
			if err != nil {
				return nil, nil, err
			}
			seen[name] = true
			entries = append(entries, skillRegistryEntry{name: name, description: description, scope: source.scope, path: absolute})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries, scanned, nil
}

func readSkillMetadata(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", errors.New("skill is missing frontmatter")
	}
	var name, description string
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "---" {
			return name, description, nil
		}
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || (strings.TrimSpace(key) != "name" && strings.TrimSpace(key) != "description") {
			continue
		}
		value = strings.TrimSpace(value)
		if value == ">" || value == "|" || value == ">-" || value == "|-" {
			var block []string
			for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.TrimSpace(lines[i+1]) == "") {
				i++
				if text := strings.TrimSpace(lines[i]); text != "" {
					block = append(block, text)
				}
			}
			value = strings.Join(block, " ")
		} else {
			value = parseYAMLScalar(value)
		}
		switch strings.TrimSpace(key) {
		case "name":
			name = value
		case "description":
			description = value
		}
	}
	return "", "", errors.New("unterminated skill frontmatter")
}

func parseYAMLScalar(value string) string {
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
	}
	return value
}

func renderSkillRegistry(project string, now time.Time, sources []string, entries []skillRegistryEntry) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Skill Registry — %s\n\n", project)
	out.WriteString("<!-- Auto-generated by elgordo skill-registry refresh. Run `elgordo skill-registry refresh --force` to regenerate. -->\n\n")
	fmt.Fprintf(&out, "Last updated: %s\n\n## Sources scanned\n\n", now.Format("2006-01-02"))
	if len(sources) == 0 {
		out.WriteString("- None\n")
	} else {
		for _, source := range sources {
			fmt.Fprintf(&out, "- %s\n", source)
		}
	}
	out.WriteString("\n## Contract\n\n**Delegator use only.** This registry is an index, not a summary. Any agent that launches subagents reads it to select relevant skills, then passes exact `SKILL.md` paths for the subagent to read before work.\n\n")
	out.WriteString("`SKILL.md` remains the source of truth. Do not inject generated summaries or compact rules by default; pass paths so subagents load the full runtime contract and preserve author intent.\n\n")
	out.WriteString("## Skills\n\n| Skill | Trigger / description | Scope | Path |\n| --- | --- | --- | --- |\n")
	for _, entry := range entries {
		fmt.Fprintf(&out, "| `%s` | %s | %s | `%s` |\n", markdownCell(entry.name), markdownCell(entry.description), entry.scope, markdownCell(entry.path))
	}
	out.WriteString("\n## Loading protocol\n\n1. Match task context and target files against the `Trigger / description` column.\n2. Pass only the matching `Path` values to the subagent under `## Skills to load before work`.\n3. Instruct the subagent to read those exact `SKILL.md` files before reading, writing, reviewing, testing, or creating artifacts.\n4. If no matching skill exists, proceed without project skill injection and report `skill_resolution: none`.\n")
	return out.String()
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}
