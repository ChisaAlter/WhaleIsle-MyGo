package pluginui

// MirrorLayout retains only the last same-document geometry while a fresh
// bitmap is pending. Keeping the native input node mounted preserves its IME
// adapter/focus. Paint and mouse still require a fully current captured frame.
func (o *Owner) MirrorLayout() *MirrorFrame {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.mirror == nil || o.closed || o.masked || o.incompatible || !o.state.Connected {
		return nil
	}
	f := o.mirror.frame
	if f == nil {
		f = o.mirror.layout
	}
	if f == nil || f.page != o.state.Page || !o.selected.equal(f.subject) {
		return nil
	}
	for _, occurrence := range f.Occurrences {
		found := false
		for _, row := range o.state.Outlets {
			if row.Handle == occurrence.Handle && row.Slot == "conversation.chat.turnTail" && row.EntryID == occurrence.Entry && row.HasTurn && row.HasTurnTail && row.Turn == occurrence.Turn && row.TurnTailSeq == occurrence.Seq && row.Available {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return f
}
func (o *Owner) mirrorLayoutCurrentLocked(f *MirrorFrame) bool {
	return f != nil && o.mirror != nil && !o.closed && !o.masked && !o.incompatible && o.state.Connected && o.state.Page == f.page && o.selected.equal(f.subject) && (o.mirror.frame == f || o.mirror.layout == f)
}
