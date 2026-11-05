package sidebar

import (
	"sync"
	"whaleisle/internal/store/session"
)

const workspaceWatermarkBase = 5
const workspaceWatermarkAutoMax = 50

var workspaceOrderedRowsMemo struct {
	sync.Mutex
	key       session.ProjectionKey
	rowsByKey map[string][]session.WorkspaceSessionRow
}

type workspaceWatermarkState struct {
	HostGeneration uint64
	Marks          map[string]int
}

type workspaceWatermarkRows struct {
	Rows                               []session.WorkspaceSessionRow
	Hidden, Watermark, Baseline, Total int
}

func workspaceReveal(m Model) string {
	if m.searchState != nil {
		return m.searchState.Reveal
	}
	return ""
}

func workspaceWatermarkAttention(m Model) map[string]bool {
	attention := map[string]bool{}
	for _, item := range m.Snapshot.Items {
		if sessionStatus(m, item) != "" || item.SessionID == displayedSessionID(m) {
			attention[item.SessionID] = true
		}
	}
	for _, item := range m.Snapshot.Items {
		if item.Running && item.Origin == "subagent" && item.ParentID != "" {
			attention[item.ParentID] = true
		}
	}
	return attention
}

// visibleWorkspaceSessionRows is shared by rendering, drag acceptance and batch
// selection: a hidden overflow row cannot become a Select visible target.
func visibleWorkspaceSessionRows(m Model, group session.WorkspaceGroup) workspaceWatermarkRows {
	var rows []session.WorkspaceSessionRow
	projKey := m.Snapshot.RosterProjectionKey()
	if projKey.Valid() {
		workspaceOrderedRowsMemo.Lock()
		if workspaceOrderedRowsMemo.key != projKey {
			workspaceOrderedRowsMemo.key = projKey
			workspaceOrderedRowsMemo.rowsByKey = make(map[string][]session.WorkspaceSessionRow, len(m.Snapshot.Workspaces)+2)
		} else {
			rows = workspaceOrderedRowsMemo.rowsByKey[group.Key]
		}
		workspaceOrderedRowsMemo.Unlock()
	}
	if rows == nil {
		pins := make([]string, 0, len(m.Snapshot.PinnedSessionIDs))
		for _, id := range m.Snapshot.PinnedSessionIDs {
			if !isArchived(m.Snapshot, id) {
				pins = append(pins, id)
			}
		}
		if m.Snapshot.WorkspaceLayout.OrderBy != "updated" {
			pins = sessionManualPins(group.Workspace.SessionIDs, pins)
		}
		rows = session.OrderedWorkspaceSessionRows(group.Items, group.Workspace.SessionIDs, pins, m.Snapshot.WorkspaceLayout.OrderBy, group.Kind == "flat")
		if projKey.Valid() {
			workspaceOrderedRowsMemo.Lock()
			if workspaceOrderedRowsMemo.key == projKey && workspaceOrderedRowsMemo.rowsByKey != nil {
				workspaceOrderedRowsMemo.rowsByKey[group.Key] = rows
			}
			workspaceOrderedRowsMemo.Unlock()
		}
	}
	// The source flat-list body is unbounded; watermarks belong to groups.
	if group.Kind == "flat" {
		return workspaceWatermarkRows{Rows: rows, Total: len(rows), Watermark: len(rows), Baseline: len(rows)}
	}
	var attention map[string]bool
	reveal := workspaceReveal(m)
	baseline, nonblank := workspaceWatermarkBase, 0
	for _, row := range rows {
		if row.Item.Blank {
			continue
		}
		nonblank++
		if nonblank > baseline {
			if attention == nil {
				// Display selection and child activity can change without a roster
				// revision. Cache row ordering, not these presentation facts.
				attention = workspaceWatermarkAttention(m)
			}
			if nonblank <= workspaceWatermarkAutoMax && attention[row.Item.SessionID] || row.Item.SessionID == reveal {
				baseline = nonblank
			}
		}
	}
	mark := baseline
	if state := m.workspaceWatermarks; state != nil && state.HostGeneration == m.Snapshot.WorkspaceLayout.HostGeneration {
		mark = max(mark, state.Marks[group.Key])
	}
	visible := make([]session.WorkspaceSessionRow, 0, min(len(rows), mark+1))
	count := 0
	for _, row := range rows {
		if !row.Item.Blank {
			count++
		}
		if row.Item.Blank || count <= mark {
			visible = append(visible, row)
		}
	}
	return workspaceWatermarkRows{Rows: visible, Hidden: len(rows) - len(visible), Watermark: mark, Baseline: baseline, Total: nonblank}
}

func retainWorkspaceWatermark(m Model, key string, mark int) {
	state := m.workspaceWatermarks
	if state == nil {
		return
	}
	if state.HostGeneration != m.Snapshot.WorkspaceLayout.HostGeneration || state.Marks == nil {
		state.HostGeneration = m.Snapshot.WorkspaceLayout.HostGeneration
		state.Marks = map[string]int{}
	}
	if mark > state.Marks[key] {
		state.Marks[key] = mark
	}
}

func resetWorkspaceWatermark(m Model, key string, groups []session.WorkspaceGroup) {
	if m.workspaceWatermarks == nil {
		return
	}
	within := map[string]bool{key: true}
	for _, group := range groups {
		if within[group.ParentID] {
			within[group.Key] = true
		}
	}
	for id := range within {
		delete(m.workspaceWatermarks.Marks, id)
	}
}
