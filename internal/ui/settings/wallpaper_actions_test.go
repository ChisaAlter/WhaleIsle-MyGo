package settings

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"reflect"
	"testing"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

func wallpaperFixture(t *testing.T) (*WallpaperEditor, *[]wallpapers.Preferences, *[]string) {
	t.Helper()
	var saved []wallpapers.Preferences
	var applied []string
	g := NewWallpaperEditor(wallpapers.Defaults(), t.TempDir(), func(p wallpapers.Preferences) error { saved = append(saved, p.Clone()); return nil }, func(p string) error { applied = append(applied, p); return nil }, func(task func() func()) { task()() })
	g.Inline = true
	return g, &saved, &applied
}
func TestWallpaperCropCancellationDoesNotCommitAndRecropsHostImage(t *testing.T) {
	g, saves, applied := wallpaperFixture(t)
	before := g.Prefs.Clone()
	var raw bytes.Buffer
	png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 30, 20)))
	g.importImage(func(context.Context) ([]byte, error) { return raw.Bytes(), nil })
	if !g.Cropping() {
		t.Fatal("crop not opened")
	}
	g.cropDraft.X = .1
	g.cancelCrop()
	if !reflect.DeepEqual(g.Prefs, before) || len(*saves) != 0 || len(*applied) != 0 {
		t.Fatal("cancel changed stored settings")
	}
	g.ImageSource = func() string { return "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw.Bytes()) }
	g.openCrop()
	if !g.Cropping() {
		t.Fatal("Host image without Original cannot recrop")
	}
	g.applyCrop()
	if len(*saves) != 1 || len(*applied) != 1 || g.Cropping() {
		t.Fatal("explicit apply did not commit")
	}
	g.ClearImage()
	if (*applied)[1] != "" || g.Saved.Original != "" {
		t.Fatal("clear did not clear both image and crop source")
	}
}
func TestWallpaperEffectsDraftCatalogSaveAndRetirement(t *testing.T) {
	g, saves, _ := wallpaperFixture(t)
	g.sourceDraft = g.Saved.Sources[0]
	g.sourceDraft.Name = "Unfinished source name"
	g.sourceEditOpen = true
	g.toggleFavorite(wallpapers.Item{ID: "favorite", SourceID: "fixture"})
	if len(*saves) != 1 || (*saves)[0].Sources[0].Name == g.sourceDraft.Name || len((*saves)[0].Favorites) != 1 {
		t.Fatal("favorite committed the source editor draft")
	}
	accepted := g.Saved.Clone()
	g.AcceptHost(accepted)
	if g.sourceDraft.Name != "Unfinished source name" || !g.sourceEditOpen {
		t.Fatal("Host refresh clobbered the open source editor")
	}
	var late func()
	g.Run = func(task func() func()) { late = task() }
	var data bytes.Buffer
	png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	g.importImage(func(context.Context) ([]byte, error) { return data.Bytes(), nil })
	g.Hide()
	late()
	if g.Cropping() || g.Busy || g.Prefs.Blur != g.Saved.Blur {
		t.Fatal("late result resurrected retired crop")
	}
}
func TestWallpaperControlsHeadlessLayout(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	g, _, _ := wallpaperFixture(t)
	g.ImageSource = func() string { return "" }
	path := ""
	a := &AppearanceAccess{Background: g, WallpaperPath: &path, OnBrowseWallpaper: func() {}}
	tt := ui.NewTester(func(c *ui.Context) {
		appearancePane(c, Model{Appearance: a, ThemeID: "deepseek.light"}, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 1800)
	for _, label := range []string{"Choose image", "Wallpaper library"} {
		if !tt.HasText(label) {
			t.Fatalf("missing action %q", label)
		}
	}
	if tt.HasText("Recrop") || tt.HasText("Clear wallpaper") || tt.HasText("Pixelate") {
		t.Fatal("image-only actions appeared without an explicit image")
	}
	var raw bytes.Buffer
	_ = png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	g.ImageSource = func() string { return "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw.Bytes()) }
	path = g.ImageSource()
	tt.Frame()
	for _, label := range []string{"Recrop", "Clear wallpaper", "Reset"} {
		if !tt.HasText(label) {
			t.Fatalf("explicit-image action missing: %s", label)
		}
	}
	if path := os.Getenv("WHALE_PET_SETTINGS_SNAPSHOT"); path != "" {
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWallpaperCropAndPointerSandboxGestures(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	g, saved, _ := wallpaperFixture(t)
	g.previewSource = image.NewRGBA(image.Rect(0, 0, 600, 400))
	g.cropDraft = wallpapers.DefaultCrop()
	g.cropOpen = true
	g.updatePreview()
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { wallpaperBackgroundPane(c, g, c.Theme().Text, c.Theme().TextMuted) })
	}, 600, 1600)
	r, ok := tt.Find("Wallpaper crop preview")
	if !ok || r.X+r.W > 601 || g.cropDraft.Aspect != .375 {
		t.Fatalf("crop did not use window aspect: %+v aspect=%g", r, g.cropDraft.Aspect)
	}
	tt.Press(r.X+r.W/2, r.Y+r.H/2)
	tt.Move(r.X+r.W/2+24, r.Y+r.H/2)
	tt.Release(r.X+r.W/2+24, r.Y+r.H/2)
	if g.cropDraft.X >= .5 {
		t.Fatal("crop drag did not move the crop")
	}
	tt.Scroll(r.X+r.W/2, r.Y+r.H/2, 0, -40)
	if g.cropDraft.Zoom <= 1 {
		t.Fatal("crop wheel did not zoom")
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Pointer effect settings"); err != nil {
		t.Fatal(err)
	}
	r, ok = tt.Find("Pointer effect preview")
	if !ok || g.pointerPreview == nil || r.X+r.W > 601 {
		t.Fatal("pointer sandbox is not locally reachable")
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if err := tt.Click("Pointer effect settings"); err != nil {
		t.Fatal(err)
	}
	if g.pointerPreview != nil || len(*saved) != 0 {
		t.Fatal("closing the sandbox retained its engine or saved a draft")
	}
}

func TestBackgroundEffectModalDraftIsolationSwatchesAndReset(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	g, saves, _ := wallpaperFixture(t)
	g.Prefs.Blur = 35
	g.Prefs.HideSidebarMask = true
	g.Prefs.Preset = "ocean"
	g.Prefs.Variant = "rays"
	g.Prefs.Speed = 120
	g.Prefs.Count = 3
	g.Saved = g.Prefs.Clone()
	path := ""
	a := &AppearanceAccess{Background: g, WallpaperPath: &path, OnBrowseWallpaper: func() {}}

	// Swatches resolve empty base/default slots via theme defaults rather than Accent.
	lightDefaults := chrome.GradientDefaultColors(false)
	darkDefaults := chrome.GradientDefaultColors(true)
	if got := backgroundPresetSwatches("default", [7]string{}, false); len(got) != 7 || got[0] != lightDefaults[0] {
		t.Fatalf("light default swatches = %+v, want %+v", got, lightDefaults)
	}
	if got := backgroundPresetSwatches("aurora", [7]string{}, true); got[0] != darkDefaults[0] || got[2] != "#34d399" {
		t.Fatalf("dark aurora swatches = %+v", got)
	}
	custom := [7]string{"#112233", "", "#445566", "", "", "", ""}
	if got := backgroundPresetSwatches("custom", custom, false); got[0] != "#112233" || got[1] != lightDefaults[1] {
		t.Fatalf("custom swatches = %+v", got)
	}

	tt := ui.NewTester(func(c *ui.Context) {
		appearancePane(c, Model{Appearance: a, ThemeID: "deepseek.light"}, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 1800)
	if err := tt.Click("Background effect settings"); err != nil {
		t.Fatalf("open background effect modal: %v", err)
	}
	tt.Frame()
	for _, want := range []string{"Base start", "Base end", "Bloom 1", "Bloom 5", "120%"} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in background effect modal", want)
		}
	}

	// Select Sunset in draft and cancel: zero saves, saved prefs unchanged.
	if err := tt.Click("Sunset"); err != nil {
		t.Fatalf("click Sunset: %v", err)
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatalf("click Cancel: %v", err)
	}
	if len(*saves) != 0 || g.Prefs.Preset != "ocean" {
		t.Fatalf("cancel modified prefs: saves=%d preset=%q", len(*saves), g.Prefs.Preset)
	}

	// Re-open, Reset restores only effect defaults while preserving Blur/HideSidebarMask, Save writes once.
	if err := tt.Click("Background effect settings"); err != nil {
		t.Fatalf("reopen background effect modal: %v", err)
	}
	var resetRect ui.Rect
	var foundReset bool
	for _, b := range tt.Interactives() {
		if b.Label == "Reset" {
			resetRect = b.Rect
			foundReset = true
		}
	}
	if !foundReset {
		t.Fatal("Reset button not found in background effect modal")
	}
	tt.ClickAt(resetRect.X+resetRect.W/2, resetRect.Y+resetRect.H/2)
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("click Save: %v", err)
	}
	if len(*saves) != 1 {
		t.Fatalf("expected 1 save, got %d", len(*saves))
	}
	got := (*saves)[0]
	if got.Preset != "default" || got.Speed != 190 || got.Count != 5 || got.Variant != "orbs" {
		t.Fatalf("reset + save effect fields = %+v", got)
	}
	if got.Blur != 35 || !got.HideSidebarMask {
		t.Fatalf("reset clobbered unrelated wallpaper prefs: blur=%v hideMask=%v", got.Blur, got.HideSidebarMask)
	}
}

func TestBackgroundEffectModalExternalHostRefreshAndCustomTransition(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	g, saves, _ := wallpaperFixture(t)
	g.Prefs.Preset = "aurora"
	g.Prefs.Variant = "orbs"
	g.Prefs.Speed = 190
	g.Prefs.Count = 5
	g.Saved = g.Prefs.Clone()
	path := ""
	a := &AppearanceAccess{Background: g, WallpaperPath: &path, OnBrowseWallpaper: func() {}}

	tt := ui.NewTester(func(c *ui.Context) {
		appearancePane(c, Model{Appearance: a, ThemeID: "deepseek.light"}, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 1800)
	if err := tt.Click("Background effect settings"); err != nil {
		t.Fatalf("open background effect modal: %v", err)
	}
	if err := tt.Click("Sunset"); err != nil {
		t.Fatalf("select Sunset: %v", err)
	}

	// Simulate an external Host theme publication arriving while the dialog is open.
	external := g.Saved.Clone()
	external.Preset = "ocean"
	external.Speed = 110
	g.AcceptHost(external)
	tt.Frame()

	if err := tt.Click("Save"); err != nil {
		t.Fatalf("save background effect modal: %v", err)
	}
	if len(*saves) != 1 || (*saves)[0].Preset != "sunset" || (*saves)[0].Speed != 190 {
		t.Fatalf("external AcceptHost clobbered open modal draft: saves=%+v", *saves)
	}

	// Editing a color slot or slider transitions preset to "custom" on save.
	if err := tt.Click("Background effect settings"); err != nil {
		t.Fatalf("reopen background effect modal: %v", err)
	}
	tt.Frame()
	if err := tt.Click("Animation style"); err != nil {
		t.Fatalf("open Animation style select: %v", err)
	}
	if err := tt.Click("Chaos"); err != nil {
		t.Fatalf("select Chaos variant: %v", err)
	}
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("save custom variant: %v", err)
	}
	if len(*saves) != 2 || (*saves)[1].Variant != "chaos" || (*saves)[1].Preset != "custom" {
		t.Fatalf("variant edit did not mark custom preset: %+v", (*saves)[1])
	}
}

func TestWallpaperPrecedenceHighGlassHintAndTransparentBlurNudge(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	g, saves, _ := wallpaperFixture(t)
	var raw bytes.Buffer
	_ = png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 16, 16)))
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw.Bytes())
	hasImg := false
	g.ImageSource = func() string {
		if hasImg {
			return dataURL
		}
		return ""
	}
	g.Prefs.Effect = "gradient"
	g.Prefs.EffectExplicit = true
	g.Prefs.Glass = 90
	g.Prefs.Blur = 5
	g.Saved = g.Prefs.Clone()

	path := ""
	a := &AppearanceAccess{Background: g, WallpaperPath: &path, OnBrowseWallpaper: func() {}}
	tt := ui.NewTester(func(c *ui.Context) {
		appearancePane(c, Model{Appearance: a, ThemeID: "deepseek.light"}, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 1800)
	if tt.HasText("Enabled; the current wallpaper takes precedence.") {
		t.Fatal("wallpaper precedence hint appeared without wallpaper")
	}

	// Set wallpaper: pauses gradient hint + high glass (>=90%) hint appear.
	hasImg = true
	path = dataURL
	tt.Frame()
	if !tt.HasText("Enabled; the current wallpaper takes precedence.") {
		t.Fatal("missing wallpaper precedence hint while gradient & wallpaper are both set")
	}
	if !tt.HasText("High glass opacity covers the wallpaper. Lower Glass opacity to let more of the image show through.") {
		t.Fatal("missing >=90% glass opacity warning hint")
	}

	// Lower glass below 90%: warning disappears.
	g.changePreference(func(p *wallpapers.Preferences) { p.Glass = 85 })
	tt.Frame()
	if tt.HasText("High glass opacity covers the wallpaper. Lower Glass opacity to let more of the image show through.") {
		t.Fatal("glass opacity hint remained when Glass < 90")
	}

	// Clear wallpaper: gradient precedence hint disappears.
	hasImg = false
	path = ""
	g.ClearImage()
	tt.Frame()
	if tt.HasText("Enabled; the current wallpaper takes precedence.") {
		t.Fatal("precedence hint remained after clearing wallpaper")
	}

	// Transparent theme one-shot Blur >= 20% nudge:
	// (1) Without wallpaper, turning Transparent ON keeps Blur=5.
	g.changePreference(func(p *wallpapers.Preferences) { p.Transparent = true })
	if g.Saved.Blur != 5 {
		t.Fatalf("transparent without wallpaper nudged blur to %v", g.Saved.Blur)
	}
	// (2) Adding a wallpaper via applyCrop while Transparent is ON nudges Blur to 20.
	g.previewSource = image.NewRGBA(image.Rect(0, 0, 32, 18))
	g.cropDraft = wallpapers.DefaultCrop()
	g.cropOpen = true
	g.applyCrop()
	if g.Saved.Blur != 20 {
		t.Fatalf("applyCrop with Transparent=true did not nudge Blur from 5 to 20, got %v", g.Saved.Blur)
	}
	// (3) User can subsequently lower Blur below 20 without re-clamping, and UI shows hint.
	hasImg = true
	path = dataURL
	g.changePreference(func(p *wallpapers.Preferences) { p.Blur = 8 })
	tt.Frame()
	if g.Saved.Blur != 8 {
		t.Fatalf("subsequent Blur=8 was re-clamped to %v", g.Saved.Blur)
	}
	if !tt.HasText("Frosted glass is below 20%, so text sits directly on the sharp wallpaper and may be hard to read. Raise the frosted-glass slider.") {
		t.Fatal("missing transparent <20% blur readability hint")
	}
	// (4) Toggling Transparent ON while wallpaper is already set with Blur < 20 nudges Blur to 20.
	g.changePreference(func(p *wallpapers.Preferences) { p.Transparent = false; p.Blur = 4 })
	g.changePreference(func(p *wallpapers.Preferences) { p.Transparent = true })
	if g.Saved.Blur != 20 || len(*saves) == 0 {
		t.Fatalf("turning Transparent ON over existing wallpaper did not nudge Blur to 20: %v", g.Saved.Blur)
	}
}

func TestWallpaperGalleryTabsFavoritesConfirmCancelAndSourceCRUD(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	g, saves, applied := wallpaperFixture(t)
	g.Items = []wallpapers.Item{
		{ID: "img-1", SourceID: "bing", Title: "Blue Coast", Copyright: "Bing", ThumbURL: "https://example.com/t1.jpg", ImageURL: "https://example.com/1.jpg"},
	}
	g.Source = "bing"
	g.filterKey = "bing\n\n100\n最近"
	g.galleryOpen = true

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { wallpaperBackgroundPane(c, g, c.Theme().Text, c.Theme().TextMuted) })
	}, 960, 1200)

	// Tabs include sources + Favorites, and grid renders item.
	if !tt.HasText("Bing") || !tt.HasText("Wallhaven") || !tt.HasText("☆ Favorites") || !tt.HasText("Blue Coast") {
		t.Fatal("gallery missing expected tabs or item")
	}

	// Favorite toggle saves item and makes it searchable under Favorites.
	if err := tt.Click("☆"); err != nil {
		t.Fatalf("click Favorite star: %v", err)
	}
	if len(*saves) != 1 || len(g.Saved.Favorites) != 1 {
		t.Fatalf("favorite not persisted: saves=%d favs=%d", len(*saves), len(g.Saved.Favorites))
	}

	// Click card opens confirm dialog; Cancel keeps gallery open and does not apply.
	if err := tt.Click("Blue Coast"); err != nil {
		t.Fatalf("click item card: %v", err)
	}
	if !tt.HasText("Use this wallpaper?") {
		t.Fatal("missing confirm wallpaper dialog")
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatalf("click Cancel on confirm dialog: %v", err)
	}
	if !g.galleryOpen || len(*applied) != 0 {
		t.Fatalf("cancel closed gallery or applied wallpaper: galleryOpen=%v applied=%d", g.galleryOpen, len(*applied))
	}

	// Manage sources: add rejects private IP, rejects duplicate URL, accepts public HTTPS, edit & delete work.
	if err := tt.Click("Manage sources"); err != nil {
		t.Fatalf("open Manage sources: %v", err)
	}
	if err := tt.Click("Add source"); err != nil {
		t.Fatalf("open Add source: %v", err)
	}
	if err := tt.SetValue("Source name", "PrivateNet"); err != nil {
		t.Fatalf("type source name: %v", err)
	}
	if err := tt.SetValue("Catalog URL", "https://127.0.0.1/walls.json"); err != nil {
		t.Fatalf("type private IP URL: %v", err)
	}
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("save private IP source: %v", err)
	}
	if !g.sourceEditOpen || g.sourceError == "" {
		t.Fatal("expected private IP catalog URL to be rejected")
	}

	// Valid public HTTPS catalog succeeds.
	if err := tt.SetValue("Catalog URL", "https://example.com/walls.json"); err != nil {
		t.Fatalf("type public URL: %v", err)
	}
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("save public source: %v", err)
	}
	if g.sourceEditOpen || len(g.Saved.Sources) != 3 {
		t.Fatalf("expected new catalog source to save, open=%v sources=%+v", g.sourceEditOpen, g.Saved.Sources)
	}

	// Adding a duplicate catalog URL is rejected.
	if err := tt.Click("Add source"); err != nil {
		t.Fatalf("reopen Add source: %v", err)
	}
	if err := tt.SetValue("Source name", "Dup"); err != nil {
		t.Fatalf("type dup name: %v", err)
	}
	if err := tt.SetValue("Catalog URL", "https://example.com/walls.json"); err != nil {
		t.Fatalf("type dup URL: %v", err)
	}
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("save dup source: %v", err)
	}
	if !g.sourceEditOpen || g.sourceError == "" {
		t.Fatal("expected duplicate catalog URL to be rejected")
	}
}
