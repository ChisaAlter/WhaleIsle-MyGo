package files

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/desktop/workspacefs"
)

func TestClosePreflightCapturesDraftAndRejectsOtherRootBeforeAnyWrite(t *testing.T) {
	m, _ := autosaveModel(t)
	disposeSave(selectedTab(m))
	*m.EditDraft = "committed current input"
	otherRoot := t.TempDir()
	other := &FileTab{Path: "a.txt", Preview: "other base", Draft: "other draft", Editing: true}
	m.Tabs.Workspaces[filepath.Clean(otherRoot)] = &WorkspaceTabs{Open: []*FileTab{other}, Active: other.Path}
	var writes atomic.Int32
	m.OnSaveEdit = func(string, string) error { writes.Add(1); return nil }
	if err := PreflightDirtyForClose(m); err == nil {
		t.Fatal("cross-root draft passed preflight")
	}
	if pending, err := SaveDirtyForClose(m); pending || err == nil {
		t.Fatalf("cross-root save pending=%v err=%v", pending, err)
	}
	if writes.Load() != 0 || selectedTab(m).save != nil || other.save != nil {
		t.Fatal("preflight scheduled a write before all roots passed")
	}
	if selectedTab(m).Draft != "committed current input" || other.Draft != "other draft" {
		t.Fatal("preflight discarded input")
	}
}

func TestDirtyLocalDraftAfterTruncatedRefreshSavesOnlyOnExplicitGesture(t *testing.T) {
	m, done := autosaveModel(t)
	*m.EditDraft = "retained complete authored draft"
	unknown := strings.Repeat("external bytes\n", workspacefs.TextPreviewLimit/15+64)
	path := filepath.Join(m.WorkspaceRoot, "a.txt")
	if err := os.WriteFile(path, []byte(unknown), 0600); err != nil {
		t.Fatal(err)
	}
	reloadSelectedFile(m)
	deadline := time.After(3 * time.Second)
	for selectedTab(m).readLoading {
		select {
		case <-done:
			syncTextRead(m)
		case <-deadline:
			t.Fatal("truncated refresh did not complete")
		}
	}
	tab := selectedTab(m)
	if !tab.Truncated || !tab.readBlocked || *m.Preview != "original-a" || *m.EditDraft != "retained complete authored draft" {
		t.Fatalf("truncated refresh adopted prefix/lost draft: %+v", tab)
	}
	observeAutosave(m)
	if tab.save != nil {
		t.Fatal("incomplete refresh restarted automatic saving")
	}
	if disk, err := os.ReadFile(path); err != nil || string(disk) != unknown {
		t.Fatal("refresh/observation overwrote unknown bytes")
	}
	if err := requestSaveDraft(m); err != nil {
		t.Fatal(err)
	}
	awaitAutosave(t, done)
	syncSaveResults(m)
	if tab.Truncated || tab.readBlocked || tab.Dirty() || *m.Preview != "retained complete authored draft" {
		t.Fatalf("explicit captured save did not settle: %+v", tab)
	}
	if disk, err := os.ReadFile(path); err != nil || string(disk) != "retained complete authored draft" {
		t.Fatalf("explicit save disk=%q err=%v", disk, err)
	}
}

func TestCloseSaveReceiptDoesNotConfirmNewerInputOrRepeatWrite(t *testing.T) {
	m, done := autosaveModel(t)
	disposeSave(selectedTab(m))
	started, release := make(chan struct{}), make(chan struct{})
	write := m.OnSaveEdit
	var writes atomic.Int32
	m.OnSaveEdit = func(rel, text string) error { writes.Add(1); close(started); <-release; return write(rel, text) }
	*m.EditDraft = "captured close input"
	if err := PreflightDirtyForClose(m); err != nil {
		t.Fatal(err)
	}
	if selectedTab(m).save != nil {
		t.Fatal("read-only preflight created a queue")
	}
	if pending, err := SaveDirtyForClose(m); !pending || err != nil {
		t.Fatalf("save pending=%v err=%v", pending, err)
	}
	awaitAutosave(t, started)
	*m.EditDraft = "new input while saving"
	for range 3 {
		if pending, err := CheckDirtyForClose(m); !pending || err != nil {
			t.Fatalf("poll pending=%v err=%v", pending, err)
		}
	}
	close(release)
	awaitAutosave(t, done)
	if pending, err := CheckDirtyForClose(m); pending || err == nil {
		t.Fatalf("newer input accepted as saved: pending=%v err=%v", pending, err)
	}
	if writes.Load() != 1 || *m.Preview != "captured close input" || *m.EditDraft != "new input while saving" || !selectedTab(m).Dirty() {
		t.Fatalf("receipt lost captured draft: writes=%d preview=%q draft=%q", writes.Load(), *m.Preview, *m.EditDraft)
	}
	if disk, err := os.ReadFile(filepath.Join(m.WorkspaceRoot, "a.txt")); err != nil || string(disk) != "captured close input" {
		t.Fatalf("disk=%q err=%v", disk, err)
	}
}

func TestExplicitSaveReadPermissionRemainsNarrow(t *testing.T) {
	for _, tc := range []struct {
		name              string
		tab               FileTab
		explicit, allowed bool
	}{
		{"dirty failed read", FileTab{Path: "a.txt", Preview: "base", Draft: "draft", readBlocked: true}, true, true},
		{"dirty truncated", FileTab{Path: "a.txt", Preview: "base", Draft: "draft", Truncated: true}, true, true},
		{"automatic failed read", FileTab{Path: "a.txt", Preview: "base", Draft: "draft", readBlocked: true}, false, false},
		{"loading", FileTab{Path: "a.txt", Preview: "base", Draft: "draft", readLoading: true}, true, false},
		{"resume", FileTab{Path: "a.txt", Preview: "base", Draft: "draft", readResume: true}, true, false},
		{"remote", FileTab{Path: "a.txt", Preview: "base", Draft: "draft", Remote: true}, true, false},
		{"table", FileTab{Path: "a.xlsx", Preview: "base", Draft: "draft"}, true, false},
		{"clean failed read", FileTab{Path: "a.txt", Preview: "base", Draft: "base", readBlocked: true}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkSavePermission(Model{OnSaveEdit: func(string, string) error { return nil }}, &tc.tab, tc.explicit)
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v err=%v", tc.allowed, err)
			}
		})
	}
}

func TestSaveReceiptSamePathCannotPublishIntoDifferentRoot(t *testing.T) {
	old, done := autosaveModel(t)
	next, _ := autosaveModel(t)
	disposeSave(selectedTab(old))
	oldTab := selectedTab(old)
	*old.EditDraft = "saved old owner"
	if err := requestSaveDraft(old); err != nil {
		t.Fatal(err)
	}
	awaitAutosave(t, done)
	applySaveResult(next, oldTab)
	if oldTab.Preview != "saved old owner" || *next.Preview != "original-a" || *next.EditDraft != "original-a" || selectedTab(next) == oldTab {
		t.Fatal("same-path receipt crossed root/occurrence ownership")
	}
}
