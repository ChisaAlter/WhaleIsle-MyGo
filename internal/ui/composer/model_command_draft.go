package composer

import "strings"

// HandleModelCommand is shared by native submission and the UI fallback. It
// never sends anything and only clears an accepted local command's text.
func (d *DraftState) HandleModelCommand(sessionID string) bool {
	if d == nil {
		return false
	}
	handled, err := false, ""
	if strings.EqualFold(strings.TrimSpace(d.Text), "/permission") {
		handled, err = d.PermissionCommand.Handle("/model", sessionID, d.TargetID, d.EditSeq > 0, len(d.Attachments))
		err = strings.ReplaceAll(strings.ReplaceAll(err, "/model", "/permission"), "a model", "permissions")
		if handled {
			d.ModelCommand.Open = false
		}
	} else {
		handled, err = d.ModelCommand.Handle(d.Text, sessionID, d.TargetID, d.EditSeq > 0, len(d.Attachments))
		if handled {
			d.PermissionCommand.Open = false
		}
	}
	if !handled {
		return false
	}
	d.Error = err
	if err == "" {
		d.Text = ""
		if d.Client != nil {
			d.Client.ClearHistory()
		}
	}
	return true
}
