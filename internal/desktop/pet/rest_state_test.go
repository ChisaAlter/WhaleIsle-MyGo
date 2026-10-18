package pet

import (
	"errors"
	"testing"
	"time"
)

func TestRestIdleThresholdsAndContact(t *testing.T) {
	start := time.Unix(1700000000, 0)
	r := restState{}
	r.touch(start)
	for _, tc := range []struct {
		elapsed          time.Duration
		dozing, sleeping bool
	}{
		{225 * time.Second, false, false}, {225*time.Second + time.Millisecond, true, false},
		{240 * time.Second, true, false}, {240*time.Second + time.Millisecond, false, true},
	} {
		r.advance(start.Add(tc.elapsed), false)
		if r.dozing != tc.dozing || r.sleeping != tc.sleeping {
			t.Fatalf("at %s: %+v", tc.elapsed, r)
		}
	}
	// Controls reset inactivity but do not undo explicit or automatic sleep.
	r.touch(start.Add(300 * time.Second))
	if !r.sleeping {
		t.Fatal("control contact woke sleep")
	}
	if err := r.wakeAt(start.Add(300*time.Second), nil); err != nil {
		t.Fatal(err)
	}
	r.advance(start.Add(524*time.Second), false)
	if r.dozing || r.sleeping {
		t.Fatal("woke into overdue nap")
	}
	r.advance(start.Add(526*time.Second), false)
	if !r.dozing {
		t.Fatal("second idle period missing")
	}
	r.touch(start.Add(527 * time.Second))
	if r.dozing {
		t.Fatal("contact did not cancel dozing")
	}
}

func TestRestIdleBlockedReloadAndBackwardClock(t *testing.T) {
	start := time.Unix(1700000000, 0)
	r := restState{}
	r.advance(start, false)
	if r.sleeping || r.dozing {
		t.Fatal("fresh page slept immediately")
	}
	r.advance(start.Add(226*time.Second), false)
	r.advance(start.Add(time.Hour), true)
	if r.dozing || r.sleeping {
		t.Fatal("busy panel did not suppress automatic rest")
	}
	r.advance(start.Add(time.Hour+240*time.Second), false)
	if r.sleeping {
		t.Fatal("suppression replayed old idle period")
	}
	r.advance(start.Add(-time.Second), false)
	if r.dozing || r.sleeping {
		t.Fatal("backwards clock caused rest")
	}
	r.advance(start.Add(241*time.Second), false)
	if !r.sleeping {
		t.Fatal("rest did not recover after backwards clock")
	}
	r = restState{}
	r.advance(start.Add(24*time.Hour), false)
	if r.sleeping || r.dozing {
		t.Fatal("page recreation restored transient sleep")
	}
}

func TestRestWakeCommitAndFreeCare(t *testing.T) {
	start := time.Unix(1700000000, 0)
	r := restState{}
	r.sleep()
	commits := 0
	failed := errors.New("save failed")
	if err := r.wakeAt(start, func() error { commits++; return failed }); err != failed || !r.sleeping {
		t.Fatal("failed explicit wake changed sleep")
	}
	if err := r.wakeAt(start, func() error { commits++; return nil }); err != nil || r.sleeping {
		t.Fatal("explicit wake failed")
	}
	_ = r.wakeAt(start, nil)
	if commits != 2 {
		t.Fatal("duplicate/free wake committed care")
	}
	r.sleep()
	_ = r.wakeAt(start, nil)
	r.advance(start.Add(240*time.Second), false)
	if r.sleeping {
		t.Fatal("successful meal/contact did not reset idle")
	}
}

func TestRestClockCancellation(t *testing.T) {
	var c restClock
	canceled := 0
	c.cancel = func() { canceled++ }
	epoch := c.generation
	c.stop()
	if c.generation == epoch || canceled != 1 || c.cancel != nil {
		t.Fatal("stop did not invalidate queued tick")
	}
	c.stop()
	if canceled != 1 {
		t.Fatal("cancel called twice")
	}
}
