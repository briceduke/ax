package checks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/yamlx"
)

const (
	TierFast = "fast"
	TierFull = "full"
	TierSlow = "slow"
)

var tierRank = map[string]int{
	TierFast: 0,
	TierFull: 1,
	TierSlow: 2,
}

// Check is one registry entry from .ax/checks.yaml.
type Check struct {
	ID       string   `yaml:"id"`
	Run      string   `yaml:"run,omitempty"`
	Builtin  string   `yaml:"builtin,omitempty"`
	Inputs   []string `yaml:"inputs"`
	Tier     string   `yaml:"tier"`
	Enforces string   `yaml:"enforces,omitempty"`
}

// LoadRegistry reads .ax/checks.yaml and rejects unknown fields.
func LoadRegistry(root string) ([]Check, error) {
	path := project.Join(root, "checks.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", project.Rel("checks.yaml"), err)
	}
	var list []Check
	if err := yamlx.Decode(data, &list); err != nil {
		return nil, fmt.Errorf("%s: %w", project.Rel("checks.yaml"), err)
	}
	for i := range list {
		if err := validateCheck(&list[i]); err != nil {
			return nil, fmt.Errorf("%s: %w", project.Rel("checks.yaml"), err)
		}
	}
	return list, nil
}

func validateCheck(c *Check) error {
	if c.ID == "" {
		return fmt.Errorf("check missing id")
	}
	hasRun := c.Run != ""
	hasBuiltin := c.Builtin != ""
	if hasRun == hasBuiltin {
		return fmt.Errorf("check %s must have exactly one of run or builtin", c.ID)
	}
	if _, ok := tierRank[c.Tier]; !ok {
		return fmt.Errorf("check %s has invalid tier %q", c.ID, c.Tier)
	}
	if c.Inputs == nil {
		c.Inputs = []string{}
	}
	return nil
}

var knownBuiltins = map[string]bool{
	"ax-format":            true,
	"ax-consistency":       true,
	"ax-staleness":         true,
	"conventional-commits": true,
}

// IsBuiltin reports whether name is a built-in check.
func IsBuiltin(name string) bool {
	return knownBuiltins[name]
}

// Append adds one check to .ax/checks.yaml.
func Append(root string, c Check) error {
	if c.Inputs == nil {
		c.Inputs = []string{}
	}
	if c.Tier == "" {
		c.Tier = TierFast
	}
	all, err := LoadRegistry(root)
	if err != nil {
		return err
	}
	for _, existing := range all {
		if existing.ID == c.ID {
			return fmt.Errorf("check %s already exists", c.ID)
		}
	}
	if err := validateCheck(&c); err != nil {
		return err
	}
	path := project.Join(root, "checks.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", project.Rel("checks.yaml"), err)
	}
	data = bytes.TrimSpace(data)
	if string(data) == "[]" {
		data = nil
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, []byte(formatCheck(c))...)
	return os.WriteFile(path, data, 0644)
}

func formatCheck(c Check) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- id: %s\n", c.ID)
	if c.Builtin != "" {
		fmt.Fprintf(&b, "  builtin: %s\n", c.Builtin)
	}
	if c.Run != "" {
		fmt.Fprintf(&b, "  run: %q\n", c.Run)
	}
	if len(c.Inputs) == 0 {
		b.WriteString("  inputs: []\n")
	} else {
		b.WriteString("  inputs:\n")
		for _, in := range c.Inputs {
			fmt.Fprintf(&b, "    - %s\n", in)
		}
	}
	fmt.Fprintf(&b, "  tier: %s\n", c.Tier)
	if c.Enforces != "" {
		fmt.Fprintf(&b, "  enforces: %s\n", c.Enforces)
	}
	return b.String()
}

// Select returns checks at or below the requested tier. Tiers are cumulative.
func Select(all []Check, tier string) ([]Check, error) {
	rank, ok := tierRank[tier]
	if !ok {
		return nil, fmt.Errorf("unknown tier %q", tier)
	}
	var out []Check
	for _, c := range all {
		if tierRank[c.Tier] <= rank {
			out = append(out, c)
		}
	}
	return out, nil
}

func expandInputs(root string, patterns []string) ([]string, error) {
	fsys := os.DirFS(root)
	seen := map[string]struct{}{}
	var files []string
	for _, pattern := range patterns {
		matches, err := doublestar.Glob(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", pattern, err)
		}
		for _, m := range matches {
			info, err := os.Stat(filepath.Join(root, filepath.FromSlash(m)))
			if err != nil {
				return nil, err
			}
			if info.IsDir() {
				continue
			}
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			files = append(files, m)
		}
	}
	sort.Strings(files)
	return files, nil
}

func hashCheck(root string, c Check) (string, error) {
	files, err := expandInputs(root, c.Inputs)
	if err != nil {
		return "", err
	}
	h := sha256Writer()
	fmt.Fprintf(h, "id=%s\nbuiltin=%s\nrun=%s\ntier=%s\nenforces=%s\n", c.ID, c.Builtin, c.Run, c.Tier, c.Enforces)
	inputs := append([]string(nil), c.Inputs...)
	sort.Strings(inputs)
	for _, in := range inputs {
		fmt.Fprintf(h, "input=%s\n", in)
	}
	if err := core.WriteFiles(h, root, files); err != nil {
		return "", err
	}
	return hexSum(h), nil
}
