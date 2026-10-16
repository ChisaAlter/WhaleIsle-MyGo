package composer

import "whaleisle/internal/store/session"

// BeginAttachmentPreparation runs on the UI thread. Retries cannot race a send
// or another preparation, so the attachment index stays stable while uploading.
func (d *DraftState) BeginAttachmentPreparation(index int, retry bool) (session.ComposerAttachment, bool) {
	if d.Sending || index < 0 || index >= len(d.Attachments) {
		return session.ComposerAttachment{}, false
	}
	a := d.Attachments[index]
	if retry && (d.Preparing > 0 || a.PrepareError == "") {
		return session.ComposerAttachment{}, false
	}
	if a.PrepareError != "" && d.Error == a.PrepareError {
		d.Error = ""
	}
	d.Attachments[index].PrepareError = ""
	d.Preparing++
	return a, true
}

// CompleteAttachmentPreparation only updates the originating draft and slot.
func (d *DraftState) CompleteAttachmentPreparation(index int, original, prepared session.ComposerAttachment, err error) {
	if d.Preparing > 0 {
		d.Preparing--
	}
	if index < 0 || index >= len(d.Attachments) || d.Attachments[index].Path != original.Path {
		return
	}
	if err != nil {
		d.Attachments[index].PrepareError = err.Error()
		d.Error = err.Error()
		return
	}
	prepared.PrepareError = ""
	d.Attachments[index] = prepared
}
