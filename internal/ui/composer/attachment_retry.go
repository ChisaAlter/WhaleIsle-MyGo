package composer

import (
	"context"
	"sync/atomic"
	"whaleisle/internal/store/session"
)

var attachmentAttempt atomic.Uint64

func (d *DraftState) AttachmentPreparing(index int) bool {
	if index < 0 || index >= len(d.Attachments) {
		return false
	}
	_, ok := d.attachmentCancels[d.Attachments[index].PreparationID]
	return ok
}
func (d *DraftState) BeginAttachmentPreparation(index int, retry bool) (session.ComposerAttachment, bool) {
	if d.Sending || index < 0 || index >= len(d.Attachments) || d.AttachmentPreparing(index) {
		return session.ComposerAttachment{}, false
	}
	a := d.Attachments[index]
	if retry && a.PrepareError == "" || !retry && a.PrepareError != "" {
		return session.ComposerAttachment{}, false
	}
	if d.Error == a.PrepareError {
		d.Error = ""
	}
	a.PrepareError = ""
	a.SentBytes, a.TotalBytes = 0, 0
	a.PreparationID = attachmentAttempt.Add(1)
	d.Attachments[index] = a
	if d.attachmentCancels == nil {
		d.attachmentCancels = map[uint64]context.CancelFunc{}
	}
	d.attachmentCancels[a.PreparationID] = nil
	d.Preparing++
	return a, true
}

// UpdateAttachmentProgress is called on Window.Update. A removed, retried or
// replaced attachment cannot consume a prior transport's receipt.
func (d *DraftState) UpdateAttachmentProgress(original session.ComposerAttachment, sent, total int64) {
	if _, ok := d.attachmentCancels[original.PreparationID]; !ok {
		return
	}
	for i, a := range d.Attachments {
		if a.PreparationID == original.PreparationID {
			d.Attachments[i].SentBytes = sent
			d.Attachments[i].TotalBytes = total
			return
		}
	}
}
func (d *DraftState) SetAttachmentCancel(a session.ComposerAttachment, cancel context.CancelFunc) {
	if _, ok := d.attachmentCancels[a.PreparationID]; ok {
		d.attachmentCancels[a.PreparationID] = cancel
	} else {
		cancel()
	}
}
func (d *DraftState) finishAttachmentAttempt(id uint64) bool {
	cancel, ok := d.attachmentCancels[id]
	if !ok {
		return false
	}
	delete(d.attachmentCancels, id)
	if cancel != nil {
		cancel()
	}
	if d.Preparing > 0 {
		d.Preparing--
	}
	return true
}
func (d *DraftState) CompleteAttachmentPreparation(_ int, original, prepared session.ComposerAttachment, err error) {
	if !d.finishAttachmentAttempt(original.PreparationID) {
		return
	}
	for i, a := range d.Attachments {
		if a.PreparationID != original.PreparationID {
			continue
		}
		if err != nil {
			d.Attachments[i].PrepareError = err.Error()
			d.Error = err.Error()
			return
		}
		prepared.PreparationID = original.PreparationID
		prepared.PrepareError = ""
		d.Attachments[i] = prepared
		return
	}
}
func (d *DraftState) RemoveAttachment(index int) bool {
	if d.Sending || index < 0 || index >= len(d.Attachments) {
		return false
	}
	a := d.Attachments[index]
	d.finishAttachmentAttempt(a.PreparationID)
	d.Attachments = append(d.Attachments[:index], d.Attachments[index+1:]...)
	if d.Error == a.PrepareError {
		d.Error = ""
	}
	return true
}
func (d *DraftState) FailedAttachmentError() string {
	for _, a := range d.Attachments {
		if a.PrepareError != "" {
			return "Retry or remove the failed attachment before sending: " + a.PrepareError
		}
	}
	return ""
}
