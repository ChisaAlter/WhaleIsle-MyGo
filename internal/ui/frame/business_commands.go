package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/store/session"
)

func selectNeighborSession(m *Model, delta int) {
	if m.Store == nil {
		return
	}
	snap := m.Store.Published().Value
	for i, item := range snap.Items {
		if item.SessionID == snap.SelectedID {
			target := i + delta
			if target < 0 || target >= len(snap.Items) {
				return
			}
			id := snap.Items[target].SessionID
			if routeRegisteredSessionOpenPtr(m, snap.Items[target]) {
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

// Retain the existing package-private helper used by existing callers.
func nativePageChord(chord shortcuts.Chord) (ui.Modifiers, ui.Key) { return NativeCommandChord(chord) }

func conversationActionItem(m Model, id string) (session.Item, bool) {
	return conversationActionItemPtr(&m, id)
}

func conversationActionItemPtr(m *Model, id string) (session.Item, bool) {
	if m.Store == nil || id == "" {
		return session.Item{}, false
	}
	for _, item := range m.Store.Published().Value.Items {
		if item.SessionID == id {
			return item, !item.Blank
		}
	}
	return session.Item{}, false
}

// Owner-specific presentation is not a generic Session mutation permission.
// A Project whose catalog is still loading must not fall through to Session.
func routeProjectConversationAction(m Model, item session.Item, action string) bool {
	return routeProjectConversationActionPtr(&m, item, action)
}

func routeProjectConversationActionPtr(m *Model, item session.Item, action string) bool {
	if p := item.Presentation; p != nil && p.Owner != "" && p.Owner != "project" && p.Owner != "whaleisle" {
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
	return routeRegisteredSessionOpenPtr(&m, item)
}

func routeRegisteredSessionOpenPtr(m *Model, item session.Item) bool {
	p := item.Presentation
	if p == nil || p.Owner == "" || p.Owner == "project" || p.Owner == "whaleisle" {
		return false
	}
	return routeProjectConversationActionPtr(m, item, "open")
}

func beginConversationRename(m Model, id, title string) {
	beginConversationRenamePtr(&m, id, title)
}

func beginConversationRenamePtr(m *Model, id, title string) {
	item, ok := conversationActionItemPtr(m, id)
	if !ok || routeProjectConversationActionPtr(m, item, "rename") {
		return
	}
	m.Store.BeginSessionRename(id, title)
}

func archiveConversation(m Model, id string, restore bool) {
	archiveConversationPtr(&m, id, restore)
}

func archiveConversationPtr(m *Model, id string, restore bool) {
	if m.Store == nil || id == "" {
		return
	}
	item, ok := conversationActionItemPtr(m, id)
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
	if routeProjectConversationActionPtr(m, item, action) {
		return
	}
	if restore {
		m.Store.UnarchiveSession(context.Background(), id)
	} else {
		m.Store.ArchiveSession(context.Background(), id)
	}
}
