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
	if _, err := os.Stat(filepath.Join(root, "core", "capabilities", "keep-files-short.md")); err != nil {
		t.Fatal(err)
	}
	cfg, err := os.ReadFile(filepath.Join(root, "ax.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "ax: 0.2.0") {
		t.Fatalf("ax.yaml = %s", cfg)
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
