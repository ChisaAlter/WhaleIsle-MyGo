package chrome

import (
	"sync"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/ui/coss"
)

const (
	ambientIdleFreezeDelay = 10 * time.Second
	// ambientFrameInterval paces GPU ambient gradient animation to ~24 fps
	// instead of 156+ Hz AnimationFrame() spinning, preventing continuous
	// full-screen backdrop cache invalidation during idle and interaction.
	ambientFrameInterval = 42 * time.Millisecond
)

var backdropPowerState struct {
	mu                sync.RWMutex
	frozen            bool
	lowPower          bool
	overlayFrozen     bool
	overlayBuildGen   uint64
	overlayBuiltGen   uint64
	overlayBuilt      bool
	overlayPainted    bool
	lastActivity      time.Time
	windowActiveCheck func() bool
}

func init() {
	coss.OnGlassOverlayActive = func(painted bool) {
		if painted {
			NoteBackdropOverlayPainted()
			return
		}
		NoteBackdropOverlayActive()
	}
}

// SetBackdropFrozen manually freezes or resumes dynamic backdrop and sheen
// animations across the UI layer.
func SetBackdropFrozen(frozen bool) {
	backdropPowerState.mu.Lock()
	backdropPowerState.frozen = frozen
	backdropPowerState.mu.Unlock()
}

// SetBackdropOverlayActive pauses or resumes ambient backdrop time progression
// while a modal or floating overlay is open so backdrop blur caches stay warm.
func SetBackdropOverlayActive(active bool) {
	backdropPowerState.mu.Lock()
	backdropPowerState.overlayFrozen = active
	backdropPowerState.mu.Unlock()
}

// NoteBackdropOverlayActive records that a frosted glass popup, menu or modal
// was built in the current frame so the ambient gradient holds its phase and
// backdrop blur cache hit rates remain near 100% during overlay interaction.
func NoteBackdropOverlayActive() {
	backdropPowerState.mu.Lock()
	backdropPowerState.overlayBuilt = true
	if backdropPowerState.overlayBuildGen > 0 {
		backdropPowerState.overlayBuiltGen = backdropPowerState.overlayBuildGen
	}
	backdropPowerState.mu.Unlock()
}

// NoteBackdropOverlayPainted records that a frosted glass overlay painted in
// the current paint pass so paint-only hover/scroll frames keep the ambient
// backdrop frozen until the overlay closes.
func NoteBackdropOverlayPainted() {
	backdropPowerState.mu.Lock()
	backdropPowerState.overlayPainted = true
	backdropPowerState.mu.Unlock()
}

func beginAmbientBackdropBuild() uint64 {
	backdropPowerState.mu.Lock()
	backdropPowerState.overlayBuildGen++
	gen := backdropPowerState.overlayBuildGen
	backdropPowerState.mu.Unlock()
	return gen
}

func consumeBackdropOverlayState(buildGen uint64) (active, needsRecheck bool) {
	backdropPowerState.mu.Lock()
	explicit := backdropPowerState.overlayFrozen
	built := backdropPowerState.overlayBuilt
	builtGen := backdropPowerState.overlayBuiltGen
	curBuildGen := backdropPowerState.overlayBuildGen
	painted := backdropPowerState.overlayPainted
	backdropPowerState.overlayBuilt = false
	backdropPowerState.overlayPainted = false
	backdropPowerState.mu.Unlock()

	if explicit {
		return true, false
	}
	if built || (buildGen > 0 && builtGen == buildGen) {
		return true, false
	}
	if !painted {
		return false, false
	}
	// If a fresh view build ran after the last built overlay and did not
	// rebuild any overlay, the overlay has closed on this frame.
	if buildGen > 0 && curBuildGen == buildGen && builtGen > 0 && builtGen < buildGen {
		return false, false
	}
	// An overlay painted without a tracked build helper (for example ui.Menu)
	// or a painter closure was reused across frames: hold this frame frozen
	// and schedule one lightweight recheck so closing resumes smoothly.
	return true, true
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
	backdropPowerState.overlayFrozen = false
	backdropPowerState.overlayBuildGen = 0
	backdropPowerState.overlayBuiltGen = 0
	backdropPowerState.overlayBuilt = false
	backdropPowerState.overlayPainted = false
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
		// View builds recreate painter closures. Keep the initial idle epoch
		// with the shared activity owner rather than restarting on every build.
		*startedAt = backdropActivityAt()
		if startedAt.IsZero() {
			*startedAt = now
			NoteBackdropActivity(now)
		}
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
