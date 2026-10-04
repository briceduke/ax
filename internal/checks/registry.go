package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/briceduke/ax/internal/core"
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

// Check is one registry entry from core/checks.yaml.
type Check struct {
	ID       string   `yaml:"id"`
	Run      string   `yaml:"run,omitempty"`
	Builtin  string   `yaml:"builtin,omitempty"`
	Inputs   []string `yaml:"inputs"`
	Tier     string   `yaml:"tier"`
	Enforces string   `yaml:"enforces,omitempty"`
}

// LoadRegistry reads core/checks.yaml and rejects unknown fields.
func LoadRegistry(root string) ([]Check, error) {
	path := filepath.Join(root, "core", "checks.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read core/checks.yaml: %w", err)
	}
	var list []Check
	if err := yamlx.Decode(data, &list); err != nil {
		return nil, fmt.Errorf("core/checks.yaml: %w", err)
	}
	for i := range list {
		if err := validateCheck(&list[i]); err != nil {
			return nil, fmt.Errorf("core/checks.yaml: %w", err)
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
