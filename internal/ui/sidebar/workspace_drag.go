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
	Scope          sidebarDragScope
}

// Value dragging keeps MyGo's threshold, Escape cancellation and autoscroll.
// DropAt supplies the release position independently of later pointer moves.

// workspaceDropBlock is one visible workspace header and its own session run.
// Blocks follow tree order; a parent's insertion extent includes every following
// deeper block. Only the encompassing list receives workspace values, so a
// child's incompatible target never swallows a valid parent-sibling reorder.
type workspaceDropBlock struct {
	Group   session.WorkspaceGroup
	Element ui.Element
}

// insertionMarker paints inside the existing edge: no height, margin or hit
// target is added, and action/menu controls retain their input priority.
func insertionMarker(e ui.Element, after bool, color ui.Color) {
	e.DrawOver(func(p *ui.Painter, r ui.Rect) {
		y := r.Y
		if after {
			y += r.H - 2
		}
		p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: 2}, color, 1)
	})
}

// workspaceDropPlacement resolves a release against visible full sections.
// The rectangles come from the committed layout, including expanded descendants.
func workspaceDropPlacement(groups []session.WorkspaceGroup, bounds []ui.Rect, source string, x, y float32) (target, edge int, after bool, before string, ok bool) {
	if len(groups) != len(bounds) {
		return
	}
	src := -1
	for i, g := range groups {
		if g.Kind == "workspace" && g.Key == source {
			src = i
			break
		}
	}
	if src < 0 {
		return
	}
	for i, g := range groups {
		if g.Kind != "workspace" || g.ParentID != groups[src].ParentID {
			continue
		}
		last := i
		for last+1 < len(groups) && groups[last+1].Depth > g.Depth {
			last++
		}
		r := bounds[i]
		bottom := bounds[last].Y + bounds[last].H
		r.H = bottom - r.Y
		if r.W <= 0 || r.H <= 0 || !r.Contains(x, y) {
			continue
		}
		after = y >= r.Y+r.H/2
		before, ok = session.WorkspaceDropAnchor(groups, source, g.Key, after)
		target, edge = i, i
		if after {
			edge = last
		}
		return
	}
	return
}

// workspaceDropGroupsTarget is called once after all visible blocks are built.
// Session values still choose their row target; workspace values choose this
// encompassing list and resolve against the source's sibling sections.
func workspaceDropGroupsTarget(list ui.Element, m Model, blocks []workspaceDropBlock, highlight ui.Color) {
	if m.Snapshot.WorkspaceLayout.Reordering || m.OnDropWorkspaceBefore == nil {
		return
	}
	groups := make([]session.WorkspaceGroup, len(blocks))
	bounds := make([]ui.Rect, len(blocks))
	for i, b := range blocks {
		groups[i], bounds[i] = b.Group, b.Element.Bounds()
	}
	place := func(v workspaceDrag, x, y float32) (int, bool, string, bool) {
		if v.HostGeneration != m.Snapshot.WorkspaceLayout.HostGeneration || v.Scope != currentSidebarDragScope(m) {
			return 0, false, "", false
		}
		_, edge, after, before, ok := workspaceDropPlacement(groups, bounds, v.ID, x, y)
		return edge, after, before, ok
	}
	if v, over := ui.DragOver[workspaceDrag](list); over {
		x, y, _ := list.PointerPosition()
		r := list.Bounds()
		target, edge, after, _, ok := workspaceDropPlacement(groups, bounds, v.ID, r.X+x, r.Y+y)
		if ok && v.HostGeneration == m.Snapshot.WorkspaceLayout.HostGeneration && v.Scope == currentSidebarDragScope(m) {
			if m.sidebarDrag != nil {
				m.sidebarDrag.Workspace = &workspaceDropMarker{Source: v, Scope: currentSidebarDragScope(m), Target: groups[target].Key, After: after}
			}
			insertionMarker(blocks[edge].Element, after, highlight)
		} else if m.sidebarDrag != nil {
			m.sidebarDrag.Workspace = nil
		}
	} else if m.sidebarDrag != nil && m.sidebarDrag.Workspace != nil {
		marker := m.sidebarDrag.Workspace
		if marker.Scope == currentSidebarDragScope(m) {
			for i, group := range groups {
				if group.Key != marker.Target {
					continue
				}
				edge := i
				if marker.After {
					for edge+1 < len(groups) && groups[edge+1].Depth > group.Depth {
						edge++
					}
				}
				insertionMarker(blocks[edge].Element, marker.After, highlight)
			}
		}
	}
	if v, x, y, dropped := ui.DropAt[workspaceDrag](list); dropped {
		if m.sidebarDrag != nil {
			m.sidebarDrag.Workspace = nil
		}
		if _, _, before, ok := place(v, x, y); ok {
			m.OnDropWorkspaceBefore(v.HostGeneration, v.ID, before)
		}
	}
}
