package composer

// RetireBinding releases transient Host/UI leases without changing authored
// text, references, attachments, selection or undo history. Persisted child
// drafts may survive reconnects; pending lookups and Escape pairs may not.
func (d *DraftState) RetireBinding() {
	if d == nil {
		return
	}
	d.references.cancelQuery()
	d.RetireNativeCommands()
	d.Suggestions = SuggestionState{}
	if d.Client != nil {
		d.Client.stop.reset()
		d.Client.regionStop = [2]stopSequence{}
		d.Client.suggestions = SuggestionState{}
	}
}
