package main

import (
	"context"
	"testing"
	"time"
)

func TestRoomPollUnmountCancelsOwnedReadersAndTimers(t *testing.T) {
	var state mainState
	groupContext, cancelGroup := context.WithCancel(context.Background())
	contactsContext, cancelContacts := context.WithCancel(context.Background())
	defer cancelGroup()
	defer cancelContacts()
	groupTimer, contactsTimer := time.NewTimer(time.Hour), time.NewTimer(time.Hour)
	defer groupTimer.Stop()
	defer contactsTimer.Stop()
	live := &state.assistantGroupLive
	live.scope, live.visible, live.rendered, live.busy, live.loaded = "group-owned", true, true, true, true
	live.cancel, live.timer = cancelGroup, groupTimer
	live.contacts.scope, live.contacts.visible, live.contacts.rendered = "page-owned", true, true
	live.contacts.busy, live.contacts.loaded = true, true
	live.contacts.cancel, live.contacts.timer = cancelContacts, contactsTimer
	live.contacts.rows = map[string]assistantContactRoomRuntime{"room": {err: "previous read error"}}
	groupGeneration, contactsGeneration := live.generation, live.contacts.generation
	state.beginAssistantGroupRuntimeFrame()
	state.finishAssistantGroupRuntimeFrame() // Neither original page painted.
	if groupContext.Err() != context.Canceled || contactsContext.Err() != context.Canceled {
		t.Fatal("hidden page retained an owned read")
	}
	if groupTimer.Stop() || contactsTimer.Stop() {
		t.Fatal("hidden page retained a scheduled repaint")
	}
	if live.visible || live.busy || live.loaded || live.scope != "" || live.contacts.visible || live.contacts.busy || live.contacts.loaded || live.contacts.scope != "" || len(live.contacts.rows) != 0 {
		t.Fatal("unmounted runtime state can be reused as current")
	}
	if live.generation == groupGeneration || live.contacts.generation == contactsGeneration {
		t.Fatal("old results were not retired on unmount")
	}
}

func TestRoomPollFrameKeepsOnlyThePageThatActuallyRendered(t *testing.T) {
	for _, keepGroup := range []bool{false, true} {
		t.Run(map[bool]string{true: "group", false: "bot-page"}[keepGroup], func(t *testing.T) {
			var state mainState
			groupContext, cancelGroup := context.WithCancel(context.Background())
			contactsContext, cancelContacts := context.WithCancel(context.Background())
			defer cancelGroup()
			defer cancelContacts()
			live := &state.assistantGroupLive
			live.scope, live.visible, live.cancel = "group", true, cancelGroup
			live.contacts.scope, live.contacts.visible, live.contacts.cancel = "page", true, cancelContacts
			state.beginAssistantGroupRuntimeFrame()
			if keepGroup {
				live.rendered = true
			} else {
				live.contacts.rendered = true
			}
			state.finishAssistantGroupRuntimeFrame()
			if keepGroup {
				if groupContext.Err() != nil || contactsContext.Err() != context.Canceled {
					t.Fatal("group visibility leaked into Bot page ownership")
				}
			} else if contactsContext.Err() != nil || groupContext.Err() != context.Canceled {
				t.Fatal("Bot page visibility leaked into group ownership")
			}
		})
	}
}
