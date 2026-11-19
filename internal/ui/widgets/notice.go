package widgets

import "github.com/egoist/mygo/ui"

// Notice reports the outcome of an action that succeeded — a saved or copied
// acknowledgement — as a toast, so a line inserted into the layout and pulled
// out again does not move everything below it. text is only evaluated while show
// holds, so it may dereference what show guards.
//
// Failures do not come through here: an error is never a toast, so a failed
// action draws widgets.Alert where it happened.
func Notice(c *ui.Context, id string, show bool, text func() string) {
	anchor := ui.Box(c.Key("notice/"+id)).Absolute().Size(0, 0).PassThrough()
	shown := ui.Local(anchor, "text", func() string { return "" })
	toast := "notice-" + id
	if !show {
		if *shown != "" {
			c.CloseToast(toast)
			*shown = ""
		}
		return
	}
	message := text()
	if message == "" {
		if *shown != "" {
			c.CloseToast(toast)
			*shown = ""
		}
		return
	}
	if *shown != message {
		c.AddToast(ui.Toast{ID: toast, Title: message, Type: "success"})
		*shown = message
	}
}
