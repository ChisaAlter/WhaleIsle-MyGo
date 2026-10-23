package chat

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestInspectorSmoothStopsCaptureBeforeQueuedFlash(t *testing.T) {
	tt, n, scroll, _ := startInspectorSmooth(t)
	end := time.Unix(7000, 0).Add(inspectorChatScrollDuration(1300))
	tt.SetTime(end)
	if !n.reveal.waiting || !n.reveal.pending || !n.reveal.started.IsZero() {
		t.Fatalf("completion skipped confirming frame:%+v", n.reveal)
	}
	tt.SetTime(end.Add(time.Millisecond))
	if n.reveal.waiting || !n.reveal.pending || !n.reveal.started.IsZero() {
		t.Fatalf("confirmation did not queue flash:%+v", n.reveal)
	}
	tt.Key(0, ui.KeyEscape)
	tt.SetTime(end.Add(2 * time.Millisecond))
	if n.reveal.started.IsZero() || scroll.Y != 1400 {
		t.Fatal("settled navigation was interrupted before queued flash")
	}
}

func TestInspectorSmoothQueuedOffscreenIntentCanBeInterrupted(t *testing.T) {
	tt, n, scroll, list, _ := inspectorRevealFixture(t)
	list.ScrollToOffset(100)
	tt.Frame()
	n.Highlight = "25"
	n.reveal.pending = true
	n.reveal.centerQueued = true
	tt.Scroll(250, 340, 0, 0)
	tt.SetTime(time.Unix(7000, 0).Add(2 * time.Second))
	if scroll.Y != 100 || n.reveal.pending || list.ScrollAnimating() || !n.reveal.started.IsZero() {
		t.Fatal("offscreen queued navigation ignored user interruption")
	}
}

func TestInspectorSmoothInputDuringConfirmingFrameCancels(t *testing.T) {
	tt, n, _, list := startInspectorSmooth(t)
	end := time.Unix(7000, 0).Add(inspectorChatScrollDuration(1300))
	tt.SetTime(end)
	tt.Scroll(250, 340, 0, 0)
	tt.SetTime(end.Add(time.Second))
	if n.reveal.pending || !n.reveal.started.IsZero() || list.ScrollAnimating() {
		t.Fatal("settlement observer removed before confirmation")
	}
}

func TestInspectorSmoothNoMovementUsesQuietFallbackAndCanBeInterrupted(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		tt, n, _, list, _ := inspectorRevealFixture(t)
		list.ScrollToOffset(100)
		tt.Frame()
		n.Highlight = "missing"
		n.reveal.pending = true
		n.reveal.waiting = true
		n.reveal.lastScroll = 100
		tt.Frame()
		base := time.Unix(7000, 0)
		tt.SetTime(base.Add(149 * time.Millisecond))
		if !n.reveal.started.IsZero() {
			t.Fatal("no-op finished before quiet")
		}
		if cancel {
			tt.Scroll(250, 340, 0, 0)
		}
		tt.SetTime(base.Add(150 * time.Millisecond))
		tt.SetTime(base.Add(151 * time.Millisecond))
		if cancel {
			if n.reveal.pending || !n.reveal.started.IsZero() {
				t.Fatal("quiet callback revived interrupted no-op")
			}
		} else if n.reveal.started.IsZero() {
			t.Fatal("no-op never settled")
		}
	}
}
