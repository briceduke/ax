package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/targets"
)

type agentWriter struct {
	run Runner
}

func (w agentWriter) Write(spec *targets.Spec, snap *Snapshot) ([]File, error) {
	dir, err := os.MkdirTemp("", "ax-compile-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	rels := expectedRels(spec, snap)
	if err := os.WriteFile(filepath.Join(dir, "TARGET"), []byte(spec.ID+"\n"), 0644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "ax-output-files.txt"), []byte(strings.Join(rels, "\n")+"\n"), 0644); err != nil {
		return nil, err
	}
	task := fmt.Sprintf("Write editor adapter files for target %s.\nRequired relative paths are listed in ax-output-files.txt.\nUse the intent, capabilities, and scaffolds in this directory.\n", spec.ID)
	if _, err := w.run.Run(dir, task); err != nil {
		return nil, err
	}
	var files []File
	for _, rel := range rels {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("agent did not write %s: %w", rel, err)
		}
		files = append(files, File{Rel: rel, Data: data})
	}
	return files, nil
}

func expectedRels(spec *targets.Spec, snap *Snapshot) []string {
	files, err := Assembler{}.Write(spec, snap)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Rel)
	}
	return out
}

func withLandings(spec *targets.Spec, snap *Snapshot, files []File) []File {
	for i, f := range files {
		if len(f.Landing) > 0 {
			continue
		}
		files[i].Landing = landingsFor(spec, snap, f.Rel)
	}
	return files
}

func landingsFor(spec *targets.Spec, snap *Snapshot, rel string) []string {
	var ids []string
	for _, cap := range snap.Capabilities {
		if core.HasHint(cap, core.HintInvocable) {
			if rel == strings.ReplaceAll(spec.Invocable, "{id}", cap.ID) {
				ids = append(ids, cap.ID)
			}
			continue
		}
		if core.HasHint(cap, core.HintIsolation) {
			if rel == strings.ReplaceAll(spec.Isolated, "{id}", cap.ID) {
				ids = append(ids, cap.ID)
			}
			continue
		}
		if rel == spec.Instructions {
			ids = append(ids, cap.ID)
		}
	}
	return ids
}
