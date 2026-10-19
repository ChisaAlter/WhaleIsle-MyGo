package files

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"whaleisle/internal/desktop/workspacefs"
)

func linePagingFixture(t *testing.T, read func(context.Context, string, int64) (workspacefs.TextPage, error)) (Model, *FileTab, <-chan struct{}) {
	t.Helper()
	selected, preview, draft := "fixture.txt", "first\nsecond", "first\nsecond"
	root := filepath.Clean("isolated-fixture-root")
	tab := &FileTab{Path: selected, Preview: preview, Draft: draft, Remote: true, Truncated: true, readID: 1,
		textPaged: true, textPage: workspacefs.TextPage{Text: preview, Version: "v1", Next: 3}}
	tabs := &Tabs{binding: "fixture-host", boundRoot: root, Workspaces: map[string]*WorkspaceTabs{
		root: {Open: []*FileTab{tab}, Active: selected},
	}}
	notified := make(chan struct{}, 16)
	m := Model{Tabs: tabs, WorkspaceRoot: root, BindingID: "fixture-host", SessionID: "fixture-session",
		SelectedFile: &selected, Preview: &preview, EditDraft: &draft, ReadTextPageContext: read,
		notify: func() { notified <- struct{}{} }}
	t.Cleanup(func() { tabs.stopTextRead(false, true) })
	return m, tab, notified
}

func TestLineNavigationContinuesBeyondFirstPage(t *testing.T) {
	var offsets []int64 // One bounded reader worker, inspected after its completion.
	m, tab, done := linePagingFixture(t, func(_ context.Context, _ string, offset int64) (workspacefs.TextPage, error) {
		offsets = append(offsets, offset)
		if offset == 3 {
			return workspacefs.TextPage{Text: "third\nfourth", Separator: "\n", Version: "v1", Next: 5}, nil
		}
		return workspacefs.TextPage{Text: "fifth\nsixth", Separator: "\n", Version: "v1", Next: 7, EOF: true}, nil
	})
	// Navigation can arrive while the initial read still owns an empty body.
	tab.Preview, tab.Draft, *m.Preview, *m.EditDraft = "", "", "", ""
	tab.textPaged, tab.readLoading = false, true
	RequestLine(m, 5)
	tab.readLoading = false
	applyTextRead(m, textReadResult{text: "first\nsecond", remote: true, truncated: true,
		page: &workspacefs.TextPage{Text: "first\nsecond", Version: "v1", Next: 3}})
	if tab.revealLine != 0 || tab.pendingLine.line != 5 || tab.lineDraft != "5" {
		t.Fatal("initial prefix consumed or clamped the target")
	}
	awaitTextRead(t, done)
	syncTextRead(m)
	if tab.revealLine != 0 || tab.pendingLine.line != 5 || !tab.readLoading {
		t.Fatal("intermediate page did not continue the original target")
	}
	awaitTextRead(t, done)
	syncTextRead(m)
	if tab.revealLine != 5 || tab.pendingLine.line != 0 || tab.lineDraft != "5" || len(offsets) != 2 || offsets[0] != 3 || offsets[1] != 5 {
		t.Fatalf("target/cursor did not settle: reveal=%d pending=%+v offsets=%v", tab.revealLine, tab.pendingLine, offsets)
	}
}

func TestLineNavigationEOFClampAndExplicitPageRetry(t *testing.T) {
	var calls atomic.Int32
	m, tab, done := linePagingFixture(t, func(_ context.Context, _ string, offset int64) (workspacefs.TextPage, error) {
		if offset != 3 {
			return workspacefs.TextPage{}, errors.New("unexpected fixture cursor")
		}
		if calls.Add(1) == 1 {
			return workspacefs.TextPage{}, errors.New("fixture offline")
		}
		return workspacefs.TextPage{Text: "third", Separator: "\n", Version: "v1", Next: 4, EOF: true}, nil
	})
	RequestLine(m, 99)
	awaitTextRead(t, done)
	syncTextRead(m)
	if tab.pendingLine.line != 99 || tab.revealLine != 0 || tab.Preview != "first\nsecond" || tab.textPage.Next != 3 || tab.Status == "" {
		t.Fatal("failed page lost target, prefix, cursor or error")
	}
	syncTextRead(m)
	if calls.Load() != 1 || tab.readLoading {
		t.Fatal("failed page automatically retried")
	}
	beginNextTextPage(m) // Existing user retry control; no new RPC/action.
	awaitTextRead(t, done)
	syncTextRead(m)
	if tab.revealLine != 3 || tab.lineDraft != "3" || tab.pendingLine.line != 0 || !tab.textPage.EOF || !tab.Truncated {
		t.Fatal("EOF did not clamp the original target while retaining read-only state")
	}
}

func TestLineNavigationTabCancelRetiresLatePageAndResumesTarget(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	var calls atomic.Int32
	m, tab, done := linePagingFixture(t, func(_ context.Context, _ string, _ int64) (workspacefs.TextPage, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-release // Obsolete generation deliberately ignores cancellation.
			return workspacefs.TextPage{Text: "obsolete", Separator: "\n", Version: "v1", Next: 4, EOF: true}, nil
		}
		return workspacefs.TextPage{Text: "third", Separator: "\n", Version: "v1", Next: 4, EOF: true}, nil
	})
	RequestLine(m, 3)
	awaitTextRead(t, started)
	m.Tabs.stopTextRead(true, false) // The existing tab-departure cancellation.
	if !tab.pendingLine.resume || tab.pendingLine.line != 3 {
		t.Fatal("tab departure lost the pending line")
	}
	*m.SelectedFile = "other.txt"
	syncTextRead(m)
	if calls.Load() != 1 {
		t.Fatal("hidden occurrence resumed a page")
	}
	*m.SelectedFile = tab.Path
	syncTextRead(m)
	once.Do(func() { close(release) })
	awaitTextRead(t, done)
	wrong := textReadOwnerFor(m, tab)
	wrong.tab++
	if _, ready := m.Tabs.textReader.take(wrong); ready {
		t.Fatal("page escaped occurrence ownership")
	}
	syncTextRead(m)
	if tab.Preview != "first\nsecond\nthird" || tab.revealLine != 3 || tab.pendingLine.line != 0 || calls.Load() != 2 {
		t.Fatal("cancelled generation was applied or target did not resume")
	}
}

func TestLineNavigationVersionAndHostRetirePendingTarget(t *testing.T) {
	m, tab, done := linePagingFixture(t, func(context.Context, string, int64) (workspacefs.TextPage, error) {
		return workspacefs.TextPage{Text: "new version", Version: "v2", Next: 4, EOF: true}, nil
	})
	RequestLine(m, 3)
	awaitTextRead(t, done)
	syncTextRead(m)
	if tab.pendingLine.line != 0 || tab.revealLine != 0 || tab.Preview != "first\nsecond" || tab.Status == "" {
		t.Fatal("changed version inherited navigation or appended incompatible bytes")
	}
	tab.pendingLine.begin()
	tab.pendingLine.deferRequest(8)
	tab.pendingLine.acceptVersion("v1")
	if tab.pendingLine.acceptVersion("v2") || tab.pendingLine.line != 0 {
		t.Fatal("reload did not retire old version navigation")
	}
	tab.pendingLine.begin()
	tab.pendingLine.deferRequest(9)
	m.Tabs.Bind("different-fixture-host", m.WorkspaceRoot)
	if tab.pendingLine.line != 0 || tab.revealLine != 0 {
		t.Fatal("Host rebind retained old navigation")
	}
}
