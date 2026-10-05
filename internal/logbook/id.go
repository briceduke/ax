package logbook

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/project"
)

var (
	idPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-[a-z0-9]+(-[a-z0-9]+)*$`)
	wordClean = regexp.MustCompile(`[^a-z0-9]+`)
)

// KindDir returns the log subdirectory for a kind.
func KindDir(kind string) string {
	switch kind {
	case "decision":
		return "decisions"
	case "observation":
		return "observations"
	case "friction":
		return "friction"
	default:
		return kind + "s"
	}
}

// KindFromDir is the inverse of KindDir.
func KindFromDir(dir string) (string, bool) {
	switch dir {
	case "decisions":
		return "decision", true
	case "observations":
		return "observation", true
	case "friction":
		return "friction", true
	default:
		return "", false
	}
}

// RelDir is .ax/<kind-dir> using slash separators.
func RelDir(kind string) string {
	return project.Rel(KindDir(kind))
}

// RelPath is .ax/<kind-dir>/<id>.md using slash separators.
func RelPath(kind, id string) string {
	return RelDir(kind) + "/" + id + ".md"
}

// TitleFromID turns a date-slug id into a short title for root-file lists.
func TitleFromID(id string) string {
	parts := strings.Split(id, "-")
	if len(parts) < 4 {
		return strings.ReplaceAll(id, "-", " ")
	}
	return strings.ReplaceAll(strings.Join(parts[3:], "-"), "-", " ")
}

// ValidID reports whether id matches the date-slug pattern.
func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

// Slugify turns text into a lowercase hyphenated slug of the first 8 words.
func Slugify(text string) string {
	words := strings.Fields(strings.ToLower(text))
	if len(words) > 8 {
		words = words[:8]
	}
	joined := strings.Join(words, "-")
	slug := wordClean.ReplaceAllString(joined, "-")
	slug = strings.Trim(slug, "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	if slug == "" {
		return "entry"
	}
	return slug
}

// MintID builds date-slug and appends -2, -3, ... on a same-day collision.
func MintID(root, kind, slug string, now time.Time) (string, error) {
	date := now.Format("2006-01-02")
	base := date + "-" + slug
	id := base
	for n := 2; ; n++ {
		path := filepath.Join(root, filepath.FromSlash(RelPath(kind, id)))
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return id, nil
		}
		if err != nil {
			return "", err
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
}
