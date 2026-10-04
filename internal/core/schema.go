package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Capability is the strict capability frontmatter schema.
type Capability struct {
	ID    string   `yaml:"id"`
	Kind  string   `yaml:"kind"`
	When  string   `yaml:"when"`
	Hints []string `yaml:"hints"`
	Body  string   `yaml:"-"`
}

// Scaffold is the strict scaffold frontmatter schema.
type Scaffold struct {
	ID          string `yaml:"id"`
	Kind        string `yaml:"kind"`
	Compensates string `yaml:"compensates"`
	Added       string `yaml:"added"`
	RetireWhen  string `yaml:"retire_when"`
	Body        string `yaml:"-"`
}

// ParseCapability reads and validates a capability markdown file.
func ParseCapability(path string) (*Capability, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cap Capability
	body, err := DecodeFrontmatter(data, &cap)
	if err != nil {
		return nil, err
	}
	cap.Body = body
	if err := validateCapability(&cap); err != nil {
		return nil, err
	}
	return &cap, nil
}

// ParseScaffold reads and validates a scaffold markdown file.
func ParseScaffold(path string) (*Scaffold, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sc Scaffold
	body, err := DecodeFrontmatter(data, &sc)
	if err != nil {
		return nil, err
	}
	sc.Body = body
	if err := validateScaffold(&sc); err != nil {
		return nil, err
	}
	return &sc, nil
}

func validateCapability(c *Capability) error {
	if c.ID == "" {
		return fmt.Errorf("missing id")
	}
	if c.Kind != "capability" {
		return fmt.Errorf("kind must be capability, got %q", c.Kind)
	}
	if strings.TrimSpace(c.When) == "" {
		return fmt.Errorf("missing when")
	}
	return nil
}

func validateScaffold(s *Scaffold) error {
	if s.ID == "" {
		return fmt.Errorf("missing id")
	}
	if s.Kind != "scaffold" {
		return fmt.Errorf("kind must be scaffold, got %q", s.Kind)
	}
	if strings.TrimSpace(s.Compensates) == "" {
		return fmt.Errorf("missing compensates")
	}
	if strings.TrimSpace(s.Added) == "" {
		return fmt.Errorf("missing added")
	}
	if strings.TrimSpace(s.RetireWhen) == "" {
		return fmt.Errorf("missing retire_when")
	}
	return nil
}

// HasHint reports whether cap lists hint.
func HasHint(c *Capability, hint string) bool {
	for _, h := range c.Hints {
		if h == hint {
			return true
		}
	}
	return false
}

const (
	HintInvocable = "user-invocable"
	HintIsolation = "isolation"
)

// LoadCapabilities parses every capability markdown file under core/capabilities.
func LoadCapabilities(root string) ([]*Capability, error) {
	rels, err := ListMarkdown(root, "core/capabilities")
	if err != nil {
		return nil, err
	}
	out := make([]*Capability, 0, len(rels))
	for _, rel := range rels {
		c, err := ParseCapability(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		out = append(out, c)
	}
	return out, nil
}

// LoadScaffolds parses scaffold files, omitting ids in without.
func LoadScaffolds(root string, without []string) ([]*Scaffold, error) {
	rels, err := ListMarkdown(root, "core/scaffolds")
	if err != nil {
		return nil, err
	}
	skip := map[string]struct{}{}
	for _, id := range without {
		skip[id] = struct{}{}
	}
	out := make([]*Scaffold, 0, len(rels))
	for _, rel := range rels {
		s, err := ParseScaffold(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		if _, omit := skip[s.ID]; omit {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// ListMarkdown returns slash-separated relative paths of *.md files in relDir.
func ListMarkdown(root, relDir string) ([]string, error) {
	dir := filepath.Join(root, filepath.FromSlash(relDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		files = append(files, relDir+"/"+e.Name())
	}
	sort.Strings(files)
	return files, nil
}
