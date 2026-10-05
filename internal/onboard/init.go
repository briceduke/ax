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

// InitOptions control project creation. Flags fill name/intent/targets for tests.
type InitOptions struct {
	Name          string
	Intent        string
	Targets       string
	ForWhom       string
	Disciplines   string
	Constraints   string
	Team          string
	ProcessWeight string
	Interview     bool
	Now           time.Time
	Stdin         io.Reader
	IsTTY         bool
}

var treeDirs = []string{
	"core/capabilities",
	"core/scaffolds",
	"core/evals",
	"log/decisions",
	"log/observations",
	"log/friction",
}

const productDockerfile = `FROM debian:bookworm-slim
WORKDIR /work
COPY . .
# ax doctor bind-mounts the host ax binary at /usr/local/bin/ax
CMD ["ax", "check"]
`

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
	if err := os.WriteFile(filepath.Join(root, "Dockerfile"), []byte(productDockerfile), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "core", "intent.md"), []byte(intentDoc(opts)), 0644); err != nil {
		return err
	}
	caps, err := builtins.Capabilities()
	if err != nil {
		return err
	}
	for _, cap := range caps {
		if err := writeCapability(root, cap); err != nil {
			return err
		}
	}
	rel, err := logbook.WriteDecision(root, "adopt ax for harness management", "", adoptBody(), opts.Now)
	if err != nil {
		return err
	}
	if opts.Interview || opts.IsTTY {
		if _, err := logbook.WriteDecision(root, "choose the product toolchain", "", toolchainBody(opts), opts.Now); err != nil {
			return err
		}
		if _, err := logbook.WriteDecision(root, "set process weight", "", processBody(opts), opts.Now); err != nil {
			return err
		}
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
	if opts.Targets == "" {
		opts.Targets = "cursor,claude-code"
	}
	interactive := opts.Interview || opts.IsTTY
	if !interactive {
		if opts.Name == "" || opts.Intent == "" {
			return fmt.Errorf("ax init requires --name and --intent when stdin is not a terminal")
		}
		return nil
	}
	in := opts.Stdin
	if in == nil {
		in = os.Stdin
	}
	r := bufio.NewReader(in)
	if err := prompt(r, "Project name: ", &opts.Name); err != nil {
		return err
	}
	if err := prompt(r, "What are you building? ", &opts.Intent); err != nil {
		return err
	}
	if err := prompt(r, "For whom? ", &opts.ForWhom); err != nil {
		return err
	}
	if err := prompt(r, "Disciplines (software, hardware, ...)? ", &opts.Disciplines); err != nil {
		return err
	}
	if err := prompt(r, "Constraints? ", &opts.Constraints); err != nil {
		return err
	}
	if err := prompt(r, "Who is on the team? ", &opts.Team); err != nil {
		return err
	}
	if err := prompt(r, "Process weight (low/medium/high)? ", &opts.ProcessWeight); err != nil {
		return err
	}
	if opts.Name == "" || opts.Intent == "" {
		return fmt.Errorf("ax init needs a name and a one-line intent")
	}
	return nil
}

func prompt(r *bufio.Reader, label string, dest *string) error {
	if strings.TrimSpace(*dest) != "" {
		return nil
	}
	fmt.Fprint(os.Stderr, label)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	*dest = strings.TrimSpace(line)
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

func intentDoc(opts InitOptions) string {
	forWhom := strings.TrimSpace(opts.ForWhom)
	if forWhom == "" {
		forWhom = strings.TrimSpace(opts.Name) + " users."
	}
	disciplines := orDefault(opts.Disciplines, "unspecified")
	constraints := orDefault(opts.Constraints, "none recorded")
	team := orDefault(opts.Team, "unspecified")
	weight := orDefault(opts.ProcessWeight, "Low")
	if opts.Interview || opts.IsTTY {
		return fmt.Sprintf(`# Intent

## What is being built

%s

## For whom

%s

## Disciplines

%s

## Constraints

%s

## Team

%s

## Values

Correctness.

## Taste

Short files.

## Process weight

%s
`, strings.TrimSpace(opts.Intent), forWhom, disciplines, constraints, team, weight)
	}
	return fmt.Sprintf(`# Intent

## What is being built

%s

## For whom

%s

## Values

Correctness.

## Taste

Short files.

## Process weight

%s
`, strings.TrimSpace(opts.Intent), forWhom, weight)
}

func orDefault(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
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

func toolchainBody(opts InitOptions) string {
	disciplines := orDefault(opts.Disciplines, "unspecified")
	return fmt.Sprintf(`## Context

The first session needs a recorded toolchain so later checks have a decision to point at.

## Options

- Leave toolchain implicit
- Write the disciplines and constraints down now

## Choice

Record disciplines: %s. Constraints: %s.

## Why

Interview answers are the starting pin. Change them with a new decision, not by editing this one.
`, disciplines, orDefault(opts.Constraints, "none recorded"))
}

func processBody(opts InitOptions) string {
	weight := orDefault(opts.ProcessWeight, "Low")
	return fmt.Sprintf(`## Context

Process weight decides how much harness text the team will tolerate.

## Options

- High: many scaffolds and checks up front
- Low: add process only after something fails

## Choice

%s process weight. Team: %s.

## Why

Start light. Add a check or scaffold when friction or a failed eval says so.
`, weight, orDefault(opts.Team, "unspecified"))
}

func writeCapability(root string, cap *core.Capability) error {
	data, err := builtins.CapabilityFile(cap.ID)
	if err != nil {
		return err
	}
	path := filepath.Join(root, "core", "capabilities", cap.ID+".md")
	return os.WriteFile(path, data, 0644)
}
