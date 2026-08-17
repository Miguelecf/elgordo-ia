package installer

import (
	"bufio"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//go:embed all:assets/opencode
var assets embed.FS

type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, args ...string) error
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecRunner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (r ExecRunner) LookPath(file string) (string, error) { return exec.LookPath(file) }

func (r ExecRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = r.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	return cmd.Run()
}

func (r ExecRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

type Options struct {
	Home                  string
	Version               string
	AcceptEngramInstall   bool
	AcceptOpenSpecInstall bool
	OverwriteUserAssets   bool
	Stdin                 io.Reader
	Stdout                io.Writer
	Stderr                io.Writer
	Runner                Runner
	Now                   func() time.Time
}

type Manifest struct {
	Version     string            `json:"version"`
	InstalledAt time.Time         `json:"installed_at"`
	Files       map[string]string `json:"files"`
}

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type assetUpdate struct {
	rel     string
	target  string
	newData []byte
	oldData []byte
	mode    fs.FileMode
	existed bool
	remove  bool
}

func DefaultOptions(version string, stdin io.Reader, stdout, stderr io.Writer) (Options, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Options{}, err
	}
	runner := ExecRunner{Stdin: stdin, Stdout: stdout, Stderr: stderr}
	return Options{
		Home:    home,
		Version: version,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Runner:  runner,
		Now:     func() time.Time { return time.Now().UTC() },
	}, nil
}

func Install(ctx context.Context, opts Options) error {
	opts = normalizeOptions(opts)
	if err := ensureSupportedPlatform(); err != nil {
		return err
	}
	if _, err := opts.Runner.LookPath("opencode"); err != nil {
		return errors.New("OpenCode is required; install it from https://opencode.ai/docs before continuing")
	}
	if err := withGlobalLock(opts, func() error {
		if err := ensureEngram(ctx, opts); err != nil {
			return err
		}
		if err := ensureOpenSpec(ctx, opts); err != nil {
			return err
		}
		if err := installAssetsUnlocked(opts); err != nil {
			return err
		}
		if err := configureContext7Unlocked(opts); err != nil {
			fmt.Fprintf(opts.Stderr, "warning: Context7 was not configured automatically: %v\n", err)
			fmt.Fprintln(opts.Stderr, "Add the Context7 remote MCP server manually, then run elgordo doctor.")
		}
		return nil
	}); err != nil {
		return err
	}
	fmt.Fprintln(opts.Stdout, "Engram and OpenSpec verified.")
	fmt.Fprintln(opts.Stdout, "ElGordo assets installed globally for OpenCode.")
	fmt.Fprintln(opts.Stdout, "Restart OpenCode, open a Git repository, run `elgordo init`, then use `/eg`.")
	return nil
}

func Sync(opts Options) error {
	opts = normalizeOptions(opts)
	if err := withGlobalLock(opts, func() error {
		if err := installAssetsUnlocked(opts); err != nil {
			return err
		}
		if err := configureContext7Unlocked(opts); err != nil {
			fmt.Fprintf(opts.Stderr, "warning: Context7 was not synchronized: %v\n", err)
		}
		return nil
	}); err != nil {
		return err
	}
	fmt.Fprintln(opts.Stdout, "ElGordo managed assets synchronized.")
	return nil
}

func InstallAssets(opts Options) error {
	opts = normalizeOptions(opts)
	return withGlobalLock(opts, func() error { return installAssetsUnlocked(opts) })
}

func installAssetsUnlocked(opts Options) error {
	configRoot := filepath.Join(opts.Home, ".config", "opencode")
	manifest, err := loadManifest(configRoot)
	if err != nil {
		return fmt.Errorf("load managed asset manifest: %w", err)
	}
	newManifest := Manifest{Version: opts.Version, InstalledAt: opts.Now(), Files: map[string]string{}}
	var paths []string
	err = fs.WalkDir(assets, "assets/opencode", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(paths)
	var updates []assetUpdate
	for _, source := range paths {
		rel := strings.TrimPrefix(source, "assets/opencode/")
		target, err := managedPath(configRoot, rel)
		if err != nil {
			return err
		}
		content, err := assets.ReadFile(source)
		if err != nil {
			return err
		}
		newHash := hashBytes(content)
		newManifest.Files[rel] = newHash
		current, err := os.ReadFile(target)
		if err == nil {
			currentHash := hashBytes(current)
			if currentHash == newHash {
				continue
			}
			ownedHash := ""
			if manifest != nil {
				ownedHash = manifest.Files[rel]
			}
			if ownedHash != currentHash && !opts.OverwriteUserAssets {
				ok, err := confirm(opts, fmt.Sprintf("Overwrite modified OpenCode asset %s?", target))
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("installation cancelled before overwriting %s", target)
				}
			}
			mode := fs.FileMode(0o644)
			if info, statErr := os.Stat(target); statErr == nil {
				mode = info.Mode().Perm()
			}
			updates = append(updates, assetUpdate{rel: rel, target: target, newData: content, oldData: current, mode: mode, existed: true})
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		} else {
			updates = append(updates, assetUpdate{rel: rel, target: target, newData: content, mode: 0o644})
		}
	}
	if manifest != nil {
		for rel, oldHash := range manifest.Files {
			if _, stillManaged := newManifest.Files[rel]; stillManaged {
				continue
			}
			target, err := managedPath(configRoot, rel)
			if err != nil {
				return fmt.Errorf("unsafe previous manifest path %q: %w", rel, err)
			}
			current, err := os.ReadFile(target)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			if hashBytes(current) != oldHash {
				fmt.Fprintf(opts.Stderr, "preserved obsolete modified asset: %s\n", target)
				continue
			}
			mode := fs.FileMode(0o644)
			if info, statErr := os.Stat(target); statErr == nil {
				mode = info.Mode().Perm()
			}
			updates = append(updates, assetUpdate{rel: rel, target: target, oldData: current, mode: mode, existed: true, remove: true})
		}
	}
	for _, update := range updates {
		if update.existed {
			if err := backupFile(opts, update.rel, update.oldData, update.mode); err != nil {
				return err
			}
		}
	}
	var applied []assetUpdate
	rollback := func() {
		for i := len(applied) - 1; i >= 0; i-- {
			update := applied[i]
			if update.existed {
				_ = writeFileAtomic(update.target, update.oldData, update.mode)
			} else {
				_ = os.Remove(update.target)
			}
		}
	}
	for _, update := range updates {
		if update.remove {
			err = os.Remove(update.target)
		} else {
			err = writeFileAtomic(update.target, update.newData, 0o644)
		}
		if err != nil {
			rollback()
			return err
		}
		applied = append(applied, update)
	}
	if err := writeJSONAtomic(filepath.Join(configRoot, "elgordo", "manifest.json"), newManifest); err != nil {
		rollback()
		return err
	}
	return nil
}

func ConfigureContext7(opts Options) error {
	opts = normalizeOptions(opts)
	return withGlobalLock(opts, func() error { return configureContext7Unlocked(opts) })
}

func configureContext7Unlocked(opts Options) error {
	path := filepath.Join(opts.Home, ".config", "opencode", "opencode.json")
	config := map[string]any{"$schema": "https://opencode.ai/config.json"}
	mode := fs.FileMode(0o600)
	var original []byte
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("existing %s is not strict JSON: %w", path, err)
		}
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		}
		original = data
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	mcp, ok := config["mcp"].(map[string]any)
	if !ok {
		mcp = map[string]any{}
		config["mcp"] = mcp
	}
	if _, exists := mcp["context7"]; exists {
		return nil
	}
	if original != nil {
		if err := backupFile(opts, "opencode.json", original, mode); err != nil {
			return fmt.Errorf("backup OpenCode config: %w", err)
		}
	}
	mcp["context7"] = map[string]any{
		"type":    "remote",
		"url":     "https://mcp.context7.com/mcp",
		"enabled": true,
	}
	return writeJSONAtomicMode(path, config, mode)
}

func Uninstall(opts Options) error {
	opts = normalizeOptions(opts)
	return withGlobalLock(opts, func() error { return uninstallUnlocked(opts) })
}

func uninstallUnlocked(opts Options) error {
	configRoot := filepath.Join(opts.Home, ".config", "opencode")
	manifest, err := loadManifest(configRoot)
	if err != nil {
		return err
	}
	if manifest == nil {
		return errors.New("ElGordo manifest not found; nothing can be safely removed")
	}
	var preserved []string
	for rel, expectedHash := range manifest.Files {
		path, err := managedPath(configRoot, rel)
		if err != nil {
			return fmt.Errorf("unsafe manifest path %q: %w", rel, err)
		}
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if hashBytes(data) != expectedHash {
			preserved = append(preserved, path)
			continue
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	if err := os.Remove(filepath.Join(configRoot, "elgordo", "manifest.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, path := range preserved {
		fmt.Fprintf(opts.Stderr, "preserved modified asset: %s\n", path)
	}
	fmt.Fprintln(opts.Stdout, "ElGordo managed assets removed. Engram and Context7 configuration were preserved.")
	return nil
}

func Doctor(ctx context.Context, opts Options) ([]Check, bool) {
	opts = normalizeOptions(opts)
	checks := []Check{}
	ok := true
	for _, command := range []string{"git", "opencode", "engram", "openspec"} {
		if path, err := opts.Runner.LookPath(command); err == nil {
			checks = append(checks, Check{Name: command, Status: "ok", Message: path})
		} else {
			checks = append(checks, Check{Name: command, Status: "error", Message: "not found in PATH"})
			ok = false
		}
	}
	if _, err := opts.Runner.LookPath("node"); err != nil {
		checks = append(checks, Check{Name: "node", Status: "error", Message: "Node.js >=20.19.0 is required; not found in PATH"})
		ok = false
	} else if output, err := opts.Runner.Output(ctx, "node", "--version"); err != nil {
		checks = append(checks, Check{Name: "node", Status: "error", Message: fmt.Sprintf("could not determine version: %v", err)})
		ok = false
	} else if !nodeVersionSupported(string(output)) {
		checks = append(checks, Check{Name: "node", Status: "error", Message: fmt.Sprintf("Node.js >=20.19.0 is required; found %q", strings.TrimSpace(string(output)))})
		ok = false
	} else {
		checks = append(checks, Check{Name: "node", Status: "ok", Message: strings.TrimSpace(string(output))})
	}
	if openSpecPath, err := opts.Runner.LookPath("openspec"); err == nil {
		if err := opts.Runner.Run(ctx, openSpecPath, "--version"); err != nil {
			checks = append(checks, Check{Name: "openspec-health", Status: "error", Message: err.Error()})
			ok = false
		} else {
			checks = append(checks, Check{Name: "openspec-health", Status: "ok", Message: "version command completed"})
		}
	}
	if _, err := opts.Runner.LookPath("engram"); err == nil {
		if err := opts.Runner.Run(ctx, "engram", "doctor", "--json"); err != nil {
			checks = append(checks, Check{Name: "engram-health", Status: "error", Message: err.Error()})
			ok = false
		} else {
			checks = append(checks, Check{Name: "engram-health", Status: "ok", Message: "doctor completed"})
		}
	}
	configRoot := filepath.Join(opts.Home, ".config", "opencode")
	manifest, err := loadManifest(configRoot)
	if err != nil || manifest == nil {
		checks = append(checks, Check{Name: "elgordo-assets", Status: "error", Message: "manifest missing or invalid"})
		ok = false
	} else {
		assetErr := verifyManagedAssets(configRoot, manifest)
		if assetErr != nil {
			checks = append(checks, Check{Name: "elgordo-assets", Status: "error", Message: assetErr.Error()})
			ok = false
		} else {
			checks = append(checks, Check{Name: "elgordo-assets", Status: "ok", Message: "version " + manifest.Version})
		}
	}
	if context7Configured(opts.Home) {
		checks = append(checks, Check{Name: "context7", Status: "ok", Message: "configured"})
	} else {
		checks = append(checks, Check{Name: "context7", Status: "warning", Message: "not configured; library documentation lookup is unavailable"})
	}
	return checks, ok
}

func ensureEngram(ctx context.Context, opts Options) error {
	engramCommand, lookupErr := opts.Runner.LookPath("engram")
	if lookupErr != nil {
		approved := opts.AcceptEngramInstall
		if !approved {
			var confirmErr error
			approved, confirmErr = confirm(opts, "Engram is required for persistent project memory. Install it now?")
			if confirmErr != nil {
				return confirmErr
			}
		}
		if !approved {
			return errors.New("Engram installation declined; install it from https://github.com/Gentleman-Programming/engram and retry")
		}
		installedCommand, err := installEngram(ctx, opts)
		if err != nil {
			return err
		}
		engramCommand = installedCommand
	}
	if err := opts.Runner.Run(ctx, engramCommand, "setup", "opencode"); err != nil {
		return fmt.Errorf("engram setup opencode failed: %w", err)
	}
	if err := opts.Runner.Run(ctx, engramCommand, "doctor", "--json"); err != nil {
		return fmt.Errorf("engram doctor failed: %w", err)
	}
	return nil
}

func ensureOpenSpec(ctx context.Context, opts Options) error {
	if _, err := opts.Runner.LookPath("node"); err != nil {
		return errors.New("Node.js >=20.19.0 is required for OpenSpec; ElGordo will not install Node automatically. Install it from https://nodejs.org/ and retry")
	}
	output, err := opts.Runner.Output(ctx, "node", "--version")
	if err != nil {
		return fmt.Errorf("could not determine Node.js version for OpenSpec: %w", err)
	}
	if !nodeVersionSupported(string(output)) {
		return fmt.Errorf("Node.js >=20.19.0 is required for OpenSpec; found %q. Upgrade Node from https://nodejs.org/ and retry", strings.TrimSpace(string(output)))
	}

	openSpecCommand, err := opts.Runner.LookPath("openspec")
	if err == nil {
		if err := opts.Runner.Run(ctx, openSpecCommand, "--version"); err != nil {
			return fmt.Errorf("openspec --version failed: %w", err)
		}
		return nil
	}
	approved := opts.AcceptOpenSpecInstall
	if !approved {
		approved, err = confirm(opts, "OpenSpec is required for versioned specifications. Install @fission-ai/openspec now with npm?")
		if err != nil {
			return err
		}
	}
	if !approved {
		return errors.New("OpenSpec installation declined; install it manually with `npm install -g @fission-ai/openspec@1.5.0` and retry")
	}
	if _, err := opts.Runner.LookPath("npm"); err != nil {
		return errors.New("npm is required to install OpenSpec; install it manually with `npm install -g @fission-ai/openspec@1.5.0` after installing npm")
	}
	if err := opts.Runner.Run(ctx, "npm", "install", "-g", "@fission-ai/openspec@1.5.0"); err != nil {
		return fmt.Errorf("npm could not install OpenSpec: %w", err)
	}
	openSpecCommand, err = opts.Runner.LookPath("openspec")
	if err != nil {
		return errors.New("OpenSpec was installed but is not in PATH; add npm's global bin directory to PATH and retry")
	}
	if err := opts.Runner.Run(ctx, openSpecCommand, "--version"); err != nil {
		return fmt.Errorf("openspec --version failed after installation: %w", err)
	}
	return nil
}

func nodeVersionSupported(value string) bool {
	version := strings.TrimSpace(value)
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	numbers := [3]int{}
	for i, part := range parts {
		if part == "" {
			return false
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return false
			}
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return false
		}
		numbers[i] = number
	}
	return numbers[0] > 20 || (numbers[0] == 20 && (numbers[1] > 19 || (numbers[1] == 19 && numbers[2] >= 0)))
}

func installEngram(ctx context.Context, opts Options) (string, error) {
	if _, err := opts.Runner.LookPath("brew"); err == nil {
		if err := opts.Runner.Run(ctx, "brew", "install", "gentleman-programming/tap/engram"); err != nil {
			return "", fmt.Errorf("Homebrew could not install Engram: %w", err)
		}
		if path, err := opts.Runner.LookPath("engram"); err == nil {
			return path, nil
		}
		return "engram", nil
	}
	if _, err := opts.Runner.LookPath("go"); err == nil {
		if err := opts.Runner.Run(ctx, "go", "install", "github.com/Gentleman-Programming/engram/cmd/engram@v1.20.0"); err != nil {
			return "", fmt.Errorf("go install could not install Engram: %w", err)
		}
		if path, err := opts.Runner.LookPath("engram"); err == nil {
			return path, nil
		}
		candidates := []string{}
		if gobin := os.Getenv("GOBIN"); gobin != "" {
			candidates = append(candidates, filepath.Join(gobin, "engram"))
		}
		candidates = append(candidates, filepath.Join(opts.Home, "go", "bin", "engram"))
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
		return "", errors.New("Engram was built but its binary could not be located; add Go's bin directory to PATH and retry")
	}
	return "", errors.New("cannot install Engram automatically: neither Homebrew nor Go is available; follow https://github.com/Gentleman-Programming/engram/blob/main/docs/INSTALLATION.md")
}

func ensureSupportedPlatform() error {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported platform %s; v0.1.0 supports macOS, Linux, and WSL", runtime.GOOS)
	}
	return nil
}

func normalizeOptions(opts Options) Options {
	if opts.Stdin == nil {
		opts.Stdin = strings.NewReader("")
	}
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Runner == nil {
		opts.Runner = ExecRunner{Stdin: opts.Stdin, Stdout: opts.Stdout, Stderr: opts.Stderr}
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	return opts
}

func confirm(opts Options, prompt string) (bool, error) {
	fmt.Fprintf(opts.Stdout, "%s [y/N] ", prompt)
	line, err := bufio.NewReader(opts.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes" || answer == "s" || answer == "si" || answer == "sí", nil
}

func backupFile(opts Options, rel string, content []byte, mode fs.FileMode) error {
	stamp := opts.Now().Format("20060102T150405.000000000Z")
	path := filepath.Join(opts.Home, ".config", "elgordo", "backups", stamp, filepath.FromSlash(rel))
	return writeFileAtomic(path, content, mode)
}

func loadManifest(configRoot string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(configRoot, "elgordo", "manifest.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func context7Configured(home string) bool {
	data, err := os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.json"))
	if err != nil {
		return false
	}
	var config map[string]any
	if json.Unmarshal(data, &config) != nil {
		return false
	}
	mcp, ok := config["mcp"].(map[string]any)
	if !ok {
		return false
	}
	entry, ok := mcp["context7"].(map[string]any)
	if !ok {
		return false
	}
	typeValue, _ := entry["type"].(string)
	url, _ := entry["url"].(string)
	enabled, _ := entry["enabled"].(bool)
	return typeValue == "remote" && url == "https://mcp.context7.com/mcp" && enabled
}

func verifyManagedAssets(configRoot string, manifest *Manifest) error {
	expected := map[string]string{}
	err := fs.WalkDir(assets, "assets/opencode", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, readErr := assets.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel := strings.TrimPrefix(path, "assets/opencode/")
		expected[rel] = hashBytes(content)
		return nil
	})
	if err != nil {
		return err
	}
	for rel, expectedHash := range expected {
		if manifest.Files[rel] != expectedHash {
			return fmt.Errorf("manifest is stale for %s", rel)
		}
		path, err := managedPath(configRoot, rel)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("managed asset %s is missing: %w", rel, err)
		}
		if hashBytes(content) != expectedHash {
			return fmt.Errorf("managed asset %s was modified", rel)
		}
	}
	for rel := range manifest.Files {
		if _, exists := expected[rel]; !exists {
			return fmt.Errorf("manifest contains obsolete asset %s", rel)
		}
	}
	return nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func writeJSONAtomic(path string, value any) error {
	return writeJSONAtomicMode(path, value, 0o644)
}

func writeJSONAtomicMode(path string, value any, mode fs.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(data, '\n'), mode)
}

func managedPath(root, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) || strings.Contains(rel, "\\") {
		return "", errors.New("path must be a non-empty relative slash path")
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes the managed root")
	}
	allowed := false
	for _, prefix := range []string{"agents" + string(filepath.Separator), "commands" + string(filepath.Separator), "skills" + string(filepath.Separator)} {
		if strings.HasPrefix(clean, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", errors.New("path is outside managed asset prefixes")
	}
	target := filepath.Join(root, clean)
	relToRoot, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) || relToRoot == ".." {
		return "", errors.New("resolved path escapes the managed root")
	}
	current := root
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("managed path contains a symbolic link")
		}
	}
	return target, nil
}

func withGlobalLock(opts Options, fn func() error) error {
	lock := filepath.Join(opts.Home, ".config", "elgordo", ".install-lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(lock, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) || !reclaimDeadGlobalLock(lock) {
			return errors.New("another ElGordo install, sync, or uninstall operation is in progress")
		}
		if err := os.Mkdir(lock, 0o700); err != nil {
			return err
		}
	}
	owner := map[string]any{"pid": os.Getpid(), "acquired_at": time.Now().UTC()}
	if err := writeJSONAtomic(filepath.Join(lock, "owner.json"), owner); err != nil {
		os.RemoveAll(lock)
		return err
	}
	defer os.RemoveAll(lock)
	return fn()
}

func reclaimDeadGlobalLock(lock string) bool {
	data, err := os.ReadFile(filepath.Join(lock, "owner.json"))
	if err != nil {
		return false
	}
	var owner struct {
		PID int `json:"pid"`
	}
	if json.Unmarshal(data, &owner) != nil || owner.PID <= 0 {
		return false
	}
	if !errors.Is(syscall.Kill(owner.PID, 0), syscall.ESRCH) {
		return false
	}
	return os.RemoveAll(lock) == nil
}

func writeFileAtomic(path string, content []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".elgordo-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(content); err != nil {
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
	return os.Rename(name, path)
}
