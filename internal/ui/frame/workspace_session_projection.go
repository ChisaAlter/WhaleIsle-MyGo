package frame

// CaptureSessionProjection collects the legacy open-file adapters before their
// owner session changes. The pane remains the sole owner of its drafts.
func (w *WorkspacePanes) CaptureSessionProjection(m Model) {
	if w == nil {
		return
	}
	if w.Root == nil {
		w.syncRoute(derefString(m.RightPaneTab))
	}
	n := w.Root.find(w.legacyFileID)
	if n == nil {
		n = w.Root.find(w.Active)
	}
	if n != nil {
		w.fileState(n, m).capture(m)
		diff := w.diffState(n)
		// Capture belongs to the outgoing occurrence, even if the selected
		// session has already changed in the store.
		m.WorkspaceRoot = diff.Root
		diff.capture(m)
	}
}

// PublishSessionProjection replaces every adapter before any new-session
// binding/read occurs; a fresh session must never inherit the previous editor.
func (w *WorkspacePanes) PublishSessionProjection(m Model) {
	if w.Root == nil {
		w.syncRoute(derefString(m.RightPaneTab))
	}
	n := w.Root.find(w.Active)
	w.fileState(n, m).publish(m)
	w.diffState(n).publish(m)
	w.legacyFileID = n.ID
	w.route = n.Kind
	if m.RightPaneTab != nil {
		*m.RightPaneTab = n.Kind
	}
}

// SuspendSession releases transient capabilities, without replaying commands,
// saving editor buffers, or deleting the layout. Suspend retires stale callbacks
// even if the same Host connection is still serving the next session.
func (w *WorkspacePanes) SuspendSession() {
	if w == nil {
		return
	}
	w.PauseContent()
	w.Root.eachResource(func(n *WorkspacePane) {
		n.Agents = nil // a later dispose must not release another session's pane ID
		if n.Files != nil {
			n.Files.Tabs.SuspendSession()
		}
	})
	w.pending = nil
	w.closing = ""
}
