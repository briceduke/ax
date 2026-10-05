package project

import (
	"path"
	"path/filepath"
)

const (
	Dir         = ".ax"
	RecordDir   = "record"
	NotesRecord = "record"
)

// Rel returns a slash-separated path under .ax/.
func Rel(elem ...string) string {
	parts := append([]string{Dir}, elem...)
	return path.Join(parts...)
}

// Join is filepath.Join(root, Rel(elem...)).
func Join(root string, elem ...string) string {
	return filepath.Join(root, filepath.FromSlash(Rel(elem...)))
}

// DirPath is the .ax directory for root.
func DirPath(root string) string {
	return filepath.Join(root, Dir)
}

// RecordPath is the visible record/ directory for root.
func RecordPath(root string) string {
	return filepath.Join(root, RecordDir)
}

// NotesVisible reports whether notes also appear in record/.
func NotesVisible(cfg *Config) bool {
	return cfg != nil && cfg.Notes == NotesRecord
}
