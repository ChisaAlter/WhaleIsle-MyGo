package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
)

// Business commands reuse existing stores and callbacks, including archive confirmation.
func addBusinessCommands(c *ui.Context, m Model, actions map[string]func()) {
	actions["pane.split"] = func() { SplitWorkspacePane(m, false) }
	actions["pane.splitDown"] = func() { SplitWorkspacePane(m, true) }
	actions["pane.fullscreen.toggle"] = func() { ToggleWorkspacePaneFullscreen(m) }
	actions["workspace.add"] = m.OnBrowseWorkspace
	actions["workspace.openLocal"] = m.OnRevealWorkspace
	actions["browser.open"] = m.OnOpenBrowserPreview
	actions["terminal.host"] = m.OnOpenHostTerminal
	actions["market.open"] = m.OnOpenMarket
	actions["schedule.open"] = m.OnOpenSchedule
	actions["assistant.open"] = m.OnOpenAssistant
	actions["launcher.open"] = m.OnOpenLauncher
	actions["workspace.files"] = m.OnOpenWorkspaceFiles
	actions["session.search"] = func() {
		if m.Sizes != nil && m.Sizes.Sidebar <= 0 {
			m.Sizes.ToggleSidebar()
		}
		if m.SessionSearchFocus != nil {
			*m.SessionSearchFocus = true
		}
		c.Invalidate()
	}
	if m.Store == nil {
		return
	}
	actions["sessions.refresh"] = func() { m.Store.RefreshList(context.Background()) }
	actions["session.rename"] = func() {
		snap := m.Store.Snapshot()
		for _, item := range snap.Items {
			if item.SessionID == snap.SelectedID {
				m.Store.BeginSessionRename(item.SessionID, item.Title)
				return
			}
		}
	}
	actions["session.fork"] = func() {
		id := m.Store.Snapshot().SelectedID
		if id != "" {
			m.Store.ForkSession(context.Background(), id)
		}
	}
	actions["session.archive"] = func() {
		id := m.Store.Snapshot().SelectedID
		if id != "" {
			m.Store.ArchiveSession(context.Background(), id)
		}
	}
	selectNeighbor := func(delta int) {
		snap := m.Store.Snapshot()
		for i, item := range snap.Items {
			if item.SessionID == snap.SelectedID {
				target := i + delta
				if target < 0 || target >= len(snap.Items) {
					return
				}
				id := snap.Items[target].SessionID
				if m.OnSelectSession != nil {
					m.OnSelectSession(id)
				} else {
					m.Store.Select(id)
				}
				return
			}
		}
	}
	actions["session.previous"] = func() { selectNeighbor(-1) }
	actions["session.next"] = func() { selectNeighbor(1) }
}

// Retain the existing package-private helper used by existing callers.
func nativePageChord(chord shortcuts.Chord) (ui.Modifiers, ui.Key) { return NativeCommandChord(chord) }
