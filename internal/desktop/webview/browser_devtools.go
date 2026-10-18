package webview

import (
	"context"

	"github.com/egoist/mygo"
)

func (h *Host) BrowserDevToolsAvailable() bool {
	w, _, _, err := h.browserPage()
	return err == nil && w.Page().BrowserDevToolsAvailable()
}

// OpenBrowserDevTools must follow an explicit current-tab UI click. Capture and
// validate Host/navigation on the main thread immediately before native dispatch,
// so an off-thread caller cannot open an inspector for a replacement navigation.
func (h *Host) OpenBrowserDevTools(clickedVersion uint64) error {
	err := error(context.Canceled)
	mygo.RunOnMain(func() {
		w, epoch, _, pageErr := h.browserPage()
		if pageErr != nil {
			err = pageErr
			return
		}
		if epoch != clickedVersion || !h.sameBrowserPage(w, epoch) {
			return
		}
		err = w.Page().OpenBrowserDevTools()
	})
	return err
}
