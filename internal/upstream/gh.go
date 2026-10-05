package upstream

import (
	"fmt"
	"os"
	"strings"
)

const axRepo = "briceduke/ax"

type ghSubmitter struct {
	run  Commander
	repo string
}

// GHSubmitter files a GitHub issue with gh against the ax repo. It does not merge anything.
func GHSubmitter(run Commander) Submitter {
	return ghSubmitter{run: run, repo: axRepo}
}

// GHIssue files a GitHub issue. Empty repo uses gh's current repository.
func GHIssue(run Commander, repo string) Submitter {
	return ghSubmitter{run: run, repo: repo}
}

func (g ghSubmitter) Submit(title, body string) (string, error) {
	f, err := os.CreateTemp("", "ax-upstream-*.md")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	args := []string{"issue", "create"}
	if g.repo != "" {
		args = append(args, "--repo", g.repo)
	}
	args = append(args, "--title", title, "--body-file", path)
	out, err := g.run(args...)
	if err != nil {
		return "", fmt.Errorf("gh issue create: %w", err)
	}
	url := strings.TrimSpace(out)
	if url == "" {
		return "submitted", nil
	}
	return url, nil
}
