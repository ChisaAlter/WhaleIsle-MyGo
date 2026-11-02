package sidebar

import (
	"reflect"
	"testing"

	"whaleisle/internal/store/session"
)

func TestWorkspaceRowActionsGiveRestingTitleFullWidthAndPreserveTargets(t *testing.T) {
	tt, m, _ := workspaceHoverFixture(t)
	var created []string
	var generations []uint64
	m.OnCreateWorkspaceSession = func(g uint64, id string) { generations = append(generations, g); created = append(created, id) }
	rest, _ := tt.Find(m.Snapshot.Workspaces[0].Title)
	moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[0].Title)
	hover, _ := tt.Find(m.Snapshot.Workspaces[0].Title)
	more, moreOK := tt.Find("Workspace actions a")
	create, createOK := tt.Find("New session a")
	if !moreOK || !createOK || more.W != 16 || more.H != 16 || create.W != 16 || create.H != 16 || more.X+more.W+15 != create.X {
		t.Fatalf("workspace glyph geometry more=%+v create=%+v", more, create)
	}
	if rest.W-hover.W != 47 || rest.X != hover.X || rest.H != hover.H {
		t.Fatalf("resting title reserved actions: rest=%+v hover=%+v", rest, hover)
	}
	if err := tt.Click("New session a"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(created, []string{"a"}) || !reflect.DeepEqual(generations, []uint64{11}) || m.Snapshot.SelectedID != "current-a" {
		t.Fatalf("create targets=%v generations=%v selected=%q", created, generations, m.Snapshot.SelectedID)
	}
	// An open row menu preserves its action group and title lane even when
	// the pointer moves off the header.
	if err := tt.Click("Workspace actions a"); err != nil {
		t.Fatal(err)
	}
	tt.Move(730, 330)
	open, _ := tt.Find(m.Snapshot.Workspaces[0].Title)
	if open.W != hover.W {
		t.Fatalf("menu-open action lane changed: hover=%+v open=%+v", hover, open)
	}
	tt.CloseMenu()
}

func TestWorkspaceHighlightUsesExpansionAndCurrentAncestors(t *testing.T) {
	groups := []session.WorkspaceGroup{
		{Key: "root", Kind: "workspace", Workspace: session.WorkspaceRow{WorkspaceID: "root"}},
		{Key: "child", ParentID: "root", Kind: "workspace", Workspace: session.WorkspaceRow{WorkspaceID: "child", SessionIDs: []string{"current"}}},
		{Key: "sibling", ParentID: "root", Kind: "workspace", Workspace: session.WorkspaceRow{WorkspaceID: "sibling", SessionIDs: []string{"other"}}},
		{Key: "::scratch", Kind: "scratch", Items: []session.Item{{SessionID: "scratch-current"}}},
	}
	for _, test := range []struct {
		name, selected                                            string
		rootClosed, childClosed, wantRoot, wantChild, wantScratch bool
	}{
		{"open current", "current", false, false, true, true, false},
		{"closed current", "current", false, true, true, false, false},
		{"closed ancestor", "current", true, false, true, true, false},
		{"both closed", "current", true, true, true, false, false},
		{"scratch current", "scratch-current", false, false, false, false, true},
		{"no current", "", false, false, false, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := Model{Snapshot: session.ViewState{SelectedID: test.selected, WorkspaceLayout: session.WorkspaceLayoutState{GroupBy: "workspace-tree", Collapsed: map[string]bool{"root": test.rootClosed, "child": test.childClosed, "::scratch": true}}}}
			active := workspaceGroupHighlights(m, groups)
			if active["root"] != test.wantRoot || active["child"] != test.wantChild || active["::scratch"] != test.wantScratch || active["sibling"] {
				t.Fatalf("highlight=%v", active)
			}
		})
	}
}

func TestWorkspaceHoverAbsentCreatedAtAndScratchNeverInventCards(t *testing.T) {
	tt, m, start := workspaceHoverFixture(t)
	m.Snapshot.Workspaces[0].CreatedAt = ""
	tt.Frame()
	moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[0].Title)
	tt.SetTime(start.Add(2 * workspaceHoverDelay))
	if tt.HasText("Workspace preview:") {
		t.Fatal("registry row without createdAt has a hover card")
	}
	// This keeps the normal workspace group and its title geometry present;
	// it suppresses only the card, with no invented creation time.
	if r, ok := tt.Find(m.Snapshot.Workspaces[0].Title); !ok || r.H != 32 {
		t.Fatalf("ordinary title lost: %+v", r)
	}
}
