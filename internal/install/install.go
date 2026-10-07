package install

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed migrate.md
var migrateBody string

const skillBody = "---\n" +
	"name: ax\n" +
	"description: Start an ax project in this folder. Invoke with /ax. Not suggested automatically.\n" +
	"disable-model-invocation: true\n" +
	"---\n\n" +
	"You start ax in this folder. The person typed /ax. Do not invent a project in a folder they never invoked this in.\n\n" +
	"If this folder has no `.ax/ax.yaml`:\n\n" +
	"1. Ask what they are making.\n" +
	"2. Ask who it is for.\n" +
	"3. Ask whether the notes are the work (hardware, research) or the code is the work (software).\n" +
	"4. If the folder already has product code, run `ax adopt`. Otherwise run `ax init --name \"<name>\" --intent \"<one line>\"`. Add `--notes record` when the notes are the work.\n" +
	"5. Saving writes `AGENTS.md` and `CLAUDE.md` before the command returns. The next chat already has the vision.\n\n" +
	"If `.ax` already exists, say so. Ordinary chat uses `AGENTS.md` or `CLAUDE.md`. They do not need /ax again for ordinary work.\n"

var skillFiles = []struct {
	rel  string
	body string
}{
	{".cursor/skills/ax/SKILL.md", skillBody},
	{".claude/skills/ax/SKILL.md", skillBody},
	{".cursor/skills/ax-migrate/SKILL.md", migrateBody},
	{".claude/skills/ax-migrate/SKILL.md", migrateBody},
}

// WriteSkills writes /ax and /ax-migrate under home/.cursor and home/.claude.
// Tests pass a temp directory. Do not default this to the real user home.
func WriteSkills(home string) error {
	home = filepath.Clean(home)
	if home == "" || home == "." {
		return fmt.Errorf("install home is empty")
	}
	for _, skill := range skillFiles {
		path := filepath.Join(home, filepath.FromSlash(skill.rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(skill.body), 0644); err != nil {
			return err
		}
	}
	return nil
}
