package fake

import "github.com/egoist/mygo/internal/platform"

// The recorder lives in an optional file so existing fake Window layout is not
// patched by this prototype. Access runs on the fake UI thread; tests use
// mygo.RunOnMain for configuration/inspection. Backend closure is still checked.
type BrowserMouseRecord struct {
	Events   []platform.BrowserMouseEvent
	Disabled bool
	Hook     func(platform.BrowserMouseEvent, func(error))
}

var browserMouseRecords = map[*Window]*BrowserMouseRecord{}

func (w *Window) BrowserMouseRecord() *BrowserMouseRecord {
	r := browserMouseRecords[w]
	if r == nil {
		r = &BrowserMouseRecord{}
		browserMouseRecords[w] = r
	}
	return r
}
func (w *Window) ForgetBrowserMouseRecord() { delete(browserMouseRecords, w) }
func (w *Window) BrowserMouseInputAvailable() bool {
	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	return !closed && !w.Opts.Surface && !w.BrowserMouseRecord().Disabled
}
func (w *Window) DispatchBrowserMouseInput(e platform.BrowserMouseEvent, done func(error)) {
	if err := platform.ValidateBrowserMouseEvent(e); err != nil {
		done(err)
		return
	}
	if !w.BrowserMouseInputAvailable() {
		done(platform.ErrUnsupported)
		return
	}
	r := w.BrowserMouseRecord()
	r.Events = append(r.Events, e)
	if r.Hook != nil {
		r.Hook(e, done)
		return
	}
	done(nil)
}
