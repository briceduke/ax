package onboard

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/briceduke/ax/internal/builtins"
	"github.com/briceduke/ax/internal/compile"
	"github.com/briceduke/ax/internal/logbook"
	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/version"
)

// Adopt writes an intent stub and registers obvious existing checks.
func Adopt(root string, now time.Time, out io.Writer) error {
	if now.IsZero() {
		now = time.Now()
	}
	if _, err := os.Stat(project.Join(root, "ax.yaml")); os.IsNotExist(err) {
		cfg := &project.Config{Ax: version.Version, Targets: []string{"cursor", "claude-code"}, Packs: []string{}}
		if err := project.Save(root, cfg); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	for _, dir := range treeDirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0755); err != nil {
			return err
		}
	}
	intentPath := project.Join(root, "intent.md")
	if _, err := os.Stat(intentPath); os.IsNotExist(err) {
		name := filepath.Base(root)
		if err := os.WriteFile(intentPath, []byte(intentDoc(InitOptions{Name: name, Intent: "Existing project " + name + "."})), 0644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	caps, err := builtins.Capabilities()
	if err != nil {
		return err
	}
	for _, cap := range caps {
		path := project.Join(root, "capabilities", cap.ID+".md")
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		data, err := builtins.CapabilityFile(cap.ID)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return err
		}
	}
	body := `## Context

This repo already existed. ax is being added from history, not from a blank start.

## Options

- Keep informal process
- Reconstruct the minimum ax tree from what is already here

## Choice

Reconstruct the minimum ax tree.

## Why

Inferred from existing files. Treat this decision as reconstructed, not freshly debated.
`
	rel, err := logbook.WriteReconstructedDecision(root, "adopt ax from existing repo", body, now)
	if err != nil {
		return err
	}
	checksPath := project.Join(root, "checks.yaml")
	if _, err := os.Stat(checksPath); os.IsNotExist(err) {
		data, err := builtins.ChecksYAML(rel)
		if err != nil {
			return err
		}
		if extra := inferredCheck(root, rel); extra != "" {
			data = append(data, []byte(extra)...)
		}
		if err := os.WriteFile(checksPath, data, 0644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := compile.Refresh(root); err != nil {
		return err
	}
	fmt.Fprintln(out, rel)
	return nil
}

func inferredCheck(root, enforces string) string {
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return ""
	}
	return fmt.Sprintf("- id: go-test\n  run: go test ./...\n  inputs: []\n  tier: full\n  enforces: %s\n", enforces)
}
