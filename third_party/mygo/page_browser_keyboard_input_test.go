package mygo

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

func TestBrowserTextDispatchAndCompositionOwnership(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true, Page: PageOptions{DisableBindings: true}})
	t.Cleanup(func() { RunOnMain(fw.ForgetBrowserTextRecord) })
	p := w.Page()
	if !p.BrowserTextInputAvailable() {
		t.Fatal("fake text provider missing")
	}
	if err := p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextCancelComposition}); err != nil {
		t.Fatal(err)
	}
	events := []BrowserTextEvent{
		{Kind: BrowserTextKeyDown, Key: "A", Code: "KeyA", WindowsVirtualKeyCode: 65, Modifiers: BrowserTextCtrl},
		{Kind: BrowserTextKeyUp, Key: "A", Code: "KeyA", WindowsVirtualKeyCode: 65},
		{Kind: BrowserTextComposition, Text: "中🙂", SelectionStart: 1, SelectionEnd: 3},
		{Kind: BrowserTextCancelComposition},
		{Kind: BrowserTextComposition, Text: "鲸屿", SelectionEnd: 2},
		{Kind: BrowserTextInsertText, Text: "鲸屿🙂"},
	}
	for _, e := range events {
		if err := p.TextInput(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextCancelComposition}); err != nil {
		t.Fatal(err)
	}
	RunOnMain(func() {
		if got := fw.BrowserTextRecord().Events; !reflect.DeepEqual(got, events) {
			t.Errorf("dispatched %+v want %+v", got, events)
		}
	})
}

func TestBrowserTextValidationUnsupportedContentAndDestroyed(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true})
	t.Cleanup(func() { RunOnMain(fw.ForgetBrowserTextRecord) })
	p := w.Page()
	if err := p.TextInput(nil, BrowserTextEvent{Kind: BrowserTextInsertText}); err == nil {
		t.Fatal("nil context accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.TextInput(ctx, BrowserTextEvent{Kind: BrowserTextInsertText}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	if err := p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextComposition, Text: "🙂", SelectionEnd: 1}); err == nil {
		t.Fatal("split surrogate accepted")
	}
	RunOnMain(func() { fw.BrowserTextRecord().Disabled = true })
	if p.BrowserTextInputAvailable() {
		t.Fatal("disabled provider available")
	}
	if err := p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextInsertText, Text: "x"}); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("unsupported: %v", err)
	}
	RunOnMain(func() {
		if len(fw.BrowserTextRecord().Events) != 0 {
			t.Error("refused input dispatched")
		}
	})
	if (*Page)(nil).BrowserTextInputAvailable() {
		t.Fatal("nil page available")
	}
	if err := (*Page)(nil).TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextInsertText}); !errors.Is(err, errDestroyed) {
		t.Fatalf("nil page: %v", err)
	}
	cw, _, _ := contentWindow(t, func(*ui.Context) {})
	if cw.pg.BrowserTextInputAvailable() {
		t.Fatal("native content available")
	}
	if err := cw.pg.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextInsertText}); !errors.Is(err, errNoPage) {
		t.Fatalf("native content: %v", err)
	}
	w.Destroy()
	if p.BrowserTextInputAvailable() {
		t.Fatal("destroyed page available")
	}
	if err := p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextInsertText}); !errors.Is(err, errDestroyed) {
		t.Fatalf("destroyed: %v", err)
	}
}

func TestBrowserTextPendingNavigationAndClose(t *testing.T) {
	for _, closeWindow := range []bool{false, true} {
		t.Run(map[bool]string{false: "navigation", true: "close"}[closeWindow], func(t *testing.T) {
			w, fw := testWindow(t, WindowOptions{Hidden: true})
			t.Cleanup(func() { RunOnMain(fw.ForgetBrowserTextRecord) })
			p := w.Page()
			started := make(chan struct{})
			var reply func(error)
			RunOnMain(func() {
				fw.BrowserTextRecord().Hook = func(_ platform.BrowserTextEvent, done func(error)) { reply = done; close(started) }
			})
			result := make(chan error, 1)
			go func() {
				result <- p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextComposition, Text: "中🙂", SelectionEnd: 3})
			}()
			<-started
			if err := p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextInsertText, Text: "later"}); !errors.Is(err, ErrBrowserTextInputBusy) {
				t.Fatalf("overlap: %v", err)
			}
			if closeWindow {
				w.Destroy()
			} else {
				RunOnMain(w.resetPage)
			}
			RunOnMain(func() { reply(nil); fw.BrowserTextRecord().Hook = nil })
			if err := <-result; !errors.Is(err, context.Canceled) && !errors.Is(err, errDestroyed) {
				t.Fatalf("retired document: %v", err)
			}
			if !closeWindow {
				if err := p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextCancelComposition}); err != nil {
					t.Fatal(err)
				}
			}
			RunOnMain(func() {
				if got := len(fw.BrowserTextRecord().Events); got != 1 {
					t.Errorf("old composition canceled into new document: %d", got)
				}
			})
		})
	}
}

func TestBrowserTextCancellationAndNativeFailure(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true})
	t.Cleanup(func() { RunOnMain(fw.ForgetBrowserTextRecord) })
	p := w.Page()
	started := make(chan struct{})
	var reply func(error)
	RunOnMain(func() {
		fw.BrowserTextRecord().Hook = func(_ platform.BrowserTextEvent, done func(error)) { reply = done; close(started) }
	})
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- p.TextInput(ctx, BrowserTextEvent{Kind: BrowserTextInsertText, Text: "committed"}) }()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("pending cancel: %v", err)
	}
	if err := p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextInsertText}); !errors.Is(err, ErrBrowserTextInputBusy) {
		t.Fatalf("overlap after cancel: %v", err)
	}
	failure := errors.New("native failure")
	RunOnMain(func() {
		reply(nil)
		fw.BrowserTextRecord().Hook = func(_ platform.BrowserTextEvent, done func(error)) { done(failure) }
	})
	if err := p.TextInput(context.Background(), BrowserTextEvent{Kind: BrowserTextInsertText, Text: "x"}); !errors.Is(err, failure) {
		t.Fatalf("native failure: %v", err)
	}
}
