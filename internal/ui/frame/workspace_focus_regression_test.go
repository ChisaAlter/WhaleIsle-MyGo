package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/store/session"
	termui "whaleisle/internal/ui/terminal"
)

func TestHostTerminalFocusAlignmentAndRightPaneHideReopenDoesNotStealComposer(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	sizes := DefaultLayout()
	sizes.Workspace = w
	route := "files"
	rightOpen := true

	var composerBtn ui.Handle
	var observedFocused *WorkspacePane
	var observedPaneFocused bool
	store := session.New()
	store.Select("sess-1")
	htStore := &hostterminal.Store{}

	m := Model{
		Sizes:             &sizes,
		Store:             store,
		HostTerminalStore: htStore,
		HostTerminalUI:    &termui.HostState{},
		RightPaneTab:      &route,
		RightPaneOpen:     true,
	}

	tt := ui.NewTester(func(c *ui.Context) {
		m.RightPaneOpen = rightOpen
		prepareWorkspace(m)
		observedFocused = focusedWorkspacePage(c, m)
		ui.Column(c).Fill().Children(func() {
			composer := ui.Button(c, "Composer").Bind(&composerBtn).Label("Composer")
			if composer.Clicked() {
				composer.Focus()
			}
			if rightOpen {
				workspaceNode(c, m, w.Root)
			}
		})
		if active := w.Root.find(w.Active); active != nil {
			observedPaneFocused = active.paneBoxFocused
		}
	}, 900, 500)

	// Start with focus in Composer; mounting files should not steal focus.
	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.Focused("Composer") {
		t.Fatal("initial render did not keep focus in Composer")
	}

	// Opening a new Host terminal sets focusPending and aligns focusedResource / paneBoxFocused.
	termID := w.NewHostTerminal()
	tt.Frame()
	tt.Frame()
	if w.focusedResource != termID {
		t.Fatalf("focusedResource = %q, want %q", w.focusedResource, termID)
	}
	if observedFocused == nil || observedFocused.ID != termID {
		t.Fatalf("focusedWorkspacePage = %v, want %s", observedFocused, termID)
	}
	if !observedPaneFocused {
		t.Fatal("host-terminal pane did not expose paneBoxFocused for inner terminal focus handoff")
	}
	actions := hostTerminalActions(Model{Store: store, HostTerminalStore: htStore, workspacePane: w.Root.find(termID)})
	if !actions.FocusTerminal {
		t.Fatal("hostTerminalActions did not propagate paneBoxFocused to FocusTerminal")
	}

	// Hiding the right pane while focused on host-terminal clears focusedResource so reopening never steals Composer focus.
	rightOpen = false
	tt.Frame()
	if w.focusedResource != "" {
		t.Fatalf("hiding right pane kept stale focusedResource = %q", w.focusedResource)
	}
	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	rightOpen = true
	tt.Frame()
	if !tt.Focused("Composer") {
		t.Fatal("reopening right pane stole focus away from Composer")
	}
}
