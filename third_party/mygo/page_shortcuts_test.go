package mygo

import (
	"runtime"
	"testing"
	"time"
)

func TestWebShortcutInstallReturnsFromLoadCallbackBeforeEval(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Web application shortcut integration is Windows-only")
	}
	parent, _ := readyWindow(t, WindowOptions{})
	child, native := readyWindow(t, WindowOptions{})
	inLoadFinished := false
	evaluations := make(chan bool, 2)
	onMain(func() {
		native.AsyncCallback = func(_ string, reply func(string, error)) {
			evaluations <- inLoadFinished
			reply(`{"ok":true}`, nil)
		}
	})
	if !child.Page().UseWebApplicationShortcuts(parent) {
		t.Fatal("shortcut integration unavailable")
	}
	select {
	case <-evaluations:
	case <-time.After(2 * time.Second):
		t.Fatal("initial shortcut installation did not run")
	}
	onMain(func() {
		inLoadFinished = true
		native.H.LoadFinished()
		inLoadFinished = false
	})
	select {
	case nested := <-evaluations:
		if nested {
			t.Fatal("shortcut Eval ran inside the native LoadFinished callback and can deadlock WebView2 completion")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shortcut script was not installed after the load callback returned")
	}
}

func TestWebShortcutPhysicalGate(t *testing.T) {
	calls := 0
	parent := &Window{webShortcuts: &webShortcutState{bindings: map[webShortcutKey]func(){{188, 1}: func() { calls++ }}}}
	child := &Window{webShortcuts: &webShortcutState{parent: parent}}
	h := &windowHandler{w: child}
	ack := webShortcutPrefix + `{"VK":188,"Mods":1}`
	child.handleWebShortcutMessage(ack)
	if calls != 0 {
		t.Fatal("page invoked without native key")
	}
	h.WebApplicationKey(188, true, false, false, false, false)
	child.handleWebShortcutMessage(webShortcutPrefix + `{"VK":65,"Mods":1}`)
	if calls != 0 {
		t.Fatal("page changed physical command")
	}
	child.handleWebShortcutMessage(ack)
	child.handleWebShortcutMessage(ack)
	if calls != 1 {
		t.Fatal("physical event did not execute exactly once")
	}
	h.WebApplicationKey(188, true, false, false, false, true)
	child.handleWebShortcutMessage(ack)
	if calls != 1 {
		t.Fatal("repeat triggered command")
	}
	h.WebApplicationKey(188, true, false, false, false, false)
	child.webShortcuts.pendingUntil = time.Now().Add(-time.Second)
	child.handleWebShortcutMessage(ack)
	if calls != 1 {
		t.Fatal("stale event triggered command")
	}
	h.WebApplicationKey(188, true, false, false, false, false)
	parent.webShortcuts.revision++
	child.handleWebShortcutMessage(ack)
	if calls != 1 {
		t.Fatal("rebound registry executed old action")
	}
}
