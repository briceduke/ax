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
	"github.com/briceduke/ax/internal/upstream"
	"github.com/briceduke/ax/internal/version"
)

const defaultRemote = "https://github.com/briceduke/ax.git"

// Fetcher copies ax machinery into a directory that has a VERSION file.
type Fetcher func() (dir string, err error)

// Commander runs an external program. Tests inject a fake.
type Commander func(args ...string) (string, error)

// Options control where machinery is copied from and optional issue filing.
type Options struct {
	From    string
	Submit  bool
	Now     time.Time
	Fetcher Fetcher
	Command Commander
}

// GitFetch clones remote with git. Tests inject run instead of calling git.
func GitFetch(remote string, run func(dir, name string, args ...string) (string, error)) Fetcher {
	if remote == "" {
		remote = defaultRemote
	}
	return func() (string, error) {
		dest, err := os.MkdirTemp("", "ax-upgrade-")
		if err != nil {
			return "", err
		}
		if err := os.RemoveAll(dest); err != nil {
			return "", err
		}
		if _, err := run("", "git", "clone", "--depth", "1", remote, dest); err != nil {
			return "", fmt.Errorf("git fetch %s: %w", remote, err)
		}
		return dest, nil
	}
}

// Run bumps the pin, recompiles, runs checks, and writes a local report.
// Nothing is merged remotely. --submit may file a GitHub issue via gh.
func Run(root string, opts Options, out io.Writer) error {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	cfg, err := project.Load(root)
	if err != nil {
		return err
	}
	old := cfg.Ax
	from := opts.From
	note := ""
	if from == "" {
		if opts.Fetcher == nil {
			return fmt.Errorf("upgrade needs --from <dir> or a git fetcher")
		}
		dir, err := opts.Fetcher()
		if err != nil {
			return err
		}
		from = dir
		note = "fetched machinery from git"
	}
	pin, err := readVersion(from)
	if err != nil {
		return err
	}
	newPin := pin
	if err := copyMachinery(from, root); err != nil {
		return err
	}
	if note == "" {
		note = "copied machinery from " + from
	}
	cfg.Ax = newPin
	if err := project.Save(root, cfg); err != nil {
		return err
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
	if opts.Submit {
		if opts.Command == nil {
			return fmt.Errorf("--submit requires gh on PATH (local report was written)")
		}
		url, err := upstream.GHIssue(upstream.Commander(opts.Command), "").Submit("ax upgrade to "+newPin, report)
		if err != nil {
			if checkErr != nil {
				return checkErr
			}
			return err
		}
		fmt.Fprintln(out, url)
	}
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
	for _, rel := range []string{"capabilities", "internal/builtins/capabilities"} {
		if err := copyCaps(filepath.Join(from, filepath.FromSlash(rel)), filepath.Join(root, "core", "capabilities")); err != nil {
			return err
		}
	}
	return nil
}

func copyCaps(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
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
