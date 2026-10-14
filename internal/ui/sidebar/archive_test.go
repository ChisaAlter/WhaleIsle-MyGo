package sidebar

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestArchiveRosterFiltersActiveSearchAndKeepsOrphans(t *testing.T) {
	q := "match"
	snap := session.ViewState{Items: []session.Item{{SessionID: "a", Title: "match archived"}, {SessionID: "b", ParentID: "a", Title: "match child"}}, ArchivedSessionIDs: []string{"a"}, SearchRPCQuery: q, SearchHitIDs: map[string]bool{"a": true, "b": true}}
	items := sidebarItems(snap, &q)
	if len(items) != 1 || items[0].SessionID != "b" {
		t.Fatalf("active items: %+v", items)
	}
	if nodes := buildTree(items, nil); len(nodes) != 1 || nodes[0].item.SessionID != "b" {
		t.Fatalf("orphan hidden: %+v", nodes)
	}
}

func TestArchivedOnlySidebarKeepsManagementReachable(t *testing.T) {
	restored, deleted, selected := "", "", ""
	snap := session.ViewState{ArchivedSessionIDs: []string{"archived"}}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Snapshot: snap, OnSelect: func(id string) { selected = id }, OnUnarchive: func(id string) { restored = id }, OnDelete: func(id string) { deleted = id }})
	}, 280, 800)
	if err := tt.Click(i18n.T(i18n.KeySidebarUnarchive)); err != nil {
		t.Fatal(err)
	}
	if restored != "archived" || selected != "" {
		t.Fatalf("restore=%q selected=%q", restored, selected)
	}
	if err := tt.Click(i18n.T(i18n.KeySidebarDelete)); err != nil {
		t.Fatal(err)
	}
	if deleted != "archived" {
		t.Fatalf("delete=%q", deleted)
	}
}

func TestActiveRowHasArchiveButNoDelete(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Snapshot: session.ViewState{Items: []session.Item{{SessionID: "a", Title: "active"}}, SelectedID: "a"}, OnArchive: func(string) {}, OnDelete: func(string) {}})
	}, 280, 800)
	if _, ok := tt.Find(i18n.T(i18n.KeySidebarArchive)); !ok {
		t.Fatal("archive missing")
	}
	if _, ok := tt.Find(i18n.T(i18n.KeySidebarDelete)); ok {
		t.Fatal("active delete exposed")
	}
}
