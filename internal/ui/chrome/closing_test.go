package chrome

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// closingModel builds the layer for one locale without touching the global
// catalog, as the quit path does with i18n.T.
func closingModel(locale string, colors theme.Colors) ClosingOverlay {
	return ClosingOverlay{
		Title:  i18n.Lookup(locale, i18n.KeyClosingTitle),
		Detail: i18n.Lookup(locale, i18n.KeyClosingDetail),
		Colors: colors,
	}
}

// TestClosingOverlayCopyMatchesNative pins the reference copy
// (reference/electron/src/main/closing-overlay.js:59-70).
func TestClosingOverlayCopyMatchesNative(t *testing.T) {
	cases := []struct{ locale, title, detail string }{
		{i18n.LocaleZH, "关闭中", "正在停止本机 Harness 服务，请稍候"},
		{i18n.LocaleEN, "Closing", "Stopping the local Harness service…"},
	}
	for _, tc := range cases {
		if got := i18n.Lookup(tc.locale, i18n.KeyClosingTitle); got != tc.title {
			t.Fatalf("%s title = %q, want %q", tc.locale, got, tc.title)
		}
		if got := i18n.Lookup(tc.locale, i18n.KeyClosingDetail); got != tc.detail {
			t.Fatalf("%s detail = %q, want %q", tc.locale, got, tc.detail)
		}
	}
}

type closingFrame struct {
	tt      *ui.Tester
	card    ui.Rect
	spinner ui.Rect
	painted bool
	start   time.Time
}

// renderClosing paints the layer over a red chat-like page so the cover and its
// metrics are measurable.
func renderClosing(t *testing.T, m ClosingOverlay, w, h int) closingFrame {
	t.Helper()
	f := closingFrame{}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(200, 30, 30)).Children(func() {
			ui.Text(c, "chat transcript").FontSize(14).TextColor(ui.RGB(255, 255, 255))
		})
		DrawClosingOverlay(c, ClosingOverlay{
			Title:  m.Title,
			Detail: m.Detail,
			Colors: m.Colors,
			Observe: func(card, spinner ui.Rect) {
				f.card, f.spinner, f.painted = card, spinner, true
			},
		})
	}, w, h)
	// The spinner turns from the moment the layer paints, so the tests pin the
	// clock: the layer's first frame is the reference's starting point.
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	f.start = start
	tt.Frame()
	f.tt = tt
	return f
}

// TestClosingOverlayGeometryMatchesNative measures the card against
// closing-overlay.js:21-52: min-width 240, padding 24, radius 24, gap 8,
// spinner 36, title 16/24 with an 8 DIP top margin, detail 12/18.
func TestClosingOverlayGeometryMatchesNative(t *testing.T) {
	colors := theme.Default().Light
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		m := closingModel(locale, colors)
		f := renderClosing(t, m, 720, 480)
		if !f.painted {
			t.Fatalf("%s: closing overlay painted no card; texts = %q", locale, f.tt.Texts())
		}
		title, ok := f.tt.Find(m.Title)
		if !ok {
			t.Fatalf("%s: missing title %q; texts = %q", locale, m.Title, f.tt.Texts())
		}
		detail, ok := f.tt.Find(m.Detail)
		if !ok {
			t.Fatalf("%s: missing detail %q; texts = %q", locale, m.Detail, f.tt.Texts())
		}
		if f.card.W < ClosingCardMinWidth {
			t.Fatalf("%s: card width %v is under the native min-width %v", locale, f.card.W, ClosingCardMinWidth)
		}
		if want := max(ClosingCardMinWidth, max(title.W, detail.W)+2*ClosingCardPadding); abs(f.card.W-want) > 1 {
			t.Fatalf("%s: card width %v, want %v for its widest line", locale, f.card.W, want)
		}
		if want := 2*ClosingCardPadding + ClosingSpinnerSize + ClosingCardGap + ClosingTitleGap +
			ClosingTitleLine + ClosingCardGap + ClosingDetailLine; abs(f.card.H-float32(want)) > 1 {
			t.Fatalf("%s: card height %v, want %v", locale, f.card.H, want)
		}
		if f.spinner.W != ClosingSpinnerSize || f.spinner.H != ClosingSpinnerSize {
			t.Fatalf("%s: spinner %v, want %v square", locale, f.spinner, ClosingSpinnerSize)
		}
		if abs(title.H-ClosingTitleLine) > 1 || abs(detail.H-ClosingDetailLine) > 1 {
			t.Fatalf("%s: line heights title %v detail %v, want %v/%v", locale, title.H, detail.H, ClosingTitleLine, ClosingDetailLine)
		}
		// The card's 8 DIP gap, plus the title's own 8 DIP top margin.
		if want := f.spinner.Y + ClosingSpinnerSize + ClosingCardGap + ClosingTitleGap; abs(title.Y-want) > 1 {
			t.Fatalf("%s: title top %v, want %v under the spinner", locale, title.Y, want)
		}
		if want := title.Y + title.H + ClosingCardGap; abs(detail.Y-want) > 1 {
			t.Fatalf("%s: detail top %v, want %v under the title", locale, detail.Y, want)
		}
		// The card's 24 DIP padding, and the whole card centered in the window.
		if want := f.card.Y + ClosingCardPadding; abs(f.spinner.Y-want) > 1 {
			t.Fatalf("%s: spinner top %v, want the card's %v padding", locale, f.spinner.Y, ClosingCardPadding)
		}
		if want := f.card.X + (f.card.W-ClosingSpinnerSize)/2; abs(f.spinner.X-want) > 1 {
			t.Fatalf("%s: spinner left %v, want %v centered", locale, f.spinner.X, want)
		}
		if abs(f.card.X-(720-f.card.W)/2) > 1 || abs(f.card.Y-(480-f.card.H)/2) > 1 {
			t.Fatalf("%s: card %v is not centered in 720x480", locale, f.card)
		}
		if abs(title.X+title.W/2-360) > 1 || abs(detail.X+detail.W/2-360) > 1 {
			t.Fatalf("%s: copy is off the window's center line", locale)
		}
	}
}

// TestClosingOverlayCardKeepsItsMinWidth covers copy short enough that the
// 240 DIP minimum, not the text, sets the card size.
func TestClosingOverlayCardKeepsItsMinWidth(t *testing.T) {
	f := renderClosing(t, ClosingOverlay{Title: "x", Detail: "y", Colors: theme.Default().Light}, 720, 480)
	if !f.painted || abs(f.card.W-ClosingCardMinWidth) > 1 {
		t.Fatalf("card width %v, want the native min-width %v", f.card.W, ClosingCardMinWidth)
	}
}

// TestClosingOverlayCoversTheWholeWindow proves the layer paints last: nothing of
// the page behind it survives, and the card is the only thing on the cover.
func TestClosingOverlayCoversTheWholeWindow(t *testing.T) {
	colors := theme.Default().Light
	f := renderClosing(t, closingModel(i18n.LocaleZH, colors), 720, 480)
	img := f.tt.Image()
	bg, field := theme.Color(colors.BG), theme.Color(colors.Field)
	for _, p := range []image.Point{{X: 1, Y: 1}, {X: 718, Y: 478}, {X: 360, Y: 12}, {X: 12, Y: 240}} {
		if got := img.RGBAAt(p.X, p.Y); distance(got, bg) > 1 {
			t.Fatalf("pixel %v = %v, want the overlay background %v over the page", p, got, bg)
		}
	}
	// The 24 DIP radius cuts the corner: the straight edge carries the raised
	// field fill, the corner falls back to the layer background.
	edge := img.RGBAAt(int(f.card.X)+3, int(f.card.Y)+int(f.card.H)/2)
	if distance(edge, field) > 4 {
		t.Fatalf("card edge = %v, want the theme field %v", edge, field)
	}
	corner := img.RGBAAt(int(f.card.X)+3, int(f.card.Y)+3)
	if distance(corner, field) < distance(corner, bg) {
		t.Fatalf("card corner %v is the field fill: the 24 DIP radius should expose the background", corner)
	}
	body := img.RGBAAt(int(f.card.X)+12, int(f.card.Y)+int(f.card.H)/2)
	if distance(body, field) > 4 {
		t.Fatalf("card body = %v, want the theme field %v", body, field)
	}
	// No red from the covered page survives anywhere, including under the card.
	red := image.Point{X: -1}
	for y := 0; y < 480 && red.X < 0; y++ {
		for x := 0; x < 720; x++ {
			if c := img.RGBAAt(x, y); c.R > 150 && c.G < 90 && c.B < 90 {
				red = image.Point{X: x, Y: y}
				break
			}
		}
	}
	if red.X >= 0 {
		t.Fatalf("the covered page shows through the closing layer at %v", red)
	}
}

// TestClosingOverlayFollowsTheTheme renders the light and dark palettes and
// checks the background, the card fill and the spinner's accent top arc.
func TestClosingOverlayFollowsTheTheme(t *testing.T) {
	for _, tc := range []struct {
		name   string
		colors theme.Colors
	}{
		{"light", theme.Default().Light},
		{"dark", theme.Default().Dark},
	} {
		f := renderClosing(t, closingModel(i18n.LocaleEN, tc.colors), 720, 480)
		if !f.painted {
			t.Fatalf("%s: no card", tc.name)
		}
		img := f.tt.Image()
		bg, field, accent := theme.Color(tc.colors.BG), theme.Color(tc.colors.Field), theme.Color(tc.colors.Accent)
		if got := img.RGBAAt(1, 1); distance(got, bg) > 1 {
			t.Fatalf("%s: background %v, want %v", tc.name, got, bg)
		}
		if got := img.RGBAAt(int(f.card.X)+12, int(f.card.Y)+int(f.card.H)/2); distance(got, field) > 1 {
			t.Fatalf("%s: card fill %v, want %v", tc.name, got, field)
		}
		// The accent paints only the top arc of the 2 DIP ring.
		cx, cy := f.spinner.X+f.spinner.W/2, f.spinner.Y+f.spinner.H/2
		top := closestPixel(img, f.spinner, accent)
		if top.X == 0 && top.Y == 0 {
			t.Fatalf("%s: the spinner has no accent pixel; accent %v", tc.name, accent)
		}
		if top.Y >= int(cy) {
			t.Fatalf("%s: accent pixel %v is not on the ring's top arc", tc.name, top)
		}
		if dx := abs(float32(top.X) - cx); dx > f.spinner.W/2 {
			t.Fatalf("%s: accent pixel %v is outside the ring", tc.name, top)
		}
	}
}

// TestClosingOverlayIgnoresEmptyCopy keeps a shutting-down window from painting
// an empty card when no copy is available.
func TestClosingOverlayIgnoresEmptyCopy(t *testing.T) {
	f := renderClosing(t, ClosingOverlay{Colors: theme.Default().Light}, 320, 240)
	if f.painted {
		t.Fatalf("empty copy painted a card at %v", f.card)
	}
}

// TestClosingSpinnerTurns covers the reference's 0.85s linear infinite spin
// (closing-overlay.js:34-42): the accent arc starts at the top and is a quarter
// of the ring round one quarter period later. A reduced-motion desktop holds it
// still.
func TestClosingSpinnerTurns(t *testing.T) {
	colors := theme.Default().Light
	f := renderClosing(t, closingModel(i18n.LocaleEN, colors), 720, 480)
	accent := theme.Color(colors.Accent)
	cx, cy := f.spinner.X+f.spinner.W/2, f.spinner.Y+f.spinner.H/2
	startX, startY, ok := accentCentroid(f.tt.Image(), f.spinner, accent)
	if !ok {
		t.Fatal("the spinner has no accent arc at its start")
	}
	if startY > cy-6 || abs(startX-cx) > 8 {
		t.Fatalf("the accent arc starts centred at %v,%v, want the ring's top", startX, startY)
	}
	f.tt.SetTime(f.start.Add(ClosingSpinPeriod / 4))
	quarterX, quarterY, _ := accentCentroid(f.tt.Image(), f.spinner, accent)
	if quarterX < cx+4 || abs(quarterY-cy) > 8 {
		t.Fatalf("a quarter turn put the accent arc at %v,%v, want it round to the right of %v,%v", quarterX, quarterY, cx, cy)
	}
	f.tt.SetTime(f.start.Add(ClosingSpinPeriod / 2))
	halfX, halfY, _ := accentCentroid(f.tt.Image(), f.spinner, accent)
	if halfY < cy+4 || abs(halfX-cx) > 8 {
		t.Fatalf("half a turn put the accent arc at %v,%v, want the ring's bottom", halfX, halfY)
	}
	// Three quarters is the left of the ring, and a whole period is back at the
	// top where it started.
	f.tt.SetTime(f.start.Add(3 * ClosingSpinPeriod / 4))
	if threeX, _, _ := accentCentroid(f.tt.Image(), f.spinner, accent); threeX > cx-4 {
		t.Fatalf("three quarters put the accent arc at x=%v, want it round to the left of %v", threeX, cx)
	}
	f.tt.SetTime(f.start.Add(ClosingSpinPeriod))
	fullX, fullY, _ := accentCentroid(f.tt.Image(), f.spinner, accent)
	if abs(fullX-startX) > 2 || abs(fullY-startY) > 2 {
		t.Fatalf("a full turn put the accent arc at %v,%v, want %v,%v", fullX, fullY, startX, startY)
	}
}

// TestClosingSpinnerHoldsStillWithReduceMotion covers the reduced-motion path.
func TestClosingSpinnerHoldsStillWithReduceMotion(t *testing.T) {
	colors := theme.Default().Light
	f := renderClosing(t, closingModel(i18n.LocaleEN, colors), 720, 480)
	f.tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	accent := theme.Color(colors.Accent)
	cy := f.spinner.Y + f.spinner.H/2
	at, atY, ok := accentCentroid(f.tt.Image(), f.spinner, accent)
	if !ok {
		t.Fatal("the spinner has no accent arc under reduce motion")
	}
	f.tt.SetTime(f.start.Add(ClosingSpinPeriod / 4))
	after, afterY, _ := accentCentroid(f.tt.Image(), f.spinner, accent)
	if at != after || atY != afterY {
		t.Fatalf("reduce motion turned the accent arc from %v,%v to %v,%v", at, atY, after, afterY)
	}
	if atY > cy-6 {
		t.Fatalf("reduce motion left the accent arc at %v,%v, want the ring's top", at, atY)
	}
}

// accentCentroid returns where the spinner's accent arc sits, as the centre of
// the arc's pixels, or false when none is close to the accent color.
func accentCentroid(img *image.RGBA, box ui.Rect, accent ui.Color) (float32, float32, bool) {
	var sx, sy, n float32
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if distance(img.RGBAAt(x, y), accent) <= 24 {
				sx, sy, n = sx+float32(x), sy+float32(y), n+1
			}
		}
	}
	if n == 0 {
		return 0, 0, false
	}
	return sx / n, sy / n, true
}

// closestPixel returns the box pixel nearest target, or the zero point when the
// box holds nothing close to it.
func closestPixel(img *image.RGBA, box ui.Rect, target ui.Color) image.Point {
	best := image.Point{}
	bestDistance := 1 << 30
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if d := distance(img.RGBAAt(x, y), target); d < bestDistance {
				best, bestDistance = image.Point{X: x, Y: y}, d
			}
		}
	}
	if bestDistance > 24*3 {
		return image.Point{}
	}
	return best
}

func abs8(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// distance sums the channel distances of two colors, as a cheap match check.
func distance(got color.RGBA, want ui.Color) int {
	return abs8(got.R, want.R) + abs8(got.G, want.G) + abs8(got.B, want.B)
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
