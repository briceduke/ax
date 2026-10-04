package targets

import (
	"embed"
	"fmt"

	"github.com/briceduke/ax/internal/core"
)

//go:embed *.md
var specFS embed.FS

// Spec is the strict target-spec frontmatter schema.
type Spec struct {
	ID           string `yaml:"id"`
	Instructions string `yaml:"instructions"`
	Invocable    string `yaml:"invocable"`
	Isolated     string `yaml:"isolated"`
	Hooks        string `yaml:"hooks"`
	HooksFormat  string `yaml:"hooks_format"`
	MCP          string `yaml:"mcp"`
	Body         string `yaml:"-"`
	Raw          []byte `yaml:"-"`
}

// Parse decodes a target spec and rejects unknown fields.
func Parse(data []byte) (*Spec, error) {
	var spec Spec
	body, err := core.DecodeFrontmatter(data, &spec)
	if err != nil {
		return nil, err
	}
	spec.Body = body
	spec.Raw = core.NormalizeLF(data)
	if err := validateSpec(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func validateSpec(s *Spec) error {
	if s.ID == "" {
		return fmt.Errorf("missing id")
	}
	if s.Instructions == "" {
		return fmt.Errorf("missing instructions")
	}
	if s.Invocable == "" {
		return fmt.Errorf("missing invocable")
	}
	if s.Isolated == "" {
		return fmt.Errorf("missing isolated")
	}
	if s.Hooks == "" {
		return fmt.Errorf("missing hooks")
	}
	if s.HooksFormat == "" {
		return fmt.Errorf("missing hooks_format")
	}
	if s.MCP == "" {
		return fmt.Errorf("missing mcp")
	}
	return nil
}

// Load reads the embedded spec named id (without .md).
func Load(id string) (*Spec, error) {
	name := id + ".md"
	data, err := specFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("unknown target %q", id)
	}
	spec, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("target %s: %w", id, err)
	}
	if spec.ID != id {
		return nil, fmt.Errorf("target %s: id %q does not match filename", id, spec.ID)
	}
	return spec, nil
}

// Extra returns hashing path/content pairs for the named specs.
func Extra(ids []string) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, id := range ids {
		spec, err := Load(id)
		if err != nil {
			return nil, err
		}
		out["targets/"+id+".md"] = spec.Raw
	}
	return out, nil
}
