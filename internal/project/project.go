package project

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/briceduke/ax/internal/version"
	"github.com/briceduke/ax/internal/yamlx"
)

// Config is the strict ax.yaml schema.
type Config struct {
	Ax      string   `yaml:"ax"`
	Targets []string `yaml:"targets"`
	Packs   []string `yaml:"packs"`
}

// Find walks up from start until it finds ax.yaml.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "ax.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no ax.yaml found from %s", start)
		}
		dir = parent
	}
}

// Load reads and validates ax.yaml at root.
func Load(root string) (*Config, error) {
	path := filepath.Join(root, "ax.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ax.yaml: %w", err)
	}
	var cfg Config
	if err := yamlx.Decode(data, &cfg); err != nil {
		return nil, fmt.Errorf("ax.yaml: %w", err)
	}
	if cfg.Ax == "" {
		return nil, fmt.Errorf("ax.yaml: missing ax version")
	}
	if cfg.Targets == nil {
		cfg.Targets = []string{}
	}
	if cfg.Packs == nil {
		cfg.Packs = []string{}
	}
	return &cfg, nil
}

// WarnVersion writes a pin mismatch warning without failing.
func WarnVersion(cfg *Config, w io.Writer) {
	if cfg.Ax == version.Version {
		return
	}
	fmt.Fprintf(w, "warning: ax.yaml pins %s but this binary is %s\n", cfg.Ax, version.Version)
}
