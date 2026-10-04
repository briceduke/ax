package logbook

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/core"
)

const decisionSkeleton = `## Context

## Options

## Choice

## Why
`

// WriteFriction writes a new one-line friction entry and returns its relative path.
func WriteFriction(root, text string, now time.Time) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("friction text is empty")
	}
	if core.CountLines(text) != 1 {
		return "", fmt.Errorf("friction body must be one line")
	}
	return writeEntry(root, "friction", text, now, func(id, date string) string {
		return fmt.Sprintf("id: %s\nkind: friction\ndate: %s\n", id, date)
	}, text+"\n")
}

// WriteObservation writes a new observation entry and returns its relative path.
func WriteObservation(root, finding, method string, now time.Time) (string, error) {
	finding = strings.TrimSpace(finding)
	method = strings.TrimSpace(method)
	if finding == "" {
		return "", fmt.Errorf("observation finding is empty")
	}
	if method == "" {
		return "", fmt.Errorf("observation --method is required")
	}
	return writeEntry(root, "observation", finding, now, func(id, date string) string {
		return fmt.Sprintf("id: %s\nkind: observation\ndate: %s\nmethod: %s\n", id, date, method)
	}, finding+"\n")
}

// WriteDecision writes a decision skeleton or piped body and returns its relative path.
func WriteDecision(root, title, supersedes, body string, now time.Time) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("decision title is empty")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		body = strings.TrimSpace(decisionSkeleton)
	}
	return writeEntry(root, "decision", title, now, func(id, date string) string {
		fm := fmt.Sprintf("id: %s\nkind: decision\ndate: %s\n", id, date)
		if supersedes != "" {
			fm += "supersedes:\n  - " + supersedes + "\n"
		}
		return fm
	}, body+"\n")
}

func writeEntry(root, kind, slugSource string, now time.Time, frontmatter func(id, date string) string, body string) (string, error) {
	slug := Slugify(slugSource)
	id, err := MintID(root, kind, slug, now)
	if err != nil {
		return "", err
	}
	date := now.Format("2006-01-02")
	content := "---\n" + frontmatter(id, date) + "---\n\n" + body
	rel := RelPath(kind, id)
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("write %s: %w", rel, err)
	}
	defer f.Close()
	if _, err := io.WriteString(f, content); err != nil {
		return "", err
	}
	return rel, nil
}
