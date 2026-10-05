package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/briceduke/ax/internal/version"
)

func TestCheckFixtures(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	for _, name := range []string{"software", "hardware"} {
		t.Run(name, func(t *testing.T) {
			src := filepath.Join(repo, "testdata", "fixtures", name)
			dst := t.TempDir()
			copyTree(t, src, dst)
			t.Chdir(dst)
			stdout, stderr := capture(t, func() error {
				return run([]string{"check"})
			})
			if stderr != "" && !strings.HasPrefix(stderr, "warning:") {
				t.Fatalf("stderr = %q", stderr)
			}
			if !strings.Contains(stdout, "ok   ax-format") {
				t.Fatalf("stdout = %q", stdout)
			}
		})
	}
}

func TestEvalHardwareWritesScores(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	src := filepath.Join(repo, "testdata", "fixtures", "hardware")
	dst := t.TempDir()
	copyTree(t, src, dst)
	t.Chdir(dst)
	stdout, stderr := capture(t, func() error {
		return run([]string{"eval", "--runs", "1"})
	})
	if stderr != "" && !strings.HasPrefix(stderr, "warning:") {
		t.Fatalf("stderr = %q", stderr)
	}
	if !strings.Contains(stdout, "cross-discipline-impact") || !strings.Contains(stdout, "harness-format") || !strings.Contains(stdout, "idle-current-range") {
		t.Fatalf("stdout = %q", stdout)
	}
	entries, err := os.ReadDir(filepath.Join(dst, "log", "observations"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 3 {
		t.Fatalf("got %d observations", len(entries))
	}
}

func TestVersionPrintsConstWithoutRepo(t *testing.T) {
	t.Chdir(t.TempDir())
	stdout, stderr := capture(t, func() error {
		return run([]string{"version"})
	})
	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
	if stdout != version.Version+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, version.Version+"\n")
	}
	if !strings.Contains(usage(), "ax version") {
		t.Fatal("usage missing ax version")
	}
}

func capture(t *testing.T, fn func() error) (string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rout, wout, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rerr, werr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = wout, werr
	runErr := fn()
	wout.Close()
	werr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	var outBuf, errBuf bytes.Buffer
	io.Copy(&outBuf, rout)
	io.Copy(&errBuf, rerr)
	if runErr != nil {
		t.Fatalf("run: %v\nstdout=%s\nstderr=%s", runErr, outBuf.String(), errBuf.String())
	}
	return outBuf.String(), errBuf.String()
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
