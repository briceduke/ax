package eval

import (
	"fmt"
	"io"
	"time"

	"github.com/briceduke/ax/internal/logbook"
)

// Runner executes an eval task in the project directory. Tests inject a fake.
type Runner interface {
	Run(dir, task string) (string, error)
}

// Options control which evals run and where scores are logged.
type Options struct {
	Without  string
	Runs     int
	Now      time.Time
	Grader   Grader
	Runner   Runner
	RunCheck CheckRunner
	LogRoot  string
	SkipLog  bool
	Filter   func(*Eval) bool
}

// Run grades every eval under root and writes one observation per eval.
func Run(root string, opts Options, out io.Writer) ([]Result, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	logRoot := opts.LogRoot
	if logRoot == "" {
		logRoot = root
	}
	evals, err := LoadAll(root)
	if err != nil {
		return nil, err
	}
	var results []Result
	for _, ev := range evals {
		if opts.Filter != nil && !opts.Filter(ev) {
			continue
		}
		res, err := runOne(root, ev, opts)
		if err != nil {
			return results, err
		}
		fmt.Fprintf(out, "%-4s %s %s\n", status(res), ev.ID, res.Score())
		if !opts.SkipLog {
			finding := fmt.Sprintf("eval %s %s", ev.ID, res.Score())
			if res.Skipped {
				finding = fmt.Sprintf("eval %s skipped: %s", ev.ID, res.Detail)
			}
			if _, err := logbook.WriteObservation(logRoot, finding, MethodEval, opts.Now); err != nil {
				return results, err
			}
		}
		results = append(results, res)
	}
	return results, nil
}

func runOne(root string, ev *Eval, opts Options) (Result, error) {
	runs := ev.Runs
	if opts.Runs > 0 {
		runs = opts.Runs
	}
	if ev.Grader.Kind == KindRubric && opts.Runner == nil && opts.Grader == nil {
		return Result{
			ID:      ev.ID,
			Passed:  0,
			Runs:    runs,
			Skipped: true,
			Detail:  "no agent CLI (agent / cursor agent / claude) available",
		}, nil
	}
	grader := opts.Grader
	if grader == nil {
		g, err := defaultGrader(ev.Grader.Kind, opts.RunCheck)
		if err != nil {
			return Result{}, err
		}
		grader = g
	}
	passed := 0
	detail := ""
	for i := 0; i < runs; i++ {
		if opts.Runner != nil {
			if _, err := opts.Runner.Run(root, ev.Task); err != nil {
				detail = err.Error()
				continue
			}
		}
		if err := grader.Grade(root, ev); err != nil {
			detail = err.Error()
			continue
		}
		passed++
	}
	return Result{ID: ev.ID, Passed: passed, Runs: runs, Detail: detail}, nil
}

func status(res Result) string {
	if res.Skipped {
		return "skip"
	}
	if res.Passed == res.Runs {
		return "ok"
	}
	if res.Passed == 0 {
		return "FAIL"
	}
	return "mix"
}
