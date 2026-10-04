package ablate

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

func TestAblateProposesRetirementWhenWithoutHolds(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	src := filepath.Join(repo, "testdata", "fixtures", "hardware")
	root := t.TempDir()
	copyTree(t, src, root)
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	err := Run(root, Options{Now: now, Runs: 1, RunCheck: checks.RunNamed}, &buf)
	if err != nil {
		t.Fatalf("ablate: %v\n%s", err, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "drop verify-pinouts") && !strings.Contains(out, "keep verify-pinouts") {
		t.Fatalf("output = %q", out)
	}
	proposal, err := os.ReadFile(filepath.Join(root, ".ax", "ablate", "proposal.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proposal), "verify-pinouts") {
		t.Fatalf("proposal = %s", proposal)
	}
	entries, err := os.ReadDir(filepath.Join(root, "log", "observations"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(root, "log", "observations", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "method: eval") && strings.Contains(string(data), "ablate") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected ablation observation")
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
