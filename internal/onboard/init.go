package onboard

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/builtins"
	"github.com/briceduke/ax/internal/compile"
	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/logbook"
	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/version"
)

// InitOptions control non-interactive project creation.
type InitOptions struct {
	Name    string
	Intent  string
	Targets string
	Now     time.Time
	Stdin   io.Reader
	IsTTY   bool
}

var treeDirs = []string{
	"core/capabilities",
	"core/scaffolds",
	"core/evals",
	"log/decisions",
	"log/observations",
	"log/friction",
}

// Init writes the minimum project tree, the first decision, and compiles.
func Init(root string, opts InitOptions, out io.Writer) error {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if err := fillInit(&opts); err != nil {
		return err
	}
	for _, dir := range treeDirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0755); err != nil {
			return err
		}
	}
	cfg := &project.Config{
		Ax:      version.Version,
		Targets: splitCSV(opts.Targets),
		Packs:   []string{},
	}
	if err := project.Save(root, cfg); err != nil {
		return err
	}
	ignore, err := builtins.Gitignore()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), ignore, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "core", "intent.md"), []byte(intentDoc(opts.Name, opts.Intent)), 0644); err != nil {
		return err
	}
	cap, err := builtins.RecordDecision()
	if err != nil {
		return err
	}
	if err := writeCapability(root, cap); err != nil {
		return err
	}
	rel, err := logbook.WriteDecision(root, "adopt ax for harness management", "", adoptBody(), opts.Now)
	if err != nil {
		return err
	}
	checksYAML, err := builtins.ChecksYAML(rel)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "core", "checks.yaml"), checksYAML, 0644); err != nil {
		return err
	}
	if err := compile.Run(root, compile.Options{}, out); err != nil {
		return err
	}
	fmt.Fprintln(out, "initialized", root)
	return nil
}

func fillInit(opts *InitOptions) error {
	if opts.Name != "" && opts.Intent != "" {
		if opts.Targets == "" {
			opts.Targets = "cursor,claude-code"
		}
		return nil
	}
	if !opts.IsTTY {
		if opts.Name == "" || opts.Intent == "" {
			return fmt.Errorf("ax init requires --name and --intent when stdin is not a terminal")
		}
	}
	in := opts.Stdin
	if in == nil {
		in = os.Stdin
	}
	r := bufio.NewReader(in)
	if opts.Name == "" {
		fmt.Fprint(os.Stderr, "Project name: ")
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		opts.Name = strings.TrimSpace(line)
	}
	if opts.Intent == "" {
		fmt.Fprint(os.Stderr, "What are you building? ")
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		opts.Intent = strings.TrimSpace(line)
	}
	if opts.Targets == "" {
		opts.Targets = "cursor,claude-code"
	}
	if opts.Name == "" || opts.Intent == "" {
		return fmt.Errorf("ax init needs a name and a one-line intent")
	}
	return nil
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func intentDoc(name, intent string) string {
	return fmt.Sprintf(`# Intent

## What is being built

%s

## For whom

%s users.

## Values

Correctness.

## Taste

Short files.

## Process weight

Low.
`, strings.TrimSpace(intent), strings.TrimSpace(name))
}

func adoptBody() string {
	return `## Context

This repo needs one place to write decisions and generate editor files.

## Options

- Ad-hoc markdown rules
- Adopt ax for harness management

## Choice

Adopt ax.

## Why

Checks replace prose rules and every failure points at this decision.
`
}

func writeCapability(root string, cap *core.Capability) error {
	data, err := builtins.CapabilityFile(cap.ID)
	if err != nil {
		return err
	}
	path := filepath.Join(root, "core", "capabilities", cap.ID+".md")
	return os.WriteFile(path, data, 0644)
}
