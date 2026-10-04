package onboard

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/briceduke/ax/internal/checks"
)

func TestInitCheckPasses(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	err := Init(root, InitOptions{
		Name:    "widget",
		Intent:  "A small test product.",
		Targets: "cursor,claude-code",
		Now:     now,
	}, &buf)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, buf.String())
	}
	buf.Reset()
	if err := checks.Run(root, checks.Options{Tier: checks.TierFull, All: true}, &buf); err != nil {
		t.Fatalf("check after init: %v\n%s", err, buf.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor", "commands", "record-decision.md")); err != nil {
		t.Fatal(err)
	}
}

func TestAdoptMarksReconstructed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	if err := Adopt(root, now, &buf); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "log", "decisions", "2026-10-04-adopt-ax-from-existing-repo.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "reconstructed: true") {
		t.Fatalf("decision = %s", data)
	}
	checksYAML, err := os.ReadFile(filepath.Join(root, "core", "checks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(checksYAML), "go-test") {
		t.Fatalf("checks.yaml = %s", checksYAML)
	}
}

func TestDoctorPATH(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	if err := Init(root, InitOptions{Name: "d", Intent: "doctor fixture", Targets: "cursor,claude-code", Now: now}, &buf); err != nil {
		t.Fatalf("init: %v\n%s", err, buf.String())
	}
	bindir := t.TempDir()
	exe := "ax"
	if runtime.GOOS == "windows" {
		exe = "ax.exe"
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bindir, exe), data, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bindir)
	buf.Reset()
	if err := Doctor(root, &buf); err != nil {
		t.Fatalf("doctor: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "ok   PATH ax") {
		t.Fatalf("output = %q", buf.String())
	}
}
