package logbook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"adopt ax for harness management", "adopt-ax-for-harness-management"},
		{"Hello, World!", "hello-world"},
		{"  ", "entry"},
		{"one two three four five six seven eight nine", "one-two-three-four-five-six-seven-eight"},
	}
	for _, tc := range cases {
		if got := Slugify(tc.in); got != tc.want {
			t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMintIDCollisionSuffix(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	dir := filepath.Join(root, ".ax", "friction")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(dir, "2026-10-02-same-slug.md")
	if err := os.WriteFile(first, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	id, err := MintID(root, "friction", "same-slug", now)
	if err != nil {
		t.Fatal(err)
	}
	if id != "2026-10-02-same-slug-2" {
		t.Fatalf("MintID() = %q, want -2 suffix", id)
	}
}

func TestValidID(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"2026-10-02-adopt-ax", true},
		{"2026-10-02-adopt-ax-2", true},
		{"2026-10-02", false},
		{"2026-10-02-Adopt-Ax", false},
		{"not-a-date-slug", false},
	}
	for _, tc := range cases {
		if got := ValidID(tc.id); got != tc.want {
			t.Errorf("ValidID(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestValidateDecision(t *testing.T) {
	valid := []byte(`---
id: 2026-10-01-adopt-ax
kind: decision
date: 2026-10-01
---

## Context

Need a harness.

## Options

- None
- Ax

## Choice

Ax.

## Why

Checks beat prose.
`)
	if err := ValidateBytes(valid, "decision", "2026-10-01-adopt-ax"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := []struct {
		name, kind, filename string
		data                 string
		want                 string
	}{
		{
			name:     "missing why",
			kind:     "decision",
			filename: "2026-10-01-x",
			data: `---
id: 2026-10-01-x
kind: decision
date: 2026-10-01
---

## Context

## Options

## Choice
`,
			want: "missing heading ## Why",
		},
		{
			name:     "id filename mismatch",
			kind:     "decision",
			filename: "2026-10-01-other",
			data: `---
id: 2026-10-01-x
kind: decision
date: 2026-10-01
---

## Context

## Options

## Choice

Use ax.

## Why
`,
			want: "does not match filename",
		},
		{
			name:     "date mismatch",
			kind:     "friction",
			filename: "2026-10-02-x",
			data: `---
id: 2026-10-02-x
kind: friction
date: 2026-10-01
---

one line
`,
			want: "does not match date",
		},
		{
			name:     "friction two lines",
			kind:     "friction",
			filename: "2026-10-02-x",
			data: `---
id: 2026-10-02-x
kind: friction
date: 2026-10-02
---

one
two
`,
			want: "body must be one line",
		},
		{
			name:     "observation missing method",
			kind:     "observation",
			filename: "2026-10-02-x",
			data: `---
id: 2026-10-02-x
kind: observation
date: 2026-10-02
---

a finding
`,
			want: "missing method",
		},
		{
			name:     "unknown field",
			kind:     "friction",
			filename: "2026-10-02-x",
			data: `---
id: 2026-10-02-x
kind: friction
date: 2026-10-02
nope: 1
---

one line
`,
			want: "yaml",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBytes([]byte(tc.data), tc.kind, tc.filename)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestWriteAndValidate(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	rel, err := WriteFriction(root, "agent re-derived the battery math", now)
	if err != nil {
		t.Fatal(err)
	}
	if rel != ".ax/friction/2026-10-02-agent-re-derived-the-battery-math.md" {
		t.Fatalf("rel = %s", rel)
	}
	if err := ValidateFile(filepath.Join(root, filepath.FromSlash(rel)), "friction"); err != nil {
		t.Fatal(err)
	}
	rel, err = WriteObservation(root, "4.1 mA while recording", "bench meter", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateFile(filepath.Join(root, filepath.FromSlash(rel)), "observation"); err != nil {
		t.Fatal(err)
	}
	rel, err = WriteDecision(root, "adopt ax for harness management", "", "## Context\n\nNeed a harness.\n\n## Options\n\n- none\n\n## Choice\n\nAdopt ax.\n\n## Why\n\nChecks.\n", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateFile(filepath.Join(root, filepath.FromSlash(rel)), "decision"); err != nil {
		t.Fatal(err)
	}
}

func TestChoiceSentence(t *testing.T) {
	ok, err := ChoiceSentence("## Context\n\n## Options\n\n## Choice\n\nAsk before adding a dependency.\n\n## Why\n\nCost.\n")
	if err != nil || ok != "Ask before adding a dependency." {
		t.Fatalf("got %q err=%v", ok, err)
	}
	cases := []struct {
		body, want string
	}{
		{"## Choice\n\n## Why\n", "choice is empty"},
		{"## Choice\n\nUse bun. Skip npm.\n\n## Why\n", "one sentence"},
		{"## Choice\n\n" + strings.Repeat("a", 201) + "\n\n## Why\n", "max 200"},
	}
	for _, tc := range cases {
		_, err := ChoiceSentence(tc.body)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("body %q error = %v, want %q", tc.body, err, tc.want)
		}
	}
}

func TestDecisionTooLong(t *testing.T) {
	var b strings.Builder
	b.WriteString("---\nid: 2026-10-01-x\nkind: decision\ndate: 2026-10-01\n---\n\n## Context\n\n## Options\n\n## Choice\n\nAx.\n\n## Why\n")
	for i := 0; i < 60; i++ {
		b.WriteString("line\n")
	}
	err := ValidateBytes([]byte(b.String()), "decision", "2026-10-01-x")
	if err == nil || !strings.Contains(err.Error(), "max 60") {
		t.Fatalf("error = %v, want max 60", err)
	}
}
