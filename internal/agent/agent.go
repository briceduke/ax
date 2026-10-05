package agent

import (
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes a headless agent task in dir.
type Runner interface {
	Run(dir, task string) (string, error)
}

// LookPath is exec.LookPath, injectable in tests.
type LookPath func(file string) (string, error)

// CmdRun runs a process. Injectable in tests.
type CmdRun func(dir, name string, args ...string) (string, error)

// CLI runs a local agent/cursor/claude binary with -p (print) mode.
type CLI struct {
	Name string
	Args []string
	Exec CmdRun
}

// Run executes the CLI in dir with the task as the prompt argument.
func (c CLI) Run(dir, task string) (string, error) {
	run := c.Exec
	if run == nil {
		run = SystemRun
	}
	args := append(append([]string{}, c.Args...), task)
	out, err := run(dir, c.Name, args...)
	if err != nil {
		return out, fmt.Errorf("%s: %w", c.Name, err)
	}
	return out, nil
}

// SystemRun is the real exec.Command CombinedOutput helper.
func SystemRun(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Detect returns a headless runner if agent, cursor, or claude is on PATH.
func Detect() Runner {
	return DetectWith(exec.LookPath, SystemRun)
}

// DetectWith is Detect with injected lookup and exec.
func DetectWith(look LookPath, run CmdRun) Runner {
	if look == nil {
		look = exec.LookPath
	}
	if run == nil {
		run = SystemRun
	}
	if _, err := look("agent"); err == nil {
		return CLI{Name: "agent", Args: []string{"-p"}, Exec: run}
	}
	if _, err := look("cursor"); err == nil {
		return CLI{Name: "cursor", Args: []string{"agent", "-p"}, Exec: run}
	}
	if _, err := look("claude"); err == nil {
		return CLI{Name: "claude", Args: []string{"-p"}, Exec: run}
	}
	return nil
}
