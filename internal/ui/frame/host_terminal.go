package frame

import (
	"errors"
	"github.com/egoist/mygo/ui"
	"strconv"
	termui "whaleisle/internal/ui/terminal"
)

func workspaceTerminalResourceMenu(menu *ui.Menu, m Model, pane *WorkspacePane) {
	if pane == nil || m.HostTerminalUI == nil {
		return
	}
	store, sid, _, _ := workspaceTerminalTitle(m, pane)
	if store == nil {
		return
	}
	if termui.HostResourceMenu(menu, m.HostTerminalUI.Occurrence(pane.ID), store, sid) {
		m.Sizes.Workspace.pending = func() { m.Sizes.Workspace.activateDockTab(pane.ID) }
	}
}

func hostTerminalActions(m Model) termui.HostActions {
	var fresh *bool
	if m.workspacePane != nil {
		fresh = &m.workspacePane.terminalFresh
	}
	session := m.Store.SelectedSessionID()
	view := m.HostTerminalStore.Snapshot(session)
	valid := func() bool {
		return m.Store.SelectedSessionID() == session && m.HostTerminalStore.CanCopySelection(view.Owner, session, view.ActiveID, view.Attachment)
	}
	return termui.HostActions{
		StartFresh:   fresh,
		CanAddToChat: m.CanAddSelectionToChat,
		AddToChat:    m.OnAddSelectionToChat,
		OpenTarget: func(selection, cwd string) error {
			openURL := func(url string) error {
				if m.OnOpenCitationURL == nil {
					return errors.New("Link opener is unavailable")
				}
				m.OnOpenCitationURL(url)
				return nil
			}
			return termui.OpenSelectionTarget(selection, cwd, termui.SelectionTargetActions{
				Valid: valid, OpenBrowser: openURL, OpenExternal: openURL,
				OpenPath: func(path string, line int) error {
					if m.OnOpenDeliverable == nil {
						return errors.New("Workspace path opener is unavailable")
					}
					if line > 0 {
						path += ":" + strconv.Itoa(line)
					}
					m.OnOpenDeliverable(path)
					return nil
				},
			})
		},
	}
}
