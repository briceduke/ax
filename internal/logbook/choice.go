package logbook

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const maxChoiceChars = 200

// ChoiceSentence is the one sentence copied into AGENTS.md and CLAUDE.md.
// The decision file keeps Context, Options, and Why.
func ChoiceSentence(body string) (string, error) {
	choice := strings.TrimSpace(ExtractSection(body, "Choice"))
	if choice == "" {
		return "", fmt.Errorf("choice is empty")
	}
	if strings.Contains(choice, "\n") || strings.Contains(choice, ". ") || strings.Contains(choice, "! ") || strings.Contains(choice, "? ") {
		return "", fmt.Errorf("choice must be one sentence")
	}
	n := utf8.RuneCountInString(choice)
	if n > maxChoiceChars {
		return "", fmt.Errorf("choice is %d characters, max %d", n, maxChoiceChars)
	}
	return choice, nil
}
