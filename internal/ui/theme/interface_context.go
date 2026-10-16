package theme

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// ApplyInterface changes only a window's root typography, preserving its palette.
func ApplyInterface(c *ui.Context) {
	font := CurrentPreferences().Interface
	next := *c.Theme()
	next.Font, next.FontSize = font.Family, font.Size
	c.SetTheme(&next)
}

// RefreshWindows applies committed or preview typography to already-open windows.
func RefreshWindows() {
	for _, window := range mygo.Windows() {
		window.Invalidate()
	}
}
