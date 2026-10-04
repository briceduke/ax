package pack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/briceduke/ax/internal/yamlx"
)

// Add copies a pack into packs/<id>/ and merges provided files into core/.
func Add(root, src string, out io.Writer) error {
	abs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	man, err := LoadManifest(abs)
	if err != nil {
		return err
	}
	dest := filepath.Join(root, "packs", man.ID)
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := copyTree(abs, dest); err != nil {
		return err
	}
	for _, kind := range man.Provides {
		if err := mergeKind(root, dest, kind); err != nil {
			return err
		}
	}
	if err := ensurePackListed(root, man.ID); err != nil {
		return err
	}
	fmt.Fprintln(out, filepath.ToSlash(filepath.Join("packs", man.ID)))
	return nil
}

func mergeKind(root, packDir, kind string) error {
	files, err := listKindFiles(packDir, kind)
	if err != nil {
		return err
	}
	for _, rel := range files {
		if kind == "checks" {
			if err := mergeChecks(root, filepath.Join(packDir, "checks.yaml")); err != nil {
				return err
			}
			continue
		}
		src := filepath.Join(packDir, filepath.FromSlash(rel))
		dst := filepath.Join(root, filepath.FromSlash("core/"+rel))
		if fileExists(dst) {
			return fmt.Errorf("refusing to overwrite core/%s", rel)
		}
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func mergeChecks(root, packChecks string) error {
	path := filepath.Join(root, "core", "checks.yaml")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	incoming, err := os.ReadFile(packChecks)
	if err != nil {
		return err
	}
	var have []map[string]interface{}
	var add []map[string]interface{}
	if len(existing) > 0 {
		if err := yamlx.Decode(existing, &have); err != nil {
			return fmt.Errorf("core/checks.yaml: %w", err)
		}
	}
	if err := yamlx.Decode(incoming, &add); err != nil {
		return fmt.Errorf("pack checks.yaml: %w", err)
	}
	seen := map[string]struct{}{}
	for _, c := range have {
		id, _ := c["id"].(string)
		seen[id] = struct{}{}
	}
	for _, c := range add {
		id, _ := c["id"].(string)
		if _, ok := seen[id]; ok {
			continue
		}
		have = append(have, c)
	}
	return writeChecksYAML(path, have)
}

func writeChecksYAML(path string, list []map[string]interface{}) error {
	var b strings.Builder
	if len(list) == 0 {
		b.WriteString("[]\n")
		return os.WriteFile(path, []byte(b.String()), 0644)
	}
	for _, c := range list {
		id, _ := c["id"].(string)
		fmt.Fprintf(&b, "- id: %s\n", id)
		if v, ok := c["builtin"].(string); ok && v != "" {
			fmt.Fprintf(&b, "  builtin: %s\n", v)
		}
		if v, ok := c["run"].(string); ok && v != "" {
			fmt.Fprintf(&b, "  run: %s\n", v)
		}
		b.WriteString("  inputs:\n")
		if inputs, ok := c["inputs"].([]interface{}); ok {
			for _, in := range inputs {
				fmt.Fprintf(&b, "    - %s\n", in)
			}
		}
		if v, ok := c["tier"].(string); ok {
			fmt.Fprintf(&b, "  tier: %s\n", v)
		}
		if v, ok := c["enforces"].(string); ok && v != "" {
			fmt.Fprintf(&b, "  enforces: %s\n", v)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

// Extract copies selected files into a pack dir, replacing product values with params.
func Extract(root, dest, id string, files []string, replacements map[string]string, out io.Writer) error {
	if id == "" {
		return fmt.Errorf("pack extract needs --id")
	}
	if dest == "" {
		return fmt.Errorf("pack extract needs a destination directory")
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	providesSet := map[string]struct{}{}
	var params []string
	seenParam := map[string]struct{}{}
	for _, name := range replacements {
		if _, ok := seenParam[name]; ok {
			continue
		}
		seenParam[name] = struct{}{}
		params = append(params, name)
	}
	sort.Strings(params)
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		text := string(data)
		for old, param := range replacements {
			text = strings.ReplaceAll(text, old, "{{"+param+"}}")
		}
		kind := kindFromRel(rel)
		if kind == "" {
			return fmt.Errorf("cannot pack %s", rel)
		}
		providesSet[kind] = struct{}{}
		targetRel := packRel(rel, kind)
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dest, filepath.FromSlash(targetRel))), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dest, filepath.FromSlash(targetRel)), []byte(text), 0644); err != nil {
			return err
		}
	}
	var provides []string
	for _, k := range []string{"capabilities", "checks", "evals", "scaffolds"} {
		if _, ok := providesSet[k]; ok {
			provides = append(provides, k)
		}
	}
	man := &Manifest{
		ID:         id,
		Version:    "0.1.0",
		Provides:   provides,
		Requires:   []string{},
		Params:     params,
		Provenance: "extracted from local project",
	}
	if err := writeManifest(dest, man); err != nil {
		return err
	}
	fmt.Fprintln(out, dest)
	return nil
}

func kindFromRel(rel string) string {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.HasPrefix(rel, "core/capabilities/"):
		return "capabilities"
	case strings.HasPrefix(rel, "core/evals/"):
		return "evals"
	case strings.HasPrefix(rel, "core/scaffolds/"):
		return "scaffolds"
	case rel == "core/checks.yaml":
		return "checks"
	default:
		return ""
	}
}

func packRel(rel, kind string) string {
	rel = filepath.ToSlash(rel)
	if kind == "checks" {
		return "checks.yaml"
	}
	base := filepath.Base(rel)
	return kind + "/" + base
}

// Bootstrap writes a six-step empty starter for a discipline.
func Bootstrap(root, discipline string, out io.Writer) error {
	discipline = strings.TrimSpace(discipline)
	if discipline == "" {
		return fmt.Errorf("usage: ax pack bootstrap <discipline>")
	}
	dir := filepath.Join(root, "packs", discipline+"-starter")
	if err := os.MkdirAll(filepath.Join(dir, "capabilities"), 0755); err != nil {
		return err
	}
	for _, sub := range []string{"evals", "scaffolds"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "checks.yaml"), []byte("[]\n"), 0644); err != nil {
		return err
	}
	man := &Manifest{
		ID:         discipline + "-starter",
		Version:    "0.1.0",
		Provides:   []string{},
		Requires:   []string{},
		Params:     []string{},
		Provenance: "bootstrap; add files only after a failure",
	}
	if err := writeManifest(dir, man); err != nil {
		return err
	}
	guide := fmt.Sprintf(`# %s starter

Add process only after something fails. Empty files are intentional.

1. Decide the toolchain for this discipline. Fill the decision skeleton under log/decisions.
2. Write a one-page intent fragment: what this discipline is for, and who it serves.
3. Leave checks empty until a failure needs a machine-testable rule.
4. Leave evals empty until a failure needs a scored task.
5. Leave capabilities empty until a job must still be done after models get stronger.
6. Leave scaffolds empty. Each one needs a friction file and a sentence that says when to delete it.
`, discipline)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(guide), 0644); err != nil {
		return err
	}
	decision := `## Context

Pick the toolchain for this discipline.

## Options

- Wait until the first failure
- Choose tools now and write them down

## Choice

## Why
`
	if err := os.WriteFile(filepath.Join(dir, "toolchain-decision.md"), []byte(decision), 0644); err != nil {
		return err
	}
	fmt.Fprintln(out, filepath.ToSlash(filepath.Join("packs", discipline+"-starter")))
	return nil
}
