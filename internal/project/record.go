package project

import (
	"io"
	"os"
	"path/filepath"
)

var noteFiles = []string{"intent.md"}

var noteDirs = []string{"decisions", "observations", "friction"}

// ShowNotes copies vision, decisions, measurements, and pain notes into record/.
func ShowNotes(root string) error {
	cfg, err := Load(root)
	if err != nil {
		return err
	}
	cfg.Notes = NotesRecord
	if err := Save(root, cfg); err != nil {
		return err
	}
	return SyncRecord(root)
}

// HideNotes removes record/ and keeps the same files only under .ax/.
func HideNotes(root string) error {
	cfg, err := Load(root)
	if err != nil {
		return err
	}
	cfg.Notes = ""
	if err := Save(root, cfg); err != nil {
		return err
	}
	return os.RemoveAll(RecordPath(root))
}

// SyncRecord mirrors note files into record/ when notes are the work.
func SyncRecord(root string) error {
	cfg, err := Load(root)
	if err != nil {
		return err
	}
	if !NotesVisible(cfg) {
		return nil
	}
	dest := RecordPath(root)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	for _, name := range noteFiles {
		src := Join(root, name)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if err := copyFile(src, filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	for _, name := range noteDirs {
		if err := mirrorDir(Join(root, name), filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	return nil
}

func mirrorDir(src, dest string) error {
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if os.IsNotExist(err) {
		return os.MkdirAll(dest, 0755)
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := copyFile(filepath.Join(src, e.Name()), filepath.Join(dest, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
