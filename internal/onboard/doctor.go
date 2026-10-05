package onboard

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/briceduke/ax/internal/checks"
	"github.com/briceduke/ax/internal/project"
)

// DoctorOptions control PATH lookup, docker, and whether a container is required.
type DoctorOptions struct {
	RequireContainer bool
	LookPath         func(file string) (string, error)
	Run              func(dir, name string, args ...string) (string, error)
}

// Doctor verifies ax.yaml, runs checks, and confirms ax is on PATH.
// Missing Docker passes with "container not verified" unless RequireContainer is set.
func Doctor(root string, opts DoctorOptions, out io.Writer) error {
	look := opts.LookPath
	if look == nil {
		look = exec.LookPath
	}
	run := opts.Run
	if run == nil {
		run = func(dir, name string, args ...string) (string, error) {
			cmd := exec.Command(name, args...)
			cmd.Dir = dir
			b, err := cmd.CombinedOutput()
			return string(b), err
		}
	}
	if _, err := project.Load(root); err != nil {
		return err
	}
	fmt.Fprintln(out, "ok   ax.yaml")
	if err := checks.Run(root, checks.Options{Tier: checks.TierFull}, out); err != nil {
		return err
	}
	if _, err := look("ax"); err != nil {
		return fmt.Errorf("ax is not on PATH")
	}
	fmt.Fprintln(out, "ok   PATH ax")
	return checkContainer(root, opts.RequireContainer, look, run, out)
}

func checkContainer(root string, require bool, look func(string) (string, error), run func(dir, name string, args ...string) (string, error), out io.Writer) error {
	unverified := func(reason string) error {
		fmt.Fprintln(out, "container not verified ("+reason+")")
		if require {
			return fmt.Errorf("container not verified (%s)", reason)
		}
		return nil
	}
	if _, err := look("docker"); err != nil {
		return unverified("docker not found")
	}
	if _, err := os.Stat(filepath.Join(root, "Dockerfile")); err != nil {
		return unverified("no Dockerfile")
	}
	axBin, err := look("ax")
	if err != nil {
		return unverified("ax not on PATH for bind-mount")
	}
	if _, err := run(root, "docker", "build", "-t", "ax-doctor", "."); err != nil {
		if require {
			return fmt.Errorf("docker build: %w", err)
		}
		return unverified("docker build failed")
	}
	args := []string{"run", "--rm", "-v", root + ":/work", "-v", axBin + ":/usr/local/bin/ax", "-w", "/work", "ax-doctor", "ax", "check"}
	if runtime.GOOS != "windows" {
		args = []string{"run", "--rm", "-v", root + ":/work", "-v", axBin + ":/usr/local/bin/ax:ro", "-w", "/work", "ax-doctor", "ax", "check"}
	}
	if _, err := run(root, "docker", args...); err != nil {
		return fmt.Errorf("docker run ax check: %w", err)
	}
	fmt.Fprintln(out, "ok   container")
	return nil
}
