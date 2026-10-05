package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/briceduke/ax/internal/project"
)

// NormalizeLF rewrites CRLF to LF so hashes match across Windows and Linux.
func NormalizeLF(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			continue
		}
		out = append(out, data[i])
	}
	return out
}

// HashFiles hashes sorted relative paths and their LF-normalized contents.
func HashFiles(root string, rels []string) (string, error) {
	h := sha256.New()
	if err := WriteFiles(h, root, rels); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// WriteFiles writes path/content pairs into h in sorted path order.
func WriteFiles(h hash.Hash, root string, rels []string) error {
	sorted := append([]string(nil), rels...)
	sort.Strings(sorted)
	for _, rel := range sorted {
		if err := writeFile(h, root, rel); err != nil {
			return err
		}
	}
	return nil
}

func writeFile(h hash.Hash, root, rel string) error {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return fmt.Errorf("hash %s: %w", rel, err)
	}
	return WriteBytes(h, rel, data)
}

// WriteBytes writes one path/content pair into h.
func WriteBytes(h hash.Hash, rel string, data []byte) error {
	if _, err := io.WriteString(h, rel+"\n"); err != nil {
		return err
	}
	if _, err := h.Write(NormalizeLF(data)); err != nil {
		return err
	}
	_, err := h.Write([]byte{0})
	return err
}

// ContentHash returns the sha256 of LF-normalized bytes.
func ContentHash(data []byte) string {
	sum := sha256.Sum256(NormalizeLF(data))
	return hex.EncodeToString(sum[:])
}

// InputHash hashes intent, capabilities, scaffolds and checks.yaml.
func InputHash(root string) (string, error) {
	return CompileHash(root, nil, nil)
}

// CompileHash hashes core inputs, extra files (target specs), and active scaffolds.
func CompileHash(root string, extra map[string][]byte, without []string) (string, error) {
	files, err := coreFiles(root, without)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if err := WriteFiles(h, root, files); err != nil {
		return "", err
	}
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := WriteBytes(h, k, extra[k]); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func coreFiles(root string, without []string) ([]string, error) {
	var files []string
	for _, p := range []string{project.Rel("intent.md"), project.Rel("checks.yaml"), ToolsFile} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err == nil {
			files = append(files, p)
		}
	}
	caps, err := ListMarkdown(root, project.Rel("capabilities"))
	if err != nil {
		return nil, err
	}
	files = append(files, caps...)
	scaffolds, err := listActiveScaffolds(root, without)
	if err != nil {
		return nil, err
	}
	files = append(files, scaffolds...)
	decs, err := ListMarkdown(root, project.Rel("decisions"))
	if err != nil {
		return nil, err
	}
	files = append(files, decs...)
	return files, nil
}

func listActiveScaffolds(root string, without []string) ([]string, error) {
	rels, err := ListMarkdown(root, project.Rel("scaffolds"))
	if err != nil {
		return nil, err
	}
	if len(without) == 0 {
		return rels, nil
	}
	skip := map[string]struct{}{}
	for _, id := range without {
		skip[id] = struct{}{}
	}
	var active []string
	for _, rel := range rels {
		sc, err := ParseScaffold(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		if _, omit := skip[sc.ID]; omit {
			continue
		}
		active = append(active, rel)
	}
	return active, nil
}
