package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type sealManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type sealManifest struct {
	Version  int                `json:"version"`
	ChangeID string             `json:"change_id"`
	Files    []sealManifestFile `json:"files"`
}

func (s *Store) buildSealManifest(state *State) (sealManifest, string, error) {
	manifestPath := fmt.Sprintf("plans/%04d.contract.json", state.Plan.Revision)
	manifest := sealManifest{
		Version:  1,
		ChangeID: state.ChangeID,
		Files: []sealManifestFile{{
			Path: ".elgordo/changes/" + state.ChangeID + "/" + state.Plan.Path,
		}},
	}
	if err := hashManifestFile(&manifest.Files[0], s.changeDir(state.ChangeID), state.Plan.Path); err != nil {
		return sealManifest{}, "", err
	}
	changeRoot := filepath.Join(s.Root, "openspec", "changes", state.ChangeID)
	if _, err := os.Stat(changeRoot); err != nil {
		return sealManifest{}, "", fmt.Errorf("OpenSpec change %q is missing; create proposal, specs, design, and tasks before sealing", state.ChangeID)
	}
	var openspecFiles []string
	err := filepath.WalkDir(changeRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(changeRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "" || strings.HasPrefix(rel, "../") || rel == ".." {
			return fmt.Errorf("OpenSpec artifact %q escapes the change directory", rel)
		}
		if strings.HasSuffix(rel, ".md") {
			openspecFiles = append(openspecFiles, rel)
		}
		return nil
	})
	if err != nil {
		return sealManifest{}, "", err
	}
	sort.Strings(openspecFiles)
	if len(openspecFiles) == 0 {
		return sealManifest{}, "", fmt.Errorf("OpenSpec change %q has no Markdown artifacts to seal", state.ChangeID)
	}
	for _, rel := range openspecFiles {
		entry := sealManifestFile{Path: "openspec/changes/" + state.ChangeID + "/" + rel}
		if err := hashManifestFile(&entry, changeRoot, rel); err != nil {
			return sealManifest{}, "", err
		}
		manifest.Files = append(manifest.Files, entry)
	}
	return manifest, manifestPath, nil
}

func hashManifestFile(entry *sealManifestFile, root, rel string) error {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	entry.SHA256 = "sha256:" + hex.EncodeToString(sum[:])
	return nil
}

func marshalManifest(manifest sealManifest) ([]byte, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
