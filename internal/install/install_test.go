package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSkillsUsesGivenHome(t *testing.T) {
	home := t.TempDir()
	if err := WriteSkills(home); err != nil {
		t.Fatal(err)
	}
	for _, skill := range skillFiles {
		data, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(skill.rel)))
		if err != nil {
			t.Fatalf("missing %s: %v", skill.rel, err)
		}
		text := string(data)
		if !strings.Contains(text, "disable-model-invocation: true") {
			t.Fatalf("%s missing disable-model-invocation:\n%s", skill.rel, text)
		}
		if strings.Contains(skill.rel, "/ax/SKILL.md") {
			if !strings.Contains(text, "ax init") || !strings.Contains(text, "ax adopt") {
				t.Fatalf("%s missing init/adopt:\n%s", skill.rel, text)
			}
		}
		if strings.Contains(skill.rel, "ax-migrate") {
			if !strings.Contains(text, "0.6.0") || !strings.Contains(text, "one sentence") {
				t.Fatalf("%s missing migration steps:\n%s", skill.rel, text)
			}
		}
	}
}

func TestWriteSkillsRejectsEmptyHome(t *testing.T) {
	if err := WriteSkills(""); err == nil {
		t.Fatal("expected error")
	}
}
