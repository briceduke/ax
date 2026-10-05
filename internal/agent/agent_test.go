package agent

import (
	"fmt"
	"strings"
	"testing"
)

func TestDetectPrefersAgentThenCursorThenClaude(t *testing.T) {
	var gotName string
	run := func(dir, name string, args ...string) (string, error) {
		gotName = name + " " + strings.Join(args, " ")
		return "ok", nil
	}
	look := func(file string) (string, error) {
		if file == "cursor" {
			return "/bin/cursor", nil
		}
		return "", fmt.Errorf("missing")
	}
	r := DetectWith(look, run)
	if r == nil {
		t.Fatal("expected runner")
	}
	if _, err := r.Run("/proj", "do the task"); err != nil {
		t.Fatal(err)
	}
	if gotName != "cursor agent -p do the task" {
		t.Fatalf("got %q", gotName)
	}
}

func TestDetectNone(t *testing.T) {
	r := DetectWith(func(string) (string, error) { return "", fmt.Errorf("missing") }, nil)
	if r != nil {
		t.Fatalf("got %#v", r)
	}
}
