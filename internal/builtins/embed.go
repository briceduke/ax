package builtins

import (
	"embed"
	"fmt"
	"strings"

	"github.com/briceduke/ax/internal/core"
)

//go:embed capabilities/*.md checks.yaml gitignore
var fs embed.FS

const enforcesToken = "{{enforces}}"

// RecordDecision is the built-in how-we-work note for logging choices.
func RecordDecision() (*core.Capability, error) {
	data, err := fs.ReadFile("capabilities/record-decision.md")
	if err != nil {
		return nil, err
	}
	return core.ParseCapabilityBytes(data)
}

// Capabilities returns every built-in capability.
func Capabilities() ([]*core.Capability, error) {
	c, err := RecordDecision()
	if err != nil {
		return nil, err
	}
	return []*core.Capability{c}, nil
}

// ChecksYAML returns the built-in check list with enforces set to rel.
func ChecksYAML(enforces string) ([]byte, error) {
	data, err := fs.ReadFile("checks.yaml")
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
	return fs.ReadFile("gitignore")
}

// CapabilityFile returns the embedded markdown for a built-in capability id.
func CapabilityFile(id string) ([]byte, error) {
	return fs.ReadFile("capabilities/" + id + ".md")
}
