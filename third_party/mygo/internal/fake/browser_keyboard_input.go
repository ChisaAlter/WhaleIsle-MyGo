package fake

import "github.com/egoist/mygo/internal/platform"

// BrowserTextRecord is configured and inspected on the fake UI thread. Hook may
// retain completion to exercise cancellation, navigation and ordered delivery.
type BrowserTextRecord struct {
	Events   []platform.BrowserTextEvent
	Disabled bool
	Hook     func(platform.BrowserTextEvent, func(error))
}

var browserTextRecords = map[*Window]*BrowserTextRecord{}

func (w *Window) BrowserTextRecord() *BrowserTextRecord {
	r := browserTextRecords[w]
	if r == nil {
		r = &BrowserTextRecord{}
		browserTextRecords[w] = r
	}
	return r
}

func (w *Window) ForgetBrowserTextRecord() { delete(browserTextRecords, w) }

func (w *Window) BrowserTextInputAvailable() bool {
	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	return !closed && !w.Opts.Surface && !w.BrowserTextRecord().Disabled
}

func (w *Window) DispatchBrowserTextInput(e platform.BrowserTextEvent, done func(error)) {
	if err := platform.ValidateBrowserTextEvent(e); err != nil {
		done(err)
		return
	}
	if !w.BrowserTextInputAvailable() {
		done(platform.ErrUnsupported)
		return
	}
	e = platform.CloneBrowserTextEvent(e)
	r := w.BrowserTextRecord()
	r.Events = append(r.Events, e)
	if r.Hook != nil {
		r.Hook(e, done)
		return
	}
	done(nil)
}
