package frame

import (
	"testing"
	"whaleisle/internal/store/hostterminal"
)

func TestHostTerminalCloseFailureKeepsPaneAndGenericDisposalNeverClosesProcess(t *testing.T) {
	w := &WorkspacePanes{}
	pane := w.leaf("host-terminal")
	w.Root = pane
	w.Active = pane.ID
	calls, disposed := 0, 0
	pane.terminalClose = func() *hostterminal.OccurrenceClose {
		calls++
		return (&hostterminal.Store{}).BeginOccurrenceClose(pane.ID, nil, 0, "session")
	}
	pane.terminalDispose = func() { disposed++ }
	if w.close(pane.ID) {
		t.Fatal("failed process close removed pane")
	}
	if calls != 1 || disposed != 0 || w.Root.find(pane.ID) != pane || w.message == "" {
		t.Fatal("failed process close lost visible retry state")
	}
	pane.disposeContent()
	if calls != 1 || disposed != 1 {
		t.Fatal("generic disposal must release renderer without process close")
	}
	pane.disposeContent()
	if disposed != 1 {
		t.Fatal("renderer disposed twice")
	}
}
