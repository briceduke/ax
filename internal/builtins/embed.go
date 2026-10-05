package builtins

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/briceduke/ax/internal/core"
)

//go:embed capabilities/*.md checks.yaml gitignore
var embedded embed.FS

const enforcesToken = "{{enforces}}"

// Capabilities returns every built-in capability, sorted by id.
func Capabilities() ([]*core.Capability, error) {
	entries, err := embedded.ReadDir("capabilities")
	if err != nil {
		return nil, err
	}
	out := make([]*core.Capability, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := embedded.ReadFile("capabilities/" + e.Name())
		if err != nil {
			return nil, err
		}
		c, err := core.ParseCapabilityBytes(data)
		if err != nil {
			return nil, fmt.Errorf("capabilities/%s: %w", e.Name(), err)
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// ChecksYAML returns the built-in check list with enforces set to rel.
func ChecksYAML(enforces string) ([]byte, error) {
	data, err := embedded.ReadFile("checks.yaml")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(enforces) == "" {
		return nil, fmt.Errorf("enforces path is empty")
	}
	return []byte(strings.ReplaceAll(string(data), enforcesToken, enforces)), nil
}

// Gitignore is the default ignore file body for a new project.
func Gitignore() ([]byte, error) {
	return embedded.ReadFile("gitignore")
}

// CapabilityFile returns the embedded markdown for a built-in capability id.
func CapabilityFile(id string) ([]byte, error) {
	return embedded.ReadFile("capabilities/" + id + ".md")
}
