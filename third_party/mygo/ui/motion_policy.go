package ui

import "sync/atomic"

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
