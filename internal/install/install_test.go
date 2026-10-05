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
	for _, rel := range skillRels {
		data, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		text := string(data)
		if !strings.Contains(text, "disable-model-invocation: true") {
			t.Fatalf("%s missing disable-model-invocation:\n%s", rel, text)
		}
		if !strings.Contains(text, "ax init") || !strings.Contains(text, "ax adopt") {
			t.Fatalf("%s missing init/adopt:\n%s", rel, text)
		}
	}
}

func TestWriteSkillsRejectsEmptyHome(t *testing.T) {
	if err := WriteSkills(""); err == nil {
		t.Fatal("expected error")
	}
}
