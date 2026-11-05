package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/store/session"
)

// Business commands reuse existing stores and callbacks, including archive confirmation.
func addBusinessCommands(c *ui.Context, m Model, actions map[string]func()) {
	actions["page.close"] = func() { closeFocusedWorkspacePage(c, m) }
	actions["page.refresh"] = func() { refreshFocusedWorkspacePage(c, m) }
	actions["sidebar.right.toggle"] = m.OnToggleTools
	actions["pane.split"] = func() { splitFocusedWorkspacePage(c, m) }
	actions["pane.splitDown"] = func() { SplitWorkspacePane(m, true) }
	actions["pane.fullscreen.toggle"] = func() { maximizeFocusedWorkspacePage(c, m) }
	actions["workspace.add"] = m.OnBrowseWorkspace
	actions["workspace.openLocal"] = m.OnOpenWorkspaceLocal
	actions["browser.open"] = m.OnOpenBrowserPreview
	actions["terminal.host"] = m.OnOpenHostTerminal
	actions["terminal.new"] = m.OnNewHostTerminal
	actions["market.open"] = m.OnOpenMarket
	actions["schedule.open"] = m.OnOpenSchedule
	actions["assistant.open"] = m.OnOpenAssistant
	actions["launcher.open"] = m.OnOpenLauncher
	actions["workspace.files"] = m.OnOpenWorkspaceFiles
	actions["session.search"] = func() {
		if m.Sizes != nil && m.Sizes.sidebarPreference() <= 0 {
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
				beginConversationRename(m, item.SessionID, item.Title)
				return
			}
		}
	}
	actions["session.fork"] = func() {
		id := m.Store.SelectedSessionID()
		if item, ok := conversationActionItem(m, id); ok && !routeProjectConversationAction(m, item, "fork") {
			m.Store.ForkSession(context.Background(), id)
		}
	}
	actions["session.archive"] = func() {
		id := m.Store.SelectedSessionID()
		archiveConversation(m, id, false)
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
				if routeRegisteredSessionOpen(m, snap.Items[target]) {
					return
				}
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

func conversationActionItem(m Model, id string) (session.Item, bool) {
	if m.Store == nil || id == "" {
		return session.Item{}, false
	}
	for _, item := range m.Store.Snapshot().Items {
		if item.SessionID == id {
			return item, !item.Blank
		}
	}
	return session.Item{}, false
}

// Owner-specific presentation is not a generic Session mutation permission.
// A Project whose catalog is still loading must not fall through to Session.
func routeProjectConversationAction(m Model, item session.Item, action string) bool {
	if p := item.Presentation; p != nil && p.Owner != "" && p.Owner != "project" && p.Owner != "whaleisle" {
		for _, entry := range m.WorkspaceNavigation {
			if entry.SessionID != item.SessionID || entry.Owner != p.Owner || entry.WorkingDirectory != p.WorkingDirectory {
				continue
			}
			allowed := action == "open" || action == "rename" && entry.CanRename || action == "archive" && entry.CanArchive || action == "restore" && entry.CanRestore
			if allowed && m.OnWorkspaceSessionAction != nil {
				m.OnWorkspaceSessionAction(item.SessionID, entry.Owner, action)
			}
			return true
		}
		// Withdrawing a source revokes its actions, not its ownership.
		return true
	}
	for _, project := range m.Projects.Listing.Projects {
		if project.CoordinatorSessionID != item.SessionID {
			continue
		}
		switch action {
		case "rename":
			if m.Projects.OnBeginRename != nil {
				m.Projects.OnBeginRename(project)
			}
		case "archive":
			if m.Projects.OnArchiveProject != nil {
				m.Projects.OnArchiveProject(project.ID)
			}
		case "restore":
			if m.Projects.OnRestoreProject != nil {
				m.Projects.OnRestoreProject(project.ID)
			}
		case "fork":
			m.Store.SetLastAction(paneLabel("项目对话不能分叉；请新建项目。", "Project conversations cannot be forked. Create a new Project."))
		}
		return true
	}
	if item.Presentation != nil && item.Presentation.Owner == "project" {
		m.Store.SetLastAction(paneLabel("正在读取项目，请从项目菜单重试。", "Project is loading. Retry from its menu."))
		return true
	}
	return false
}

func routeRegisteredSessionOpen(m Model, item session.Item) bool {
	p := item.Presentation
	if p == nil || p.Owner == "" || p.Owner == "project" || p.Owner == "whaleisle" {
		return false
	}
	return routeProjectConversationAction(m, item, "open")
}

func beginConversationRename(m Model, id, title string) {
	item, ok := conversationActionItem(m, id)
	if !ok || routeProjectConversationAction(m, item, "rename") {
		return
	}
	m.Store.BeginSessionRename(id, title)
}

func archiveConversation(m Model, id string, restore bool) {
	if m.Store == nil || id == "" {
		return
	}
	item, ok := conversationActionItem(m, id)
	if !ok {
		if !restore {
			return
		}
		// Archived catalog rows are not members of the active Session list.
		// Preserve their existing restore contract without requiring selection.
		item = session.Item{SessionID: id}
	}
	action := "archive"
	if restore {
		action = "restore"
	}
	if routeProjectConversationAction(m, item, action) {
		return
	}
	if restore {
		m.Store.UnarchiveSession(context.Background(), id)
	} else {
		m.Store.ArchiveSession(context.Background(), id)
	}
}
