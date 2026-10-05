package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindWalksUp(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(Join(root), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Join(root, "ax.yaml"), []byte("ax: 0.1.0\ntargets: []\npacks: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	got, err := Find(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("Find() = %q, want %q", got, root)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(Join(root), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Join(root, "ax.yaml"), []byte("ax: 0.1.0\ntargets: []\npacks: []\nextra: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "yaml") {
		t.Fatalf("Load() error = %v, want unknown field", err)
	}
}

func TestLoadRequiresVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(Join(root), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Join(root, "ax.yaml"), []byte("targets: []\npacks: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "missing ax version") {
		t.Fatalf("Load() error = %v, want missing version", err)
	}
}
