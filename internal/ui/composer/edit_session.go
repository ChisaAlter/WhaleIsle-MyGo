package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// BeginEdit borrows the permanent composer without destroying its draft.
func (d *DraftState) BeginEdit(id string, seq float64, body string) bool {
	if id == "" || seq <= 0 || d.Sending || d.Preparing > 0 || d.EditSeq > 0 {
		return false
	}
	d.Client.Sync()
	d.stashedReferences = append([]draftReference(nil), d.Client.references...)
	d.stashedText, d.stashedAttachments = d.Text, d.Attachments
	d.Text, d.Attachments, d.Error = body, nil, ""
	d.EditSeq, d.EditSessionID, d.EditFocus = seq, id, true
	d.TargetID = id
	d.Client.Sync()
	n := ui.UTF16Len(body)
	d.Client.SetSelectionUTF16(n, n)
	return true
}

// EndEdit is used only after acceptance or explicit cancellation; failures keep editing.
func (d *DraftState) EndEdit() {
	if d.EditSeq <= 0 || d.Sending {
		return
	}
	d.Text, d.Attachments = d.stashedText, d.stashedAttachments
	d.stashedText, d.stashedAttachments = "", nil
	d.EditSeq, d.EditSessionID, d.TargetID, d.Error = 0, "", "", ""
	d.EditFocus = true
	d.Client.Sync()
	d.Client.references = append(d.Client.references[:0], d.stashedReferences...)
	d.stashedReferences = nil
	d.Client.shape()
}

func (d *DraftState) EditAllowed(s session.ViewState) bool {
	return d.EditSeq > 0 && d.EditSessionID == s.SelectedID && session.CanEditMessage(s, d.EditSeq)
}
