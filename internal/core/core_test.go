package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashFilesNormalizesCRLF(t *testing.T) {
	root := t.TempDir()
	lfDir := filepath.Join(root, "lf")
	crlfDir := filepath.Join(root, "crlf")
	if err := os.MkdirAll(lfDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(crlfDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lfDir, "a.md"), []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crlfDir, "a.md"), []byte("one\r\ntwo\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	lfHash, err := HashFiles(lfDir, []string{"a.md"})
	if err != nil {
		t.Fatal(err)
	}
	crlfHash, err := HashFiles(crlfDir, []string{"a.md"})
	if err != nil {
		t.Fatal(err)
	}
	if lfHash != crlfHash {
		t.Fatalf("LF hash %s != CRLF hash %s", lfHash, crlfHash)
	}
}

func TestContentHashNormalizesCRLF(t *testing.T) {
	if ContentHash([]byte("a\r\nb")) != ContentHash([]byte("a\nb")) {
		t.Fatal("ContentHash should ignore CRLF vs LF")
	}
}

func TestParseCapabilityAndScaffold(t *testing.T) {
	root := t.TempDir()
	capPath := filepath.Join(root, "cap.md")
	scPath := filepath.Join(root, "sc.md")
	if err := os.WriteFile(capPath, []byte("---\nid: record-decision\nkind: capability\nwhen: a choice is made\nhints: [user-invocable]\n---\n\nWrite it down.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scPath, []byte("---\nid: verify-pinouts\nkind: scaffold\ncompensates: invented pins\nadded: 2026-10-02 (friction/2026-10-02-wrong-pin)\nretire_when: eval passes\n---\n\nQuote the table.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseCapability(capPath); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseScaffold(scPath); err != nil {
		t.Fatal(err)
	}
}

func TestParseCapabilityRejectsUnknownField(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "cap.md")
	if err := os.WriteFile(path, []byte("---\nid: x\nkind: capability\nwhen: now\nhints: []\nextra: 1\n---\n\nbody\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ParseCapability(path)
	if err == nil || !strings.Contains(err.Error(), "yaml") {
		t.Fatalf("error = %v, want unknown field", err)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	yamlBytes, body, err := SplitFrontmatter([]byte("---\nid: x\n---\n\nhello\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(yamlBytes), "id: x") {
		t.Fatalf("yaml = %q", yamlBytes)
	}
	if strings.TrimSpace(body) != "hello" {
		t.Fatalf("body = %q", body)
	}
}

func TestInputHashIncludesCoreFiles(t *testing.T) {
	root := t.TempDir()
	writeCore(t, root)
	h1, err := InputHash(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core", "intent.md"), []byte("# changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	h2, err := InputHash(root)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Fatal("intent change should change core hash")
	}
}

func TestCompileHashIncludesTargetSpec(t *testing.T) {
	root := t.TempDir()
	writeCore(t, root)
	h1, err := CompileHash(root, map[string][]byte{"targets/cursor.md": []byte("spec-a\n")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := CompileHash(root, map[string][]byte{"targets/cursor.md": []byte("spec-b\n")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Fatal("target spec change should change compile hash")
	}
	h3, err := CompileHash(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h3 {
		t.Fatal("adding a target spec should change compile hash")
	}
}

func TestCompileHashExcludesScaffold(t *testing.T) {
	root := t.TempDir()
	writeCore(t, root)
	dir := filepath.Join(root, "core", "scaffolds")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: verify-pinouts\nkind: scaffold\ncompensates: invented pins\nadded: 2026-10-02 (friction/2026-10-02-wrong-pin)\nretire_when: eval passes\n---\n\nQuote the table.\n"
	if err := os.WriteFile(filepath.Join(dir, "verify-pinouts.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	full, err := CompileHash(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	without, err := CompileHash(root, nil, []string{"verify-pinouts"})
	if err != nil {
		t.Fatal(err)
	}
	if full == without {
		t.Fatal("--without should change compile hash")
	}
}

func writeCore(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "core"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core", "intent.md"), []byte("# Intent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core", "checks.yaml"), []byte("[]\n"), 0644); err != nil {
		t.Fatal(err)
	}
}
