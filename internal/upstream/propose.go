package upstream

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/logbook"
)

var layers = map[string]struct{}{
	"machinery": {},
	"schema":    {},
	"target":    {},
	"pack":      {},
}

// Options control the local payload and optional remote submit.
type Options struct {
	Friction    string
	Layer       string
	Change      string
	HelpsOthers bool
	PrivateFile string
	Now         time.Time
	Submitter   Submitter
	Submit      bool
	Command     Commander
}

// Submitter is the optional GitHub (or other) client. Tests inject a fake. Default is file-only.
type Submitter interface {
	Submit(title, body string) (string, error)
}

// Commander runs an external program. Tests inject a fake; production uses gh.
type Commander func(args ...string) (string, error)

// Payload is the sanitized issue/PR text written under .ax/upstream/.
type Payload struct {
	Friction    string
	Evidence    string
	Change      string
	Layer       string
	HelpsOthers bool
}

// Run writes sanitized issue.md and pr.md. It does not merge anything.
func Run(root string, opts Options, out io.Writer) error {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if opts.Layer == "" {
		opts.Layer = "machinery"
	}
	if _, ok := layers[opts.Layer]; !ok {
		return fmt.Errorf("layer must be machinery, schema, target, or pack")
	}
	payload, err := build(root, opts)
	if err != nil {
		return err
	}
	sanitizer, err := loadSanitizer(root, opts.PrivateFile)
	if err != nil {
		return err
	}
	payload = sanitizer.Apply(payload)
	dir := filepath.Join(root, ".ax", "upstream")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	issue := renderIssue(payload)
	pr := renderPR(payload)
	if err := os.WriteFile(filepath.Join(dir, "issue.md"), []byte(issue), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "pr.md"), []byte(pr), 0644); err != nil {
		return err
	}
	fmt.Fprintln(out, ".ax/upstream/issue.md")
	fmt.Fprintln(out, ".ax/upstream/pr.md")
	submitter := opts.Submitter
	if submitter == nil && opts.Submit {
		if opts.Command == nil {
			return fmt.Errorf("--submit requires gh on PATH")
		}
		submitter = GHSubmitter(opts.Command)
	}
	if submitter != nil {
		url, err := submitter.Submit("ax upstream: "+opts.Layer, issue)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, url)
	}
	return nil
}

func build(root string, opts Options) (Payload, error) {
	friction, err := logbook.LoadFriction(root)
	if err != nil {
		return Payload{}, err
	}
	var notes []string
	for _, f := range friction {
		if opts.Friction != "" && f.ID != opts.Friction {
			continue
		}
		notes = append(notes, f.ID+": "+f.Body)
	}
	if opts.Friction != "" && len(notes) == 0 {
		return Payload{}, fmt.Errorf("friction %s not found", opts.Friction)
	}
	change := strings.TrimSpace(opts.Change)
	if change == "" {
		change = "See friction notes. Propose the smallest core or machinery fix that removes the repeated pain."
	}
	evidence := ".ax/friction and .ax/observations in this repo"
	if opts.Friction != "" {
		evidence = logbook.RelPath("friction", opts.Friction)
	}
	return Payload{
		Friction:    strings.Join(notes, "\n"),
		Evidence:    evidence,
		Change:      change,
		Layer:       opts.Layer,
		HelpsOthers: opts.HelpsOthers,
	}, nil
}

func renderIssue(p Payload) string {
	helps := "no"
	if p.HelpsOthers {
		helps = "yes"
	}
	return fmt.Sprintf(`# Upstream issue

## Friction

%s

## Evidence

%s

## Proposed change

%s

## Layer

%s

## Would this help another project?

%s
`, p.Friction, p.Evidence, p.Change, p.Layer, helps)
}

func renderPR(p Payload) string {
	return fmt.Sprintf(`# Upstream pull request

Do not merge automatically. A person reviews this.

%s
`, renderIssue(p))
}

type sanitizer struct {
	root    string
	private []string
}

func loadSanitizer(root, extraFile string) (*sanitizer, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	s := &sanitizer{root: abs}
	path := extraFile
	if path == "" {
		path = filepath.Join(root, ".ax", "private.txt")
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		s.private = append(s.private, line)
	}
	return s, nil
}

var secretPat = regexp.MustCompile(`(?i)(ghp_[A-Za-z0-9]+|sk-[A-Za-z0-9]+|api[_-]?key\s*[=:]\s*\S+)`)

func (s *sanitizer) Apply(p Payload) Payload {
	p.Friction = s.scrub(p.Friction)
	p.Evidence = s.scrub(p.Evidence)
	p.Change = s.scrub(p.Change)
	return p
}

func (s *sanitizer) scrub(text string) string {
	text = strings.ReplaceAll(text, s.root, "<project>")
	text = strings.ReplaceAll(text, filepath.ToSlash(s.root), "<project>")
	for _, p := range s.private {
		if p == "" {
			continue
		}
		text = strings.ReplaceAll(text, p, "<redacted>")
	}
	text = secretPat.ReplaceAllString(text, "<redacted>")
	return text
}
