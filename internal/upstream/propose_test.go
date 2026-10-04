package upstream

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/briceduke/ax/internal/logbook"
)

func TestSanitizePrivatePathsAndSecrets(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	if _, err := logbook.WriteFriction(root, "token ghp_notarealtoken leaked from C:\\secret-lab", now); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".ax"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ax", "private.txt"), []byte("C:\\secret-lab\n"), 0644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err = Run(root, Options{
		Layer:       "schema",
		Change:      "remove " + abs + " from docs; token ghp_notarealtoken",
		HelpsOthers: true,
		Now:         now,
	}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	issue, err := os.ReadFile(filepath.Join(root, ".ax", "upstream", "issue.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(issue)
	if strings.Contains(text, abs) {
		t.Fatalf("absolute path leaked:\n%s", text)
	}
	if strings.Contains(text, "ghp_notarealtoken") {
		t.Fatalf("token leaked:\n%s", text)
	}
	if strings.Contains(text, `C:\secret-lab`) {
		t.Fatalf("private path leaked:\n%s", text)
	}
	if !strings.Contains(text, "<redacted>") || !strings.Contains(text, "<project>") {
		t.Fatalf("expected redaction markers:\n%s", text)
	}
	if !strings.Contains(text, "schema") || !strings.Contains(text, "yes") {
		t.Fatalf("missing fields:\n%s", text)
	}
}

type fakeHost struct {
	title string
	body  string
}

func (f *fakeHost) Submit(title, body string) (string, error) {
	f.title = title
	f.body = body
	return "https://example.invalid/issue/1", nil
}

func TestSubmitUsesInjectedClient(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	if _, err := logbook.WriteFriction(root, "docs were stale", now); err != nil {
		t.Fatal(err)
	}
	host := &fakeHost{}
	var buf strings.Builder
	if err := Run(root, Options{Now: now, Submitter: host, HelpsOthers: false}, &buf); err != nil {
		t.Fatal(err)
	}
	if host.title == "" || host.body == "" {
		t.Fatal("submitter was not called")
	}
	if !strings.Contains(buf.String(), "https://example.invalid/issue/1") {
		t.Fatalf("output = %q", buf.String())
	}
}
