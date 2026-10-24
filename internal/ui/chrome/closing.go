package chrome

import (
	"math"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// Closing layer metrics, taken from the native closing overlay's CSS
// (reference/electron/src/main/closing-overlay.js:4-56): a full-window cover
// with a centered 24 DIP card.
const (
	ClosingCardMinWidth = 240
	ClosingCardPadding  = 24
	ClosingCardRadius   = 24
	ClosingCardGap      = 8
	ClosingSpinnerSize  = 36
	ClosingSpinnerRing  = 2
	ClosingTitleSize    = 16
	ClosingTitleLine    = 24
	// ClosingTitleGap is the title's `margin: 8px 0 0`, which stacks on the
	// card's 8 DIP gap, exactly as the reference's flex column does.
	ClosingTitleGap   = 8
	ClosingDetailSize = 12
	ClosingDetailLine = 18
)

// ClosingOverlay is the shell's closing layer: while the app stops its local
// Harness service the window shows this cover instead of the last page, as the
// native shell does before harness.shutdown
// (reference/electron/src/main/index.js:1434-1437).
type ClosingOverlay struct {
	Title  string
	Detail string
	Colors theme.Colors
	// Observe receives the painted card and spinner boxes. The headless
	// geometry test measures the reference metrics through it; production
	// callers leave it nil.
	Observe func(card, spinner ui.Rect)
}

// DrawClosingOverlay paints the closing layer above everything else in the
// window. Call it last in the window's view so it covers chat, dialogs and
// settings. The layer captures input for as long as the quit runs.
func DrawClosingOverlay(c *ui.Context, m ClosingOverlay) {
	if m.Title == "" && m.Detail == "" {
		return
	}
	colors := m.Colors
	if colors.BG == "" {
		colors = theme.Default().Light
	}
	background, field := Color(colors.BG), Color(colors.Field)
	line, accent, ink := Color(colors.Border), Color(colors.Accent), Color(colors.FG)
	muted := Color(colors.Muted)
	if colors.Muted == "" {
		muted = ink
	}
	ui.Overlay(c, func() {
		w, h := c.Size()
		layer := ui.Column(c.Key("closing-overlay")).Absolute().Left(0).Top(0).Width(w).Height(h).
			Modal().AlignItems(ui.Center).Justify(ui.Center).
			Draw(func(p *ui.Painter, r ui.Rect) { p.Fill(r, background, 0) })
		layer.Children(func() {
			// The reference card: min-width 240, 24 padding, 1px border, 24
			// radius, 8 gap, and the two-layer 0 0 1px / 0 12px 32px shadow.
			card := ui.Column(c.Key("closing-overlay-card")).
				MinWidth(ClosingCardMinWidth).Padding(ClosingCardPadding).Gap(ClosingCardGap).
				AlignItems(ui.Center).Background(field).Radius(ClosingCardRadius).
				Shadow(0, 0, 1, 0, ui.RGBA(0, 0, 0, .2)).
				Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .08))
			card.Children(func() {
				ui.Box(c).Width(ClosingSpinnerSize).Height(ClosingSpinnerSize).
					Draw(func(p *ui.Painter, r ui.Rect) { paintClosingSpinner(p, r, line, accent) })
				ui.Text(c, m.Title).FontSize(theme.InterfaceSize(ClosingTitleSize)).FontWeight(500).
					FixedLineHeight(theme.InterfaceSize(ClosingTitleLine)).TextColor(ink).
					Margin(ClosingTitleGap, 0, 0, 0)
				ui.Text(c, m.Detail).FontSize(theme.InterfaceSize(ClosingDetailSize)).
					FixedLineHeight(theme.InterfaceSize(ClosingDetailLine)).TextColor(muted)
			})
			card.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if line.A != 0 && r.W > 0 && r.H > 0 {
					p.Stroke(r, line, ClosingCardRadius, 1)
				}
				if m.Observe != nil {
					m.Observe(r, ui.Rect{
						X: r.X + (r.W-ClosingSpinnerSize)/2,
						Y: r.Y + ClosingCardPadding,
						W: ClosingSpinnerSize,
						H: ClosingSpinnerSize,
					})
				}
			})
		})
	})
}

// paintClosingSpinner draws the reference spinner: a 36 DIP ring with a 2 DIP
// border whose top arc alone carries the accent, as
// `border-top-color` does in closing-overlay.js:33-40. The CSS 0.85s rotation
// is not reproduced: the layer only lives for the bounded quit paint, so an
// animated frame would never reach the screen before the process stops.
func paintClosingSpinner(p *ui.Painter, r ui.Rect, ring, accent ui.Color) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	p.Stroke(r, ring, r.W/2, ClosingSpinnerRing)
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	radius := r.W/2 - ClosingSpinnerRing/2
	// Screen coordinates put the top at 270掳; `border-top-color` colors the
	// quarter arc between the top two corners, 225掳..315掳.
	const segments = 12
	for i := 0; i < segments; i++ {
		a0 := math.Pi * (1.25 + 0.5*float64(i)/segments)
		a1 := math.Pi * (1.25 + 0.5*float64(i+1)/segments)
		p.Line(
			cx+radius*float32(math.Cos(a0)), cy+radius*float32(math.Sin(a0)),
			cx+radius*float32(math.Cos(a1)), cy+radius*float32(math.Sin(a1)),
			ClosingSpinnerRing, accent,
		)
	}
}
