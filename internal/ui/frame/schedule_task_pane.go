package frame

import (
	"context"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	schedstore "whaleisle/internal/store/schedule"
	scheduleui "whaleisle/internal/ui/schedule"
)

// The layout owns bindings; only identity is persisted, not task instructions
// or history. Each viewed session keeps its own target on the same page seat.
type ScheduleTaskTarget struct {
	SessionID string `json:"sessionId"`
	ID        string `json:"id"`
}
type SchedulePaneState struct {
	Targets  map[string]ScheduleTaskTarget `json:"targets"`
	panels   map[string]*scheduleui.Panel
	rendered bool
}

func (s *SchedulePaneState) deactivate() {
	if s == nil {
		return
	}
	for _, panel := range s.panels {
		panel.Deactivate()
	}
	s.panels = nil
}

func scheduleScope(origin, viewer string) string { return origin + "\x00" + viewer }
func (w *WorkspacePanes) OpenScheduleTask(viewer string, row schedstore.Entry) {
	w.syncRoute("schedule-task")
	n := w.Root.find(w.Active)
	if n.Schedule == nil {
		n.Schedule = &SchedulePaneState{}
	}
	if n.Schedule.Targets == nil {
		n.Schedule.Targets = map[string]ScheduleTaskTarget{}
	}
	n.Schedule.Targets[scheduleScope(row.Origin, viewer)] = ScheduleTaskTarget{SessionID: row.SessionID, ID: row.ID}
}
func schedulePaneTarget(m Model) (*SchedulePaneState, string, ScheduleTaskTarget) {
	if m.workspacePane == nil || m.workspacePane.Schedule == nil || m.Store == nil || m.ScheduleStore == nil {
		return nil, "", ScheduleTaskTarget{}
	}
	state := m.workspacePane.Schedule
	scope := scheduleScope(m.ScheduleStore.Snapshot().Origin, m.Store.SelectedSessionID())
	return state, scope, state.Targets[scope]
}
func schedulePaneTitle(m Model) string {
	_, _, target := schedulePaneTarget(m)
	if m.ScheduleStore != nil {
		for _, row := range m.ScheduleStore.Snapshot().Rows {
			if row.ID == target.ID && row.SessionID == target.SessionID {
				return row.Title
			}
		}
	}
	return paneLabel("任务详情", "Task details")
}
func scheduleTaskPane(c *ui.Context, m Model) {
	state, scope, target := schedulePaneTarget(m)
	if state == nil {
		ui.Text(c, paneLabel("选择会话以查看任务", "Select a session to view its task"))
		return
	}
	closeTask := func() {
		delete(state.Targets, scope)
		for key := range state.panels {
			if strings.HasPrefix(key, scope+"\x00") {
				state.panels[key].Deactivate()
				delete(state.panels, key)
			}
		}
		if m.RightPaneTab != nil {
			*m.RightPaneTab = "files"
		}
	}
	if target.ID == "" {
		ui.Text(c, paneLabel("从会话任务或任务卡打开详情", "Open a task from its conversation entry or card"))
		return
	}
	m.ScheduleStore.EnsureContext(context.Background(), target.SessionID, nil)
	if state.panels == nil {
		state.panels = map[string]*scheduleui.Panel{}
	}
	sessions := renderSnapshot(m)
	links := map[string]string{}
	if sessions.ClientReady && !sessions.ListLoading && sessions.ListErr == "" {
		archived := map[string]bool{}
		for _, id := range sessions.ArchivedSessionIDs {
			archived[id] = true
		}
		for _, item := range sessions.Items {
			if !archived[item.SessionID] {
				links[item.SessionID] = item.Title
			}
		}
	}
	// One editor instance per task keeps its history and draft with that target.
	key := scope + "\x00" + target.SessionID + "\x00" + target.ID
	state.rendered = true
	for old, panel := range state.panels {
		if old != key {
			panel.Deactivate()
			delete(state.panels, old)
		}
	}
	panel := state.panels[key]
	if panel == nil {
		panel = &scheduleui.Panel{}
		state.panels[key] = panel
	}
	panel.View(c, scheduleui.Model{Store: m.ScheduleStore, SessionID: target.SessionID, DetailID: target.ID, DetailSessionID: target.SessionID,
		OnClose:      closeTask,
		SessionLinks: links, OnOpenSession: m.OnOpenScheduleSession, ThemeID: m.ThemeID, SystemDark: m.SystemDark, OnInvalidate: m.OnScheduleInvalidate,
		OnCopyHistory: func(text string) { mygo.Clipboard.WriteText(text) },
	})
}
