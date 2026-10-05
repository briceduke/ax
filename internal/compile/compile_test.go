package compile

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/briceduke/ax/internal/checks"
	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/version"
	"github.com/briceduke/ax/targets"
)

func TestSkipWhenHashMatches(t *testing.T) {
	root := writeCompileProject(t)
	var buf bytes.Buffer
	if err := Run(root, Options{}, &buf); err != nil {
		t.Fatalf("compile 1: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "ok   cursor") || !strings.Contains(buf.String(), "ok   claude-code") {
		t.Fatalf("output = %q", buf.String())
	}
	buf.Reset()
	if err := Run(root, Options{}, &buf); err != nil {
		t.Fatalf("compile 2: %v\n%s", err, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "skip cursor") || !strings.Contains(out, "skip claude-code") {
		t.Fatalf("expected skip, got %q", out)
	}
}

func TestRecompileWhenCoreChanges(t *testing.T) {
	root := writeCompileProject(t)
	var buf bytes.Buffer
	if err := Run(root, Options{}, &buf); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ax", "intent.md"), []byte("# Intent\n\nchanged\n"), 0644); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := Run(root, Options{}, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "skip cursor") || strings.Contains(out, "skip claude-code") {
		t.Fatalf("expected recompile, got %q", out)
	}
	if !strings.Contains(out, "ok   cursor") {
		t.Fatalf("output = %q", out)
	}
}

func TestValidatorRejectsMissingLanding(t *testing.T) {
	root := writeCompileProject(t)
	var buf bytes.Buffer
	err := Run(root, Options{Writer: omitLanding{}}, &buf)
	if err == nil || !strings.Contains(err.Error(), "did not land") {
		t.Fatalf("error = %v, want missing landing", err)
	}
}

func TestWithoutExcludesScaffold(t *testing.T) {
	root := writeCompileProject(t)
	var buf bytes.Buffer
	if err := Run(root, Options{Without: "verify-pinouts"}, &buf); err != nil {
		t.Fatalf("compile: %v\n%s", err, buf.String())
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Fatal("main tree should stay clean")
	}
	work := filepath.Join(root, ".ax", "worktrees", "without-verify-pinouts")
	data, err := os.ReadFile(filepath.Join(work, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "Quote the table") {
		t.Fatalf("excluded scaffold still in adapters:\n%s", data)
	}
	if !strings.Contains(buf.String(), "wrote .ax/worktrees/without-verify-pinouts") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestFixtureCompileCheckStaleness(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	src := filepath.Join(repo, "testdata", "fixtures", "hardware")
	root := t.TempDir()
	copyTestTree(t, src, root)
	if err := os.WriteFile(filepath.Join(root, ".ax", "ax.yaml"), []byte("ax: "+version.Version+"\ntargets: [cursor, claude-code]\npacks: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Run(root, Options{}, &buf); err != nil {
		t.Fatalf("compile: %v\n%s", err, buf.String())
	}
	assertAdapters(t, root)
	buf.Reset()
	if err := checks.Run(root, checks.Options{Tier: checks.TierFull, All: true}, &buf); err != nil {
		t.Fatalf("check after compile: %v\n%s", err, buf.String())
	}
	agents := filepath.Join(root, "AGENTS.md")
	orig, err := os.ReadFile(agents)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agents, append(orig, []byte("tamper\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	err = checks.Run(root, checks.Options{Tier: checks.TierFull, All: true}, &buf)
	if err == nil || !strings.Contains(buf.String(), "content hash mismatch") {
		t.Fatalf("edited adapter: err=%v out=%q", err, buf.String())
	}
	if err := os.WriteFile(agents, orig, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ax", "intent.md"), []byte("# Intent\n\nedited without compile\n"), 0644); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	err = checks.Run(root, checks.Options{Tier: checks.TierFull, All: true}, &buf)
	if err == nil || !strings.Contains(buf.String(), "core hash changed") {
		t.Fatalf("core edit: err=%v out=%q", err, buf.String())
	}
}

func TestToolsCompileIntoMCP(t *testing.T) {
	root := writeCompileProject(t)
	if err := os.WriteFile(filepath.Join(root, ".ax", "tools.yaml"), []byte("- name: notes\n  command: python\n  args: [notes.py]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Run(root, Options{}, &buf); err != nil {
		t.Fatalf("compile: %v\n%s", err, buf.String())
	}
	for _, rel := range []string{".cursor/mcp.json", ".mcp.json"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text, `"notes"`) || !strings.Contains(text, `"python"`) || !strings.Contains(text, `"notes.py"`) {
			t.Fatalf("%s = %s", rel, text)
		}
	}
}

func TestAgentWriterValidates(t *testing.T) {
	root := writeCompileProject(t)
	var called int
	fake := runnerFunc(func(dir, task string) (string, error) {
		called++
		name := strings.TrimSpace(readFile(t, filepath.Join(dir, "TARGET")))
		spec, err := targets.Load(name)
		if err != nil {
			return "", err
		}
		hash, err := core.CompileHash(root, nil, nil)
		if err != nil {
			return "", err
		}
		snap, err := loadSnapshot(root, hash)
		if err != nil {
			return "", err
		}
		files, err := Assembler{}.Write(spec, snap)
		if err != nil {
			return "", err
		}
		for _, f := range files {
			path := filepath.Join(dir, filepath.FromSlash(f.Rel))
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return "", err
			}
			if err := os.WriteFile(path, f.Data, 0644); err != nil {
				return "", err
			}
		}
		return "ok", nil
	})
	var buf bytes.Buffer
	if err := Run(root, Options{UseAgent: true, Agent: fake}, &buf); err != nil {
		t.Fatalf("compile: %v\n%s", err, buf.String())
	}
	if called < 2 {
		t.Fatalf("agent called %d times, want both targets", called)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
}

func TestAgentWriterMissingFileFails(t *testing.T) {
	root := writeCompileProject(t)
	fake := runnerFunc(func(dir, task string) (string, error) {
		return "did nothing", nil
	})
	var buf bytes.Buffer
	err := Run(root, Options{UseAgent: true, Agent: fake}, &buf)
	if err == nil || !strings.Contains(err.Error(), "did not write") {
		t.Fatalf("error = %v", err)
	}
}

type runnerFunc func(dir, task string) (string, error)

func (f runnerFunc) Run(dir, task string) (string, error) {
	return f(dir, task)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestBuiltinRecordDecisionLands(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".ax/capabilities", ".ax/scaffolds"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".ax", "ax.yaml"), []byte("ax: "+version.Version+"\ntargets: [cursor, claude-code]\npacks: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ax", "intent.md"), []byte("# Intent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ax", "checks.yaml"), []byte("[]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Run(root, Options{}, &buf); err != nil {
		t.Fatalf("compile: %v\n%s", err, buf.String())
	}
	wantBody := map[string]string{
		"record-decision": "ax log decision",
		"log-friction":    "ax log friction",
		"sync-harness":    "ax compile",
	}
	for id, needle := range wantBody {
		for _, rel := range skillRels(id) {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatalf("missing %s: %v", rel, err)
			}
			if !bytes.Contains(data, []byte(needle)) {
				t.Fatalf("%s missing %q:\n%s", rel, needle, data)
			}
		}
	}
	for _, rel := range skillRels("record-decision") {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(data, []byte("enforces:")) {
			t.Fatalf("%s missing check prompt:\n%s", rel, data)
		}
	}
	for _, rel := range []string{"AGENTS.md", "CLAUDE.md"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		if !bytes.Contains(data, []byte("own-the-harness")) {
			t.Fatalf("%s missing own-the-harness:\n%s", rel, data)
		}
		if !bytes.Contains(data, []byte("markdown on stdin")) {
			t.Fatalf("%s missing decision stdin contract:\n%s", rel, data)
		}
	}
}

func TestTargetSpecUnknownField(t *testing.T) {
	_, err := targets.Parse([]byte("---\nid: x\ninstructions: A.md\ninvocable: i\nisolated: s\nhooks: h\nhooks_format: cursor-hooks\nmcp: m\nextra: 1\n---\n\nbody\n"))
	if err == nil || !strings.Contains(err.Error(), "yaml") {
		t.Fatalf("error = %v, want unknown field", err)
	}
}

type omitLanding struct{}

func (omitLanding) Write(spec *targets.Spec, snap *Snapshot) ([]File, error) {
	files, err := Assembler{}.Write(spec, snap)
	if err != nil {
		return nil, err
	}
	var out []File
	for _, f := range files {
		if len(f.Landing) > 0 && f.Rel != spec.Instructions {
			continue
		}
		out = append(out, f)
	}
	return out, nil
}

func assertAdapters(t *testing.T, root string) {
	t.Helper()
	mustExist := []string{
		"AGENTS.md",
		"CLAUDE.md",
		".cursor/hooks.json",
		".claude/settings.json",
		".cursor/mcp.json",
		".mcp.json",
		".ax/manifest.yaml",
	}
	mustExist = append(mustExist, skillRels("record-decision")...)
	mustExist = append(mustExist, skillRels("log-friction")...)
	mustExist = append(mustExist, skillRels("sync-harness")...)
	for _, rel := range mustExist {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	for _, rel := range []string{".cursor/mcp.json", ".mcp.json", ".cursor/hooks.json", ".claude/settings.json"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("<!--")) {
			t.Fatalf("%s contains a comment header", rel)
		}
	}
	for _, rel := range []string{"AGENTS.md", "CLAUDE.md", ".cursor/skills/record-decision/SKILL.md"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(data, []byte("generated by ax. do not edit.")) {
			t.Fatalf("%s missing generated header", rel)
		}
		if rel == "AGENTS.md" || rel == "CLAUDE.md" {
			if !bytes.Contains(data, []byte("own-the-harness")) {
				t.Fatalf("%s missing own-the-harness:\n%s", rel, data)
			}
			if !bytes.Contains(data, []byte("markdown on stdin")) {
				t.Fatalf("%s missing decision stdin contract:\n%s", rel, data)
			}
		}
	}
	skill := filepath.Join(root, filepath.FromSlash(".claude/skills/record-decision/SKILL.md"))
	var fm struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	data, err := os.ReadFile(skill)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.DecodeFrontmatter(data, &fm); err != nil {
		t.Fatalf("skill frontmatter: %v", err)
	}
}

func skillRels(id string) []string {
	return []string{
		".cursor/skills/" + id + "/SKILL.md",
		".claude/skills/" + id + "/SKILL.md",
	}
}

func writeCompileProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{".ax/capabilities", ".ax/scaffolds", ".ax/friction"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".ax", "ax.yaml"), []byte("ax: "+version.Version+"\ntargets: [cursor, claude-code]\npacks: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ax", "intent.md"), []byte("# Intent\n\nA compile fixture.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ax", "checks.yaml"), []byte("[]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	capInv := "---\nid: record-decision\nkind: capability\nwhen: a choice is made\nhints: [user-invocable]\n---\n\nWrite it down.\n"
	if err := os.WriteFile(filepath.Join(root, ".ax", "capabilities", "record-decision.md"), []byte(capInv), 0644); err != nil {
		t.Fatal(err)
	}
	capIso := "---\nid: isolate-work\nkind: capability\nwhen: work must not share context\nhints: [isolation]\n---\n\nUse a fresh agent.\n"
	if err := os.WriteFile(filepath.Join(root, ".ax", "capabilities", "isolate-work.md"), []byte(capIso), 0644); err != nil {
		t.Fatal(err)
	}
	sc := "---\nid: verify-pinouts\nkind: scaffold\ncompensates: invented pins\nadded: 2026-10-02 (friction/2026-10-02-wrong-pin)\nretire_when: eval passes\n---\n\nQuote the table.\n"
	if err := os.WriteFile(filepath.Join(root, ".ax", "scaffolds", "verify-pinouts.md"), []byte(sc), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func copyTestTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
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
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
