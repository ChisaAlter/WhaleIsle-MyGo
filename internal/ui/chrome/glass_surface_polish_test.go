package chrome

import (
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/design"
)

func TestApplyMenuSurfaceBlursAndFallsBackCleanly(t *testing.T) {
	t.Cleanup(func() {
		ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})

	paintStripes := func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			for i, x := 0, float32(0); x < r.W; i, x = i+1, x+4 {
				col := ui.RGB(20, 120, 240)
				if i%2 == 1 {
					col = ui.RGB(240, 120, 20)
				}
				p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 4, H: r.H}, col, 0)
			}
		})
	}

	// Off: keeps solid Popover background with zero stripe leakage.
	ConfigureBackdrop("", wallpapers.Defaults())
	design.SetGlass(false, nil)
	var bOff ui.Rect
	ttOff := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			paintStripes(c)
			bOff = ApplyMenuSurface(ui.Box(c).Absolute().Left(30).Top(20).Width(120).Height(80), design.For(c), 12).Bounds()
		})
	}, 200, 140)
	ttOff.Frame()
	if span := testChannelSpan(ttOff, int(bOff.Y+bOff.H/2), int(bOff.X+20), int(bOff.X+bOff.W-20)); span != 0 {
		t.Fatalf("expected opaque fallback with glass off, got stripe span=%d", span)
	}

	// On: blurs backdrop sharply (40px) and paints top inner specular rim.
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Popover = p.Popover.Alpha(0.58)
	})
	var bOn ui.Rect
	ttOn := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			paintStripes(c)
			bOn = ApplyMenuSurface(ui.Box(c).Absolute().Left(30).Top(20).Width(120).Height(80), design.For(c), 12).Bounds()
		})
	}, 200, 140)
	ttOn.Frame()
	raw := testChannelSpan(ttOn, 8, 10, 190)
	blurred := testChannelSpan(ttOn, int(bOn.Y+bOn.H/2), int(bOn.X+20), int(bOn.X+bOn.W-20))
	if blurred > raw/4 {
		t.Fatalf("ApplyMenuSurface did not blur stripes enough: blurred=%d raw=%d", blurred, raw)
	}
	img := ttOn.Image()
	rimLum := RelativeLuminance(pixel(img.RGBAAt(int(bOn.X+bOn.W/2), int(bOn.Y+1))))
	inLum := RelativeLuminance(pixel(img.RGBAAt(int(bOn.X+bOn.W/2), int(bOn.Y+10))))
	if rimLum <= inLum {
		t.Fatalf("ApplyMenuSurface missing top inner specular rim: rim=%.3f inside=%.3f", rimLum, inLum)
	}
}

func TestDialogMaskBlurSoftensBackdropUnderGlass(t *testing.T) {
	t.Cleanup(func() {
		ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	url := pngDataURL(t, color.RGBA{R: 200, G: 200, B: 200, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	ConfigureBackdrop(url, prefs)

	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			for i, x := 0, float32(0); x < r.W; i, x = i+1, x+2 {
				col := ui.RGB(20, 40, 220)
				if i%2 == 1 {
					col = ui.RGB(230, 180, 30)
				}
				p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 2, H: r.H}, col, 0)
			}
		}).Children(func() {
			ui.DialogBase(c.Key("mask-blur-dlg"), &open, func(backdrop, panel ui.Element) {
				DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 120, 12).Children(func() {
					ui.Box(c).Size(80, 40)
				})
			})
		})
	}, 260, 180)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))
	// Outside the centered dialog (x=10..50, y=20), 2px mask blur softens 2px stripes.
	if span := testChannelSpan(tt, 20, 10, 50); span >= 120 {
		t.Fatalf("expected 2px mask blur to soften fine stripes behind modal mask, got span=%d", span)
	}
}
