package settings

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/egoist/mygo/ui"
	"image"
	"image/png"
	"os"
	"reflect"
	"testing"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/i18n"
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
