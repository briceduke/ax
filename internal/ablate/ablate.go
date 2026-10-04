package ablate

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/compile"
	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/eval"
	"github.com/briceduke/ax/internal/logbook"
)

// Options control ablation timing and graders.
type Options struct {
	Now      time.Time
	Grader   eval.Grader
	RunCheck eval.CheckRunner
	Runs     int
}

// Run compiles without each scaffold, scores related evals, and proposes retirement.
func Run(root string, opts Options, out io.Writer) error {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	scaffolds, err := core.LoadScaffolds(root, nil)
	if err != nil {
		return err
	}
	evals, err := eval.LoadAll(root)
	if err != nil {
		return err
	}
	var lines []string
	for _, sc := range scaffolds {
		related := relatedTo(evals, sc.ID)
		if len(related) == 0 {
			fmt.Fprintf(out, "skip %s (no related evals)\n", sc.ID)
			continue
		}
		withScores, err := scoreEvals(root, related, opts, root)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := compile.Run(root, compile.Options{Without: sc.ID}, &buf); err != nil {
			return err
		}
		work := compile.WorktreeDir(root, sc.ID)
		withoutScores, err := scoreEvals(work, related, opts, root)
		if err != nil {
			return err
		}
		retire := withoutAtLeast(withScores, withoutScores)
		finding := fmt.Sprintf("ablate %s with=%s without=%s retire=%t", sc.ID, joinScores(withScores), joinScores(withoutScores), retire)
		if _, err := logbook.WriteObservation(root, finding, eval.MethodEval, opts.Now); err != nil {
			return err
		}
		fmt.Fprintf(out, "%-4s %s with=%s without=%s\n", retireStatus(retire), sc.ID, joinScores(withScores), joinScores(withoutScores))
		if retire {
			lines = append(lines, fmt.Sprintf("- Retire scaffold %s: scores without it are at least as good.", sc.ID))
		} else {
			lines = append(lines, fmt.Sprintf("- Keep scaffold %s: scores dropped without it.", sc.ID))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "- No scaffolds had related evals.")
	}
	report := "# Ablation\n\nA human still decides. Nothing was deleted.\n\n" + joinLines(lines) + "\n"
	path := filepath.Join(root, ".ax", "ablate", "proposal.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(report), 0644); err != nil {
		return err
	}
	fmt.Fprintln(out, ".ax/ablate/proposal.md")
	return nil
}

func relatedTo(evals []*eval.Eval, scaffoldID string) []*eval.Eval {
	var out []*eval.Eval
	for _, ev := range evals {
		if eval.Relates(ev, "scaffold", scaffoldID) {
			out = append(out, ev)
		}
	}
	return out
}

func scoreEvals(gradeRoot string, evals []*eval.Eval, opts Options, logRoot string) ([]eval.Result, error) {
	ids := map[string]struct{}{}
	for _, ev := range evals {
		ids[ev.ID] = struct{}{}
	}
	var buf bytes.Buffer
	return eval.Run(gradeRoot, eval.Options{
		Runs:     opts.Runs,
		Now:      opts.Now,
		Grader:   opts.Grader,
		RunCheck: opts.RunCheck,
		LogRoot:  logRoot,
		SkipLog:  true,
		Filter: func(ev *eval.Eval) bool {
			_, ok := ids[ev.ID]
			return ok
		},
	}, &buf)
}

func withoutAtLeast(with, without []eval.Result) bool {
	byID := map[string]eval.Result{}
	for _, r := range with {
		byID[r.ID] = r
	}
	if len(without) == 0 {
		return false
	}
	for _, w := range without {
		base, ok := byID[w.ID]
		if !ok {
			return false
		}
		if w.Passed < base.Passed {
			return false
		}
	}
	return true
}

func joinScores(results []eval.Result) string {
	parts := make([]string, 0, len(results))
	for _, r := range results {
		parts = append(parts, r.ID+"="+r.Score())
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ",")
}

func joinLines(lines []string) string {
	out := ""
	for _, line := range lines {
		out += line + "\n"
	}
	return out
}

func retireStatus(retire bool) string {
	if retire {
		return "drop"
	}
	return "keep"
}
