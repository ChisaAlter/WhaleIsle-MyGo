package sidebar

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestArchiveFiltersRenderInPlaceAndKeepMissingSummary(t *testing.T) {
	for _, grouping := range []string{"workspace", "workspace-tree", "flat"} {
		t.Run(grouping, func(t *testing.T) {
			filter, restored, selected := "default", "", ""
			snap := session.ViewState{
				WorkspaceReady: true,
				ScratchCwd:     "/scratch",
				Items: []session.Item{
					{SessionID: "active", Title: "Active fixture", Cwd: "/project"},
					{SessionID: "archived", Title: "Archived fixture", Cwd: "/project"},
					{SessionID: "orphan", Title: "Orphan fixture", Cwd: "/removed"},
					{SessionID: "child", Title: "Subagent fixture", Origin: "subagent", Cwd: "/project"},
				},
				Workspaces:         []session.WorkspaceRow{{WorkspaceID: "workspace", Title: "Project fixture", Path: "/project", SessionIDs: []string{"active", "archived", "child"}}},
				ArchivedSessionIDs: []string{"archived", "missing", "orphan", "child"},
				WorkspaceLayout:    session.WorkspaceLayoutState{GroupBy: grouping, Collapsed: map[string]bool{"workspace": false}},
			}
			m := Model{Snapshot: snap, BrowseArchivedFilter: &filter, OnSelect: func(id string) { selected = id }, OnUnarchive: func(id string) { restored = id }}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 1000)
			label := i18n.T(i18n.KeySidebarArchivedSection)
			if !tt.HasText("Active fixture") || tt.HasText("Archived fixture") || !tt.HasText(label) {
				t.Fatal("default filter did not hide archived rows behind the collapsed section")
			}
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			if !tt.HasText("Active fixture") || !tt.HasText("Archived fixture") || tt.HasText("Orphan fixture") || tt.HasText("Subagent fixture") {
				t.Fatal("archive section did not respect active and registered membership")
			}
			if _, ok := tt.Find(sessionActionsLabel("missing")); !ok {
				t.Fatal("missing summary has no recovery action")
			}
			filter = "show"
			tt.Frame()
			if !tt.HasText("Active fixture") || !tt.HasText("Archived fixture") || tt.HasText("Orphan fixture") || tt.HasText("Subagent fixture") {
				t.Fatal("show filter did not retain live and archived rows")
			}
			if _, ok := tt.Find(sessionActionsLabel("archived")); !ok {
				t.Fatal("archived row not manageable in place")
			}
			if err := tt.Click(sessionActionsLabel("archived")); err != nil {
				t.Fatal(err)
			}
			if err := tt.ChooseMenuItem(i18n.T(i18n.KeySidebarUnarchive)); err != nil {
				t.Fatal(err)
			}
			if restored != "archived" || selected != "" {
				t.Fatalf("archive action restored=%q selected=%q", restored, selected)
			}
			filter = "only"
			tt.Frame()
			if tt.HasText("Active fixture") || !tt.HasText("Archived fixture") || tt.HasText("Orphan fixture") || tt.HasText("Subagent fixture") {
				t.Fatal("only filter did not restrict in-place rows to registered archives")
			}
			if _, ok := tt.Find(sessionActionsLabel("missing")); !ok {
				t.Fatal("missing summary lost recovery action under only filter")
			}
			m.HideArchivedList = true
			filter = "default"
			tt.Frame()
			if tt.HasText(label) || tt.HasText("Archived fixture") || !tt.HasText("Active fixture") {
				t.Fatal("hiding bottom section changed default main-list filter")
			}
			filter = "show"
			tt.Frame()
			if !tt.HasText("Archived fixture") {
				t.Fatal("hiding the bottom section suppressed the show filter")
			}
			m.Snapshot.ArchivedSessionIDs = []string{"archived"}
			tt.Frame()
			if tt.HasText(label) {
				t.Fatal("show filter duplicated a known archived row in the bottom section")
			}
			filter = "only"
			tt.Frame()
			if tt.HasText(label) {
				t.Fatal("only filter duplicated a known archived row in the bottom section")
			}
		})
	}
}

func TestArchiveOnlyTreeKeepsNearestVisibleAncestor(t *testing.T) {
	filter := "only"
	snap := session.ViewState{WorkspaceReady: true, Workspaces: []session.WorkspaceRow{
		{WorkspaceID: "root", Title: "Root", Path: "/project", SessionIDs: []string{"root-archive"}},
		{WorkspaceID: "empty", Title: "Empty middle", Path: "/project/middle"},
		{WorkspaceID: "child", Title: "Child", Path: "/project/middle/child", SessionIDs: []string{"child-archive"}},
	}, Items: []session.Item{{SessionID: "root-archive", Title: "Root archived"}, {SessionID: "child-archive", Title: "Child archived"}}, ArchivedSessionIDs: []string{"root-archive", "child-archive"}, WorkspaceLayout: session.WorkspaceLayoutState{GroupBy: "workspace-tree", Collapsed: map[string]bool{"root": true, "child": false}}}
	m := Model{BrowseArchivedFilter: &filter, Snapshot: snap}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 1000)
	if !tt.HasText("Root") || tt.HasText("Empty middle") || tt.HasText("Child") || len(VisibleBatchTargets(m)) != 0 {
		t.Fatal("visible descendants escaped a collapsed retained ancestor")
	}
	m.Snapshot.WorkspaceLayout.Collapsed["root"] = false
	tt.Frame()
	if !tt.HasText("Child") || !tt.HasText("Child archived") || tt.HasText("Empty middle") {
		t.Fatal("visible descendant did not attach to its nearest retained ancestor")
	}
	if targets := VisibleBatchTargets(m); len(targets) != 2 {
		t.Fatalf("expanded tree batch targets=%v", targets)
	}
}

func TestArchiveFilterVisibleBatchTargetsMatchRows(t *testing.T) {
	filter := "default"
	expanded := false
	m := Model{BrowseArchivedFilter: &filter, archiveExpanded: &expanded, Snapshot: session.ViewState{
		WorkspaceReady:     true,
		Items:              []session.Item{{SessionID: "active", Title: "Active"}, {SessionID: "stored", Title: "Stored"}},
		Workspaces:         []session.WorkspaceRow{{WorkspaceID: "w", SessionIDs: []string{"active", "stored"}}},
		WorkspaceLayout:    session.WorkspaceLayoutState{Collapsed: map[string]bool{"w": false}},
		ArchivedSessionIDs: []string{"stored", "missing"},
	}}
	ids := func() []string {
		var result []string
		for _, target := range VisibleBatchTargets(m) {
			result = append(result, target.ID)
		}
		return result
	}
	if got := ids(); !reflect.DeepEqual(got, []string{"active"}) {
		t.Fatalf("closed: %v", got)
	}
	expanded = true
	if got := ids(); !reflect.DeepEqual(got, []string{"active", "stored", "missing"}) {
		t.Fatalf("open: %v", got)
	}
	filter = "show"
	if got := ids(); !reflect.DeepEqual(got, []string{"active", "stored", "missing"}) {
		t.Fatalf("show: %v", got)
	}
	filter = "only"
	if got := ids(); !reflect.DeepEqual(got, []string{"stored", "missing"}) {
		t.Fatalf("only: %v", got)
	}
	expanded = false
	if got := ids(); !reflect.DeepEqual(got, []string{"stored"}) {
		t.Fatalf("only with missing section closed: %v", got)
	}
}
