package onboard

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/briceduke/ax/internal/checks"
	"github.com/briceduke/ax/internal/compile"
	"github.com/briceduke/ax/internal/logbook"
	"github.com/briceduke/ax/internal/project"
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
	for _, id := range []string{"own-the-harness", "log-friction", "sync-harness", "record-decision"} {
		if _, err := os.Stat(filepath.Join(root, ".ax", "capabilities", id+".md")); err != nil {
			t.Fatalf("init missing capability %s: %v", id, err)
		}
	}
	for _, id := range []string{"record-decision", "log-friction", "sync-harness"} {
		for _, dir := range []string{
			filepath.Join(root, ".cursor", "skills", id, "SKILL.md"),
			filepath.Join(root, ".claude", "skills", id, "SKILL.md"),
		} {
			if _, err := os.Stat(dir); err != nil {
				t.Fatalf("missing skill %s: %v", dir, err)
			}
		}
	}
	for _, rel := range []string{"AGENTS.md", "CLAUDE.md"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "own-the-harness") {
			t.Fatalf("%s missing own-the-harness:\n%s", rel, data)
		}
		if !strings.Contains(string(data), "adopt ax for harness management") {
			t.Fatalf("%s missing decision line:\n%s", rel, data)
		}
		if strings.Contains(string(data), "Ad-hoc markdown rules") {
			t.Fatalf("%s pasted a full decision writeup:\n%s", rel, data)
		}
	}
	for _, name := range []string{"core", "log", "packs", "ax.yaml", ".generated"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("software init left %s at root: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "record")); !os.IsNotExist(err) {
		t.Fatal("software init should not create record/")
	}
	if _, err := os.Stat(filepath.Join(root, ".ax", "ax.yaml")); err != nil {
		t.Fatal(err)
	}
}

func TestInitNotesRecordMirrorsNotes(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	err := Init(root, InitOptions{
		Name:    "probe",
		Intent:  "A hardware notebook.",
		Targets: "cursor,claude-code",
		Notes:   "record",
		Now:     now,
	}, &buf)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, buf.String())
	}
	for _, rel := range []string{
		"record/intent.md",
		"record/decisions/2026-10-04-adopt-ax-for-harness-management.md",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	if err := project.HideNotes(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "record")); !os.IsNotExist(err) {
		t.Fatal("hide should remove record/")
	}
	if err := project.ShowNotes(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "record", "intent.md")); err != nil {
		t.Fatal(err)
	}
}

func TestLogDecisionRewritesAgents(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	if err := Init(root, InitOptions{Name: "w", Intent: "widget", Targets: "cursor,claude-code", Now: now}, &buf); err != nil {
		t.Fatalf("init: %v\n%s", err, buf.String())
	}
	later := now.Add(time.Hour)
	if _, err := logbook.WriteDecision(root, "use bun", "", "## Context\n\nRuntime.\n\n## Options\n\n- npm\n- bun\n\n## Choice\n\nbun.\n\n## Why\n\nSpeed.\n", later); err != nil {
		t.Fatal(err)
	}
	if err := compile.Refresh(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "use bun") {
		t.Fatalf("AGENTS.md missing new decision:\n%s", data)
	}
	if strings.Contains(string(data), "Runtime.") {
		t.Fatalf("AGENTS.md pasted decision body:\n%s", data)
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
	data, err := os.ReadFile(filepath.Join(root, ".ax", "decisions", "2026-10-04-adopt-ax-from-existing-repo.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "reconstructed: true") {
		t.Fatalf("decision = %s", data)
	}
	checksYAML, err := os.ReadFile(filepath.Join(root, ".ax", "checks.yaml"))
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
	if err := Doctor(root, DoctorOptions{
		LookPath: func(file string) (string, error) {
			if file == "docker" {
				return "", fmt.Errorf("missing")
			}
			return exec.LookPath(file)
		},
	}, &buf); err != nil {
		t.Fatalf("doctor: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "ok   PATH ax") {
		t.Fatalf("output = %q", buf.String())
	}
	if !strings.Contains(buf.String(), "container not verified") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestInitInterviewWritesIntentAndDecisions(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	stdin := strings.NewReader("field teams\nsoftware\nbattery life\ntwo people\nlow\n")
	var buf bytes.Buffer
	err := Init(root, InitOptions{
		Name:      "widget",
		Intent:    "A small recorder for field notes.",
		Targets:   "cursor,claude-code",
		Interview: true,
		Stdin:     stdin,
		Now:       now,
	}, &buf)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, buf.String())
	}
	intent, err := os.ReadFile(filepath.Join(root, ".ax", "intent.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(intent)
	for _, want := range []string{"field teams", "software", "battery life", "two people", "low"} {
		if !strings.Contains(text, want) {
			t.Fatalf("intent missing %q:\n%s", want, text)
		}
	}
	for _, name := range []string{
		"2026-10-04-choose-the-product-toolchain.md",
		"2026-10-04-set-process-weight.md",
	} {
		if _, err := os.Stat(filepath.Join(root, ".ax", "decisions", name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "Dockerfile")); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := checks.Run(root, checks.Options{Tier: checks.TierFull, All: true}, &buf); err != nil {
		t.Fatalf("check after interview: %v\n%s", err, buf.String())
	}
}

func TestDoctorRequireContainerFailsWithoutDocker(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	if err := Init(root, InitOptions{Name: "d", Intent: "doctor fixture", Targets: "cursor,claude-code", Now: now}, &buf); err != nil {
		t.Fatalf("init: %v\n%s", err, buf.String())
	}
	buf.Reset()
	err := Doctor(root, DoctorOptions{
		RequireContainer: true,
		LookPath: func(file string) (string, error) {
			if file == "ax" {
				return "/bin/ax", nil
			}
			return "", fmt.Errorf("missing")
		},
	}, &buf)
	if err == nil || !strings.Contains(err.Error(), "container not verified") {
		t.Fatalf("error = %v out=%q", err, buf.String())
	}
}

func TestDoctorContainerUsesFakeDocker(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	if err := Init(root, InitOptions{Name: "d", Intent: "doctor fixture", Targets: "cursor,claude-code", Now: now}, &buf); err != nil {
		t.Fatalf("init: %v\n%s", err, buf.String())
	}
	var cmds []string
	buf.Reset()
	err := Doctor(root, DoctorOptions{
		LookPath: func(file string) (string, error) {
			return "/bin/" + file, nil
		},
		Run: func(dir, name string, args ...string) (string, error) {
			cmds = append(cmds, name+" "+strings.Join(args, " "))
			return "", nil
		},
	}, &buf)
	if err != nil {
		t.Fatalf("doctor: %v\n%s", err, buf.String())
	}
	if len(cmds) < 2 || !strings.Contains(cmds[0], "docker build") || !strings.Contains(cmds[1], "docker run") {
		t.Fatalf("cmds = %#v", cmds)
	}
	if !strings.Contains(buf.String(), "ok   container") {
		t.Fatalf("output = %q", buf.String())
	}
}
