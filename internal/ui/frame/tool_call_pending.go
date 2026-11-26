package frame

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
)

// Retains one view request, not a history-loading job. Only already-loaded
// snapshots are considered; newer requests replace older ones.
type pendingToolCall struct {
	sessionID, callID string
	hostGeneration    uint64
}

func (p *pendingToolCall) request(m Model, snapshot session.ViewState, callID string) {
	p.requestTarget(toolCallNavTargetOf(&m), snapshot, callID)
}

func (p *pendingToolCall) requestTarget(t toolCallNavTarget, snapshot session.ViewState, callID string) {
	if strings.TrimSpace(callID) == "" || t.chatPaneTab == nil || t.trajSelectedID == nil {
		return
	}
	*p = pendingToolCall{sessionID: snapshot.SelectedID, hostGeneration: snapshot.ComposerHostGeneration, callID: callID}
	*t.chatPaneTab = "trajectory"
	*t.trajSelectedID = ""
	p.applyTarget(t, snapshot)
}

func (p *pendingToolCall) apply(m Model, snapshot session.ViewState) {
	p.applyTarget(toolCallNavTargetOf(&m), snapshot)
}

func (p *pendingToolCall) applyTarget(t toolCallNavTarget, snapshot session.ViewState) {
	if p.callID == "" {
		return
	}
	if p.sessionID != snapshot.SelectedID || p.hostGeneration != snapshot.ComposerHostGeneration || t.disableDeveloperTools || t.hideChatViewTabs ||
		t.chatPaneTab == nil || *t.chatPaneTab != "trajectory" {
		*p = pendingToolCall{}
		return
	}
	if focusToolCallTrajectoryTarget(t, snapshot, p.callID) {
		*p = pendingToolCall{}
	}
}

func toolCallNavigationFrame(c *ui.Context, m *Model, snapshot session.ViewState) func(string) {
	pending := ui.Local(c.Root(), "trajectory-inspect-call", func() pendingToolCall { return pendingToolCall{} })
	if pending.callID != "" {
		pending.applyTarget(toolCallNavTargetOf(m), snapshot)
	}
	return inspectToolCallNavigationPtr(m, snapshot, pending)
}
