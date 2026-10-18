package composer

import "strings"

// CanSteerPendingQueue guards the separate empty-editor accelerated gesture.
// It never converts text, attachments, an edit or a command popup into a batch.
func (d *DraftState) CanSteerPendingQueue(sessionID string, accelerated bool) bool {
	return d != nil && accelerated && sessionID != "" && !d.Sending && d.Preparing == 0 && d.EditSeq == 0 && strings.TrimSpace(d.Text) == "" && len(d.Attachments) == 0 && !d.ModelCommand.Visible(sessionID)
}
