package composer

import "whaleisle/internal/store/session"

type GroupReplyDraft struct {
	ThreadID, Text string
	Attachments    []session.ComposerAttachment
}

func (d *DraftState) BeginGroupReply(id string) bool {
	if d == nil || d.Sending || d.Preparing > 0 || d.EditSeq > 0 || d.GroupReply != nil {
		return false
	}
	if _, err := session.EncodeGroupThreadReply(id, ""); err != nil {
		return false
	}
	d.GroupReply = &GroupReplyDraft{id, d.Text, append([]session.ComposerAttachment(nil), d.Attachments...)}
	d.Text = session.GroupThreadReplyCommand + " " + id + " "
	d.Error = ""
	d.Client.Sync()
	return true
}
func (d *DraftState) CancelGroupReply() {
	if d == nil || d.GroupReply == nil || d.Sending {
		return
	}
	d.Text = d.GroupReply.Text
	d.Attachments = append([]session.ComposerAttachment(nil), d.GroupReply.Attachments...)
	d.GroupReply = nil
	d.Error = ""
	d.Client.Sync()
}
