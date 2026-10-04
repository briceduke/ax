package onboard

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/briceduke/ax/internal/checks"
	"github.com/briceduke/ax/internal/project"
)

// Doctor verifies ax.yaml, runs checks, and confirms ax is on PATH.
// Docker is optional and never fails the command when missing.
func Doctor(root string, out io.Writer) error {
	if _, err := project.Load(root); err != nil {
		return err
	}
	fmt.Fprintln(out, "ok   ax.yaml")
	if err := checks.Run(root, checks.Options{Tier: checks.TierFull}, out); err != nil {
		return err
	}
	if _, err := exec.LookPath("ax"); err != nil {
		return fmt.Errorf("ax is not on PATH")
	}
	fmt.Fprintln(out, "ok   PATH ax")
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintln(out, "skip docker (optional)")
		return nil
	}
	fmt.Fprintln(out, "ok   docker")
	return nil
}
