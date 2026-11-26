package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

type toolCallNavTarget struct {
	store                 *session.Store
	disableDeveloperTools bool
	hideChatViewTabs      bool
	chatPaneTab           *string
	trajSelectedID        *string
	trajSearch            *string
	trajTurnsCollapsed    *bool
	trajCallsCollapsed    *bool
	trajDetailTab         *string
	chatList              *ui.ListState
}

func toolCallNavTargetOf(m *Model) toolCallNavTarget {
	return toolCallNavTarget{
		store:                 m.Store,
		disableDeveloperTools: m.DisableDeveloperTools,
		hideChatViewTabs:      m.HideChatViewTabs,
		chatPaneTab:           m.ChatPaneTab,
		trajSelectedID:        m.TrajSelectedID,
		trajSearch:            m.TrajSearch,
		trajTurnsCollapsed:    m.TrajTurnsCollapsed,
		trajCallsCollapsed:    m.TrajCallsCollapsed,
		trajDetailTab:         m.TrajDetailTab,
		chatList:              m.ChatList,
	}
}

func inspectToolCallNavigation(m Model, captured session.ViewState, pending *pendingToolCall) func(string) {
	return inspectToolCallNavigationPtr(&m, captured, pending)
}

func inspectToolCallNavigationPtr(m *Model, captured session.ViewState, pending *pendingToolCall) func(string) {
	if m.Store == nil || m.DisableDeveloperTools || m.HideChatViewTabs || m.ChatPaneTab == nil || m.TrajSelectedID == nil {
		return nil
	}
	target := toolCallNavTargetOf(m)
	selectedID := captured.SelectedID
	hostGen := captured.ComposerHostGeneration
	return func(callID string) {
		live := target.store.Published().Value
		if live.SelectedID != selectedID || live.ComposerHostGeneration != hostGen {
			return
		}
		if pending != nil {
			pending.requestTarget(target, live, callID)
		}
	}
}

func focusToolCallTrajectory(m Model, snapshot session.ViewState, callID string) bool {
	return focusToolCallTrajectoryTarget(toolCallNavTargetOf(&m), snapshot, callID)
}

func focusToolCallTrajectoryTarget(t toolCallNavTarget, snapshot session.ViewState, callID string) bool {
	if t.chatPaneTab == nil || t.trajSelectedID == nil {
		return false
	}
	id, index, ok := chat.TrajectoryToolCallTarget(snapshot.Messages, callID)
	if !ok {
		return false
	}
	*t.chatPaneTab = "trajectory"
	*t.trajSelectedID = id
	if t.trajSearch != nil {
		*t.trajSearch = ""
	}
	if t.trajTurnsCollapsed != nil {
		*t.trajTurnsCollapsed = false
	}
	if t.trajCallsCollapsed != nil {
		*t.trajCallsCollapsed = false
	}
	if t.trajDetailTab != nil {
		*t.trajDetailTab = "args"
	}
	if t.chatList != nil {
		t.chatList.Key = nil
		t.chatList.FollowEnd = false
		t.chatList.ScrollTo(index, ui.Center)
	}
	return true
}
