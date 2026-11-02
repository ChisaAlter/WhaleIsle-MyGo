package fake

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestBrowserTextRecorderRefusalAndIndependentRanges(t *testing.T) {
	w := &Window{Opts: &platform.WindowOptions{}}
	t.Cleanup(w.ForgetBrowserTextRecord)
	if !w.BrowserTextInputAvailable() {
		t.Fatal("fake provider missing")
	}
	start, end := 0, 2
	e := platform.BrowserTextEvent{Kind: platform.BrowserTextComposition, Text: "中", SelectionEnd: 1, ReplacementStart: &start, ReplacementEnd: &end}
	called := false
	w.DispatchBrowserTextInput(e, func(err error) {
		called = true
		if err != nil {
			t.Error(err)
		}
	})
	start, end = 6, 9
	r := w.BrowserTextRecord()
	if !called || len(r.Events) != 1 || *r.Events[0].ReplacementStart != 0 || *r.Events[0].ReplacementEnd != 2 {
		t.Fatalf("record %+v", r)
	}
	r.Disabled = true
	if platform.BrowserTextInputProvider(w) != nil {
		t.Fatal("disabled fake provider advertised support")
	}
	w.DispatchBrowserTextInput(e, func(err error) {
		if !errors.Is(err, platform.ErrUnsupported) {
			t.Errorf("disabled input: %v", err)
		}
	})
	r.Disabled = false
	w.closed = true
	w.DispatchBrowserTextInput(e, func(err error) {
		if !errors.Is(err, platform.ErrUnsupported) {
			t.Errorf("closed input: %v", err)
		}
	})
	if len(r.Events) != 1 {
		t.Fatal("refused events recorded")
	}
}

func TestBrowserTextRecorderHookReceivesNativeFailure(t *testing.T) {
	w := &Window{Opts: &platform.WindowOptions{}}
	t.Cleanup(w.ForgetBrowserTextRecord)
	failure := errors.New("fixture input failed")
	var reply func(error)
	w.BrowserTextRecord().Hook = func(_ platform.BrowserTextEvent, done func(error)) { reply = done }
	var got error
	w.DispatchBrowserTextInput(platform.BrowserTextEvent{Kind: platform.BrowserTextInsertText, Text: "鲸屿🙂"}, func(err error) { got = err })
	if reply == nil {
		t.Fatal("hook not called")
	}
	reply(failure)
	if !errors.Is(got, failure) {
		t.Fatalf("failure lost: %v", got)
	}
}
