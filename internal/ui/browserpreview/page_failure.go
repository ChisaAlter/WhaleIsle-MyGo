package browserpreview

import (
	desktop "whaleisle/internal/desktop/webview"

	"whaleisle/internal/errcopy"
)

// SetPageFailureHandler connects the shell's explicit recovery dialog without
// transferring navigation or filesystem authority to the Browser controller.
func (b *Controller) SetPageFailureHandler(fn func(desktop.PageFailure)) { b.onFailure = fn }
func (b *Controller) pageFailure(t *Tab) func(desktop.PageFailure) {
	return func(failure desktop.PageFailure) {
		if failure.Current == nil || !failure.Current() {
			return
		}
		apply := func() {
			if !failure.Current() {
				return
			}
			for _, tab := range b.Tabs {
				if tab == t {
					// Kind is the runtime's stable token (render-process-gone);
					// Message is its English prose. The classifier reads both and
					// the chrome paints only the resulting copy.
					t.Error = errcopy.SentenceText(failure.Kind + " " + failure.Message)
					if b.onFailure != nil {
						b.onFailure(failure)
					}
					return
				}
			}
		}
		if b.parent != nil {
			b.parent.Update(apply)
		} else {
			apply()
		}
	}
}
