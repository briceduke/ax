package eval

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseAndScriptGrade(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "core/evals/harness-format.md", `---
id: harness-format
related: [scaffold/verify-pinouts]
runs: 3
---
task: >
  Format checks still pass.
grader:
  kind: script
  check: ax-format
  file: core/intent.md
  number:
    file: measurements/idle-ma.txt
    min: 3
    max: 6
`)
	writeFile(t, root, "core/intent.md", "# Intent\n")
	writeFile(t, root, "measurements/idle-ma.txt", "4.1\n")
	ev, err := ParseFile(filepath.Join(root, "core", "evals", "harness-format.md"))
	if err != nil {
		t.Fatal(err)
	}
	if ev.ID != "harness-format" || ev.Runs != 3 {
		t.Fatalf("eval = %+v", ev)
	}
	if !Relates(ev, "scaffold", "verify-pinouts") {
		t.Fatal("expected related scaffold")
	}
	g := scriptGrader{runCheck: func(root, id string) error {
		if id != "ax-format" {
			t.Fatalf("check id = %s", id)
		}
		return nil
	}}
	if err := g.Grade(root, ev); err != nil {
		t.Fatal(err)
	}
}

func TestRubricStandInAndInjectedGrader(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "core/evals/readable.md", `---
id: readable
related: []
runs: 2
---
task: intent names the product
grader:
  kind: rubric
  must: ["hardware fixture"]
`)
	writeFile(t, root, "core/intent.md", "A hardware fixture used to test ax.\n")
	ev, err := ParseFile(filepath.Join(root, "core", "evals", "readable.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := (rubricGrader{}).Grade(root, ev); err != nil {
		t.Fatal(err)
	}
	fake := graderFunc(func(root string, ev *Eval) error {
		return nil
	})
	var buf bytes.Buffer
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	results, err := Run(root, Options{Now: now, Grader: fake, Runs: 1}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Score() != "1/1" {
		t.Fatalf("results = %+v", results)
	}
	if !strings.Contains(buf.String(), "ok   readable 1/1") {
		t.Fatalf("output = %q", buf.String())
	}
	obs := filepath.Join(root, "log", "observations", "2026-10-04-eval-readable-1-1.md")
	data, err := os.ReadFile(obs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "method: eval") {
		t.Fatalf("observation = %s", data)
	}
}

func TestUnknownEvalFieldRejected(t *testing.T) {
	_, err := ParseBytes([]byte(`---
id: x
related: []
runs: 1
extra: 1
---
task: t
grader:
  kind: script
  file: core/intent.md
`))
	if err == nil || !strings.Contains(err.Error(), "yaml") {
		t.Fatalf("error = %v, want unknown field", err)
	}
}

func TestRubricSkipsWithoutRunner(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "core/evals/readable.md", `---
id: readable
related: []
runs: 1
---
task: intent names the product
grader:
  kind: rubric
  must: ["hardware fixture"]
`)
	writeFile(t, root, "core/intent.md", "A hardware fixture used to test ax.\n")
	var buf bytes.Buffer
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	results, err := Run(root, Options{Now: now, Runs: 1}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Skipped {
		t.Fatalf("results = %+v", results)
	}
	if !strings.Contains(buf.String(), "skip readable") {
		t.Fatalf("output = %q", buf.String())
	}
	found := false
	entries, err := os.ReadDir(filepath.Join(root, "log", "observations"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(root, "log", "observations", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "skipped:") && strings.Contains(string(data), "method: eval") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected skipped eval observation")
	}
}

func TestFakeRunnerThenGrade(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "core/evals/ran.md", `---
id: ran
related: []
runs: 1
---
task: write the marker file
grader:
  kind: script
  file: marker.txt
`)
	var ranTask string
	fake := runnerFunc(func(dir, task string) (string, error) {
		ranTask = task
		return "", os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("ok\n"), 0644)
	})
	var buf bytes.Buffer
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	results, err := Run(root, Options{Now: now, Runner: fake}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if ranTask != "write the marker file" {
		t.Fatalf("task = %q", ranTask)
	}
	if len(results) != 1 || results[0].Skipped || results[0].Score() != "1/1" {
		t.Fatalf("results = %+v", results)
	}
	if !strings.Contains(buf.String(), "ok   ran 1/1") {
		t.Fatalf("output = %q", buf.String())
	}
}

type runnerFunc func(dir, task string) (string, error)

func (f runnerFunc) Run(dir, task string) (string, error) {
	return f(dir, task)
}

type graderFunc func(root string, ev *Eval) error

func (f graderFunc) Grade(root string, ev *Eval) error {
	return f(root, ev)
}

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
