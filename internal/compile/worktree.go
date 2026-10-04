package compile

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/briceduke/ax/internal/core"
)

// WorktreeDir is the local copy used for compile --without experiments.
func WorktreeDir(root, without string) string {
	return filepath.Join(root, ".ax", "worktrees", "without-"+without)
}

func isolate(root, without string) (string, error) {
	if without == "" {
		return root, nil
	}
	rel, err := scaffoldRel(root, without)
	if err != nil {
		return "", err
	}
	dest := WorktreeDir(root, without)
	if err := os.RemoveAll(dest); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return "", err
	}
	if err := gitWorktreeAdd(root, dest); err != nil {
		if rmErr := os.RemoveAll(dest); rmErr != nil {
			return "", rmErr
		}
		if err := copyTree(root, dest); err != nil {
			return "", err
		}
	}
	path := filepath.Join(dest, filepath.FromSlash(rel))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("exclude scaffold %s: %w", without, err)
	}
	return dest, nil
}

func scaffoldRel(root, id string) (string, error) {
	rels, err := core.ListMarkdown(root, "core/scaffolds")
	if err != nil {
		return "", err
	}
	for _, rel := range rels {
		sc, err := core.ParseScaffold(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		if sc.ID == id {
			return rel, nil
		}
	}
	return "", fmt.Errorf("scaffold %q not found", id)
}

func gitWorktreeAdd(root, dest string) error {
	rev := exec.Command("git", "rev-parse", "--verify", "HEAD")
	rev.Dir = root
	if err := rev.Run(); err != nil {
		return err
	}
	cmd := exec.Command("git", "worktree", "add", "--detach", dest, "HEAD")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
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
		base := filepath.Base(path)
		if (base == ".git" || base == ".ax") && info.IsDir() && path != src {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}
