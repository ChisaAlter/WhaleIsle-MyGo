package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestReferenceDirectoryEnterSettles(t *testing.T) {
	d, m, _ := suggestionFixture(`@my`)
	m.FilePaths = []string{"my folder/"}
	if !handleSuggestKey(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}) {
		t.Fatal("not consumed")
	}
	if d.Text != `@"my folder/" ` {
		t.Fatalf("directory did not settle: %q", d.Text)
	}
	if _, _, _, _, open := activeSuggestion(m, d.Client); open {
		t.Fatal("settled directory reopened")
	}
}

func TestReferenceDirectoryTabDrillsAndCrumbsReturn(t *testing.T) {
	d, m, _ := suggestionFixture(`@src`)
	m.FilePaths = []string{"src/nested/"}
	if !handleSuggestKey(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyTab}) || d.Text != "@src/nested/" {
		t.Fatalf("Tab drill: %q", d.Text)
	}
	crumbs := referenceBreadcrumbs(m)
	if len(crumbs) != 3 || crumbs[0].insert != "@" || crumbs[1].insert != "@src/" || !crumbs[2].current {
		t.Fatalf("crumbs %+v", crumbs)
	}
	_, _, tok, _, _ := activeSuggestion(m, d.Client)
	applySuggestDrill(m, suggestItem{drill: crumbs[1].insert}, tok)
	if d.Text != "@src/" || len(referenceBreadcrumbs(m)) != 2 {
		t.Fatal("crumb failed")
	}
	d.Client.ReplaceText(nil, "other")
	if len(referenceBreadcrumbs(m)) != 0 {
		t.Fatal("typed query retained stale breadcrumb")
	}
}

func TestReferenceTypedPathHasNoBreadcrumbAndShiftTabDoesNotPick(t *testing.T) {
	d, m, _ := suggestionFixture(`@src/`)
	m.FilePaths = []string{"src/nested/"}
	if len(referenceBreadcrumbs(m)) != 0 {
		t.Fatal("typed query gained drill trail")
	}
	handleSuggestKey(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyTab, Mods: ui.Shift})
	if d.Text != "@src/" {
		t.Fatal("Shift Tab changed draft")
	}
}

func TestReferenceOutsideComposerDismissal(t *testing.T) {
	d, m, _ := suggestionFixture("@r")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.Button(c, "Outside composer")
			View(c, m)
		})
	}, 800, 600)
	tt.Click("Composer draft")
	if _, _, _, _, open := activeSuggestion(m, d.Client); !open {
		t.Fatal("editor click wrongly dismissed completion")
	}
	tt.Click("Outside composer")
	if _, _, _, _, open := activeSuggestion(m, d.Client); open {
		t.Fatal("outside click did not dismiss")
	}
	if d.Text != "@r" {
		t.Fatal("outside click changed draft")
	}
}

func TestReferenceEmptyDirectoryRetainsBackNavigation(t *testing.T) {
	d, m, _ := suggestionFixture("@src")
	m.FilePaths = []string{"src/empty/"}
	handleSuggestKey(d.Client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyTab})
	m.FilePaths = nil
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	if _, ok := tt.Find("工作区 / Workspace"); !ok {
		t.Fatal("empty folder lost breadcrumb")
	}
	tt.Click("工作区 / Workspace")
	if d.Text != "@" {
		t.Fatalf("root navigation %q", d.Text)
	}
}
