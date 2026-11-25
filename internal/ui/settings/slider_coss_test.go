package settings

import (
	"image"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// assertNoTicksBelowSlider checks the rendered pixels directly beneath a slider
// track (below the 3px thumb shadow and within the band where StepSlider drew
// ticks) to ensure no vertical tick marks are painted.
func assertNoTicksBelowSlider(t *testing.T, tt *ui.Tester, track ui.Rect) {
	t.Helper()
	img := tt.Image()
	bounds := img.Bounds()
	xStart := max(bounds.Min.X, int(math.Ceil(float64(track.X+8))))
	xEnd := min(bounds.Max.X, int(math.Floor(float64(track.X+track.W-8))))
	yStart := max(bounds.Min.Y, int(math.Ceil(float64(track.Y+track.H+4))))
	yEnd := min(bounds.Max.Y, int(math.Floor(float64(track.Y+track.H+10))))
	if xEnd-xStart < 40 || yEnd <= yStart {
		t.Fatalf("invalid inspection band x=[%d,%d) y=[%d,%d) for track=%+v", xStart, xEnd, yStart, yEnd, track)
	}
	for y := yStart; y < yEnd; y++ {
		base := img.RGBAAt(xStart, y)
		for x := xStart + 1; x < xEnd; x++ {
			got := img.RGBAAt(x, y)
			if got != base {
				t.Fatalf("unexpected non-uniform pixel beneath slider at (%d,%d): got=%+v want=%+v (track=%+v)", x, y, got, base, track)
			}
		}
	}
}

func assertSnappedStep(t *testing.T, got, lo, step float64) {
	t.Helper()
	steps := (got - lo) / step
	if math.Abs(steps-math.Round(steps)) > 1e-6 {
		t.Fatalf("value %g is not a multiple of step %g from %g", got, step, lo)
	}
}

func TestWallpaperCropModalSliderNoTicksAndLayout(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	for _, tc := range []struct {
		name string
		w, h int
	}{
		{"wide", 900, 900},
		{"narrow", 380, 760},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, _, _ := wallpaperFixture(t)
			g.previewSource = image.NewRGBA(image.Rect(0, 0, 600, 400))
			g.cropDraft = wallpapers.DefaultCrop()
			g.cropOpen = true

			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					wallpaperCropModal(c, g, c.Theme().Text, c.Theme().TextMuted)
				})
			}, tc.w, tc.h)
			tt.Frame()

			track := sliderTrack(t, tt, "Zoom")
			if track.H != 16 {
				t.Fatalf("expected coss.Slider height 16, got %g (track=%+v)", track.H, track)
			}
			if track.W < 176 {
				t.Fatalf("expected coss.Slider width >= 176, got %g (track=%+v)", track.W, track)
			}
			if track.X < 0 || track.X+track.W > float32(tc.w) {
				t.Fatalf("slider overflows viewport %dx%d: %+v", tc.w, tc.h, track)
			}
			rowRect, ok := tt.Find("Zoom")
			if !ok {
				t.Fatal("Zoom row not found")
			}
			if track.X < rowRect.X || track.X+track.W > rowRect.X+rowRect.W {
				t.Fatalf("slider %+v overflows Zoom row %+v", track, rowRect)
			}

			assertNoTicksBelowSlider(t, tt, track)

			beforePreview := g.preview
			dragSlider(tt, track, .5)
			if g.cropDraft.Zoom <= 1 || g.cropDraft.Zoom > 3 {
				t.Fatalf("drag did not move zoom as expected: %g", g.cropDraft.Zoom)
			}
			assertSnappedStep(t, g.cropDraft.Zoom, 1, 0.05)
			if g.preview == beforePreview {
				t.Fatal("zoom drag did not update preview")
			}

			afterDrag := g.cropDraft.Zoom
			tt.Key(0, ui.KeyRight)
			if math.Abs((g.cropDraft.Zoom-afterDrag)-0.05) > 1e-6 {
				t.Fatalf("KeyRight expected +0.05 from %g, got %g", afterDrag, g.cropDraft.Zoom)
			}
			tt.Key(0, ui.KeyLeft)
			if math.Abs(g.cropDraft.Zoom-afterDrag) > 1e-6 {
				t.Fatalf("KeyLeft expected back to %g, got %g", afterDrag, g.cropDraft.Zoom)
			}
		})
	}
}

func TestWallpaperSliderStepBehaviorAndDisabled(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	for _, tc := range []struct {
		name string
		w, h int
	}{
		{"horizontal", 600, 200},
		{"stacked", 320, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			val := 1.0
			disabled := false
			changedCalls := 0
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Background(c.Theme().Background).Padding(16).Children(func() {
					if wallpaperSliderStep(c, "zoom-test", "Zoom", "", &val, 1, 3, 0.05, disabled, c.Theme().Text, c.Theme().TextMuted) {
						changedCalls++
					}
				})
			}, tc.w, tc.h)
			tt.Frame()

			track := sliderTrack(t, tt, "Zoom")
			if track.H != 16 || track.W < 176 {
				t.Fatalf("unexpected slider geometry %+v", track)
			}
			if track.X < 0 || track.X+track.W > float32(tc.w) {
				t.Fatalf("slider overflows %dx%d: %+v", tc.w, tc.h, track)
			}
			assertNoTicksBelowSlider(t, tt, track)

			dragSlider(tt, track, .5)
			if changedCalls == 0 || math.Abs(val-2.0) > 0.1 {
				t.Fatalf("drag failed: val=%g changedCalls=%d", val, changedCalls)
			}
			assertSnappedStep(t, val, 1, 0.05)

			prev := val
			tt.Key(0, ui.KeyRight)
			if math.Abs((val-prev)-0.05) > 1e-6 {
				t.Fatalf("KeyRight step failed: prev=%g val=%g", prev, val)
			}

			disabled = true
			tt.Frame()
			lockedVal := val
			callsBefore := changedCalls
			dragSlider(tt, track, .9)
			tt.Key(0, ui.KeyLeft)
			if val != lockedVal || changedCalls != callsBefore {
				t.Fatalf("disabled slider changed: val=%g->%g calls=%d->%d", lockedVal, val, callsBefore, changedCalls)
			}
		})
	}
}

func TestMigratedCossSlidersNoTicksAndInteractions(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	t.Run("immediate-slider", func(t *testing.T) {
		g, saves, _ := wallpaperFixture(t)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Background(c.Theme().Background).Padding(16).Children(func() {
				wallpaperImmediateSlider(c, g, "terminal-opacity", "Terminal opacity", g.currentPreferences().Terminal, 40, 100, 5, false, func(p *wallpapers.Preferences, v float64) {
					p.Terminal = v
				}, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 500, 160)
		tt.Frame()

		track := sliderTrack(t, tt, "Terminal opacity")
		if track.H != 16 || track.W < 176 {
			t.Fatalf("unexpected immediate slider geometry %+v", track)
		}
		assertNoTicksBelowSlider(t, tt, track)

		dragSlider(tt, track, .2)
		if len(*saves) == 0 {
			t.Fatal("immediate slider drag did not save")
		}
		got := (*saves)[len(*saves)-1].Terminal
		assertSnappedStep(t, got, 40, 5)
	})

	t.Run("typography-size-slider", func(t *testing.T) {
		ed := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Background(c.Theme().Background).Padding(16).Children(func() {
				typographySizeField(c, ed, "font-size-interface", "Size", "interface", &ed.Draft.Interface, 12, 22, c.Theme().TextMuted)
			})
		}, 500, 180)
		tt.Frame()

		var track ui.Rect
		for _, r := range tt.FindAll("Size") {
			if r.H == 16 && r.W > track.W {
				track = r
			}
		}
		if track.W < 176 || track.H != 16 {
			t.Fatalf("unexpected typography slider geometry %+v among %v", track, tt.FindAll("Size"))
		}
		assertNoTicksBelowSlider(t, tt, track)

		before := ed.Draft.Interface.Size
		dragSlider(tt, track, .8)
		if ed.Draft.Interface.Size == before {
			t.Fatalf("typography slider drag did not change size (still %g)", before)
		}
		assertSnappedStep(t, float64(ed.Draft.Interface.Size), 12, 1)
	})

	t.Run("effect-number-slider", func(t *testing.T) {
		draft := &effectDraft{
			Style: map[string]any{
				"mode":       "custom",
				"intensity":  70.0,
				"bloom":      70.0,
				"period":     10.0,
				"trackWidth": 2.0,
				"glowBlur":   10.0,
			},
		}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Background(c.Theme().Background).Padding(16).Children(func() {
				renderEffectControls(c, draft, false, false, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 600, 1200)
		tt.Frame()

		track := sliderTrack(t, tt, "effect-intensity")
		if track.H != 16 || track.W < 176 {
			t.Fatalf("unexpected effect slider geometry %+v", track)
		}
		assertNoTicksBelowSlider(t, tt, track)

		dragSlider(tt, track, .9)
		got, _ := effectValue(draft.Style, "intensity").(float64)
		if got <= 70 {
			t.Fatalf("effect-intensity drag did not raise value: %g", got)
		}
		assertSnappedStep(t, got, 0, 5)
	})
}
