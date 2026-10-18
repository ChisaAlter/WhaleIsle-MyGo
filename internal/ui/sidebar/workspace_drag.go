package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

type workspaceRowKey struct {
	HostGeneration uint64
	ID             string
}
type workspaceDrag struct {
	HostGeneration uint64
	ID             string
}

// The existing MyGo value-drag API owns threshold, Escape cancel, pointer
// capture and autoscroll. No native file transfer or framework change is used.
func workspaceDropTarget(target ui.Element, m Model, g session.WorkspaceGroup, groups []session.WorkspaceGroup, after bool, highlight ui.Color) {
	if g.Kind != "workspace" || m.Snapshot.WorkspaceLayout.Reordering {
		return
	}
	accept := func(value workspaceDrag) (string, bool) {
		if value.HostGeneration != m.Snapshot.WorkspaceLayout.HostGeneration {
			return "", false
		}
		return session.WorkspaceDropAnchor(groups, value.ID, g.Key, after)
	}
	if value, over := ui.DragOver[workspaceDrag](target); over {
		if _, ok := accept(value); ok {
			target.Border(1, highlight)
		}
	}
	if value, dropped := ui.Drop[workspaceDrag](target); dropped && m.OnDropWorkspaceBefore != nil {
		if before, ok := accept(value); ok {
			m.OnDropWorkspaceBefore(value.HostGeneration, value.ID, before)
		}
	}
}
