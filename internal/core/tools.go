package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/briceduke/ax/internal/yamlx"
)

const ToolsFile = "core/tools.yaml"

// Tool is one local stdio MCP server compiled into editor mcp.json files.
type Tool struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

// LoadTools reads optional core/tools.yaml. Missing file is empty, not an error.
func LoadTools(root string) ([]Tool, error) {
	path := filepath.Join(root, filepath.FromSlash(ToolsFile))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}
	var tools []Tool
	if err := yamlx.Decode(data, &tools); err != nil {
		return nil, fmt.Errorf("core/tools.yaml: %w", err)
	}
	for i, t := range tools {
		if strings.TrimSpace(t.Name) == "" || strings.TrimSpace(t.Command) == "" {
			return nil, fmt.Errorf("core/tools.yaml: item %d needs name and command", i)
		}
		if t.Args == nil {
			tools[i].Args = []string{}
		}
	}
	return tools, nil
}
