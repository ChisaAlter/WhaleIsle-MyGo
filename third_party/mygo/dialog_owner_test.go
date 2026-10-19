package mygo

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestMessageBeforeShowChecksQueuedOwner(t *testing.T) {
	// Run this on the actual fake-backend UI loop. Dialog.Message queues its
	// native show even when called on that thread; the earlier invalidation
	// must be observed at show time, not when Message was called.
	onMain(func() {
		previous := fb.MessageResult
		defer func() { fb.MessageResult = previous }()
		fb.MessageResult = platform.MessageBoxResult{Response: 1}
		current, checked := true, false
		postMain(func() { current = false })
		res, err := Dialog.Message(MessageOptions{
			Buttons: []string{"Decline", "Accept"},
			BeforeShow: func() bool {
				checked = true
				if !isMainThread() {
					t.Error("BeforeShow ran off main thread")
				}
				return current
			},
		})
		if err != nil || res.Button != 0 || !checked {
			t.Errorf("stale queued dialog was not canceled: result=%+v err=%v checked=%v", res, err, checked)
		}
	})
}

func TestMessageBeforeShowCurrentAndDefault(t *testing.T) {
	onMain(func() {
		previous := fb.MessageResult
		defer func() { fb.MessageResult = previous }()
		fb.MessageResult = platform.MessageBoxResult{Response: 1}
		for _, hook := range []func() bool{nil, func() bool { return true }} {
			res, err := Dialog.Message(MessageOptions{Buttons: []string{"Decline", "Accept"}, BeforeShow: hook})
			if err != nil || res.Button != 1 {
				t.Errorf("current/default dialog changed: %+v %v", res, err)
			}
		}
		res, err := Dialog.Message(MessageOptions{Buttons: []string{"Accept", "Cancel"}, CheckboxChecked: true, BeforeShow: func() bool { return false }})
		if err != nil || res.Button != 1 || !res.CheckboxChecked {
			t.Errorf("cancel defaults changed: %+v %v", res, err)
		}
	})
}
