package chrome

import (
	"sync"
	"time"

	"github.com/egoist/mygo"
)

const ambientIdleFreezeDelay = 10 * time.Second

var backdropPowerState struct {
	mu                sync.RWMutex
	frozen            bool
	lowPower          bool
	lastActivity      time.Time
	windowActiveCheck func() bool
}

// SetBackdropFrozen manually freezes or resumes dynamic backdrop and sheen
// animations across the UI layer.
func SetBackdropFrozen(frozen bool) {
	backdropPowerState.mu.Lock()
	backdropPowerState.frozen = frozen
	backdropPowerState.mu.Unlock()
}

// SetLowPowerMode freezes or resumes high-frequency backdrop and sheen
// animations when low-power mode is active.
func SetLowPowerMode(enabled bool) {
	backdropPowerState.mu.Lock()
	backdropPowerState.lowPower = enabled
	backdropPowerState.mu.Unlock()
}

// NoteBackdropActivity records active UI/pointer interaction at now so ambient
// gradient and sheen animations remain live until ambientIdleFreezeDelay elapses.
func NoteBackdropActivity(now time.Time) {
	if now.IsZero() {
		return
	}
	backdropPowerState.mu.Lock()
	last := backdropPowerState.lastActivity
	if last.IsZero() || now.After(last) || last.Sub(now) > time.Hour {
		backdropPowerState.lastActivity = now
	}
	backdropPowerState.mu.Unlock()
}

func backdropActivityAt() time.Time {
	backdropPowerState.mu.RLock()
	t := backdropPowerState.lastActivity
	backdropPowerState.mu.RUnlock()
	return t
}

func setBackdropWindowActiveHook(fn func() bool) {
	backdropPowerState.mu.Lock()
	backdropPowerState.windowActiveCheck = fn
	backdropPowerState.mu.Unlock()
}

func resetBackdropPowerForTest() {
	backdropPowerState.mu.Lock()
	backdropPowerState.frozen = false
	backdropPowerState.lowPower = false
	backdropPowerState.lastActivity = time.Time{}
	backdropPowerState.windowActiveCheck = nil
	backdropPowerState.mu.Unlock()
}

func backdropWindowActive() bool {
	backdropPowerState.mu.RLock()
	frozen := backdropPowerState.frozen
	lowPower := backdropPowerState.lowPower
	hook := backdropPowerState.windowActiveCheck
	backdropPowerState.mu.RUnlock()
	if frozen || lowPower {
		return false
	}
	if hook != nil {
		return hook()
	}
	if !mygo.App.IsReady() {
		// Headless ui.Tester runs without an OS window loop; stay active unless frozen.
		return true
	}
	win := mygo.FocusedWindow()
	if win == nil || win.IsDestroyed() {
		return false
	}
	return win.IsVisible() && !win.IsMinimized() && win.IsFocused()
}

func backdropIdleSince(start, now time.Time) bool {
	if start.IsZero() || now.IsZero() {
		return false
	}
	return now.Sub(start) >= ambientIdleFreezeDelay
}

func shouldFreezeAmbientAnimation(reduced bool, startedAt *time.Time, now time.Time) bool {
	if reduced {
		return false
	}
	if startedAt.IsZero() {
		*startedAt = now
	}
	if act := backdropActivityAt(); act.After(*startedAt) && !now.Before(act) {
		*startedAt = act
	}
	return !backdropWindowActive() || backdropIdleSince(*startedAt, now)
}

func shouldFreezeSheenAnimation(now time.Time) bool {
	if !backdropWindowActive() {
		return true
	}
	if act := backdropActivityAt(); !act.IsZero() && !now.Before(act) && backdropIdleSince(act, now) {
		return true
	}
	return false
}
