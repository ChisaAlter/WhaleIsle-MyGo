package ui

// DialogOptions controls a dialog's dismissal without letting a blocked click
// or Escape reach the page behind it. An OnDismiss callback owns the transition:
// it may close a nested confirmation first, or set open to false to close the
// entire dialog. The zero value preserves DialogBase's existing behavior.
type DialogOptions struct {
	DismissDisabled bool
	OnDismiss       func()
}

func DialogBaseWithOptions(c *Context, open *bool, options DialogOptions, fn func(backdrop Element, panel Element)) Element {
	ctx := c.build()
	if ctx == nil {
		return Element{}
	}
	return wrapElement(coreDialogBaseOptions(ctx, open, options, func(backdrop, panel *node) {
		fn(wrapElement(backdrop), wrapElement(panel))
	}))
}
