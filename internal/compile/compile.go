package compile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/briceduke/ax/internal/builtins"
	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/version"
	"github.com/briceduke/ax/targets"
)

// Options control which targets compile and where output goes.
type Options struct {
	Target  string
	Without string
	Writer  Writer
}

// Writer turns a core snapshot into adapter files for one target spec.
type Writer interface {
	Write(spec *targets.Spec, snap *Snapshot) ([]File, error)
}

// File is one generated adapter.
type File struct {
	Rel     string
	Data    []byte
	Landing []string
}

// Snapshot is the core inputs the writer sees.
type Snapshot struct {
	Intent       string
	Capabilities []*core.Capability
	Scaffolds    []*core.Scaffold
	Hash         string
	Version      string
}

const fastCheckCommand = "ax check --tier fast"

// Run compiles enabled targets (or --target) into adapter files.
func Run(root string, opts Options, out io.Writer) error {
	compileRoot, err := isolate(root, opts.Without)
	if err != nil {
		return err
	}
	if compileRoot != root {
		fmt.Fprintf(out, "wrote %s\n", slashRel(root, compileRoot))
	}
	cfg, err := project.Load(compileRoot)
	if err != nil {
		return err
	}
	names, err := selectTargets(cfg.Targets, opts.Target)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Fprintln(out, "no targets enabled")
		return nil
	}
	extra, err := targets.Extra(cfg.Targets)
	if err != nil {
		return err
	}
	hash, err := core.CompileHash(compileRoot, extra, nil)
	if err != nil {
		return err
	}
	snap, err := loadSnapshot(compileRoot, hash)
	if err != nil {
		return err
	}
	writer := opts.Writer
	if writer == nil {
		writer = Assembler{}
	}
	old, err := core.LoadManifest(compileRoot)
	if err != nil {
		return err
	}
	man := mergeManifest(old, hash, names)
	for _, name := range names {
		status, adapters, err := compileTarget(compileRoot, name, snap, writer, old, hash)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%-4s %s\n", status, name)
		for rel, sum := range adapters {
			man.Adapters[rel] = sum
		}
	}
	return core.SaveManifest(compileRoot, man)
}

func selectTargets(enabled []string, only string) ([]string, error) {
	if only == "" {
		return append([]string(nil), enabled...), nil
	}
	for _, name := range enabled {
		if name == only {
			return []string{only}, nil
		}
	}
	return nil, fmt.Errorf("target %q is not enabled in ax.yaml", only)
}

func loadSnapshot(root, hash string) (*Snapshot, error) {
	intent := ""
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash("core/intent.md")))
	if err == nil {
		intent = string(core.NormalizeLF(data))
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	caps, err := core.LoadCapabilities(root)
	if err != nil {
		return nil, err
	}
	caps, err = mergeBuiltinCapabilities(caps)
	if err != nil {
		return nil, err
	}
	scaffolds, err := core.LoadScaffolds(root, nil)
	if err != nil {
		return nil, err
	}
	return &Snapshot{
		Intent:       intent,
		Capabilities: caps,
		Scaffolds:    scaffolds,
		Hash:         hash,
		Version:      version.Version,
	}, nil
}

func compileTarget(root, name string, snap *Snapshot, writer Writer, old *core.Manifest, hash string) (string, map[string]string, error) {
	spec, err := targets.Load(name)
	if err != nil {
		return "", nil, err
	}
	files, err := writer.Write(spec, snap)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", name, err)
	}
	for i := range files {
		files[i].Data = applyHeader(files[i].Rel, files[i].Data, snap.Hash)
		files[i].Data = withTrailingLF(core.NormalizeLF(files[i].Data))
	}
	if err := Validate(spec, files, snap.Capabilities); err != nil {
		return "", nil, fmt.Errorf("%s: %w", name, err)
	}
	adapters := map[string]string{}
	for _, f := range files {
		adapters[f.Rel] = core.ContentHash(f.Data)
	}
	if canSkip(root, old, hash, adapters) {
		return "skip", adapters, nil
	}
	for _, f := range files {
		path := filepath.Join(root, filepath.FromSlash(f.Rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", nil, err
		}
		if err := os.WriteFile(path, f.Data, 0644); err != nil {
			return "", nil, err
		}
	}
	return "ok", adapters, nil
}

func canSkip(root string, old *core.Manifest, hash string, adapters map[string]string) bool {
	if old == nil || old.CoreHash != hash || old.AxVersion != version.Version {
		return false
	}
	for rel, want := range adapters {
		got, ok := old.Adapters[rel]
		if !ok || got != want {
			return false
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return false
		}
		if core.ContentHash(data) != want {
			return false
		}
	}
	return true
}

func mergeManifest(old *core.Manifest, hash string, compiled []string) *core.Manifest {
	man := &core.Manifest{
		CoreHash:  hash,
		AxVersion: version.Version,
		Adapters:  map[string]string{},
	}
	if old == nil || old.CoreHash != hash {
		return man
	}
	for rel, sum := range old.Adapters {
		if ownedBy(rel, compiled) {
			continue
		}
		man.Adapters[rel] = sum
	}
	return man
}

func ownedBy(rel string, names []string) bool {
	for _, name := range names {
		spec, err := targets.Load(name)
		if err != nil {
			continue
		}
		if rel == spec.Instructions || rel == spec.Hooks || rel == spec.MCP {
			return true
		}
		if underPattern(rel, spec.Invocable) || underPattern(rel, spec.Isolated) {
			return true
		}
	}
	return false
}

func underPattern(rel, pattern string) bool {
	prefix, _, ok := strings.Cut(pattern, "{id}")
	if !ok {
		return rel == pattern
	}
	return strings.HasPrefix(rel, prefix)
}

func slashRel(root, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(rel)
}

func mergeBuiltinCapabilities(caps []*core.Capability) ([]*core.Capability, error) {
	builtinsCaps, err := builtins.Capabilities()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]*core.Capability, 0, len(caps)+len(builtinsCaps))
	for _, c := range caps {
		seen[c.ID] = struct{}{}
		out = append(out, c)
	}
	for _, c := range builtinsCaps {
		if _, ok := seen[c.ID]; ok {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func withTrailingLF(data []byte) []byte {
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return append(append([]byte{}, data...), '\n')
	}
	return data
}
