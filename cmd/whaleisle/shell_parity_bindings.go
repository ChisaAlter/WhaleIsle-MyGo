package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/frame"
)

// Shared sidebar and conversation frames consume the same live global registry.
func (s *mainState) bindShellParity(c *ui.Context, m *frame.Model, win *mygo.Window) {
	m.WorkspaceNavigation = s.pluginWorkspaceNavigation()
	m.OnWorkspaceSessionAction = s.pluginWorkspaceSessionAction
	m.GlobalPluginSlot = s.renderPluginGlobalSlot
	m.OnCloseForegroundModal = func() bool { return s.closeForegroundModal(c) }
	if win != nil {
		// Close retains the window's existing tray/unsaved/task protection route.
		m.OnCloseWindow = win.Close
	}
}

func (s *mainState) closeForegroundModal(c *ui.Context) bool {
	if c.CloseTopDialog() {
		return true
	}
	// Other modal/popover owners keep their own close/save/draft policy. Never
	// interpret an unrecognized child dialog as permission to close its parent.
	return !c.MenuOverlayClear()
}

// Deferred after visible page owners have had first refusal. Identical binding
// identity lets a route transition consume a delivery from the previous route.
func (s *mainState) processNonFramePageClose(c *ui.Context) {
	s.mu.Lock()
	win, editor, quitting := s.win, s.shortcutEditor, s.quitting
	s.mu.Unlock()
	if quitting || editor != nil && (editor.Recording != "" || editor.ResetPending) {
		return
	}
	chord, bound := s.pageShortcutChords()["page.close"]
	if !bound || !frame.NativeCommandPressed(c, chord, "page.close"+s.pageShortcutEncoded(), "page-close") {
		return
	}
	if !s.closeForegroundModal(c) && win != nil {
		win.Close()
	}
}
