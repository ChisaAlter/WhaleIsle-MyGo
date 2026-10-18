package frame

import "github.com/egoist/mygo/ui"

// MainCaptionFileDrop owns only the application caption's native UI ancestry.
// It never registers a window fallback: resource/child/WebView blank areas do
// not thereby become destinations for the parent conversation's attachments.
func MainCaptionFileDrop(c *ui.Context, m Model, content func()) {
	area := ui.Column(c.Key("main-caption-files")).FillWidth()
	area.Children(func() {
		if navigationFileDropAllowed(m) {
			conversationFileDrop(c, m, area, m.Store.Snapshot().SelectedID)
		}
		content()
	})
}
