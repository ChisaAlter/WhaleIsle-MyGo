package webview

import (
	"context"
	"errors"
	"time"
)

func (h *Host) BrowserCookieToolsAvailable() bool {
	w, _, _, err := h.browserPage()
	return err == nil && w.Page().BrowserCookieToolsAvailable()
}

// ClearBrowserLoginCookies requires the navigation version the user confirmed.
// The narrow native provider independently verifies the Browser-only profile.
// Never call automatically or retry it after an uncertain native completion.
func (h *Host) ClearBrowserLoginCookies(ctx context.Context, confirmedVersion uint64) error {
	if ctx == nil {
		return errors.New("browser: Cookie operation context is required")
	}
	w, epoch, _, err := h.browserPage()
	if err != nil {
		return err
	}
	if epoch != confirmedVersion {
		return context.Canceled
	}
	ctx, stop := pageOperationContext(ctx, w, 15*time.Second)
	defer stop()
	if ctx.Err() != nil || !h.sameBrowserPage(w, epoch) {
		return context.Canceled
	}
	err = w.Page().ClearBrowserLoginCookies(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !h.sameBrowserPage(w, epoch) {
		return context.Canceled
	}
	return err
}
