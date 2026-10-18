package composer

import (
	"errors"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/filelink"
)

// InsertWorkspaceReference is a draft edit, preserving native undo, attachments
// and the selected range. It never uploads a file or submits a prompt.
func InsertWorkspaceReference(d *DraftState, ref filelink.Reference, binding, sessionID, root string) error {
	if !ref.Matches(binding, sessionID, root) {
		return errors.New("the file belongs to a different conversation or workspace")
	}
	if d == nil || d.Sending {
		return errors.New("wait for the current send to finish before adding a file")
	}
	if d.Client == nil {
		d.Client = NewDraftClient(&d.Text)
	}
	if _, marked := d.Client.MarkedRange(); marked {
		return errors.New("finish text composition before adding a file")
	}
	selection := d.Client.Selection().Range
	start, end := ui.UTF16ByteOffset(d.Text, selection.Start), ui.UTF16ByteOffset(d.Text, selection.End)
	replacement, ok := filelink.Insertion(d.Text, start, end, ref.Path)
	if !ok {
		return errors.New("this file path cannot be inserted safely")
	}
	d.Client.ReplaceText(&selection, replacement)
	d.Error = ""
	d.EditFocus = true
	return nil
}

func wireWorkspaceReferenceDrop(area ui.Element, m Model) {
	if ref, ok := ui.Drop[filelink.Reference](area); ok && m.OnWorkspaceReference != nil {
		if err := m.OnWorkspaceReference(ref); err != nil && m.State != nil {
			m.State.Error = err.Error()
		}
	}
}
