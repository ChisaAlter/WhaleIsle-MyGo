package widgets

import "github.com/egoist/mygo/ui"

// Notice reports the outcome of an action (a saved/copied acknowledgement, a failure) as a
// toast. A line that is inserted into the layout when the outcome arrives, and pulled out
// again when it clears, moves everything below it; the layout stays put and the toast
// carries the news. text is only evaluated while show holds, so it may dereference what
// show guards. tone is "error", "success" or "".
func Notice(c *ui.Context, id, tone string, show bool, text func() string) {
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
		return
	}
	if *shown != message {
		c.AddToast(ui.Toast{ID: toast, Title: message, Type: tone})
		*shown = message
	}
}
