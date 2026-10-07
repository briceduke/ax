package compile

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/briceduke/ax/internal/core"
	"github.com/briceduke/ax/internal/logbook"
	"github.com/briceduke/ax/targets"
)

const (
	hooksClaudeSettings = "claude-settings"
	hooksCursorHooks    = "cursor-hooks"
)

// Assembler is a deterministic Writer that follows a target spec.
type Assembler struct{}

// Write emits adapter files named by spec. It does not invent extra tool paths.
func (Assembler) Write(spec *targets.Spec, snap *Snapshot) ([]File, error) {
	var files []File
	instructions, err := instructionsFile(spec, snap)
	if err != nil {
		return nil, err
	}
	files = append(files, instructions)
	for _, cap := range snap.Capabilities {
		if core.HasHint(cap, core.HintInvocable) {
			files = append(files, capabilityFile(spec.Invocable, cap))
		}
		if core.HasHint(cap, core.HintIsolation) {
			files = append(files, capabilityFile(spec.Isolated, cap))
		}
	}
	hooks, err := hooksFile(spec)
	if err != nil {
		return nil, err
	}
	files = append(files, hooks)
	files = append(files, mcpFile(spec, snap.Tools))
	return files, nil
}

func instructionsFile(spec *targets.Spec, snap *Snapshot) (File, error) {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(snap.Intent))
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	var landing []string
	alwaysOn := false
	writeAlwaysOn := func() {
		if alwaysOn {
			return
		}
		b.WriteString("## Always-on\n\n")
		alwaysOn = true
	}
	for _, sc := range snap.Scaffolds {
		writeAlwaysOn()
		fmt.Fprintf(&b, "### %s\n\nCompensates: %s\n\n%s\n\n", sc.ID, sc.Compensates, strings.TrimSpace(sc.Body))
	}
	for _, cap := range snap.Capabilities {
		if core.HasHint(cap, core.HintInvocable) || core.HasHint(cap, core.HintIsolation) {
			continue
		}
		writeAlwaysOn()
		fmt.Fprintf(&b, "### %s\n\nWhen: %s\n\n%s\n\n", cap.ID, cap.When, strings.TrimSpace(cap.Body))
		landing = append(landing, cap.ID)
	}
	if len(snap.Decisions) > 0 {
		b.WriteString("## Decisions in force\n\n")
		for _, d := range snap.Decisions {
			sentence, err := logbook.ChoiceSentence(d.Body)
			if err != nil {
				return File{}, fmt.Errorf("decision %s: %w", d.ID, err)
			}
			fmt.Fprintf(&b, "- %s (`.ax/decisions/%s.md`)\n", sentence, d.ID)
		}
		b.WriteString("\n")
	}
	return File{Rel: spec.Instructions, Data: []byte(strings.TrimSpace(b.String()) + "\n"), Landing: landing}, nil
}

func capabilityFile(pattern string, cap *core.Capability) File {
	rel := strings.ReplaceAll(pattern, "{id}", cap.ID)
	body := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", cap.ID, strconv.Quote(cap.When), strings.TrimSpace(cap.Body))
	return File{Rel: rel, Data: []byte(body), Landing: []string{cap.ID}}
}

func hooksFile(spec *targets.Spec) (File, error) {
	var data []byte
	var err error
	switch spec.HooksFormat {
	case hooksClaudeSettings:
		data, err = json.MarshalIndent(claudeSettings{
			Hooks: map[string][]claudeHookGroup{
				"PostToolUse": {{
					Matcher: "Write|Edit",
					Hooks:   []claudeHook{{Type: "command", Command: fastCheckCommand}},
				}},
			},
		}, "", "  ")
	case hooksCursorHooks:
		data, err = json.MarshalIndent(cursorHooksFile{
			Version: 1,
			Hooks: map[string][]cursorHook{
				"afterFileEdit": {{Command: fastCheckCommand}},
			},
		}, "", "  ")
	default:
		return File{}, fmt.Errorf("unknown hooks_format %q", spec.HooksFormat)
	}
	if err != nil {
		return File{}, err
	}
	return File{Rel: spec.Hooks, Data: data}, nil
}

func mcpFile(spec *targets.Spec, tools []core.Tool) File {
	servers := map[string]mcpServer{}
	for _, t := range tools {
		servers[t.Name] = mcpServer{Command: t.Command, Args: t.Args}
	}
	data, err := json.MarshalIndent(struct {
		MCPServers map[string]mcpServer `json:"mcpServers"`
	}{MCPServers: servers}, "", "  ")
	if err != nil {
		data = []byte("{\"mcpServers\":{}}")
	}
	return File{Rel: spec.MCP, Data: data}
}

type mcpServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

type claudeSettings struct {
	Hooks map[string][]claudeHookGroup `json:"hooks"`
}

type claudeHookGroup struct {
	Matcher string       `json:"matcher"`
	Hooks   []claudeHook `json:"hooks"`
}

type claudeHook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type cursorHooksFile struct {
	Version int                     `json:"version"`
	Hooks   map[string][]cursorHook `json:"hooks"`
}

type cursorHook struct {
	Command string `json:"command"`
}
