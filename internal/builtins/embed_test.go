package builtins

import (
	"strings"
	"testing"

	"github.com/briceduke/ax/internal/core"
)

func TestCapabilitiesIncludesHarnessNotes(t *testing.T) {
	caps, err := Capabilities()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*core.Capability{}
	for _, c := range caps {
		got[c.ID] = c
	}
	want := []struct {
		id          string
		invocable   bool
		mustContain string
	}{
		{id: "own-the-harness", invocable: false, mustContain: "markdown on stdin"},
		{id: "log-friction", invocable: true, mustContain: "ax log friction"},
		{id: "sync-harness", invocable: true, mustContain: "ax compile"},
		{id: "record-decision", invocable: true, mustContain: "pipe markdown on stdin"},
	}
	if len(caps) != len(want) {
		t.Fatalf("got %d capabilities, want %d", len(caps), len(want))
	}
	for i := 1; i < len(caps); i++ {
		if caps[i-1].ID > caps[i].ID {
			t.Fatalf("capabilities not sorted: %s before %s", caps[i-1].ID, caps[i].ID)
		}
	}
	for _, w := range want {
		c, ok := got[w.id]
		if !ok {
			t.Fatalf("missing %s", w.id)
		}
		if core.HasHint(c, core.HintInvocable) != w.invocable {
			t.Fatalf("%s invocable=%v, want %v", w.id, core.HasHint(c, core.HintInvocable), w.invocable)
		}
		if !strings.Contains(c.Body, w.mustContain) {
			t.Fatalf("%s body missing %q:\n%s", w.id, w.mustContain, c.Body)
		}
	}
}
