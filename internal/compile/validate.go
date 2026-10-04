package compile

import (
	"fmt"
	"strings"

	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/targets"
)

type namedFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Validate checks required files, frontmatter, and capability landings.
func Validate(spec *targets.Spec, files []File, caps []*core.Capability) error {
	byRel := map[string]File{}
	landed := map[string]struct{}{}
	for _, f := range files {
		byRel[f.Rel] = f
		for _, id := range f.Landing {
			landed[id] = struct{}{}
		}
	}
	for _, rel := range []string{spec.Instructions, spec.Hooks, spec.MCP} {
		if _, ok := byRel[rel]; !ok {
			return fmt.Errorf("missing required adapter %s", rel)
		}
	}
	for _, f := range files {
		if err := parseAdapterFrontmatter(f); err != nil {
			return fmt.Errorf("%s: %w", f.Rel, err)
		}
	}
	for _, cap := range caps {
		if core.HasHint(cap, core.HintInvocable) {
			rel := strings.ReplaceAll(spec.Invocable, "{id}", cap.ID)
			if _, ok := byRel[rel]; !ok {
				return fmt.Errorf("capability %s did not land at %s", cap.ID, rel)
			}
		}
		if core.HasHint(cap, core.HintIsolation) {
			rel := strings.ReplaceAll(spec.Isolated, "{id}", cap.ID)
			if _, ok := byRel[rel]; !ok {
				return fmt.Errorf("capability %s did not land at %s", cap.ID, rel)
			}
		}
		if _, ok := landed[cap.ID]; !ok {
			return fmt.Errorf("capability %s did not land anywhere", cap.ID)
		}
	}
	return nil
}

func parseAdapterFrontmatter(f File) error {
	if strings.HasSuffix(strings.ToLower(f.Rel), ".json") {
		return nil
	}
	text := string(core.NormalizeLF(f.Data))
	if !strings.HasPrefix(text, "---\n") {
		return nil
	}
	var fm namedFrontmatter
	if _, err := core.DecodeFrontmatter(f.Data, &fm); err != nil {
		return err
	}
	if fm.Name == "" {
		return fmt.Errorf("missing name")
	}
	return nil
}
