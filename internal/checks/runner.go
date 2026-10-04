package checks

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/briceduke/ax/internal/logbook"
)

// ErrFailed means at least one check failed.
var ErrFailed = errors.New("checks failed")

// Options control which checks run and whether the cache is used.
type Options struct {
	Tier string
	All  bool
}

type cacheFile struct {
	Hashes map[string]string `json:"hashes"`
}

func sha256Writer() hash.Hash {
	return sha256.New()
}

func hexSum(h hash.Hash) string {
	return hex.EncodeToString(h.Sum(nil))
}

func cachePath(root string) string {
	return filepath.Join(root, ".ax", "cache", "checks.json")
}

func loadCache(root string) (*cacheFile, error) {
	data, err := os.ReadFile(cachePath(root))
	if os.IsNotExist(err) {
		return &cacheFile{Hashes: map[string]string{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var c cacheFile
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("cache: %w", err)
	}
	if c.Hashes == nil {
		c.Hashes = map[string]string{}
	}
	return &c, nil
}

func saveCache(root string, c *cacheFile) error {
	dir := filepath.Dir(cachePath(root))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath(root), append(data, '\n'), 0644)
}

// Run executes selected checks and writes ok/skip/FAIL lines to out.
func Run(root string, opts Options, out io.Writer) error {
	if opts.Tier == "" {
		opts.Tier = TierFull
	}
	all, err := LoadRegistry(root)
	if err != nil {
		return err
	}
	selected, err := Select(all, opts.Tier)
	if err != nil {
		return err
	}
	cache, err := loadCache(root)
	if err != nil {
		return err
	}
	failed := false
	for _, check := range selected {
		status, detail := runOne(root, check, opts, cache)
		fmt.Fprintf(out, "%-4s %s\n", status, check.ID)
		if status == "FAIL" {
			failed = true
			writeFailure(out, root, check, detail)
		}
	}
	if failed {
		return ErrFailed
	}
	return nil
}

func runOne(root string, check Check, opts Options, cache *cacheFile) (string, string) {
	if check.Builtin != "" {
		if err := runBuiltin(root, check.Builtin); err != nil {
			return "FAIL", err.Error()
		}
		return "ok", ""
	}
	sum, err := hashCheck(root, check)
	if err != nil {
		return "FAIL", err.Error()
	}
	if !opts.All && cache.Hashes[check.ID] == sum {
		return "skip", ""
	}
	output, err := execRun(root, check.Run)
	if err != nil {
		detail := strings.TrimSpace(output)
		if detail == "" {
			detail = err.Error()
		} else {
			detail = detail + "\n" + err.Error()
		}
		return "FAIL", detail
	}
	cache.Hashes[check.ID] = sum
	if err := saveCache(root, cache); err != nil {
		return "FAIL", err.Error()
	}
	return "ok", ""
}

func execRun(root, run string) (string, error) {
	argv := strings.Fields(run)
	if len(argv) == 0 {
		return "", fmt.Errorf("empty run")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = root
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeFailure(out io.Writer, root string, check Check, detail string) {
	if tail := tailLines(detail, 20); tail != "" {
		fmt.Fprintln(out, tail)
	}
	if check.Enforces == "" {
		return
	}
	fmt.Fprintf(out, "enforces %s\n", check.Enforces)
	why := loadWhy(root, check.Enforces)
	if why == "" {
		return
	}
	fmt.Fprintf(out, "## Why\n%s\n", why)
}

func loadWhy(root, enforces string) string {
	rel := strings.TrimSuffix(filepath.ToSlash(enforces), ".md") + ".md"
	path := filepath.Join(root, filepath.FromSlash(rel))
	d, err := logbook.ParseDecision(path)
	if err != nil {
		return ""
	}
	return logbook.ExtractSection(d.Body, "Why")
}

func tailLines(s string, n int) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
