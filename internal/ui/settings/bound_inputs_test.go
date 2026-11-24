package settings

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/i18n"
)

// sliderTrack returns the track rect of the named slider among the label
// matches (the row block and its text share the same name; the track is the
// widest short rect).
func sliderTrack(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	best := ui.Rect{}
	for _, r := range tt.FindAll(label) {
		if r.H > 0 && r.H < 40 && r.W > best.W {
			best = r
		}
	}
	if best.W == 0 {
		t.Fatalf("slider %q not found: %v", label, tt.FindAll(label))
	}
	return best
}

// dragSlider presses inside a slider track; the press sets the value at the
// pressed position, so releasing further right raises it.
func dragSlider(tt *ui.Tester, r ui.Rect, frac float32) {
	tt.Press(r.X+r.W*frac, r.Y+r.H/2)
	tt.Release(r.X+r.W*frac, r.Y+r.H/2)
}

// Moving a slider in the background-effects dialog must mark the preset
// "custom". The pre/post snapshot compare ran before applyInputs flushed the
// slider write, so the preset stayed "default".
func TestBackgroundEffectSliderMarksPresetCustom(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	g, saves, _ := wallpaperFixture(t)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			wallpaperBackgroundPane(c, g, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 600, 1600)
	if err := tt.Click("Background effect settings"); err != nil {
		t.Fatalf("background effect dialog not reachable: %v", err)
	}
	dragSlider(tt, sliderTrack(t, tt, "Speed"), .9)
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("save button not reachable: %v", err)
	}
	if len(*saves) == 0 {
		t.Fatal("save did not run")
	}
	saved := (*saves)[len(*saves)-1]
	if saved.Preset != "custom" || saved.Speed <= 190 {
		t.Fatalf("slider change lost: saved preset=%q speed=%g", saved.Preset, saved.Speed)
	}
}

// The same snapshot-compare defect existed in the pointer-effects dialog.
func TestPointerEffectSliderMarksPresetCustom(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	g, saves, _ := wallpaperFixture(t)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			wallpaperBackgroundPane(c, g, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 600, 1600)
	if err := tt.Click("Pointer effect settings"); err != nil {
		t.Fatalf("pointer effect dialog not reachable: %v", err)
	}
	dragSlider(tt, sliderTrack(t, tt, "Pointer speed"), .9)
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("save button not reachable: %v", err)
	}
	if len(*saves) == 0 {
		t.Fatal("save did not run")
	}
	saved := (*saves)[len(*saves)-1]
	if saved.CursorPreset != "custom" || saved.CursorSpeed <= 100 {
		t.Fatalf("slider change lost: saved cursorPreset=%q speed=%g", saved.CursorPreset, saved.CursorSpeed)
	}
}

// Moving the crop-zoom slider must rebuild the preview. The pre/post compare
// read the bound zoom before applyInputs flushed it, so updatePreview never
// ran and the preview image went stale.
func TestWallpaperCropZoomSliderUpdatesPreview(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	g, _, _ := wallpaperFixture(t)
	g.previewSource = image.NewRGBA(image.Rect(0, 0, 600, 400))
	g.cropDraft = wallpapers.DefaultCrop()
	g.cropOpen = true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			wallpaperBackgroundPane(c, g, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 600, 1600)
	tt.Frame()
	before := g.preview
	if before == nil {
		t.Fatal("crop preview never built")
	}
	dragSlider(tt, sliderTrack(t, tt, "Zoom"), .9)
	if g.cropDraft.Zoom <= 1 {
		t.Fatalf("zoom slider did not change the bound value: %g", g.cropDraft.Zoom)
	}
	if g.preview == before {
		t.Fatal("zoom change did not rebuild the preview")
	}
}
