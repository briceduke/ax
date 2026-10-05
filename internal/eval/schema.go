package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/yamlx"
)

const (
	KindScript = "script"
	KindRubric = "rubric"
	MethodEval = "eval"
)

// Eval is one harness test under .ax/evals.
type Eval struct {
	ID      string     `yaml:"id"`
	Related []string   `yaml:"related"`
	Runs    int        `yaml:"runs"`
	Task    string     `yaml:"-"`
	Grader  GraderSpec `yaml:"-"`
}

// GraderSpec is the YAML grader block in an eval body.
type GraderSpec struct {
	Kind   string       `yaml:"kind"`
	Must   []string     `yaml:"must,omitempty"`
	Check  string       `yaml:"check,omitempty"`
	File   string       `yaml:"file,omitempty"`
	Number *NumberBound `yaml:"number,omitempty"`
}

// NumberBound is a script grader that checks a file's first number.
type NumberBound struct {
	File string  `yaml:"file"`
	Min  float64 `yaml:"min"`
	Max  float64 `yaml:"max"`
}

type evalBody struct {
	Task   string     `yaml:"task"`
	Grader GraderSpec `yaml:"grader"`
}

// ParseFile reads and validates one eval markdown file.
func ParseFile(path string) (*Eval, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseBytes(data)
}

// ParseBytes validates eval markdown bytes.
func ParseBytes(data []byte) (*Eval, error) {
	var ev Eval
	body, err := core.DecodeFrontmatter(data, &ev)
	if err != nil {
		return nil, err
	}
	var spec evalBody
	if err := yamlx.Decode([]byte(body), &spec); err != nil {
		return nil, fmt.Errorf("eval body: %w", err)
	}
	ev.Task = spec.Task
	ev.Grader = spec.Grader
	if ev.Related == nil {
		ev.Related = []string{}
	}
	return &ev, validateEval(&ev)
}

func validateEval(ev *Eval) error {
	if ev.ID == "" {
		return fmt.Errorf("missing id")
	}
	if ev.Runs < 1 {
		return fmt.Errorf("runs must be at least 1")
	}
	if strings.TrimSpace(ev.Task) == "" {
		return fmt.Errorf("missing task")
	}
	switch ev.Grader.Kind {
	case KindScript:
		if ev.Grader.Check == "" && ev.Grader.File == "" && ev.Grader.Number == nil {
			return fmt.Errorf("script grader needs check, file, or number")
		}
		if ev.Grader.Number != nil && ev.Grader.Number.File == "" {
			return fmt.Errorf("number grader missing file")
		}
	case KindRubric:
		if len(ev.Grader.Must) == 0 {
			return fmt.Errorf("rubric grader needs must")
		}
	default:
		return fmt.Errorf("unknown grader kind %q", ev.Grader.Kind)
	}
	return nil
}

// LoadAll parses every eval under .ax/evals.
func LoadAll(root string) ([]*Eval, error) {
	rels, err := core.ListMarkdown(root, project.Rel("evals"))
	if err != nil {
		return nil, err
	}
	out := make([]*Eval, 0, len(rels))
	for _, rel := range rels {
		ev, err := ParseFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		out = append(out, ev)
	}
	return out, nil
}

// Relates reports whether ev is tied to a capability or scaffold id.
func Relates(ev *Eval, kind, id string) bool {
	want := kind + "/" + id
	for _, r := range ev.Related {
		if r == want || r == id {
			return true
		}
	}
	return false
}
