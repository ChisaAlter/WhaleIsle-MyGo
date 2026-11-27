package composer

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)

func TestComposerCardNoOuterShadowAnd12pxMenuRadius(t *testing.T) {
	defer design.SetGlass(false, nil)
	for _, glass := range []bool{false, true} {
		design.SetGlass(glass, func(_ design.Scheme, p *design.Palette) {
			p.Card = p.Card.Alpha(0.80)
			p.Popover = p.Popover.Alpha(0.70)
		})
		var menuBounds ui.Rect
		tt := ui.NewTester(func(c *ui.Context) {
			t := design.For(c)
			card := ui.Box(c).Width(200).Height(60)
			composerCardElevation(card, t, false, false)
			menuBounds = menuPanelLook(c, ui.Box(c).Width(160).Height(80)).Bounds()
		}, 300, 200)
		tt.Frame()
		if menuBounds.W != 160 {
			t.Fatalf("menuPanelLook width=%v, want 160", menuBounds.W)
		}
	}
}

func stripeColorRange(img *image.RGBA, r ui.Rect) int {
	x0 := max(img.Bounds().Min.X, int(r.X)+6)
	x1 := min(img.Bounds().Max.X, int(r.X+r.W)-6)
	y := min(img.Bounds().Max.Y-1, max(img.Bounds().Min.Y, int(r.Y+r.H/2)))
	loR, hiR := 255, 0
	for x := x0; x < x1; x++ {
		px := img.RGBAAt(x, y)
		loR = min(loR, int(px.R))
		hiR = max(hiR, int(px.R))
	}
	if hiR < loR {
		return 0
	}
	return hiR - loR
}

func paintStripedBackdrop(p *ui.Painter, r ui.Rect) {
	for x := r.X; x < r.X+r.W; x += 8 {
		col := ui.RGB(235, 45, 45)
		if int(x/8)%2 == 1 {
			col = ui.RGB(25, 190, 235)
		}
		p.Fill(ui.Rect{X: x, Y: r.Y, W: 8, H: r.H}, col, 0)
	}
}

func TestComposerSuppressBackdropBlurAndEditStatusGlassSurface(t *testing.T) {
	defer design.SetGlass(false, nil)
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Card = ui.RGBA(252, 253, 255, 0.45)
		p.Popover = ui.RGBA(252, 253, 254, 0.44)
	})

	d := NewDraftState("")
	suppress := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(60, 0, 0, 0).Draw(paintStripedBackdrop).Children(func() {
			View(c, Model{
				State:                d,
				Draft:                &d.Text,
				Editing:              true,
				SuppressBackdropBlur: suppress,
			})
		})
	}, 520, 260)
	tt.Frame()

	cardRect, ok := tt.Find("Composer card")
	if !ok {
		t.Fatal("composer card not found")
	}
	editRect, ok := tt.Find("Composer edit status")
	if !ok {
		t.Fatal("composer edit status not found")
	}
	blurImg := tt.Image()
	// Sample inside the composer card away from text/toolbar and inside edit status.
	cardSample := ui.Rect{X: cardRect.X + 40, Y: cardRect.Y + 12, W: 140, H: 12}
	blurredCardSpan := stripeColorRange(blurImg, cardSample)
	blurredEditSpan := stripeColorRange(blurImg, ui.Rect{X: editRect.X + 4, Y: editRect.Y + 2, W: editRect.W - 8, H: 6})

	// SuppressBackdropBlur switches the composer card and edit status to cheap
	// translucent fills without live blur, preserving the crisp underlying stripes.
	suppress = true
	tt.Frame()
	plainImg := tt.Image()
	plainCardSpan := stripeColorRange(plainImg, cardSample)
	plainEditSpan := stripeColorRange(plainImg, ui.Rect{X: editRect.X + 4, Y: editRect.Y + 2, W: editRect.W - 8, H: 6})

	if blurredCardSpan >= plainCardSpan/2 || plainCardSpan < 50 {
		t.Fatalf("composer card blur suppression did not switch between live blur and plain fill: blurredSpan=%d plainSpan=%d", blurredCardSpan, plainCardSpan)
	}
	if blurredEditSpan >= plainEditSpan/2 || plainEditSpan < 40 {
		t.Fatalf("composer-edit-status did not apply unified glass blur surface: blurredSpan=%d plainSpan=%d", blurredEditSpan, plainEditSpan)
	}
}
