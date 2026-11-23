// Package coss draws WhaleIsle's native controls in the coss ui design
// system (docs/design/coss, MIT): the same sizes, paddings, radii, borders,
// per-state colors, inset highlights, focus rings and motion as the coss
// React components, on the vendored MyGo toolkit's bases (ButtonBase,
// TextInputBase, SelectBase, PopoverBase, DialogBase, …), which keep the
// pointer, keyboard, focus and accessibility behavior.
//
// Each component is a function taking *ui.Context, its bound value where it
// has one, and a typed options struct whose zero value is coss's default
// variant and size. Colors come from design.For(c): the window's scheme and
// the Host accent as primary. Desktop metrics are coss's ≥640px (sm:)
// values; the 375px/coarse-pointer variants do not apply to a desktop app.
//
// View code here never does I/O, starts goroutines or timers: animation
// runs on Element.Animate/Loop and the toolkit's transitions. There is no
// toast: the app gives feedback inline (docs/error-copy.md), with Alert and
// Field errors.
package coss
