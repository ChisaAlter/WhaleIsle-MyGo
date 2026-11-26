package chat

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestPlanLeavingOnSendTimingAndRecovery(t *testing.T) {
	t0 := time.Unix(1000, 0)
	var tr PlanLeavingTracker

	// Idle without pending stays visible.
	if tr.Update(t0, false, "p1") {
		t.Fatal("idle plan should not be leaving")
	}
	// Pending submission triggers immediate exit.
	if !tr.Update(t0, true, "p1") {
		t.Fatal("pending submission should immediately hide plan dock")
	}
	// Stays hidden before 8s safety ceiling while pending.
	if !tr.Update(t0.Add(7*time.Second), true, "p1") {
		t.Fatal("plan should stay leaving before 8s max")
	}
	// Pending clears -> 1s grace period starts.
	if !tr.Update(t0.Add(2*time.Second), false, "p1") {
		t.Fatal("plan should remain leaving during 1s grace after pending clears")
	}
	if tr.Update(t0.Add(3*time.Second), false, "p1") {
		t.Fatal("plan should restore after 1s grace expires")
	}

	// 8s max safety timeout restores even if pending never cleared.
	tr = PlanLeavingTracker{}
	if !tr.Update(t0, true, "p1") {
		t.Fatal("expected leaving")
	}
	if tr.Update(t0.Add(8*time.Second), true, "p1") {
		t.Fatal("8s max should release leaving state")
	}

	// Fresh plan signature clears leaving state early.
	tr = PlanLeavingTracker{}
	tr.Update(t0, false, "p1")
	tr.Update(t0.Add(10*time.Millisecond), true, "p1")
	if tr.Update(t0.Add(20*time.Millisecond), false, "p2") {
		t.Fatal("new plan arrival should clear leaving immediately")
	}
}

func TestSystemRowSettle400msVsUserImmediate(t *testing.T) {
	if SystemRowSettleDuration != 400*time.Millisecond {
		t.Fatalf("SystemRowSettleDuration = %v, want 400ms", SystemRowSettleDuration)
	}
	t0 := time.Unix(1000, 0)
	var tr SystemRowSettleTracker
	hasRows, settling := tr.Update(t0, 2, false, false)
	if !hasRows || settling {
		t.Fatalf("with rows: got (%v, %v), want (true, false)", hasRows, settling)
	}

	// System drain to 0 keeps dock height reserved for 400ms.
	hasRows, settling = tr.Update(t0.Add(10*time.Millisecond), 0, false, false)
	if !hasRows || !settling {
		t.Fatalf("right after system drain: got (%v, %v), want (true, true)", hasRows, settling)
	}
	hasRows, settling = tr.Update(t0.Add(250*time.Millisecond), 0, false, false)
	if !hasRows || !settling {
		t.Fatalf("before 400ms settle: got (%v, %v), want (true, true)", hasRows, settling)
	}
	hasRows, settling = tr.Update(t0.Add(410*time.Millisecond), 0, false, false)
	if hasRows || settling {
		t.Fatalf("after 400ms settle: got (%v, %v), want (false, false)", hasRows, settling)
	}

	// User-triggered removal settles immediately.
	tr.Update(t0, 1, false, false)
	hasRows, settling = tr.Update(t0.Add(10*time.Millisecond), 0, true, false)
	if hasRows || settling {
		t.Fatalf("user removal should settle immediately, got (%v, %v)", hasRows, settling)
	}
}

func TestDockSlideProgressTimingAndImmediateClose(t *testing.T) {
	if DockSlideDuration != 200*time.Millisecond {
		t.Fatalf("DockSlideDuration = %v, want 200ms", DockSlideDuration)
	}
	open := false
	immediateClose := false
	var got float32
	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		host := ui.Box(c).Key("dock").Fill()
		got = DockSlideProgress(c, host, open, immediateClose)
	}, 100, 100)
	tt.SetTime(start)
	if got != 0 {
		t.Fatalf("initial closed progress = %v, want 0", got)
	}
	open = true
	tt.SetTime(start.Add(time.Millisecond))
	tt.SetTime(start.Add(100 * time.Millisecond))
	if got <= 0 || got >= 1 {
		t.Fatalf("mid slide-in = %v, want (0,1)", got)
	}
	tt.SetTime(start.Add(DockSlideDuration + 2*time.Millisecond))
	if got != 1 {
		t.Fatalf("settled open = %v, want 1", got)
	}
	// Immediate close (TaskDock on send / closed track) snaps to 0.
	open = false
	immediateClose = true
	tt.SetTime(start.Add(220 * time.Millisecond))
	if got != 0 {
		t.Fatalf("immediate close = %v, want 0", got)
	}
}
