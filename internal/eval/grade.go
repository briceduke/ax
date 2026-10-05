package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Result is the score of one eval after all runs.
type Result struct {
	ID      string
	Passed  int
	Runs    int
	Detail  string
	Skipped bool
	Retire  bool
	With    string
	Without string
}

func (r Result) Score() string {
	if r.Skipped {
		return "skipped"
	}
	return fmt.Sprintf("%d/%d", r.Passed, r.Runs)
}

// Grader scores one eval against a project root. Injected in tests; default is script/rubric stand-ins.
type Grader interface {
	Grade(root string, ev *Eval) error
}

// CheckRunner runs one named project check. Wired from main so this package does not import checks.
type CheckRunner func(root, id string) error

type scriptGrader struct {
	runCheck CheckRunner
}

func (g scriptGrader) Grade(root string, ev *Eval) error {
	spec := ev.Grader
	if spec.Check != "" {
		if g.runCheck == nil {
			return fmt.Errorf("script grader check %s: no check runner", spec.Check)
		}
		if err := g.runCheck(root, spec.Check); err != nil {
			return fmt.Errorf("check %s: %w", spec.Check, err)
		}
	}
	if spec.File != "" {
		path := filepath.Join(root, filepath.FromSlash(spec.File))
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("file %s: %w", spec.File, err)
		}
	}
	if spec.Number != nil {
		if err := checkNumber(root, spec.Number); err != nil {
			return err
		}
	}
	return nil
}

var firstNumber = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)

func checkNumber(root string, bound *NumberBound) error {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(bound.File)))
	if err != nil {
		return fmt.Errorf("number file %s: %w", bound.File, err)
	}
	s := firstNumber.FindString(string(data))
	if s == "" {
		return fmt.Errorf("number file %s: no number found", bound.File)
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("number file %s: %w", bound.File, err)
	}
	if n < bound.Min || n > bound.Max {
		return fmt.Errorf("number %v not in [%v, %v]", n, bound.Min, bound.Max)
	}
	return nil
}

type rubricGrader struct{}

func (rubricGrader) Grade(root string, ev *Eval) error {
	text, err := collectText(root)
	if err != nil {
		return err
	}
	var missing []string
	for _, must := range ev.Grader.Must {
		if !strings.Contains(text, must) {
			missing = append(missing, must)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required text: %s", strings.Join(missing, ", "))
	}
	return nil
}

func collectText(root string) (string, error) {
	var b strings.Builder
	for _, dir := range []string{"core", "log"} {
		err := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if info.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			b.Write(data)
			b.WriteByte('\n')
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

func defaultGrader(kind string, runCheck CheckRunner) (Grader, error) {
	switch kind {
	case KindScript:
		return scriptGrader{runCheck: runCheck}, nil
	case KindRubric:
		return rubricGrader{}, nil
	default:
		return nil, fmt.Errorf("unknown grader kind %q", kind)
	}
}
