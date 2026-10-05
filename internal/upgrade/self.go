package upgrade

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/briceduke/ax/internal/version"
)

func replaceExe(current string, data []byte) error {
	if current == "" {
		return fmt.Errorf("empty executable path")
	}
	if len(data) == 0 {
		return fmt.Errorf("empty binary")
	}
	dir := filepath.Dir(current)
	next := filepath.Join(dir, filepath.Base(current)+".new")
	old := current + ".old"
	if err := os.WriteFile(next, data, 0755); err != nil {
		return err
	}
	_ = os.Remove(old)
	if err := os.Rename(current, old); err != nil {
		_ = os.Remove(next)
		return err
	}
	if err := os.Rename(next, current); err != nil {
		_ = os.Rename(old, current)
		return err
	}
	_ = os.Remove(old)
	return nil
}

func currentExe(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	return os.Executable()
}

func installBinary(opts Options, ver string) (string, error) {
	if opts.SkipBinary || opts.Get == nil {
		return "skipped", nil
	}
	if ver == version.Version {
		return "already " + ver, nil
	}
	data, err := downloadRelease(opts.Get, ver)
	if err != nil {
		return "", err
	}
	exe, err := currentExe(opts.CurrentExe)
	if err != nil {
		return "", err
	}
	if err := replaceExe(exe, data); err != nil {
		return "", err
	}
	return "replaced " + ver, nil
}

// Self replaces this ax binary with the latest GitHub Release. No project needed.
func Self(opts Options, out io.Writer) error {
	if opts.Get == nil {
		return fmt.Errorf("upgrade needs a downloader")
	}
	tag, err := LatestTag(opts.Get)
	if err != nil {
		return err
	}
	ver := pinFromTag(tag)
	note, err := installBinary(opts, ver)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, note)
	return nil
}
