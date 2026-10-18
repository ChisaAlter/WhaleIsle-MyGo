package composer

import (
	"errors"
	"strings"
)

// AppendSelectionDraft is a local draft edit only. It neither submits a turn nor
// prepares/uploads an attachment. Existing text and attachments are preserved.
func AppendSelectionDraft(d *DraftState, fragment string) error {
	if d == nil || d.Sending {
		return errors.New("the composer is currently sending; try again when it finishes")
	}
	if strings.TrimSpace(fragment) == "" {
		return errors.New("select some text first")
	}
	if d.Client != nil {
		if _, marked := d.Client.MarkedRange(); marked {
			return errors.New("finish the current text composition before adding a selection")
		}
	}
	if d.Text == "" {
		d.Text = fragment
	} else {
		d.Text += " " + fragment
	}
	if d.Client == nil {
		d.Client = NewDraftClient(&d.Text)
	} else {
		d.Client.Sync()
	}
	d.Error = ""
	return nil
}
