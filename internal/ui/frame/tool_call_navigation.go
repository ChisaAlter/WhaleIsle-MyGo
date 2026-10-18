package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

func inspectToolCallNavigation(m Model, captured session.ViewState, pending *pendingToolCall) func(string) {
	if m.Store == nil || m.DisableDeveloperTools || m.HideChatViewTabs || m.ChatPaneTab == nil || m.TrajSelectedID == nil {
		return nil
	}
	return func(callID string) {
		live := m.Store.Snapshot()
		if live.SelectedID != captured.SelectedID || live.ComposerHostGeneration != captured.ComposerHostGeneration {
			return
		}
		if pending != nil {
			pending.request(m, live, callID)
		}
	}
}

func focusToolCallTrajectory(m Model, snapshot session.ViewState, callID string) bool {
	if m.ChatPaneTab == nil || m.TrajSelectedID == nil {
		return false
	}
	id, index, ok := chat.TrajectoryToolCallTarget(snapshot.Messages, callID)
	if !ok {
		return false
	}
	*m.ChatPaneTab = "trajectory"
	*m.TrajSelectedID = id
	if m.TrajSearch != nil {
		*m.TrajSearch = ""
	}
	if m.TrajTurnsCollapsed != nil {
		*m.TrajTurnsCollapsed = false
	}
	if m.TrajCallsCollapsed != nil {
		*m.TrajCallsCollapsed = false
	}
	if m.TrajDetailTab != nil {
		*m.TrajDetailTab = "args"
	}
	if m.ChatList != nil {
		m.ChatList.Key = nil
		m.ChatList.FollowEnd = false
		m.ChatList.ScrollTo(index, ui.Center)
	}
	return true
}
