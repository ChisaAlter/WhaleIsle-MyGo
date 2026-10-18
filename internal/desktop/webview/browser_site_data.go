package webview

import (
	"context"
	"errors"
	"time"
)

func (h *Host) BrowserSiteDataToolsAvailable() bool {
	w, _, _, err := h.browserPage()
	return err == nil && w.Page().BrowserSiteDataToolsAvailable()
}

// ClearBrowserSiteData accepts only the page/navigation explicitly confirmed by
// the user. Native code independently validates its dedicated partition owner.
// A submitted cleanup cannot be undone by cancellation and is never retried.
func (h *Host) ClearBrowserSiteData(ctx context.Context, confirmedVersion uint64) error {
	if ctx == nil {
		return errors.New("browser: site-data operation context is required")
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
	err = w.Page().ClearBrowserSiteData(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !h.sameBrowserPage(w, epoch) {
		return context.Canceled
	}
	return err
}
