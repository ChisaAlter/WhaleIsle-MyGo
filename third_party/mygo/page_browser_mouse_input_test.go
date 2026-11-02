package mygo

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestPrototypeBrowserMouseDispatchAndCancel(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true, Page: PageOptions{DisableBindings: true}})
	t.Cleanup(func() { RunOnMain(fw.ForgetBrowserMouseRecord) })
	p := w.Page()
	if !p.BrowserMouseInputAvailable() {
		t.Fatal("fake typed input unavailable")
	}
	press := BrowserMouseEvent{Kind: BrowserMousePress, X: 24.5, Y: 60, Button: BrowserMouseLeft, Buttons: 1, Modifiers: 10, ClickCount: 1}
	if err := p.MouseInput(context.Background(), press); err != nil {
		t.Fatal(err)
	}
	if err := p.MouseInput(context.Background(), BrowserMouseEvent{Kind: BrowserMouseCancel}); err != nil {
		t.Fatal(err)
	}
	RunOnMain(func() {
		events := fw.BrowserMouseRecord().Events
		if len(events) != 4 || events[0] != press || events[1].Kind != BrowserMouseMove || events[1].X != -1 || events[1].Buttons != 1 || events[2].Kind != BrowserMouseRelease || events[2].Button != BrowserMouseLeft || events[2].Buttons != 0 || events[2].ClickCount != 0 || events[3].Kind != BrowserMouseMove || events[3].Buttons != 0 {
			t.Errorf("events=%+v", events)
		}
	})
}

func TestPrototypeBrowserMouseValidationUnsupportedAndContext(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true})
	t.Cleanup(func() { RunOnMain(fw.ForgetBrowserMouseRecord) })
	p := w.Page()
	invalid := []BrowserMouseEvent{{Kind: 99}, {X: math.NaN()}, {DeltaY: 1}, {Buttons: 32}, {Modifiers: 16}, {Kind: BrowserMousePress, Button: BrowserMouseLeft}, {Kind: BrowserMouseRelease, Button: BrowserMouseLeft, Buttons: 1}, {Kind: BrowserMouseCancel, Buttons: 1}, {Kind: BrowserMouseWheel, ClickCount: 1}}
	for _, e := range invalid {
		if err := p.MouseInput(context.Background(), e); err == nil {
			t.Errorf("accepted %+v", e)
		}
	}
	if err := p.MouseInput(nil, BrowserMouseEvent{}); err == nil {
		t.Error("nil context accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.MouseInput(ctx, BrowserMouseEvent{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancel=%v", err)
	}
	RunOnMain(func() { fw.BrowserMouseRecord().Disabled = true })
	if p.BrowserMouseInputAvailable() {
		t.Fatal("disabled provider available")
	}
	if err := p.MouseInput(context.Background(), BrowserMouseEvent{}); !errors.Is(err, ErrBrowserMouseInputUnsupported) {
		t.Fatalf("unsupported=%v", err)
	}
	RunOnMain(func() {
		if got := len(fw.BrowserMouseRecord().Events); got != 0 {
			t.Errorf("invalid calls dispatched %d events", got)
		}
	})
	if (*Page)(nil).BrowserMouseInputAvailable() {
		t.Fatal("nil page available")
	}
}

func TestPrototypeBrowserMouseBusyAndNavigationDrop(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true})
	t.Cleanup(func() { RunOnMain(fw.ForgetBrowserMouseRecord) })
	p := w.Page()
	started := make(chan struct{})
	var reply func(error)
	RunOnMain(func() {
		fw.BrowserMouseRecord().Hook = func(_ platform.BrowserMouseEvent, done func(error)) { reply = done; close(started) }
	})
	result := make(chan error, 1)
	go func() {
		result <- p.MouseInput(context.Background(), BrowserMouseEvent{Kind: BrowserMousePress, Button: BrowserMouseLeft, Buttons: 1, ClickCount: 1})
	}()
	<-started
	if err := p.MouseInput(context.Background(), BrowserMouseEvent{}); !errors.Is(err, ErrBrowserMouseInputBusy) {
		t.Fatalf("overlap=%v", err)
	}
	RunOnMain(func() { w.resetPage(); reply(nil); fw.BrowserMouseRecord().Hook = nil })
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("navigation=%v", err)
	}
	if err := p.MouseInput(context.Background(), BrowserMouseEvent{Kind: BrowserMouseCancel}); err != nil {
		t.Fatal(err)
	}
	RunOnMain(func() {
		events := fw.BrowserMouseRecord().Events
		if len(events) != 3 || events[1].Kind != BrowserMouseMove || events[1].Buttons != 0 || events[2].Kind != BrowserMouseMove {
			t.Errorf("old press released into new document: %+v", events)
		}
	})
}

func TestPrototypeBrowserMouseCancelStopsAtDocumentChange(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true})
	t.Cleanup(func() { RunOnMain(fw.ForgetBrowserMouseRecord) })
	p := w.Page()
	if err := p.MouseInput(context.Background(), BrowserMouseEvent{Kind: BrowserMousePress, Button: BrowserMouseLeft, Buttons: 1, ClickCount: 1}); err != nil {
		t.Fatal(err)
	}
	RunOnMain(func() {
		fw.BrowserMouseRecord().Hook = func(_ platform.BrowserMouseEvent, done func(error)) { w.resetPage(); done(nil) }
	})
	if err := p.MouseInput(context.Background(), BrowserMouseEvent{Kind: BrowserMouseCancel}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel navigation=%v", err)
	}
	RunOnMain(func() {
		if got := len(fw.BrowserMouseRecord().Events); got != 2 {
			t.Errorf("continued old cleanup after navigation: %d", got)
		}
	})
}
