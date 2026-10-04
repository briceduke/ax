package core

import (
	"fmt"
	"strings"

	"github.com/briceduke/ax/internal/yamlx"
)

// SplitFrontmatter splits a markdown file into YAML frontmatter and body.
func SplitFrontmatter(data []byte) ([]byte, string, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, "", fmt.Errorf("missing opening frontmatter delimiter")
	}
	rest := text[4:]
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		if strings.HasSuffix(rest, "\n---") {
			return []byte(rest[:len(rest)-4]), "", nil
		}
		return nil, "", fmt.Errorf("missing closing frontmatter delimiter")
	}
	yamlBytes := []byte(rest[:end])
	body := strings.TrimPrefix(rest[end+5:], "\n")
	return yamlBytes, body, nil
}

// DecodeFrontmatter splits a markdown file and decodes frontmatter into v.
func DecodeFrontmatter(data []byte, v interface{}) (string, error) {
	yamlBytes, body, err := SplitFrontmatter(data)
	if err != nil {
		return "", err
	}
	if err := yamlx.Decode(yamlBytes, v); err != nil {
		return "", err
	}
	return body, nil
}

// CountLines counts LF-normalized lines, ignoring a trailing newline.
func CountLines(text string) int {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}
