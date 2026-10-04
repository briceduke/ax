package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/yamlx"
)

// Manifest is pack.yaml.
type Manifest struct {
	ID         string   `yaml:"id"`
	Version    string   `yaml:"version"`
	Provides   []string `yaml:"provides"`
	Requires   []string `yaml:"requires"`
	Params     []string `yaml:"params"`
	Provenance string   `yaml:"provenance"`
}

func validateManifest(m *Manifest) error {
	if m.ID == "" {
		return fmt.Errorf("pack missing id")
	}
	if m.Version == "" {
		return fmt.Errorf("pack missing version")
	}
	if m.Provides == nil {
		m.Provides = []string{}
	}
	if m.Requires == nil {
		m.Requires = []string{}
	}
	if m.Params == nil {
		m.Params = []string{}
	}
	return nil
}

// LoadManifest reads pack.yaml from dir.
func LoadManifest(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "pack.yaml"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yamlx.Decode(data, &m); err != nil {
		return nil, fmt.Errorf("pack.yaml: %w", err)
	}
	if err := validateManifest(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func writeManifest(dir string, m *Manifest) error {
	if err := validateManifest(m); err != nil {
		return err
	}
	body := fmt.Sprintf("id: %s\nversion: %s\nprovides: %s\nrequires: %s\nparams: %s\nprovenance: %s\n",
		m.ID, m.Version, formatList(m.Provides), formatList(m.Requires), formatList(m.Params), m.Provenance)
	return os.WriteFile(filepath.Join(dir, "pack.yaml"), []byte(body), 0644)
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func listKindFiles(dir, kind string) ([]string, error) {
	var rels []string
	switch kind {
	case "capabilities", "evals", "scaffolds":
		entries, err := os.ReadDir(filepath.Join(dir, kind))
		if os.IsNotExist(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			rels = append(rels, kind+"/"+e.Name())
		}
	case "checks":
		if _, err := os.Stat(filepath.Join(dir, "checks.yaml")); err == nil {
			rels = append(rels, "checks.yaml")
		}
	}
	return rels, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ensurePackListed(root, id string) error {
	cfg, err := project.Load(root)
	if err != nil {
		return err
	}
	for _, p := range cfg.Packs {
		if p == id {
			return nil
		}
	}
	cfg.Packs = append(cfg.Packs, id)
	return project.Save(root, cfg)
}
