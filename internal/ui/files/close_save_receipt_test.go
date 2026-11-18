package files

import (
	"path/filepath"
	"testing"
	"time"
)

func settleQueue(t *testing.T, tab *FileTab) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		save := tab.save.Snapshot()
		if save.Revision > 0 && !save.Running && !save.Pending {
			return
		}
		if !time.Now().Before(deadline) {
			t.Fatalf("queue never settled: %+v", save)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func closeReport(t *testing.T, draft string) (Model, *FileTab) {
	t.Helper()
	root := filepath.Clean(t.TempDir())
	tab := &FileTab{Path: "report.md", Preview: "saved Project report", Draft: draft}
	tab.Editing = true
	tabs := &Tabs{Workspaces: map[string]*WorkspaceTabs{root: {Open: []*FileTab{tab}, Active: "report.md"}}}
	selected := "report.md"
	return Model{
		Tabs:          tabs,
		WorkspaceRoot: root,
		SelectedFile:  &selected,
		OnSaveEdit:    func(rel, text string) error { return nil },
	}, tab
}

// The close decision must read a receipt the tab has not applied as work still
// settling. Run and pending are both false in the state that failed: the write
// had completed and bumped the queue's revision after the receipt was read, so
// only the revision comparison can tell that the save was not yet reported.
func TestCloseTabSettlingDefersUnappliedReceipt(t *testing.T) {
	cases := []struct {
		name                   string
		running, pending       bool
		queueRevision, applied uint64
		want                   bool
	}{
		{name: "receipt landed after the read", queueRevision: 1, applied: 0, want: true},
		{name: "write in flight", running: true, queueRevision: 1, applied: 1, want: true},
		{name: "queued behind a write", pending: true, queueRevision: 1, applied: 1, want: true},
		{name: "settled and applied", queueRevision: 2, applied: 2, want: false},
		{name: "nothing ever saved", queueRevision: 0, applied: 0, want: false},
	}
	for _, c := range cases {
		if got := closeTabSettling(c.running, c.pending, c.queueRevision, c.applied); got != c.want {
			t.Errorf("%s: settling=%v, want %v", c.name, got, c.want)
		}
	}
}

// The receipt boundary itself: once a confirmed write has been applied, the tab
// holds the queue's revision and the close sees nothing left to save.
func TestCheckDirtyForCloseAppliesConfirmedWrite(t *testing.T) {
	m, tab := closeReport(t, "edited Project report")
	if err := prepareSaveQueue(m, tab, true); err != nil {
		t.Fatal(err)
	}
	tab.save.Flush(tab.Draft)
	settleQueue(t, tab)

	pending, err := CheckDirtyForClose(m)
	if err != nil || pending {
		t.Fatalf("close did not settle on a confirmed write: pending=%v err=%v", pending, err)
	}
	if got, want := tab.saveRevision, tab.save.Snapshot().Revision; got != want {
		t.Fatalf("tab applied revision %d, queue is at %d", got, want)
	}
	if tab.Dirty() {
		t.Fatal("confirmed write left the tab dirty")
	}
}

// The mirror case: an edit that really is newer than the confirmed text must
// still keep the pane open. Deferring on an unapplied receipt cannot hide it,
// because the next poll applies the receipt and the refusal returns.
func TestCheckDirtyForCloseStillRefusesGenuinelyNewerDraft(t *testing.T) {
	m, tab := closeReport(t, "edited Project report")
	if err := prepareSaveQueue(m, tab, true); err != nil {
		t.Fatal(err)
	}
	tab.save.Flush(tab.Draft)
	settleQueue(t, tab)
	if _, err := CheckDirtyForClose(m); err != nil {
		t.Fatal(err)
	}
	tab.Draft = "edited Project report again"

	pending, err := CheckDirtyForClose(m)
	if err == nil {
		t.Fatalf("close accepted a genuinely newer draft (pending=%v)", pending)
	}
	if !tab.Dirty() {
		t.Fatal("the newer edit was discarded rather than kept")
	}
}
