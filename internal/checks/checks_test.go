package checks

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/version"
	"github.com/briceduke/ax/targets"
)

func TestSelectTiersCumulative(t *testing.T) {
	all := []Check{
		{ID: "fast", Tier: TierFast, Builtin: "ax-format"},
		{ID: "full", Tier: TierFull, Run: "true"},
		{ID: "slow", Tier: TierSlow, Run: "true"},
	}
	cases := []struct {
		tier string
		want []string
	}{
		{TierFast, []string{"fast"}},
		{TierFull, []string{"fast", "full"}},
		{TierSlow, []string{"fast", "full", "slow"}},
	}
	for _, tc := range cases {
		got, err := Select(all, tc.tier)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("tier %s: got %d checks, want %d", tc.tier, len(got), len(tc.want))
		}
		for i := range tc.want {
			if got[i].ID != tc.want[i] {
				t.Fatalf("tier %s [%d] = %s, want %s", tc.tier, i, got[i].ID, tc.want[i])
			}
		}
	}
}

func TestHashCheckNormalizesCRLF(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "in.txt"), []byte("a\nb\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c := Check{ID: "x", Run: "true", Inputs: []string{"in.txt"}, Tier: TierFast}
	h1, err := hashCheck(root, c)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "in.txt"), []byte("a\r\nb\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	h2, err := hashCheck(root, c)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("hash changed across line endings: %s vs %s", h1, h2)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		os.Exit(2)
	}
	switch args[0] {
	case "pass":
		os.Exit(0)
	case "fail":
		os.Stderr.WriteString("helper failed\n")
		os.Exit(1)
	case "count":
		path := args[1]
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			os.Exit(3)
		}
		f.WriteString("x\n")
		f.Close()
		os.Exit(0)
	default:
		os.Exit(2)
	}
}

func helperRun(mode string, extra ...string) string {
	parts := []string{os.Args[0], "-test.run=TestHelperProcess", "--", mode}
	parts = append(parts, extra...)
	return strings.Join(parts, " ")
}

func TestRunnerExternalPassFailAndCache(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		return
	}
	root := t.TempDir()
	os.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Cleanup(func() { os.Unsetenv("GO_WANT_HELPER_PROCESS") })
	writeProject(t, root, helperRun("pass"), "")
	var buf bytes.Buffer
	if err := Run(root, Options{Tier: TierFull}, &buf); err != nil {
		t.Fatalf("pass run: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "ok   helper") {
		t.Fatalf("output = %q", buf.String())
	}

	countFile := filepath.Join(root, "count.txt")
	writeProject(t, root, helperRun("count", countFile), "")

	buf.Reset()
	if err := Run(root, Options{Tier: TierFull}, &buf); err != nil {
		t.Fatalf("count 1: %v\n%s", err, buf.String())
	}
	buf.Reset()
	if err := Run(root, Options{Tier: TierFull}, &buf); err != nil {
		t.Fatalf("count skip: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "skip helper") {
		t.Fatalf("expected skip, got %q", buf.String())
	}
	data, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "x") != 1 {
		t.Fatalf("helper ran %d times, want 1", strings.Count(string(data), "x"))
	}
	buf.Reset()
	if err := Run(root, Options{Tier: TierFull, All: true}, &buf); err != nil {
		t.Fatalf("--all: %v\n%s", err, buf.String())
	}
	data, err = os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "x") != 2 {
		t.Fatalf("helper ran %d times after --all, want 2", strings.Count(string(data), "x"))
	}

	writeProject(t, root, helperRun("fail"), "log/decisions/2026-10-01-adopt-ax-for-harness-management")
	writeDecision(t, root)
	buf.Reset()
	err = Run(root, Options{Tier: TierFull, All: true}, &buf)
	if err == nil {
		t.Fatal("expected fail")
	}
	out := buf.String()
	if !strings.Contains(out, "FAIL helper") {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, "helper failed") {
		t.Fatalf("missing helper output: %q", out)
	}
	if !strings.Contains(out, "enforces log/decisions/2026-10-01-adopt-ax-for-harness-management") {
		t.Fatalf("missing enforces: %q", out)
	}
	if !strings.Contains(out, "Checks replace prose rules") {
		t.Fatalf("missing Why: %q", out)
	}
}

func TestE2EFixturesPassThenCorruptFails(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		return
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	for _, name := range []string{"software", "hardware"} {
		t.Run(name, func(t *testing.T) {
			src := filepath.Join(repo, "testdata", "fixtures", name)
			dst := t.TempDir()
			copyTree(t, src, dst)
			var buf bytes.Buffer
			if err := Run(dst, Options{Tier: TierFull}, &buf); err != nil {
				t.Fatalf("expected pass: %v\n%s", err, buf.String())
			}
			bad := filepath.Join(dst, "log", "friction", "2026-10-02-broken.md")
			if err := os.MkdirAll(filepath.Dir(bad), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(bad, []byte("---\nid: 2026-10-02-broken\nkind: friction\ndate: 2026-10-02\n---\n\none\ntwo\n"), 0644); err != nil {
				t.Fatal(err)
			}
			buf.Reset()
			err := Run(dst, Options{Tier: TierFull, All: true}, &buf)
			if err == nil {
				t.Fatal("expected fail after corrupt entry")
			}
			out := buf.String()
			if !strings.Contains(out, "FAIL ax-format") {
				t.Fatalf("output = %q", out)
			}
			if !strings.Contains(out, "enforces ") {
				t.Fatalf("missing enforces: %q", out)
			}
			if !strings.Contains(out, "## Why") {
				t.Fatalf("missing Why: %q", out)
			}
		})
	}
}

func TestStaleness(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		return
	}
	root := t.TempDir()
	writeBuiltinProject(t, root, []string{})
	if err := checkStaleness(root); err != nil {
		t.Fatalf("empty targets, no manifest: %v", err)
	}
	writeBuiltinProject(t, root, []string{"cursor"})
	err := checkStaleness(root)
	if err == nil || !strings.Contains(err.Error(), "run ax compile") {
		t.Fatalf("error = %v, want run ax compile", err)
	}
	extra, err := extraFor(t, []string{"cursor"})
	if err != nil {
		t.Fatal(err)
	}
	coreHash, err := core.CompileHash(root, extra, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".generated"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	man := "core_hash: " + coreHash + "\nax_version: " + version.Version + "\nadapters:\n  AGENTS.md: " + core.ContentHash([]byte("hello\n")) + "\n"
	if err := os.WriteFile(filepath.Join(root, ".generated", "manifest.yaml"), []byte(man), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkStaleness(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core", "intent.md"), []byte("# changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	err = checkStaleness(root)
	if err == nil || !strings.Contains(err.Error(), "core hash changed") {
		t.Fatalf("error = %v, want core hash changed", err)
	}
	if err := os.WriteFile(filepath.Join(root, "core", "intent.md"), []byte("# Intent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkStaleness(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("tampered\n"), 0644); err != nil {
		t.Fatal(err)
	}
	err = checkStaleness(root)
	if err == nil || !strings.Contains(err.Error(), "content hash mismatch") {
		t.Fatalf("error = %v, want mismatch", err)
	}
	writeBuiltinProject(t, root, []string{"cursor", "claude-code"})
	extra, err = extraFor(t, []string{"cursor", "claude-code"})
	if err != nil {
		t.Fatal(err)
	}
	coreHash, err = core.CompileHash(root, extra, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	man = "core_hash: " + coreHash + "\nax_version: " + version.Version + "\nadapters:\n  AGENTS.md: " + core.ContentHash([]byte("hello\n")) + "\n"
	if err := os.WriteFile(filepath.Join(root, ".generated", "manifest.yaml"), []byte(man), 0644); err != nil {
		t.Fatal(err)
	}
	err = checkStaleness(root)
	if err == nil || !strings.Contains(err.Error(), "run ax compile") {
		t.Fatalf("error = %v, want run ax compile for missing target", err)
	}
}

func extraFor(t *testing.T, ids []string) (map[string][]byte, error) {
	t.Helper()
	return targets.Extra(ids)
}

func writeProject(t *testing.T, root, run, enforces string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "core"), 0755); err != nil {
		t.Fatal(err)
	}
	ax := "ax: " + version.Version + "\ntargets: []\npacks: []\n"
	if err := os.WriteFile(filepath.Join(root, "ax.yaml"), []byte(ax), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core", "intent.md"), []byte("# Intent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	enforcesLine := ""
	if enforces != "" {
		enforcesLine = "  enforces: " + enforces + "\n"
	}
	checks := "- id: helper\n  run: " + strconv.Quote(run) + "\n  inputs: []\n  tier: full\n" + enforcesLine
	if err := os.WriteFile(filepath.Join(root, "core", "checks.yaml"), []byte(checks), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeBuiltinProject(t *testing.T, root string, targets []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "core"), 0755); err != nil {
		t.Fatal(err)
	}
	targetYAML := "[]"
	if len(targets) > 0 {
		targetYAML = "[" + strings.Join(targets, ", ") + "]"
	}
	ax := "ax: " + version.Version + "\ntargets: " + targetYAML + "\npacks: []\n"
	if err := os.WriteFile(filepath.Join(root, "ax.yaml"), []byte(ax), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core", "intent.md"), []byte("# Intent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core", "checks.yaml"), []byte("[]\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeDecision(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "log", "decisions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	body := `---
id: 2026-10-01-adopt-ax-for-harness-management
kind: decision
date: 2026-10-01
---

## Context

Need rules.

## Options

- Prose
- Ax

## Choice

Ax.

## Why

Checks replace prose rules.
`
	path := filepath.Join(dir, "2026-10-01-adopt-ax-for-harness-management.md")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
