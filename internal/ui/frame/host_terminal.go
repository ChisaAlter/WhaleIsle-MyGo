package frame

import (
	"errors"
	"strconv"
	termui "whaleisle/internal/ui/terminal"
)

func hostTerminalActions(m Model) termui.HostActions {
	session := m.Store.Snapshot().SelectedID
	view := m.HostTerminalStore.Snapshot(session)
	valid := func() bool {
		return m.Store.Snapshot().SelectedID == session && m.HostTerminalStore.CanCopySelection(view.Owner, session, view.ActiveID, view.Attachment)
	}
	return termui.HostActions{
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
