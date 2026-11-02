package pluginui

// retireMirrorTextLocked detaches native text authority across owner lifecycle
// boundaries. Browser operations already submitted retain their document guard;
// no queued operation or late acknowledgement can revive these clients.
func (o *Owner) retireMirrorTextLocked() {
	if o.mirror == nil || o.mirror.text == nil {
		return
	}
	state := o.mirror.text
	state.closed = true
	state.queue = nil
	for _, client := range state.clients {
		client.focus = nil
		client.marked = nil
		client.pending = 0
	}
	o.mirror.text = nil
}
