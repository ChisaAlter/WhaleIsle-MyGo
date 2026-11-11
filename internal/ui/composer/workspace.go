package composer

import (
	"strings"
	"whaleisle/internal/store/session"
)

// RequiresWorkspace follows ConversationContent's inert posture. A ready list
// is authoritative: a deleted workspace must not be resurrected from its cwd.
// While that list opens, the selected Session's cwd bridges its chip label.
func RequiresWorkspace(snap session.ViewState, hero bool) bool {
	if snap.SelectedID == "" {
		return true
	}
	if !hero || strings.HasPrefix(snap.SelectedID, "session-whale-") {
		return false
	}
	for _, item := range snap.Items {
		if item.SessionID == snap.SelectedID && item.Presentation != nil && item.Presentation.Owner == "dsh-whale:assistant" {
			return false
		}
	}
	for _, workspace := range snap.Workspaces {
		for _, id := range workspace.SessionIDs {
			if id == snap.SelectedID {
				return false
			}
		}
	}
	if snap.WorkspaceReady && snap.ScratchCwd != "" && snap.SelectedCwd == snap.ScratchCwd {
		return false
	}
	return snap.WorkspaceReady || snap.SelectedCwd == ""
}
