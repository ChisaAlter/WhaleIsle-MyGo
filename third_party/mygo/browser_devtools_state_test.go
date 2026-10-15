package mygo

import (
	"errors"
	"testing"
)

type browserDevToolsFake struct {
	ready bool // Native capability includes verified Browser partition ownership.
	calls int
	err   error
}

func (f *browserDevToolsFake) BrowserDevToolsAvailable() bool { return f.ready }
func (f *browserDevToolsFake) OpenBrowserDevTools() error     { f.calls++; return f.err }

func TestBrowserDevToolsNarrowDispatch(t *testing.T) {
	f := &browserDevToolsFake{ready: true}
	if browserDevToolsProvider(f, false) != nil || !errors.Is(openBrowserDevTools(f, false), ErrBrowserDevToolsUnsupported) || f.calls != 0 {
		t.Fatal("pages with native bindings must never open this inspector")
	}
	for _, unsupported := range []any{nil, struct{}{}} {
		if browserDevToolsProvider(unsupported, true) != nil || !errors.Is(openBrowserDevTools(unsupported, true), ErrBrowserDevToolsUnsupported) {
			t.Fatal("missing backend capability must fail closed")
		}
	}
	if browserDevToolsProvider(f, true) != f {
		t.Fatal("ready, isolated-from-bindings Browser capability must be discoverable")
	}
	f.ready = false // Closing, loading or losing partition ownership after paint.
	if !errors.Is(openBrowserDevTools(f, true), ErrBrowserDevToolsUnsupported) || f.calls != 0 {
		t.Fatal("opening must recheck native readiness instead of trusting painted state")
	}
	f.ready = true
	if err := openBrowserDevTools(f, true); err != nil || f.calls != 1 {
		t.Fatal("one explicit dispatch must invoke the ready provider once")
	}
	f.err = errors.New("synthetic runtime/policy rejection")
	if err := openBrowserDevTools(f, true); !errors.Is(err, f.err) || f.calls != 2 {
		t.Fatal("native rejection must propagate without retries or broader fallback")
	}
}
