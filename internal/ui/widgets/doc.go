// Package widgets holds the design primitives that WhaleIsle's surfaces share,
// so that collapsing rows, status marks, settings and confirmations look and
// behave the same everywhere instead of each surface growing its own text
// button and ASCII arrow.
//
// The primitives own their look, their metrics, their motion and their
// keyboard behavior; they own no product words. Every string they show comes
// from the caller, already localized, and the same goes for anything they
// announce to assistive technology.
//
// # Rows that open and shut
//
// DisclosureRow is the line a tool call, a reasoning block or a turn's process
// collapses into: a leading glyph, a title, a 2 DIP separator dot, a summary
// cut with an ellipsis to the room left, and a chevron that turns as the row
// opens. The whole 24 DIP line is the target, by pointer or by Enter and
// Space, and it takes a quiet fill while hovered:
//
//	open := app.open[id]
//	widgets.DisclosureRow(c, widgets.DisclosureRowOptions{
//		Icon:    icons.Terminal,
//		Title:   "bash",
//		Summary: command,
//		Open:    &open,
//		Running: running,
//		Body:    func() { terminal(c, output) },
//		OnToggle: func(open bool) { app.open[id] = open },
//	})
//
// # Status
//
// StateDot shows idle, running, done, warning or error in a 10 DIP slot, its
// dot 6 DIPs wide, breathing while running, and StateColor gives the matching
// color for the words beside it:
//
//	ui.Text(c, status).TextColor(widgets.StateColor(c.Theme(), state))
//	widgets.StateDot(c, state)
//
// TextShimmer sweeps a highlight across a label while work goes on, and
// stands still, without asking for frames, when it does not.
//
// # Settings
//
// A setting is one row: SettingRow puts the title at 14/22 over its
// description at 12/18 on the left and the control on the right, with 16 DIPs
// of room above and below and a hairline under it that the last row of a group
// drops. SwitchRow is that row with a Switch — a 36×20 track, 2 DIPs of
// padding and a 16 DIP thumb — for what is on or off:
//
//	widgets.SwitchRow(c, widgets.SwitchRowOptions{
//		Title:       labels.CloseToTray,
//		Description: labels.CloseToTrayHint,
//		On:          &app.closeToTray,
//		Last:        true,
//	})
//
// # Confirmations
//
// ConfirmDialog asks inside the window, in place of the system's message box,
// and reports the answer in the frame it is given:
//
//	if widgets.ConfirmDialog(c, widgets.ConfirmDialogOptions{
//		Open:         &app.confirm,
//		Title:        title,
//		Message:      body,
//		ConfirmLabel: confirmLabel,
//		CancelLabel:  cancelLabel,
//		Danger:       true,
//	}) {
//		app.doIt()
//	}
//
// # Motion
//
// The motion tokens — OverlayMotion, PopoverMotion, SwapMotion, FlipMotion —
// keep durations and curves in one place, and Motion.Animate moves a value
// along one of them, jumping straight there when the desktop asks for less
// motion. Nothing here animates past a reduce-motion setting.
package widgets
