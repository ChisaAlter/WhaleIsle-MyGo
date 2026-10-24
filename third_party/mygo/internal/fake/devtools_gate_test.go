package fake

import "testing"

// TestOpenDevToolsHonoursTheGate pins round206: a disabled runtime gate blocks
// the programmatic OpenDevTools call, and enabling it lets the call through.
func TestOpenDevToolsHonoursTheGate(t *testing.T) {
	w := &Window{}
	w.SetDevToolsEnabled(false)
	w.OpenDevTools()
	if w.IsDevToolsOpened() {
		t.Fatal("a gated page opened DevTools")
	}

	w.SetDevToolsEnabled(true)
	w.OpenDevTools()
	if !w.IsDevToolsOpened() {
		t.Fatal("an enabled page did not open DevTools")
	}

	// A page that never set the gate keeps the creation-time behaviour.
	bare := &Window{}
	bare.OpenDevTools()
	if !bare.IsDevToolsOpened() {
		t.Fatal("an ungated page did not open DevTools")
	}
}
