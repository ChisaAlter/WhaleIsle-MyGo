package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/ui/chrome"
)

func TestPluginRootRetainsNativeCaptionAndClientBody(t *testing.T) {
	for _, width := range []int{1200, 400} {
		var body ui.Rect
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				PluginRootCaption(c, chrome.DefaultTitleBarColors())
				ui.Box(c).Grow(1).MinHeight(0).FillWidth().Draw(func(_ *ui.Painter, r ui.Rect) { body = r })
			})
		}, width, 800)
		r, ok := tt.Find("Plugin root native caption")
		if !ok || r.H != 48 || r.Y != 0 || r.W != float32(width) || body.Y != 48 || body.H != 752 || body.W != float32(width) {
			t.Fatalf("width=%v native caption=%+v body=%+v", width, r, body)
		}
	}
}
