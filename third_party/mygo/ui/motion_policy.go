package ui

import (
	"sync/atomic"
	"time"
)

// reduceMotionForced is the application's own request for less motion. The
// desktop's setting asks for it too (Preferences.ReduceMotion); this one is the
// product's, for a mode of its own that turns interface transitions off.
var reduceMotionForced atomic.Bool

// SetReduceMotion makes the application behave as if the desktop asked for less
// motion, on top of the desktop's own setting: Animate goes to its target at
// once, transitions, router, menu, toast and list motion are skipped, and
// decorative effects stay still. A product uses it for a policy of its own, such
// as a light appearance mode that turns interface transitions off.
//
// Pass false to hand the decision back to the desktop's setting. A window
// repaints the change after a call to Window.Invalidate.
func SetReduceMotion(on bool) { reduceMotionForced.Store(on) }

// DialogExit is how a closing DialogBase overlay leaves: the whole overlay,
// its backdrop and panel as the last frame showed them, fades out over
// Duration with Ease (EaseOut when nil), while taking neither the pointer
// nor the focus. The zero value, the default, removes it at once.
type DialogExit struct {
	Duration time.Duration
	Ease     Easing
}

// dialogExitCfg is the application's SetDialogExit choice; unset, the zero
// DialogExit, which keeps a closing dialog's immediate removal.
var dialogExitCfg atomic.Value // DialogExit

// SetDialogExit is how the application's dialogs leave: a Duration over 0
// keeps a closing DialogBase overlay as a fading copy of the frame before,
// and 0 keeps the immediate removal it had. A dialog's own DialogOptions
// .NoExit keeps it immediate no matter this setting, as does less motion
// (the desktop's ReduceMotion or SetReduceMotion).
func SetDialogExit(e DialogExit) { dialogExitCfg.Store(e) }

// dialogExit is the application's SetDialogExit choice, the zero value
// while none was made.
func dialogExit() DialogExit {
	e, _ := dialogExitCfg.Load().(DialogExit)
	return e
}
