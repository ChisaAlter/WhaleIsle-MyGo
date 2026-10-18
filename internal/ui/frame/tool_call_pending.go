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
	if strings.TrimSpace(callID) == "" || m.ChatPaneTab == nil || m.TrajSelectedID == nil {
		return
	}
	*p = pendingToolCall{sessionID: snapshot.SelectedID, hostGeneration: snapshot.ComposerHostGeneration, callID: callID}
	*m.ChatPaneTab = "trajectory"
	*m.TrajSelectedID = ""
	p.apply(m, snapshot)
}

func (p *pendingToolCall) apply(m Model, snapshot session.ViewState) {
	if p.callID == "" {
		return
	}
	if p.sessionID != snapshot.SelectedID || p.hostGeneration != snapshot.ComposerHostGeneration || m.DisableDeveloperTools || m.HideChatViewTabs ||
		m.ChatPaneTab == nil || *m.ChatPaneTab != "trajectory" {
		*p = pendingToolCall{}
		return
	}
	if focusToolCallTrajectory(m, snapshot, p.callID) {
		*p = pendingToolCall{}
	}
}

func toolCallNavigationFrame(c *ui.Context, m Model, snapshot session.ViewState) func(string) {
	pending := ui.Local(c.Root(), "trajectory-inspect-call", func() pendingToolCall { return pendingToolCall{} })
	pending.apply(m, snapshot)
	return inspectToolCallNavigation(m, snapshot, pending)
}
