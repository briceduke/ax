package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/briceduke/ax/internal/version"
)

func TestCheckFixtures(t *testing.T) {
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
			t.Chdir(dst)
			stdout, stderr := capture(t, func() error {
				return run([]string{"check"})
			})
			if stderr != "" && !strings.HasPrefix(stderr, "warning:") {
				t.Fatalf("stderr = %q", stderr)
			}
			if !strings.Contains(stdout, "ok   ax-format") {
				t.Fatalf("stdout = %q", stdout)
			}
		})
	}
}

func TestEvalHardwareWritesScores(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	src := filepath.Join(repo, "testdata", "fixtures", "hardware")
	dst := t.TempDir()
	copyTree(t, src, dst)
	t.Chdir(dst)
	stdout, stderr := capture(t, func() error {
		return run([]string{"eval", "--runs", "1"})
	})
	if stderr != "" && !strings.HasPrefix(stderr, "warning:") {
		t.Fatalf("stderr = %q", stderr)
	}
	if !strings.Contains(stdout, "cross-discipline-impact") || !strings.Contains(stdout, "harness-format") || !strings.Contains(stdout, "idle-current-range") {
		t.Fatalf("stdout = %q", stdout)
	}
	entries, err := os.ReadDir(filepath.Join(dst, ".ax", "observations"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 3 {
		t.Fatalf("got %d observations", len(entries))
	}
}

func TestVersionPrintsConstWithoutRepo(t *testing.T) {
	t.Chdir(t.TempDir())
	stdout, stderr := capture(t, func() error {
		return run([]string{"version"})
	})
	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
	if stdout != version.Version+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, version.Version+"\n")
	}
	if !strings.Contains(usage(), "ax version") {
		t.Fatal("usage missing ax version")
	}
	if !strings.Contains(usage(), "ax install") || !strings.Contains(usage(), "ax notes") || !strings.Contains(usage(), "ax help") {
		t.Fatal("usage missing install/notes/help")
	}
	if !strings.Contains(usage(), "Markdown on stdin") {
		t.Fatal("usage missing decision stdin contract")
	}
}

func TestLogHelpPrintsDecisionStdin(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"log", "--help"},
		{"log", "-h"},
		{"help", "log"},
		{"log", "friction", "help"},
		{"log", "decision", "-h"},
		{"init", "--help"},
		{"notes", "--help"},
		{"help"},
	} {
		stdout, stderr := capture(t, func() error {
			return run(args)
		})
		if stderr != "" {
			t.Fatalf("%v stderr = %q", args, stderr)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Fatalf("%v stdout = %q", args, stdout)
		}
	}
}

func TestNotesWithoutArgsPrintsStatus(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	stdout, stderr := capture(t, func() error {
		return run([]string{"init", "--name", "widget", "--intent", "A small test product."})
	})
	if stderr != "" && !strings.HasPrefix(stderr, "warning:") {
		t.Fatalf("init stderr = %q", stderr)
	}
	if !strings.Contains(stdout, "initialized") {
		t.Fatalf("init stdout = %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "record")); !os.IsNotExist(err) {
		t.Fatal("init should not create record/")
	}
	stdout, stderr = capture(t, func() error {
		return run([]string{"notes"})
	})
	if stderr != "" && !strings.HasPrefix(stderr, "warning:") {
		t.Fatalf("notes stderr = %q", stderr)
	}
	if stdout != "notes: hidden\n" {
		t.Fatalf("notes stdout = %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "record")); !os.IsNotExist(err) {
		t.Fatal("ax notes must not turn record/ on")
	}
	stdout, stderr = capture(t, func() error {
		return run([]string{"notes", "show"})
	})
	if stderr != "" && !strings.HasPrefix(stderr, "warning:") {
		t.Fatalf("show stderr = %q", stderr)
	}
	if stdout != "notes: record\n" {
		t.Fatalf("show stdout = %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "record", "intent.md")); err != nil {
		t.Fatal(err)
	}
}

func TestLogDecisionPinsAlwaysOnAndCheck(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	capture(t, func() error {
		return run([]string{"init", "--name", "widget", "--intent", "A small test product."})
	})
	body := "## Context\n\nCommits.\n\n## Options\n\n- freeform\n- conventional\n\n## Choice\n\nCommit messages use Conventional Commits.\n\n## Why\n\nHistory stays readable.\n"
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldIn := os.Stdin
	os.Stdin = r
	go func() {
		_, _ = w.Write([]byte(body))
		w.Close()
	}()
	stdout, stderr := capture(t, func() error {
		return run([]string{"log", "decision", "use conventional commit messages", "--always-on", "--when", "you write a commit message", "--check", "conventional-commits"})
	})
	os.Stdin = oldIn
	if stderr != "" && !strings.HasPrefix(stderr, "warning:") {
		t.Fatalf("stderr = %q", stderr)
	}
	if !strings.Contains(stdout, ".ax/decisions/") || !strings.Contains(stdout, ".ax/capabilities/use-conventional-commit-messages.md") {
		t.Fatalf("stdout = %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, ".ax", "capabilities", "use-conventional-commit-messages.md")); err != nil {
		t.Fatal(err)
	}
	checksYAML, err := os.ReadFile(filepath.Join(root, ".ax", "checks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(checksYAML), "builtin: conventional-commits") {
		t.Fatalf("checks.yaml = %s", checksYAML)
	}
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "### use-conventional-commit-messages") {
		t.Fatalf("AGENTS.md missing always-on note:\n%s", agents)
	}
}

func TestInstallWritesSkillsUnderHome(t *testing.T) {
	home := t.TempDir()
	stdout, stderr := capture(t, func() error {
		return run([]string{"install", "--home", home})
	})
	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
	if !strings.Contains(stdout, "wrote /ax and /ax-migrate skills") {
		t.Fatalf("stdout = %q", stdout)
	}
	for _, rel := range []string{
		".cursor/skills/ax/SKILL.md",
		".claude/skills/ax/SKILL.md",
		".cursor/skills/ax-migrate/SKILL.md",
		".claude/skills/ax-migrate/SKILL.md",
	} {
		data, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		if !strings.Contains(string(data), "disable-model-invocation: true") {
			t.Fatalf("%s = %s", rel, data)
		}
	}
}

func capture(t *testing.T, fn func() error) (string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rout, wout, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rerr, werr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = wout, werr
	runErr := fn()
	wout.Close()
	werr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	var outBuf, errBuf bytes.Buffer
	io.Copy(&outBuf, rout)
	io.Copy(&errBuf, rerr)
	if runErr != nil {
		t.Fatalf("run: %v\nstdout=%s\nstderr=%s", runErr, outBuf.String(), errBuf.String())
	}
	return outBuf.String(), errBuf.String()
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
