package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/eval"
	"github.com/briceduke/ax/internal/logbook"
	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/version"
	"github.com/briceduke/ax/targets"
)

func runBuiltin(root, name string) error {
	switch name {
	case "ax-format":
		return checkFormat(root)
	case "ax-consistency":
		return checkConsistency(root)
	case "ax-staleness":
		return checkStaleness(root)
	default:
		return fmt.Errorf("unknown builtin %s", name)
	}
}

func checkFormat(root string) error {
	var errs []string
	errs = append(errs, formatLog(root)...)
	errs = append(errs, formatCapabilities(root)...)
	errs = append(errs, formatScaffolds(root)...)
	errs = append(errs, formatEvals(root)...)
	return joinErrors(errs)
}

func formatLog(root string) []string {
	var errs []string
	logRoot := filepath.Join(root, "log")
	dirs, err := os.ReadDir(logRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return []string{err.Error()}
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		kind, ok := logbook.KindFromDir(d.Name())
		if !ok {
			errs = append(errs, "unknown log directory: "+d.Name())
			continue
		}
		entries, err := os.ReadDir(filepath.Join(logRoot, d.Name()))
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			rel := "log/" + d.Name() + "/" + e.Name()
			path := filepath.Join(root, filepath.FromSlash(rel))
			if err := logbook.ValidateFile(path, kind); err != nil {
				errs = append(errs, rel+": "+err.Error())
			}
		}
	}
	return errs
}

func formatCapabilities(root string) []string {
	files, err := core.ListMarkdown(root, "core/capabilities")
	if err != nil {
		return []string{err.Error()}
	}
	var errs []string
	for _, rel := range files {
		if _, err := core.ParseCapability(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			errs = append(errs, rel+": "+err.Error())
		}
	}
	return errs
}

func formatScaffolds(root string) []string {
	files, err := core.ListMarkdown(root, "core/scaffolds")
	if err != nil {
		return []string{err.Error()}
	}
	var errs []string
	for _, rel := range files {
		if _, err := core.ParseScaffold(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			errs = append(errs, rel+": "+err.Error())
		}
	}
	return errs
}

func formatEvals(root string) []string {
	files, err := core.ListMarkdown(root, "core/evals")
	if err != nil {
		return []string{err.Error()}
	}
	var errs []string
	for _, rel := range files {
		if _, err := eval.ParseFile(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			errs = append(errs, rel+": "+err.Error())
		}
	}
	return errs
}

func checkConsistency(root string) error {
	var errs []string
	registry, err := LoadRegistry(root)
	if err != nil {
		return err
	}
	decisions, err := loadDecisions(root)
	if err != nil {
		return err
	}
	errs = append(errs, checkEnforces(root, registry, decisions)...)
	errs = append(errs, checkSupersedes(root, decisions)...)
	errs = append(errs, checkScaffoldAdded(root)...)
	return joinErrors(errs)
}

func loadDecisions(root string) (map[string]*logbook.Decision, error) {
	files, err := core.ListMarkdown(root, "log/decisions")
	if err != nil {
		return nil, err
	}
	out := map[string]*logbook.Decision{}
	for _, rel := range files {
		d, err := logbook.ParseDecision(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		out[d.ID] = d
	}
	return out, nil
}

func checkEnforces(root string, registry []Check, decisions map[string]*logbook.Decision) []string {
	superseded := map[string]struct{}{}
	for _, d := range decisions {
		for _, id := range d.Supersedes {
			superseded[id] = struct{}{}
		}
	}
	var errs []string
	for _, c := range registry {
		if c.Enforces == "" {
			continue
		}
		id, err := resolveDecision(root, c.Enforces)
		if err != nil {
			errs = append(errs, fmt.Sprintf("check %s: %v", c.ID, err))
			continue
		}
		if _, ok := superseded[id]; ok {
			errs = append(errs, fmt.Sprintf("check %s enforces superseded decision %s", c.ID, id))
		}
	}
	return errs
}

func checkSupersedes(root string, decisions map[string]*logbook.Decision) []string {
	var errs []string
	for _, d := range decisions {
		for _, id := range d.Supersedes {
			if !logbook.ValidID(id) {
				errs = append(errs, fmt.Sprintf("decision %s supersedes invalid id %s", d.ID, id))
				continue
			}
			path := filepath.Join(root, filepath.FromSlash(logbook.RelPath("decision", id)))
			if _, err := os.Stat(path); err != nil {
				errs = append(errs, fmt.Sprintf("decision %s supersedes missing %s", d.ID, id))
			}
		}
	}
	return errs
}

var frictionRef = regexp.MustCompile(`(?:log/)?friction/([a-z0-9]+(?:-[a-z0-9]+)*)`)

func checkScaffoldAdded(root string) []string {
	files, err := core.ListMarkdown(root, "core/scaffolds")
	if err != nil {
		return []string{err.Error()}
	}
	var errs []string
	for _, rel := range files {
		sc, err := core.ParseScaffold(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			errs = append(errs, rel+": "+err.Error())
			continue
		}
		m := frictionRef.FindStringSubmatch(sc.Added)
		if m == nil {
			errs = append(errs, rel+": added does not reference a friction entry")
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(logbook.RelPath("friction", m[1])))
		if _, err := os.Stat(path); err != nil {
			errs = append(errs, rel+": added link log/friction/"+m[1]+".md does not exist")
		}
	}
	return errs
}

func resolveDecision(root, enforces string) (string, error) {
	rel := strings.TrimSuffix(filepath.ToSlash(enforces), ".md") + ".md"
	path := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("enforces %s does not exist", enforces)
	}
	id := strings.TrimSuffix(filepath.Base(rel), ".md")
	if _, err := logbook.ParseDecision(path); err != nil {
		return "", fmt.Errorf("enforces %s is not a valid decision: %w", enforces, err)
	}
	return id, nil
}

func checkStaleness(root string) error {
	cfg, err := project.Load(root)
	if err != nil {
		return err
	}
	man, err := core.LoadManifest(root)
	if err != nil {
		return err
	}
	if man == nil {
		if len(cfg.Targets) == 0 {
			return nil
		}
		return fmt.Errorf("run ax compile")
	}
	extra, err := targets.Extra(cfg.Targets)
	if err != nil {
		return err
	}
	coreHash, err := core.CompileHash(root, extra, nil)
	if err != nil {
		return err
	}
	var errs []string
	if man.CoreHash != coreHash {
		errs = append(errs, "core hash changed; run ax compile")
	}
	if man.AxVersion != version.Version {
		errs = append(errs, fmt.Sprintf("manifest ax version %s does not match binary %s", man.AxVersion, version.Version))
	}
	for _, name := range cfg.Targets {
		spec, err := targets.Load(name)
		if err != nil {
			return err
		}
		if _, ok := man.Adapters[spec.Instructions]; !ok {
			errs = append(errs, "run ax compile")
			break
		}
	}
	for rel, want := range man.Adapters {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			errs = append(errs, fmt.Sprintf("adapter %s: %v", rel, err))
			continue
		}
		got := core.ContentHash(data)
		if got != want {
			errs = append(errs, fmt.Sprintf("adapter %s content hash mismatch", rel))
		}
	}
	return joinErrors(errs)
}

func joinErrors(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, "\n"))
}
