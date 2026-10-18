package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestSessionAncestryForkAndMissing(t *testing.T) {
	items := []session.Item{{SessionID: "root", Title: "Root"}, {SessionID: "fork", ParentID: "root", Title: "Fork"}, {SessionID: "leaf", ParentID: "fork", Title: "Leaf"}}
	got := sessionAncestry(items, "leaf")
	if len(got) != 3 || got[0].SessionID != "root" || got[2].SessionID != "leaf" {
		t.Fatalf("ancestry: %+v", got)
	}
	items[1].ParentID = "unknown"
	got = sessionAncestry(items, "leaf")
	if len(got) != 2 || got[0].SessionID != "fork" {
		t.Fatalf("invented missing ancestor: %+v", got)
	}
	items[1].ParentID = "leaf"
	if got = sessionAncestry(items, "leaf"); len(got) != 2 {
		t.Fatalf("cycle: %+v", got)
	}
	if got = sessionAncestry(items, "unknown"); len(got) != 0 {
		t.Fatal("invented current summary")
	}
}

func TestSessionAncestryNavigation(t *testing.T) {
	items := []session.Item{{SessionID: "root", Title: "Root"}, {SessionID: "fork", ParentID: "root", Title: "Fork"}}
	selected := ""
	tt := ui.NewTester(func(c *ui.Context) {
		paintSessionAncestry(c, sessionAncestry(items, "fork"), func(id string) { selected = id })
	}, 600, 100)
	if !tt.HasText("Fork") {
		t.Fatal("current title missing")
	}
	if err := tt.Click("Root"); err != nil {
		t.Fatal(err)
	}
	if selected != "root" {
		t.Fatal("ancestor did not navigate")
	}
	selected = ""
	_ = tt.Click("Fork")
	if selected != "" {
		t.Fatal("current crumb must be text")
	}
}
