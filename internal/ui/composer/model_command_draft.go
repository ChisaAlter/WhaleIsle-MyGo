package composer

// HandleModelCommand is shared by native submission and the UI fallback. It
// never sends anything and only clears an accepted local command's text.
func (d *DraftState) HandleModelCommand(sessionID string) bool {
	if d == nil {
		return false
	}
	handled, err := d.ModelCommand.Handle(d.Text, sessionID, d.TargetID, d.EditSeq > 0, len(d.Attachments))
	if !handled {
		return false
	}
	d.Error = err
	if err == "" {
		d.Text = ""
		if d.Client != nil {
			d.Client.Sync()
		}
	}
	return true
}
