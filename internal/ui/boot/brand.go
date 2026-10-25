package boot

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// The reference brand block (boot.html:22, boot.css:222-268) is a serif
// wordmark with a highlight crossing it every 6s, the island's own two-character
// mark under it, and the harness line in tracked capitals. The page it replaces
// drew a 24 DIP sans title over the mascot bitmap.

const (
	// brandSize and brandLine are .brand's 44/54 DIP; a short window keeps
	// 34/42 (boot.css:222-225,453-459).
	brandSize      = float32(44)
	brandLine      = float32(54)
	brandShortSize = float32(34)
	brandShortLine = float32(42)
	// brandTracking is .brand's letter-spacing, 0.08em.
	brandTracking = .08
	// brandSheenPeriod is brand-sheen's 6s cycle: the highlight crosses between
	// 22% and 55% of it and the letters rest for the rest of the cycle
	// (boot.css:510-520).
	brandSheenPeriod = 6 * time.Second
	// brandMarkSize, brandMarkTracking, brandSubSize and brandSubTracking are
	// .tag-mark's 10.5px/0.12em and .brand-sub's 8px/0.32em (boot.css:255-268).
	brandMarkSize     = float32(10.5)
	brandMarkTracking = .12
	brandSubSize      = float32(8)
	brandSubTracking  = .32
)

// brandFont is the reference brand's serif stack: Didot on macOS, Bodoni MT and
// Playfair Display where they exist, then Georgia and the generic serif.
const brandFont = "Didot, Bodoni MT, Playfair Display, Georgia, Times New Roman, serif"

// brandShine is the sheen's color, --boot-shine: white in both schemes.
var brandShine = ui.RGBA(255, 255, 255, 1)

// brandMark is the island's own name, the tag the reference prints under the
// wordmark; it reads the same in every locale.
var brandMark = "鲸屿"

// bootBrand paints .brand: the wordmark with its sheen, the mark and the harness
// line, centred over the sky.
func bootBrand(c *ui.Context, short bool, fg, muted ui.Color, dark bool) {
	size, line := brandSize, brandLine
	if short {
		size, line = brandShortSize, brandShortLine
	}
	// --boot-brand-base is the ink in light and the same ink at 68% in dark
	// (boot-tokens.css:36,80).
	base := fg
	if dark {
		base = fg.Alpha(.68)
	}
	ui.Column(c).Key("boot-brand").AlignItems(ui.Center).Children(func() {
		wordmark(c, "Whale Isle", size, line, base)
		// .tag-mark, then .brand-sub in the code face's tracked capitals.
		ui.Text(c, brandMark).FontSize(theme.InterfaceSize(brandMarkSize)).
			LetterSpacing(theme.InterfaceSize(brandMarkSize)*brandMarkTracking).
			TextColor(fg).Margin(11, 0, 0, 0)
		ui.Text(c, "BASED ON DEEPSEEK HARNESS").Font(theme.RenderPreferences().Code.Family).
			FontSize(theme.InterfaceSize(brandSubSize)).
			LetterSpacing(theme.InterfaceSize(brandSubSize)*brandSubTracking).
			TextColor(muted).Margin(6, 0, 0, 0)
	})
}

// wordmark paints the serif wordmark with its sheen crossing it: a bright copy of
// the letters clipped to a moving band, since the painter fills text with one
// color. It reserves the .brand line box, so the status line below sits still.
func wordmark(c *ui.Context, text string, size, line float32, base ui.Color) {
	span := ui.Span{Text: text, Font: brandFont, Size: size, Weight: 400, Color: base, LetterSpacing: size * brandTracking}
	w, h := c.MeasureText(0, span)
	if h <= 0 {
		h = line
	}
	if w <= 0 {
		// A face the system does not have still reserves room for the word.
		w = size * 5.5
	}
	mark := ui.Box(c).Key("boot-wordmark").Width(w).Height(h).Shrink(0)
	mark.Label(text)
	sweep := float32(-1)
	if !c.Preferences().ReduceMotion {
		// Outside its window brand-sheen keeps the highlight off the letters.
		if phase := mark.Loop("boot-sheen", brandSheenPeriod, ui.Linear); phase >= .22 && phase <= .55 {
			sweep = (phase - .22) / .33
		}
	}
	shine := span
	shine.Color = brandShine
	mark.Draw(func(p *ui.Painter, r ui.Rect) {
		p.RichText(r.X, r.Y, 0, span)
		if sweep < 0 {
			return
		}
		band := max(r.W*.45, 28)
		x := r.X + r.W + band - (r.W+2*band)*sweep
		p.Clip(ui.Rect{X: x, Y: r.Y, W: band, H: r.H}, 0, func() {
			p.RichText(r.X, r.Y, 0, shine)
		})
	})
}

// statusDots is the three-dot "starting" suffix .status shows while the Host
// comes up (boot.css:322-341): 4 DIP dots blinking on a 1.3s cycle, 0.2s apart.
// It replaces the spinner the page used to turn at the head of the status line.
func statusDots(c *ui.Context, color ui.Color) {
	row := ui.Row(c).Key("boot-status-dots").Gap(4).AlignItems(ui.Center).Shrink(0).Role(ui.RoleNone)
	now := c.Now()
	reduced := c.Preferences().ReduceMotion
	row.Children(func() {
		for i := 0; i < 3; i++ {
			dot := ui.Box(c).Size(4, 4).Radius(2).Shrink(0)
			alpha := float32(.22)
			if !reduced {
				// The context's frame request rebuilds the view, so the phase
				// alone decides each dot's opacity.
				c.AnimationFrame()
				phase := float32(now.UnixMilli()%1300)/1300 + float32(i)*.2/1.3
				alpha = dotBlink(phase - float32(int(phase)))
			}
			dot.Background(color.Alpha(alpha))
		}
	})
}

// dotBlink is dot-blink (boot.css:530-534): opacity 0.22 at 0%, 70% and 100%, 1
// at 30%.
func dotBlink(t float32) float32 {
	switch {
	case t < .3:
		return .22 + .78*(t/.3)
	case t < .7:
		return 1 - .78*((t-.3)/.4)
	}
	return .22
}
