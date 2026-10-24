package frame

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// captionButtonSize and the three right offsets are the native hit geometry:
// 32x32 at Top 12, measured from the right edge (close 8, maximize 40,
// minimize 72).
const (
	captionButtonSize = 32
	captionButtonTop  = 12
)

// captionCloseHover is the close button's hover fill: #e81123 with a white
// glyph, from both native caption stylesheets
// (reference/electron/src/renderer/window-controls.css:43-46 and
// reference/electron/src/main/harness-chrome-inject.js:119-122).
var captionCloseHover = ui.RGB(0xe8, 0x11, 0x23)

// captionWindowActions is the window a painted control trio acts on. The
// native window is absent in a headless tester, so the buttons take it as data.
type captionWindowActions struct {
	Close, Minimize, ToggleMaximize func()
	Maximized                       func() bool
}

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
	drawCaptionButtons(c, ink, captionWindowActions{
		Close:          win.Close,
		Minimize:       win.Minimize,
		ToggleMaximize: win.ToggleMaximize,
		Maximized:      win.IsMaximized,
	})
}

// drawCaptionButtons paints minimize, maximize and close over the window's top
// right corner. Normal state is unchanged: no fill, glyph in ink. On hover the
// close button fills #e81123 and switches to a white glyph, while minimize and
// maximize take the interactive hover background, each exactly as the native
// caption stylesheets do (window-controls.css:39-46).
func drawCaptionButtons(c *ui.Context, ink ui.Color, w captionWindowActions) {
	label := func(zh, en string) string {
		if strings.HasPrefix(i18n.Locale(), "zh") {
			return zh
		}
		return en
	}
	hoverFill := c.Theme().SurfaceHover
	control := func(key string, right float32, name string, fill, hoverInk ui.Color, paint func(*ui.Painter, ui.Rect, ui.Color), run func()) {
		button := ui.ButtonBase(c.Key(key)).Absolute().Top(captionButtonTop).Right(right).
			Size(captionButtonSize, captionButtonSize).
			Padding(0).BorderWidth(0).Radius(8).Background(ui.Transparent).
			Label(name).Tooltip(name)
		button.Draw(func(p *ui.Painter, r ui.Rect) {
			tone := ink
			if button.Hovered() {
				p.Fill(r, fill, 8)
				tone = hoverInk
			}
			paint(p, r, tone)
		})
		if button.Clicked() && run != nil {
			run()
		}
	}
	maximized := w.Maximized != nil && w.Maximized()
	// Order from the right edge: close (8), maximize (40), minimize (72).
	control("caption-close", 8, label("关闭", "Close"), captionCloseHover, ui.RGB(255, 255, 255),
		func(p *ui.Painter, r ui.Rect, tone ui.Color) {
			// The reference close glyph is an ✕. Painter.Line draws an
			// axis-aligned bar (paint.go:874), so the cross is a dotted diagonal.
			captionDiagonal(p, r.X+r.W/2-4, r.Y+r.H/2-4, r.X+r.W/2+4, r.Y+r.H/2+4, tone)
			captionDiagonal(p, r.X+r.W/2-4, r.Y+r.H/2+4, r.X+r.W/2+4, r.Y+r.H/2-4, tone)
		}, w.Close)
	control("caption-minimize", 72, label("最小化", "Minimize"), hoverFill, ink,
		func(p *ui.Painter, r ui.Rect, tone ui.Color) {
			p.Fill(ui.Rect{X: r.X + r.W/2 - 5, Y: r.Y + r.H/2, W: 10, H: 1}, tone, 0)
		}, w.Minimize)
	control("caption-maximize", 40, label("最大化", "Maximize"), hoverFill, ink,
		func(p *ui.Painter, r ui.Rect, tone ui.Color) {
			if maximized {
				// Restore: two offset corners read as a smaller window.
				p.Stroke(ui.Rect{X: r.X + r.W/2 - 5, Y: r.Y + r.H/2 - 3, W: 8, H: 8}, tone, 0, 1)
				p.Stroke(ui.Rect{X: r.X + r.W/2 - 3, Y: r.Y + r.H/2 - 6, W: 8, H: 8}, tone, 0, 1)
				return
			}
			p.Stroke(ui.Rect{X: r.X + r.W/2 - 5, Y: r.Y + r.H/2 - 5, W: 10, H: 10}, tone, 0, 1)
		}, w.ToggleMaximize)
}

// captionDiagonal paints a 1 DIP diagonal stroke as a run of dots, because the
// toolkit's Painter.Line only draws axis-aligned bars (third_party/mygo/ui
// paint.go:874). Nine dots from end to end stay connected at 1x.
func captionDiagonal(p *ui.Painter, x0, y0, x1, y1 float32, tone ui.Color) {
	const steps = 8
	for i := 0; i <= steps; i++ {
		t := float32(i) / steps
		p.Fill(ui.Rect{
			X: float32(int(x0 + (x1-x0)*t + .5)),
			Y: float32(int(y0 + (y1-y0)*t + .5)),
			W: 1, H: 1,
		}, tone, 0)
	}
}
