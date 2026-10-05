package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/yamlx"
)

// Manifest is the staleness sidecar at .ax/manifest.yaml.
type Manifest struct {
	CoreHash  string            `yaml:"core_hash"`
	AxVersion string            `yaml:"ax_version"`
	Adapters  map[string]string `yaml:"adapters"`
}

// ManifestPath is the slash-separated path of the compile manifest.
func ManifestPath() string {
	return project.Rel("manifest.yaml")
}

// LoadManifest reads .ax/manifest.yaml. Missing file returns (nil, nil).
func LoadManifest(root string) (*Manifest, error) {
	path := filepath.Join(root, filepath.FromSlash(ManifestPath()))
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var man Manifest
	if err := yamlx.Decode(data, &man); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if man.Adapters == nil {
		man.Adapters = map[string]string{}
	}
	return &man, nil
}

// SaveManifest writes .ax/manifest.yaml with sorted adapter keys.
func SaveManifest(root string, man *Manifest) error {
	if err := os.MkdirAll(project.DirPath(root), 0755); err != nil {
		return err
	}
	if man.Adapters == nil {
		man.Adapters = map[string]string{}
	}
	data := encodeManifest(man)
	return os.WriteFile(project.Join(root, "manifest.yaml"), data, 0644)
}

func encodeManifest(man *Manifest) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "core_hash: %s\n", man.CoreHash)
	fmt.Fprintf(&b, "ax_version: %s\n", man.AxVersion)
	keys := make([]string, 0, len(man.Adapters))
	for k := range man.Adapters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		b.WriteString("adapters: {}\n")
		return []byte(b.String())
	}
	b.WriteString("adapters:\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "  %s: %s\n", k, man.Adapters[k])
	}
	return []byte(b.String())
}
