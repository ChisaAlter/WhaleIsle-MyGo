package pluginui

// MirrorDiagnostics returns cached pipeline stages. It never evaluates renderer
// code or exposes a browser capability to a native control.
func (o *Owner) MirrorDiagnostics() any {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.mirror == nil {
		return nil
	}
	m := o.mirror
	return map[string]any{"generation": m.generation, "inflight": m.inflight, "inputBusy": m.inputBusy, "error": m.error, "pointerStage": m.pointerStage, "textFocusError": m.textFocusError}
}
func (o *Owner) mirrorPointerStage(m *mirrorState, stage string) {
	o.mu.Lock()
	if o.mirror == m {
		m.pointerStage = stage
	}
	o.mu.Unlock()
}
