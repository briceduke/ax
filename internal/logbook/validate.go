package logbook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/core"
)

const (
	maxDecisionLines    = 60
	maxObservationLines = 30
)

var decisionHeadings = []string{"## Context", "## Options", "## Choice", "## Why"}

// Decision is the strict decision frontmatter schema.
type Decision struct {
	ID            string   `yaml:"id"`
	Kind          string   `yaml:"kind"`
	Date          string   `yaml:"date"`
	Supersedes    []string `yaml:"supersedes,omitempty"`
	Reconstructed bool     `yaml:"reconstructed,omitempty"`
	Body          string   `yaml:"-"`
}

// Observation is the strict observation frontmatter schema.
type Observation struct {
	ID     string `yaml:"id"`
	Kind   string `yaml:"kind"`
	Date   string `yaml:"date"`
	Method string `yaml:"method"`
	Body   string `yaml:"-"`
}

// Friction is the strict friction frontmatter schema.
type Friction struct {
	ID    string   `yaml:"id"`
	Kind  string   `yaml:"kind"`
	Date  string   `yaml:"date"`
	Links []string `yaml:"links,omitempty"`
	Body  string   `yaml:"-"`
}

// ValidateFile checks a log markdown file against its kind schema.
func ValidateFile(path, kind string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return ValidateBytes(data, kind, strings.TrimSuffix(filepath.Base(path), ".md"))
}

// ValidateBytes checks entry bytes against kind and expected filename id.
func ValidateBytes(data []byte, kind, filenameID string) error {
	switch kind {
	case "decision":
		d, err := parseDecision(data)
		if err != nil {
			return err
		}
		return checkIdentity(d.ID, d.Kind, d.Date, kind, filenameID, core.CountLines(string(data)), maxDecisionLines)
	case "observation":
		o, err := parseObservation(data)
		if err != nil {
			return err
		}
		return checkIdentity(o.ID, o.Kind, o.Date, kind, filenameID, core.CountLines(string(data)), maxObservationLines)
	case "friction":
		f, err := parseFriction(data)
		if err != nil {
			return err
		}
		if err := checkIdentity(f.ID, f.Kind, f.Date, kind, filenameID, 0, 0); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown log kind %q", kind)
	}
}

// ParseDecision reads a decision file.
func ParseDecision(path string) (*Decision, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseDecision(data)
}

func parseDecision(data []byte) (*Decision, error) {
	var d Decision
	body, err := core.DecodeFrontmatter(data, &d)
	if err != nil {
		return nil, err
	}
	d.Body = body
	if err := requireHeadings(d.Body, decisionHeadings); err != nil {
		return nil, err
	}
	if _, err := ChoiceSentence(d.Body); err != nil {
		return nil, err
	}
	return &d, nil
}

// ParseObservation reads an observation file.
func ParseObservation(path string) (*Observation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseObservation(data)
}

func parseObservation(data []byte) (*Observation, error) {
	var o Observation
	body, err := core.DecodeFrontmatter(data, &o)
	if err != nil {
		return nil, err
	}
	o.Body = strings.TrimSpace(body)
	if o.Method == "" {
		return nil, fmt.Errorf("missing method")
	}
	if o.Body == "" {
		return nil, fmt.Errorf("body is empty")
	}
	return &o, nil
}

func parseFriction(data []byte) (*Friction, error) {
	var f Friction
	body, err := core.DecodeFrontmatter(data, &f)
	if err != nil {
		return nil, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("body is empty")
	}
	if core.CountLines(body) != 1 {
		return nil, fmt.Errorf("body must be one line")
	}
	f.Body = body
	return &f, nil
}

// LoadDecisions parses every decision under .ax/decisions.
func LoadDecisions(root string) ([]*Decision, error) {
	rels, err := core.ListMarkdown(root, RelDir("decision"))
	if err != nil {
		return nil, err
	}
	out := make([]*Decision, 0, len(rels))
	for _, rel := range rels {
		d, err := ParseDecision(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		out = append(out, d)
	}
	return out, nil
}

// InForce returns decisions that no later decision has superseded.
func InForce(decisions []*Decision) []*Decision {
	superseded := map[string]struct{}{}
	for _, d := range decisions {
		for _, id := range d.Supersedes {
			superseded[id] = struct{}{}
		}
	}
	out := make([]*Decision, 0, len(decisions))
	for _, d := range decisions {
		if _, ok := superseded[d.ID]; ok {
			continue
		}
		out = append(out, d)
	}
	return out
}

// LoadFriction parses every friction entry under .ax/friction.
func LoadFriction(root string) ([]*Friction, error) {
	rels, err := core.ListMarkdown(root, RelDir("friction"))
	if err != nil {
		return nil, err
	}
	out := make([]*Friction, 0, len(rels))
	for _, rel := range rels {
		f, err := ParseFrictionFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		out = append(out, f)
	}
	return out, nil
}

// LoadObservations parses every observation entry under .ax/observations.
func LoadObservations(root string) ([]*Observation, error) {
	rels, err := core.ListMarkdown(root, RelDir("observation"))
	if err != nil {
		return nil, err
	}
	out := make([]*Observation, 0, len(rels))
	for _, rel := range rels {
		o, err := ParseObservation(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		out = append(out, o)
	}
	return out, nil
}

func checkIdentity(id, kind, date, wantKind, filenameID string, lines, maxLines int) error {
	if id == "" {
		return fmt.Errorf("missing id")
	}
	if !ValidID(id) {
		return fmt.Errorf("id %q is not date-slug form", id)
	}
	if kind != wantKind {
		return fmt.Errorf("kind must be %s, got %q", wantKind, kind)
	}
	if date == "" {
		return fmt.Errorf("missing date")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("date %q is not YYYY-MM-DD", date)
	}
	if !strings.HasPrefix(id, date+"-") {
		return fmt.Errorf("id date prefix %q does not match date %q", id[:min(10, len(id))], date)
	}
	if filenameID != "" && id != filenameID {
		return fmt.Errorf("id %q does not match filename %q", id, filenameID)
	}
	if maxLines > 0 && lines > maxLines {
		return fmt.Errorf("entry is %d lines, max %d", lines, maxLines)
	}
	return nil
}

func requireHeadings(body string, headings []string) error {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	at := 0
	for _, want := range headings {
		found := false
		for at < len(lines) {
			if strings.TrimSpace(lines[at]) == want {
				found = true
				at++
				break
			}
			at++
		}
		if !found {
			return fmt.Errorf("missing heading %s", want)
		}
	}
	return nil
}

// ExtractSection returns the body of a markdown ## heading.
func ExtractSection(body, heading string) string {
	want := "## " + heading
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == want {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			end = i
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

// ParseFrictionFile reads a friction entry from disk.
func ParseFrictionFile(path string) (*Friction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseFriction(data)
}
