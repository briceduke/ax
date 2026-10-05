package upgrade

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestUpgradeFromFakeSource(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	src := filepath.Join(repo, "testdata", "fixtures", "software")
	root := t.TempDir()
	copyTestTree(t, src, root)
	from := t.TempDir()
	if err := os.WriteFile(filepath.Join(from, "VERSION"), []byte("0.2.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(from, "capabilities"), 0755); err != nil {
		t.Fatal(err)
	}
	capBody := "---\nid: keep-files-short\nkind: capability\nwhen: a file is added\nhints: []\n---\n\nKeep files short.\n"
	if err := os.WriteFile(filepath.Join(from, "capabilities", "keep-files-short.md"), []byte(capBody), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	if err := Run(root, Options{From: from, Now: now}, &buf); err != nil {
		t.Fatalf("upgrade: %v\n%s", err, buf.String())
	}
	report, err := os.ReadFile(filepath.Join(root, ".ax", "upgrade", "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(report)
	if !strings.Contains(text, "old pin: 0.1.0") || !strings.Contains(text, "new pin: 0.2.0") {
		t.Fatalf("report = %s", text)
	}
	if _, err := os.Stat(filepath.Join(root, ".ax", "capabilities", "keep-files-short.md")); err != nil {
		t.Fatal(err)
	}
	cfg, err := os.ReadFile(filepath.Join(root, ".ax", "ax.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "ax: 0.2.0") {
		t.Fatalf("ax.yaml = %s", cfg)
	}
}

func TestUpgradeFromFakeFetcherAndSubmit(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	src := filepath.Join(repo, "testdata", "fixtures", "software")
	root := t.TempDir()
	copyTestTree(t, src, root)
	from := t.TempDir()
	if err := os.WriteFile(filepath.Join(from, "VERSION"), []byte("0.2.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(from, "internal", "builtins", "capabilities"), 0755); err != nil {
		t.Fatal(err)
	}
	capBody := "---\nid: keep-files-short\nkind: capability\nwhen: a file is added\nhints: []\n---\n\nKeep files short.\n"
	if err := os.WriteFile(filepath.Join(from, "internal", "builtins", "capabilities", "keep-files-short.md"), []byte(capBody), 0644); err != nil {
		t.Fatal(err)
	}
	var ghArgs []string
	var buf bytes.Buffer
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	err := Run(root, Options{
		Now: now,
		Fetcher: func() (string, error) {
			return from, nil
		},
		Submit: true,
		Command: func(args ...string) (string, error) {
			ghArgs = args
			return "https://example.invalid/issues/3", nil
		},
	}, &buf)
	if err != nil {
		t.Fatalf("upgrade: %v\n%s", err, buf.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".ax", "capabilities", "keep-files-short.md")); err != nil {
		t.Fatal(err)
	}
	if len(ghArgs) < 2 || ghArgs[0] != "issue" {
		t.Fatalf("gh args = %#v", ghArgs)
	}
	if !strings.Contains(buf.String(), "https://example.invalid/issues/3") {
		t.Fatalf("output = %q", buf.String())
	}
	report, err := os.ReadFile(filepath.Join(root, ".ax", "upgrade", "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "fetched machinery from git") {
		t.Fatalf("report = %s", report)
	}
}

func TestGitFetchUsesInjectedGit(t *testing.T) {
	var got []string
	fetch := GitFetch("https://example.invalid/ax.git", func(dir, name string, args ...string) (string, error) {
		got = append([]string{name}, args...)
		dest := args[len(args)-1]
		if err := os.MkdirAll(dest, 0755); err != nil {
			return "", err
		}
		return "", os.WriteFile(filepath.Join(dest, "VERSION"), []byte("0.9.0\n"), 0644)
	})
	dir, err := fetch()
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "git" || got[1] != "clone" {
		t.Fatalf("got = %#v", got)
	}
	data, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "0.9.0" {
		t.Fatalf("VERSION = %s", data)
	}
}

func copyTestTree(t *testing.T, src, dst string) {
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
