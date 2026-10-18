package files

// SuspendSession retires callbacks while retaining the original Host owner of
// plan resources. Binding to an empty Host would discard those resource tabs;
// a session switch is only a pause. The next Bind still checks the real Host
// and cancels stale capabilities if that Host changed while this pane was away.
func (t *Tabs) SuspendSession() {
	t.Pause()
	t.CancelPlanRead()
	t.closeFloatingPreview()
	t.binding = ""
}
