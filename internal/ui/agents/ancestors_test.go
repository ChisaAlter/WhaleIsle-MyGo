package agents

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestAgentAncestorDirectSelection(t *testing.T) {
	depth := -1
	tt := ui.NewTester(func(c *ui.Context) {
		paintAgentAncestors(c, []session.AgentNavigationAncestor{{Depth: 0, SessionID: "root", Title: "Root"}, {Depth: 1, SessionID: "a", Title: "A"}}, func(d int) { depth = d })
	}, 500, 200)
	tt.Click("Agent ancestor root")
	if depth != 0 {
		t.Fatalf("root not directly reachable %d", depth)
	}
	tt.Click("Agent ancestor a")
	if depth != 1 {
		t.Fatalf("intermediate not directly reachable %d", depth)
	}
}
