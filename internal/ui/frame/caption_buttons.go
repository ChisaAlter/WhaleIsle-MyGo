package frame

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// captionWindowButtons paints the window controls the native caption overlay
// leaves out. mygo draws only the close button unless the window carries
// WS_MINIMIZEBOX and WS_MAXIMIZEBOX (internal/windows captionBar.shownButtons),
// and this window is created without them, so the real window showed a lone ✕
// at its right edge. Keeping the native close and adding the missing two to its
// left yields the reference's order: terminal, right panel, minimize, maximize,
// close.
//
// It draws nothing when the overlay already reserves a three-button strip, so a
// future window style cannot end up with doubled controls.
func captionWindowButtons(c *ui.Context, ink ui.Color) {
	if runtime.GOOS != "windows" {
		// macOS and Linux keep their own window controls.
		return
	}
	if c.TitleBar().Right > 130 {
		return
	}
	win := mygo.FocusedWindow()
	if win == nil {
		if windows := mygo.Windows(); len(windows) > 0 {
			win = windows[0]
		}
	}
	if win == nil {
		return
	}
	label := func(zh, en string) string {
		if strings.HasPrefix(i18n.Locale(), "zh") {
			return zh
		}
		return en
	}
	control := func(key string, right float32, name string, paint func(*ui.Painter, ui.Rect), run func()) {
		button := ui.ButtonBase(c.Key(key)).Absolute().Top(12).Right(right).Size(32, 32).
			Padding(0).BorderWidth(0).Radius(8).Background(ui.Transparent).
			Label(name).Tooltip(name)
		button.Draw(paint)
		if button.Clicked() && run != nil {
			run()
		}
	}
	// Order from the right edge: close (8), maximize (40), minimize (72).
	control("caption-close", 8, label("关闭", "Close"), func(p *ui.Painter, r ui.Rect) {
		p.Line(r.X+r.W/2-4, r.Y+r.H/2-4, r.X+r.W/2+4, r.Y+r.H/2+4, 1, ink)
		p.Line(r.X+r.W/2-4, r.Y+r.H/2+4, r.X+r.W/2+4, r.Y+r.H/2-4, 1, ink)
	}, win.Close)
	control("caption-minimize", 72, label("最小化", "Minimize"), func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X + r.W/2 - 5, Y: r.Y + r.H/2, W: 10, H: 1}, ink, 0)
	}, win.Minimize)
	control("caption-maximize", 40, label("最大化", "Maximize"), func(p *ui.Painter, r ui.Rect) {
		if win.IsMaximized() {
			// Restore: two offset corners read as a smaller window.
			p.Stroke(ui.Rect{X: r.X + r.W/2 - 5, Y: r.Y + r.H/2 - 3, W: 8, H: 8}, ink, 0, 1)
			p.Stroke(ui.Rect{X: r.X + r.W/2 - 3, Y: r.Y + r.H/2 - 6, W: 8, H: 8}, ink, 0, 1)
			return
		}
		p.Stroke(ui.Rect{X: r.X + r.W/2 - 5, Y: r.Y + r.H/2 - 5, W: 10, H: 10}, ink, 0, 1)
	}, win.ToggleMaximize)
}
