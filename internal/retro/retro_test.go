package retro

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/briceduke/ax/internal/logbook"
)

func TestRetroEmitsDeletion(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	src := filepath.Join(repo, "testdata", "fixtures", "hardware")
	root := t.TempDir()
	copyTree(t, src, root)
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	if _, err := logbook.WriteFriction(root, "agent invented a pin assignment that was not in the datasheet", now); err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	p, err := Run(root, Options{Now: now}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p.Rel)))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateProposal(data); err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "## Deletions") {
		t.Fatalf("proposal missing deletions:\n%s", text)
	}
	if !strings.Contains(text, "verify-pinouts") {
		t.Fatalf("proposal missing scaffold deletion:\n%s", text)
	}
}

func TestValidateProposalRejectsEmptyDeletions(t *testing.T) {
	data := []byte(`---
id: 2026-10-04-retro
date: 2026-10-04
---

## Problems

- x

## Proposed evals

- none

## Core changes

- none

## Deletions

`)
	err := ValidateProposal(data)
	if err == nil || !strings.Contains(err.Error(), "deletion") {
		t.Fatalf("error = %v", err)
	}
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
