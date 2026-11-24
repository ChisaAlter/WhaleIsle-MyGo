package terminal

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/hostterminal"
)

// renameStoreFixture builds a bound store with one adopted terminal in an
// active rename. The zero-value client fails its RPCs, which leaves the entry
// in a non-closed phase that still permits renaming.
func renameStoreFixture(t *testing.T) (*hostterminal.Store, uint64) {
	t.Helper()
	store := &hostterminal.Store{}
	store.Bind(&dsh.Client{}, &dsh.Mux{}, true, nil)
	t.Cleanup(func() { store.Bind(nil, nil, false, nil) })
	store.Show("session", true)
	var owner uint64
	for i := 0; i < 200; i++ {
		if o := store.Snapshot("session").Value.Owner; o != 0 {
			owner = o
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if owner == 0 {
		t.Fatal("store published no owner generation")
	}
	if !store.AdoptRecovery(owner, "host-a", "session", &hostterminal.RecoveryTarget{
		Authority: "host-a", SessionID: "session", TerminalID: "term", Title: "old",
	}) {
		t.Fatal("recovery adoption rejected")
	}
	store.BeginRename(owner, "session", "term")
	if !store.RenameSnapshot(owner, "session", "term").Editing {
		t.Fatal("rename did not begin")
	}
	return store, owner
}

// Typing in the DockKit rename input must reach the store. The bound draft
// used to be compared before applyInputs flushed the edit, so every
// keystroke was dropped.
func TestHostTitleEditorTypedEditsReachStore(t *testing.T) {
	store, owner := renameStoreFixture(t)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			HostTitleEditor(c, store, "session")
		})
	}, 400, 120)
	tt.Type("x")
	if d := store.RenameSnapshot(owner, "session", "term").Draft; d != "x" {
		t.Fatalf("typed edit lost: draft=%q", d)
	}
}

// The same defect exists in the rename row painted by hostRenameView.
func TestHostRenameViewTypedEditsReachStore(t *testing.T) {
	store, owner := renameStoreFixture(t)
	muted := ui.RGB(120, 120, 120)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			hostRenameView(c, store, store.Snapshot("session").Value, "session", muted)
		})
	}, 400, 200)
	tt.Type("x")
	if d := store.RenameSnapshot(owner, "session", "term").Draft; d != "oldx" {
		t.Fatalf("typed edit lost: draft=%q", d)
	}
}
