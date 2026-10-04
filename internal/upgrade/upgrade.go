package upgrade

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/checks"
	"github.com/briceduke/ax/internal/compile"
	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/version"
)

// Options control where machinery is copied from.
type Options struct {
	From string
	Now  time.Time
}

// Run bumps the pin when --from is set, recompiles, runs checks, and writes a local report.
func Run(root string, opts Options, out io.Writer) error {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	cfg, err := project.Load(root)
	if err != nil {
		return err
	}
	old := cfg.Ax
	note := "fetch is not implemented; this binary is " + version.Version + "."
	newPin := old
	if opts.From != "" {
		pin, err := readVersion(opts.From)
		if err != nil {
			return err
		}
		newPin = pin
		if err := copyMachinery(opts.From, root); err != nil {
			return err
		}
		note = "copied machinery from " + opts.From
		cfg.Ax = newPin
		if err := project.Save(root, cfg); err != nil {
			return err
		}
	}
	var compileBuf strings.Builder
	if err := compile.Run(root, compile.Options{}, &compileBuf); err != nil {
		return err
	}
	var checkBuf strings.Builder
	checkErr := checks.Run(root, checks.Options{Tier: checks.TierFull, All: true}, &checkBuf)
	checkStatus := "ok"
	if checkErr != nil {
		checkStatus = "FAIL"
	}
	report := fmt.Sprintf(`# Upgrade report

date: %s
old pin: %s
new pin: %s
binary: %s
note: %s
compile:
%s
check: %s
%s
A person still approves. Nothing was merged remotely.
`, opts.Now.Format("2006-01-02"), old, newPin, version.Version, note, indent(compileBuf.String()), checkStatus, indent(checkBuf.String()))
	path := filepath.Join(root, ".ax", "upgrade", "report.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(report), 0644); err != nil {
		return err
	}
	fmt.Fprintln(out, ".ax/upgrade/report.md")
	return checkErr
}

func readVersion(from string) (string, error) {
	for _, name := range []string{"VERSION", "version"} {
		data, err := os.ReadFile(filepath.Join(from, name))
		if err == nil {
			v := strings.TrimSpace(string(data))
			if v == "" {
				return "", fmt.Errorf("%s: empty version", name)
			}
			return v, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("no VERSION file in %s", from)
}

func copyMachinery(from, root string) error {
	src := filepath.Join(from, "capabilities")
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	dest := filepath.Join(root, "core", "capabilities")
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		target := filepath.Join(dest, e.Name())
		if _, err := os.Stat(target); err == nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

func indent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "  (none)"
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}
