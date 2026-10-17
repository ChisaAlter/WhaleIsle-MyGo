package sidebar

import (
	"github.com/egoist/mygo/ui"
	"slices"
	"whaleisle/internal/store/session"
)

type sessionDrag struct {
	Generation    uint64
	Workspace, ID string
	Pinned        bool
}
type sessionDragContext struct {
	Workspace string
	Order     []string
	Rows      []session.WorkspaceSessionRow
}
type sessionRowKey struct {
	Generation    uint64
	Workspace, ID string
}

// Resolve the visible insertion marker against the full Host membership, never
// replacing it with the filtered rows (which would discard hidden members).
func sessionDragAnchor(order []string, rows []session.WorkspaceSessionRow, source, target string, after bool) (string, bool) {
	var src, dst *session.WorkspaceSessionRow
	for i := range rows {
		if rows[i].Item.SessionID == source {
			src = &rows[i]
		}
		if rows[i].Item.SessionID == target {
			dst = &rows[i]
		}
	}
	if src == nil || dst == nil || source == target || src.Depth != 0 || dst.Depth != 0 || src.Item.Blank || src.Item.Pinned != dst.Item.Pinned || !slices.Contains(order, source) || !slices.Contains(order, target) {
		return "", false
	}
	if dst.Item.Blank {
		after = true
	}
	var section []string
	for _, r := range rows {
		if r.Depth == 0 && r.Item.Pinned == src.Item.Pinned {
			section = append(section, r.Item.SessionID)
		}
	}
	old := slices.Index(section, source)
	visible := slices.Delete(slices.Clone(section), old, old+1)
	at := slices.Index(visible, target)
	if after {
		at++
	}
	if old == at {
		return "", false
	}
	if !after {
		return target, true
	}
	next := slices.Clone(order)
	i := slices.Index(next, source)
	next = slices.Delete(next, i, i+1)
	i = slices.Index(next, target) + 1
	if i < len(next) {
		return next[i], true
	}
	return "", true
}

func sessionDragSource(e ui.Element, m Model, it session.Item) {
	ctx := m.sessionDrag
	if ctx == nil || it.Blank || m.Snapshot.WorkspaceLayout.Reordering {
		return
	}
	for _, r := range ctx.Rows {
		if r.Item.SessionID == it.SessionID && r.Depth == 0 && slices.Contains(ctx.Order, it.SessionID) {
			e.Drag(sessionDrag{m.Snapshot.WorkspaceLayout.HostGeneration, ctx.Workspace, it.SessionID, it.Pinned})
			return
		}
	}
}
func sessionDropTarget(e ui.Element, m Model, it session.Item, after bool, highlight ui.Color) {
	ctx := m.sessionDrag
	if ctx == nil || m.Snapshot.WorkspaceLayout.Reordering || m.OnMoveWorkspaceSessionBefore == nil {
		return
	}
	accept := func(v sessionDrag) (string, bool) {
		if v.Generation != m.Snapshot.WorkspaceLayout.HostGeneration || v.Workspace != ctx.Workspace || v.Pinned != it.Pinned {
			return "", false
		}
		return sessionDragAnchor(ctx.Order, ctx.Rows, v.ID, it.SessionID, after)
	}
	if v, over := ui.DragOver[sessionDrag](e); over {
		if _, ok := accept(v); ok {
			e.Border(1, highlight)
		}
	}
	if v, dropped := ui.Drop[sessionDrag](e); dropped {
		if before, ok := accept(v); ok {
			m.OnMoveWorkspaceSessionBefore(ctx.Workspace, v.ID, before)
		}
	}
}

// Pinning remains a leading partition; manual position within it is owned by
// the same Host session order as the unpinned partition.
func sessionManualPins(order, pins []string) []string {
	out := make([]string, 0, len(pins))
	for _, id := range order {
		if slices.Contains(pins, id) && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	for _, id := range pins {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}
