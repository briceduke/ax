package pack

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAddExtractBootstrap(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	src := filepath.Join(repo, "testdata", "fixtures", "software")
	root := t.TempDir()
	copyTestTree(t, src, root)

	packDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(packDir, "capabilities"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "pack.yaml"), []byte("id: short-files\nversion: 0.1.0\nprovides: [capabilities]\nrequires: []\nparams: []\nprovenance: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cap := "---\nid: keep-files-short\nkind: capability\nwhen: a file is added\nhints: []\n---\n\nKeep files short.\n"
	if err := os.WriteFile(filepath.Join(packDir, "capabilities", "keep-files-short.md"), []byte(cap), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Add(root, packDir, &buf); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".ax", "capabilities", "keep-files-short.md")); err != nil {
		t.Fatal(err)
	}
	cfg, err := os.ReadFile(filepath.Join(root, ".ax", "ax.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "short-files") {
		t.Fatalf("ax.yaml = %s", cfg)
	}

	outDir := t.TempDir()
	buf.Reset()
	if err := Extract(root, outDir, "short-files", []string{".ax/capabilities/keep-files-short.md"}, map[string]string{"Keep files short": "RULE"}, &buf); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(outDir, "capabilities", "keep-files-short.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "{{RULE}}") {
		t.Fatalf("extract = %s", got)
	}
	buf.Reset()
	if err := Bootstrap(root, "pcb", &buf); err != nil {
		t.Fatal(err)
	}
	guide, err := os.ReadFile(filepath.Join(root, ".ax", "packs", "pcb-starter", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(guide), "1.") || !strings.Contains(string(guide), "6.") {
		t.Fatalf("bootstrap guide missing steps:\n%s", guide)
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
