package composer

import "strings"

// ModelCommandState is transient UI state; selection remains owned by the
// existing per-session model directory and its confirmed Host projection.
type ModelCommandState struct {
	Open      bool
	SessionID string
	popup     *nativeCommandPopup
}

func (p *ModelCommandState) Visible(sessionID string) bool {
	return p.Open && sessionID != "" && p.SessionID == sessionID
}

func (p *ModelCommandState) Sync(sessionID string) {
	if p.SessionID != sessionID {
		p.close()
	}
}

// Handle claims only the bare local command. Failed requests remain local and
// preserve the draft; they can never become a prompt or Host command.
func (p *ModelCommandState) Handle(text, sessionID, targetID string, editing bool, attachments int) (bool, string) {
	if editing || !strings.EqualFold(strings.TrimSpace(text), "/model") {
		return false, ""
	}
	p.close()
	if sessionID == "" || (targetID != "" && targetID != sessionID) {
		return true, "Select the draft's session before choosing a model"
	}
	if attachments > 0 {
		return true, CommandAttachmentsUnsupportedNotice("model")
	}
	p.SessionID, p.Open = sessionID, true
	return true, ""
}

func (p *ModelCommandState) close() {
	if p.popup != nil {
		p.popup.close()
	}
	p.popup = nil
	p.Open = false
}
