package retro

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/eval"
	"github.com/briceduke/ax/internal/logbook"
)

const (
	headingProblems      = "## Problems"
	headingProposedEvals = "## Proposed evals"
	headingCoreChanges   = "## Core changes"
	headingDeletions     = "## Deletions"
)

// Options control retro input and output.
type Options struct {
	Now time.Time
}

// Proposal is a human-reviewed change set. Nothing here is merged automatically.
type Proposal struct {
	ID       string
	Date     string
	Rel      string
	Problems []string
	Evals    []string
	Changes  []string
	Deletes  []string
}

// Run writes a dated proposal under .ax/retro/.
func Run(root string, opts Options, out io.Writer) (*Proposal, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	p, err := build(root, opts.Now)
	if err != nil {
		return nil, err
	}
	body := render(p)
	if err := ValidateProposal([]byte(body)); err != nil {
		return nil, err
	}
	path := filepath.Join(root, filepath.FromSlash(p.Rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return nil, err
	}
	fmt.Fprintln(out, p.Rel)
	return p, nil
}

func build(root string, now time.Time) (*Proposal, error) {
	since, err := lastRetroDate(root)
	if err != nil {
		return nil, err
	}
	friction, err := logbook.LoadFriction(root)
	if err != nil {
		return nil, err
	}
	recent := filterSince(friction, since)
	clusters := clusterFriction(recent)
	evals, err := eval.LoadAll(root)
	if err != nil {
		return nil, err
	}
	obs, err := logbook.LoadObservations(root)
	if err != nil {
		return nil, err
	}
	scores := latestEvalScores(obs)
	scaffolds, err := core.LoadScaffolds(root, nil)
	if err != nil {
		return nil, err
	}
	date := now.Format("2006-01-02")
	p := &Proposal{
		ID:   date + "-retro",
		Date: date,
		Rel:  ".ax/retro/" + date + "-proposal.md",
	}
	if len(clusters) == 0 {
		p.Problems = []string{"- No new friction since the last retro."}
	} else {
		for _, c := range clusters {
			p.Problems = append(p.Problems, fmt.Sprintf("- %s (%d notes)", c.label, c.count))
		}
	}
	p.Evals = proposedEvals(clusters, evals)
	p.Changes = coreChanges(scores, scaffolds)
	p.Deletes = deletions(scaffolds, scores)
	return p, nil
}

type cluster struct {
	label string
	count int
	key   string
}

func filterSince(all []*logbook.Friction, since string) []*logbook.Friction {
	if since == "" {
		return all
	}
	var out []*logbook.Friction
	for _, f := range all {
		if f.Date > since {
			out = append(out, f)
		}
	}
	return out
}

func clusterFriction(items []*logbook.Friction) []cluster {
	groups := map[string][]*logbook.Friction{}
	order := []string{}
	for _, f := range items {
		key := signature(f.Body)
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], f)
	}
	out := make([]cluster, 0, len(order))
	for _, key := range order {
		list := groups[key]
		out = append(out, cluster{label: list[0].Body, count: len(list), key: key})
	}
	return out
}

func signature(text string) string {
	words := strings.Fields(strings.ToLower(text))
	seen := map[string]struct{}{}
	var toks []string
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?")
		if len(w) < 4 {
			continue
		}
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		toks = append(toks, w)
	}
	sort.Strings(toks)
	if len(toks) == 0 {
		return strings.TrimSpace(strings.ToLower(text))
	}
	return strings.Join(toks, " ")
}

func proposedEvals(clusters []cluster, evals []*eval.Eval) []string {
	var out []string
	for _, c := range clusters {
		if c.count < 2 {
			continue
		}
		if covered(c, evals) {
			continue
		}
		out = append(out, fmt.Sprintf("- Recurring friction %q has no eval; add a script grader for it.", c.label))
	}
	if len(out) == 0 {
		return []string{"- None. Existing evals cover the recurring friction, or there is not enough repetition yet."}
	}
	return out
}

func covered(c cluster, evals []*eval.Eval) bool {
	needles := strings.Fields(c.key)
	for _, ev := range evals {
		blob := strings.ToLower(ev.ID + " " + ev.Task + " " + strings.Join(ev.Related, " "))
		for _, n := range needles {
			if strings.Contains(blob, n) {
				return true
			}
		}
	}
	return false
}

func latestEvalScores(obs []*logbook.Observation) map[string]string {
	out := map[string]string{}
	for _, o := range obs {
		if o.Method != eval.MethodEval {
			continue
		}
		id, score, ok := parseEvalFinding(o.Body)
		if !ok {
			continue
		}
		out[id] = score
	}
	return out
}

func parseEvalFinding(body string) (string, string, bool) {
	fields := strings.Fields(strings.TrimSpace(body))
	if len(fields) < 3 || fields[0] != "eval" {
		return "", "", false
	}
	return fields[1], fields[2], true
}

func coreChanges(scores map[string]string, scaffolds []*core.Scaffold) []string {
	if len(scores) == 0 {
		return []string{"- No eval scores yet. Keep core as-is until ax eval writes observations."}
	}
	lines := []string{"- Keep instructions that still have a failing related eval."}
	if len(scaffolds) > 0 {
		lines = append(lines, "- Prefer deleting a scaffold over adding a new one.")
	}
	return lines
}

func deletions(scaffolds []*core.Scaffold, scores map[string]string) []string {
	if len(scaffolds) > 0 {
		sc := scaffolds[0]
		note := fmt.Sprintf("- Delete scaffold %s (%s).", sc.ID, sc.RetireWhen)
		if score, ok := scores["harness-format"]; ok {
			note = fmt.Sprintf("- Delete scaffold %s; latest harness-format score is %s.", sc.ID, score)
		}
		return []string{note}
	}
	return []string{"- Delete the shortest always-on how-we-work note that duplicates a check."}
}

func lastRetroDate(root string) (string, error) {
	dir := filepath.Join(root, ".ax", "retro")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	latest := ""
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if len(e.Name()) < 10 {
			continue
		}
		d := e.Name()[:10]
		if _, err := time.Parse("2006-01-02", d); err != nil {
			continue
		}
		if d > latest {
			latest = d
		}
	}
	return latest, nil
}

func render(p *Proposal) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\nid: %s\ndate: %s\n---\n\n", p.ID, p.Date)
	writeSection(&b, headingProblems, p.Problems)
	writeSection(&b, headingProposedEvals, p.Evals)
	writeSection(&b, headingCoreChanges, p.Changes)
	writeSection(&b, headingDeletions, p.Deletes)
	return b.String()
}

func writeSection(b *strings.Builder, heading string, lines []string) {
	b.WriteString(heading)
	b.WriteString("\n\n")
	for _, line := range lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

type proposalFront struct {
	ID   string `yaml:"id"`
	Date string `yaml:"date"`
}

// ValidateProposal checks the required sections and at least one deletion.
func ValidateProposal(data []byte) error {
	var fm proposalFront
	body, err := core.DecodeFrontmatter(data, &fm)
	if err != nil {
		return err
	}
	if fm.ID == "" || fm.Date == "" {
		return fmt.Errorf("proposal missing id or date")
	}
	for _, h := range []string{headingProblems, headingProposedEvals, headingCoreChanges, headingDeletions} {
		if !strings.Contains(body, h) {
			return fmt.Errorf("proposal missing %s", h)
		}
	}
	del := logbook.ExtractSection(body, "Deletions")
	if !strings.Contains(del, "- ") {
		return fmt.Errorf("proposal must propose at least one deletion")
	}
	return nil
}
