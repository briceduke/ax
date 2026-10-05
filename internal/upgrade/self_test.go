package upgrade

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReplaceExe(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "ax")
	if err := os.WriteFile(cur, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := replaceExe(cur, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(cur)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("exe = %q", got)
	}
}

func TestUnpackZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("ax.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("inside")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := unpackAx(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "inside" {
		t.Fatalf("payload = %q", got)
	}
}

func TestSelfReplacesFromLatest(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "ax")
	if err := os.WriteFile(cur, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	get := func(url string) ([]byte, error) {
		if strings.Contains(url, "/releases/latest") {
			return []byte(`{"tag_name":"v9.9.9"}`), nil
		}
		if strings.Contains(url, "v9.9.9") {
			return []byte("fresh"), nil
		}
		return nil, os.ErrNotExist
	}
	var out bytes.Buffer
	err := Self(Options{Get: get, CurrentExe: cur}, &out)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(cur)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fresh" {
		t.Fatalf("exe = %q", got)
	}
	if !strings.Contains(out.String(), "replaced 9.9.9") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestUpgradeInstallsBinary(t *testing.T) {
	if runtime.GOOS == "" {
		t.Fatal("goos")
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	src := filepath.Join(repo, "testdata", "fixtures", "software")
	root := t.TempDir()
	copyTestTree(t, src, root)
	from := t.TempDir()
	if err := os.WriteFile(filepath.Join(from, "VERSION"), []byte("9.9.9\n"), 0644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "ax")
	if err := os.WriteFile(exe, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	get := func(url string) ([]byte, error) {
		if strings.Contains(url, "9.9.9") {
			return []byte("fresh"), nil
		}
		return nil, os.ErrNotExist
	}
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	if err := Run(root, Options{From: from, Now: now, Get: get, CurrentExe: exe}, &buf); err != nil {
		t.Fatalf("upgrade: %v\n%s", err, buf.String())
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fresh" {
		t.Fatalf("exe = %q", got)
	}
	report, err := os.ReadFile(filepath.Join(root, ".ax", "upgrade", "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "binary: replaced 9.9.9") {
		t.Fatalf("report = %s", report)
	}
}
