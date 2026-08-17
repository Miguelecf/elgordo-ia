package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Miguelecf/elgordo-ia/internal/installer"
	"github.com/Miguelecf/elgordo-ia/internal/workflow"
)

type Dependencies struct {
	Version string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Getwd   func() (string, error)
}

type app struct {
	deps Dependencies
}

func Run(args []string, deps Dependencies) int {
	if deps.Version == "" {
		deps.Version = "dev"
	}
	if deps.Stdin == nil {
		deps.Stdin = os.Stdin
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	if deps.Getwd == nil {
		deps.Getwd = os.Getwd
	}
	a := app{deps: deps}
	if err := a.run(args); err != nil {
		fmt.Fprintln(deps.Stderr, "error:", err)
		return 1
	}
	return 0
}

func (a app) run(args []string) error {
	if len(args) == 0 {
		a.usage()
		return nil
	}
	switch args[0] {
	case "help", "-h", "--help":
		a.usage()
		return nil
	case "version", "--version":
		fmt.Fprintf(a.deps.Stdout, "elgordo %s\n", a.deps.Version)
		return nil
	case "install":
		return a.install(args[1:])
	case "sync":
		return a.sync(args[1:])
	case "doctor":
		return a.doctor(args[1:])
	case "uninstall":
		return a.uninstall(args[1:])
	case "init":
		return a.initProject(args[1:])
	case "skill-registry":
		return a.skillRegistry(args[1:])
	case "status":
		return a.status(args[1:])
	case "change":
		return a.change(args[1:])
	case "plan":
		return a.plan(args[1:])
	case "execution":
		return a.execution(args[1:])
	case "code":
		return a.code(args[1:])
	case "qa":
		return a.qa(args[1:])
	case "final":
		return a.final(args[1:])
	default:
		return fmt.Errorf("unknown command %q; run elgordo help", args[0])
	}
}

func (a app) install(args []string) error {
	fs := newFlagSet("install", a.deps.Stderr)
	acceptEngram := fs.Bool("accept-engram-install", false, "approve Engram installation when missing")
	overwriteAssets := fs.Bool("overwrite-assets", false, "approve replacement of user-modified managed assets")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: elgordo install [--accept-engram-install] [--overwrite-assets]")
	}
	opts, err := installer.DefaultOptions(a.deps.Version, a.deps.Stdin, a.deps.Stdout, a.deps.Stderr)
	if err != nil {
		return err
	}
	opts.AcceptEngramInstall = *acceptEngram
	opts.OverwriteUserAssets = *overwriteAssets
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return installer.Install(ctx, opts)
}

func (a app) sync(args []string) error {
	fs := newFlagSet("sync", a.deps.Stderr)
	overwriteAssets := fs.Bool("overwrite-assets", false, "approve replacement of user-modified managed assets")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: elgordo sync [--overwrite-assets]")
	}
	opts, err := installer.DefaultOptions(a.deps.Version, a.deps.Stdin, a.deps.Stdout, a.deps.Stderr)
	if err != nil {
		return err
	}
	opts.OverwriteUserAssets = *overwriteAssets
	return installer.Sync(opts)
}

func (a app) doctor(args []string) error {
	fs := newFlagSet("doctor", a.deps.Stderr)
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: elgordo doctor [--json]")
	}
	opts, err := installer.DefaultOptions(a.deps.Version, a.deps.Stdin, a.deps.Stdout, a.deps.Stderr)
	if err != nil {
		return err
	}
	if *jsonOutput {
		opts.Runner = installer.ExecRunner{Stdin: a.deps.Stdin, Stdout: io.Discard, Stderr: io.Discard}
	}
	checks, ok := installer.Doctor(context.Background(), opts)
	if *jsonOutput {
		if err := writeJSON(a.deps.Stdout, map[string]any{"ok": ok, "checks": checks}); err != nil {
			return err
		}
	} else {
		for _, check := range checks {
			fmt.Fprintf(a.deps.Stdout, "%-16s %-7s %s\n", check.Name, check.Status, check.Message)
		}
	}
	if !ok {
		return errors.New("required ElGordo checks failed")
	}
	return nil
}

func (a app) uninstall(args []string) error {
	fs := newFlagSet("uninstall", a.deps.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: elgordo uninstall")
	}
	opts, err := installer.DefaultOptions(a.deps.Version, a.deps.Stdin, a.deps.Stdout, a.deps.Stderr)
	if err != nil {
		return err
	}
	return installer.Uninstall(opts)
}

func (a app) initProject(args []string) error {
	fs := newFlagSet("init", a.deps.Stderr)
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: elgordo init [--json]")
	}
	store, err := a.store(false)
	if err != nil {
		return err
	}
	if err := store.Init(); err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSON(a.deps.Stdout, map[string]any{"initialized": true, "repository": store.Root})
	}
	fmt.Fprintf(a.deps.Stdout, "Initialized ElGordo in %s\n", store.Root)
	fmt.Fprintln(a.deps.Stdout, "Local workflow artifacts, Engram config, and the skill registry are excluded from Git.")
	return nil
}

func (a app) skillRegistry(args []string) error {
	if len(args) == 0 || args[0] != "refresh" {
		return errors.New("usage: elgordo skill-registry refresh [--force]")
	}
	fs := newFlagSet("skill-registry refresh", a.deps.Stderr)
	force := fs.Bool("force", false, "rewrite the registry even when unchanged")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: elgordo skill-registry refresh [--force]")
	}
	store, err := a.store(true)
	if err != nil {
		return err
	}
	result, err := store.RefreshSkillRegistry(*force)
	if err != nil {
		return err
	}
	status := "unchanged"
	if result.Updated {
		status = "updated"
	}
	fmt.Fprintf(a.deps.Stdout, "Skill registry %s: %s (%d skills)\n", status, filepath.Join(store.Root, ".atl", "skill-registry.md"), result.Count)
	return nil
}

func (a app) status(args []string) error {
	fs := newFlagSet("status", a.deps.Stderr)
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: elgordo status [--json]")
	}
	store, err := a.store(true)
	if err != nil {
		return err
	}
	state, err := store.ActiveState()
	if err != nil {
		if errors.Is(err, workflow.ErrNoActiveChange) {
			if *jsonOutput {
				return writeJSON(a.deps.Stdout, map[string]any{"initialized": true, "active_change": nil})
			}
			fmt.Fprintln(a.deps.Stdout, "No active change.")
			return nil
		}
		return err
	}
	if *jsonOutput {
		return writeJSON(a.deps.Stdout, state)
	}
	fmt.Fprintf(a.deps.Stdout, "Change: %s\nPhase: %s\nPlan: %s (revision %d)\n", state.ChangeID, state.Phase, state.Plan.Path, state.Plan.Revision)
	if state.Plan.SHA256 != "" {
		fmt.Fprintf(a.deps.Stdout, "Sealed: %s\n", state.Plan.SHA256)
	}
	return nil
}

func (a app) change(args []string) error {
	if len(args) == 0 || args[0] != "start" {
		return errors.New("usage: elgordo change start <slug> [--title <title>]")
	}
	fs := newFlagSet("change start", a.deps.Stderr)
	title := fs.String("title", "", "human-readable title")
	jsonOutput := fs.Bool("json", false, "print JSON")
	if len(args) < 2 || strings.HasPrefix(args[1], "-") {
		return errors.New("usage: elgordo change start <slug> [--title <title>]")
	}
	slug := args[1]
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: elgordo change start <slug> [--title <title>]")
	}
	store, err := a.store(true)
	if err != nil {
		return err
	}
	state, err := store.StartChange(slug, *title)
	if err != nil {
		return err
	}
	return a.printState(state, *jsonOutput)
}

func (a app) plan(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: elgordo plan <hash|ready|verify|seal|replan>")
	}
	store, err := a.store(true)
	if err != nil {
		return err
	}
	switch args[0] {
	case "hash":
		if len(args) != 1 {
			return errors.New("usage: elgordo plan hash")
		}
		hash, err := store.PlanHash()
		if err != nil {
			return err
		}
		fmt.Fprintln(a.deps.Stdout, hash)
		return nil
	case "verify":
		if len(args) != 1 {
			return errors.New("usage: elgordo plan verify")
		}
		if err := store.VerifyPlan(); err != nil {
			return err
		}
		fmt.Fprintln(a.deps.Stdout, "Plan seal is valid.")
		return nil
	case "ready":
		jsonOutput, err := parseJSONOnly("plan ready", args[1:], a.deps.Stderr)
		if err != nil {
			return err
		}
		state, err := store.PlanReady()
		if err != nil {
			return err
		}
		return a.printState(state, jsonOutput)
	case "seal":
		fs := newFlagSet("plan seal", a.deps.Stderr)
		expected := fs.String("expect", "", "expected SHA-256")
		jsonOutput := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("usage: elgordo plan seal --expect <sha256> [--json]")
		}
		state, err := store.SealPlan(*expected)
		if err != nil {
			return err
		}
		return a.printState(state, *jsonOutput)
	case "replan":
		fs := newFlagSet("plan replan", a.deps.Stderr)
		reason := fs.String("reason", "", "replanning reason")
		jsonOutput := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("usage: elgordo plan replan --reason <text> [--json]")
		}
		state, err := store.ForceReplan(*reason)
		if err != nil {
			return err
		}
		return a.printState(state, *jsonOutput)
	default:
		return fmt.Errorf("unknown plan command %q", args[0])
	}
}

func (a app) execution(args []string) error {
	if len(args) == 0 || args[0] != "ready" {
		return errors.New("usage: elgordo execution ready")
	}
	jsonOutput, err := parseJSONOnly("execution ready", args[1:], a.deps.Stderr)
	if err != nil {
		return err
	}
	store, err := a.store(true)
	if err != nil {
		return err
	}
	state, err := store.ExecutionReady()
	if err != nil {
		return err
	}
	return a.printState(state, jsonOutput)
}

func (a app) code(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: elgordo code <approve|reject>")
	}
	store, err := a.store(true)
	if err != nil {
		return err
	}
	if args[0] == "approve" {
		jsonOutput, err := parseJSONOnly("code approve", args[1:], a.deps.Stderr)
		if err != nil {
			return err
		}
		state, err := store.CodeApprove()
		if err != nil {
			return err
		}
		return a.printState(state, jsonOutput)
	}
	if args[0] != "reject" {
		return fmt.Errorf("unknown code command %q", args[0])
	}
	route, reason, jsonOutput, err := parseRouteReason("code reject", args[1:], a.deps.Stderr)
	if err != nil {
		return err
	}
	state, err := store.CodeReject(route, reason)
	if err != nil {
		return err
	}
	return a.printState(state, jsonOutput)
}

func (a app) qa(args []string) error {
	if len(args) == 0 || args[0] != "submit" {
		return errors.New("usage: elgordo qa submit --verdict <pass|fail> [--route <execution|planning> --reason <text>]")
	}
	fs := newFlagSet("qa submit", a.deps.Stderr)
	verdict := fs.String("verdict", "", "pass or fail")
	route := fs.String("route", "", "execution or planning")
	reason := fs.String("reason", "", "failure reason")
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: elgordo qa submit --verdict <pass|fail> [--route <execution|planning> --reason <text>] [--json]")
	}
	if strings.EqualFold(*verdict, "pass") && (*route != "" || *reason != "") {
		return errors.New("QA pass cannot include route or reason")
	}
	store, err := a.store(true)
	if err != nil {
		return err
	}
	state, err := store.QASubmit(*verdict, *route, *reason)
	if err != nil {
		return err
	}
	return a.printState(state, *jsonOutput)
}

func (a app) final(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: elgordo final <approve|reject>")
	}
	store, err := a.store(true)
	if err != nil {
		return err
	}
	if args[0] == "approve" {
		jsonOutput, err := parseJSONOnly("final approve", args[1:], a.deps.Stderr)
		if err != nil {
			return err
		}
		state, err := store.FinalApprove()
		if err != nil {
			return err
		}
		return a.printState(state, jsonOutput)
	}
	if args[0] != "reject" {
		return fmt.Errorf("unknown final command %q", args[0])
	}
	route, reason, jsonOutput, err := parseRouteReason("final reject", args[1:], a.deps.Stderr)
	if err != nil {
		return err
	}
	state, err := store.FinalReject(route, reason)
	if err != nil {
		return err
	}
	return a.printState(state, jsonOutput)
}

func (a app) store(requireInit bool) (*workflow.Store, error) {
	cwd, err := a.deps.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := workflow.FindRepositoryRoot(cwd)
	if err != nil {
		return nil, err
	}
	if requireInit {
		if _, err := os.Stat(root + string(os.PathSeparator) + ".elgordo" + string(os.PathSeparator) + "project.json"); err != nil {
			return nil, errors.New("project is not initialized; run elgordo init")
		}
	}
	return workflow.NewStore(root, a.deps.Version), nil
}

func (a app) printState(state *workflow.State, jsonOutput bool) error {
	if jsonOutput {
		return writeJSON(a.deps.Stdout, state)
	}
	fmt.Fprintf(a.deps.Stdout, "%s -> %s\n", state.ChangeID, state.Phase)
	return nil
}

func (a app) usage() {
	fmt.Fprintln(a.deps.Stdout, `ElGordo IA - human-led engineering workflow for OpenCode

Usage:
  elgordo install [--yes]
  elgordo sync [--yes]
  elgordo doctor [--json]
  elgordo uninstall
  elgordo init
  elgordo skill-registry refresh [--force]
  elgordo status [--json]
  elgordo change start <slug> [--title <title>]
  elgordo plan <hash|ready|verify|seal|replan>
  elgordo execution ready
  elgordo code <approve|reject>
  elgordo qa submit --verdict <pass|fail>
  elgordo final <approve|reject>
  elgordo version`)
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func parseRouteReason(name string, args []string, stderr io.Writer) (string, string, bool, error) {
	fs := newFlagSet(name, stderr)
	route := fs.String("route", "", "execution or planning")
	reason := fs.String("reason", "", "rejection reason")
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return "", "", false, err
	}
	if fs.NArg() != 0 {
		return "", "", false, errors.New("unexpected positional arguments")
	}
	return *route, *reason, *jsonOutput, nil
}

func parseJSONOnly(name string, args []string, stderr io.Writer) (bool, error) {
	fs := newFlagSet(name, stderr)
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return false, err
	}
	if fs.NArg() != 0 {
		return false, errors.New("unexpected positional arguments")
	}
	return *jsonOutput, nil
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
