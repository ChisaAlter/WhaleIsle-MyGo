package files

import (
	"errors"
	"path/filepath"
	"testing"
	"whaleisle/internal/desktop/workspacefs"
)

func TestB26PageFailureAndReadOnlyEOF(t *testing.T) {
	tab := &FileTab{Preview: "first", Draft: "first", Truncated: true, textPaged: true, textPage: workspacefs.TextPage{Next: 6}}
	applyNextTextPage(Model{}, tab, textReadResult{err: errors.New("offline")})
	if tab.Preview != "first" || tab.textPage.Next != 6 || tab.Status != "offline" {
		t.Fatal("failure discarded prefix/cursor")
	}
	applyNextTextPage(Model{}, tab, textReadResult{page: &workspacefs.TextPage{Text: "second", Separator: "\n", Next: 8, EOF: true}})
	if tab.Preview != "first\nsecond" || !tab.Truncated || !tab.textPage.EOF || tab.Status != "" {
		t.Fatal("append or EOF save guard")
	}
	tab.Draft = "private edit"
	applyNextTextPage(Model{}, tab, textReadResult{page: &workspacefs.TextPage{Text: "unexpected"}})
	if tab.Draft != "private edit" || tab.Preview != "first\nsecond" {
		t.Fatal("overwrote draft")
	}
}

func TestB26ClosingTabResumesReboundPagedResource(t *testing.T) {
	selected, preview, draft := "other.txt", "other", "other"
	paged := &FileTab{Path: "large.txt", Preview: "prefix", Draft: "prefix", textPaged: true, readBlocked: true, readResume: true}
	other := &FileTab{Path: selected, Preview: preview, Draft: draft}
	root := filepath.Clean(filepath.FromSlash("/work"))
	tabs := &Tabs{Workspaces: map[string]*WorkspaceTabs{root: {Open: []*FileTab{paged, other}, Active: selected}}}
	model := Model{Tabs: tabs, WorkspaceRoot: root, SelectedFile: &selected, Preview: &preview, EditDraft: &draft}
	if !closeTab(model, "other.txt", false) || selected != "large.txt" || !paged.readResume {
		t.Fatal("paged Host rebind was stranded behind readBlocked")
	}
}
