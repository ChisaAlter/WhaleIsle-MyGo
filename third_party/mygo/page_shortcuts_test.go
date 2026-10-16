package mygo

import (
	"testing"
	"time"
)

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
