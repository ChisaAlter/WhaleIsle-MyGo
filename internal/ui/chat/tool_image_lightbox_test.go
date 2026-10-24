package chat

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestToolImageLightboxFitsModalAndRestoresOpener(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(oldLocale) })
	path := toolImageViewBitmap(t, 960, 540)
	displayed := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			if ui.Button(c, "Image opener").Clicked() {
				displayed = path
			}
			ui.Button(c, "Behind image")
		})
		ImageLightboxOverlay(c, displayed, theme.DefaultID, false, func() { displayed = "" })
	}, 640, 480)
	tt.SetTime(time.Unix(12100, 0))
	tt.SetTitleBar(ui.TitleBar{Height: 48})
	if err := tt.Click("Image opener"); err != nil {
		t.Fatal(err)
	}
	tt.SetTime(time.Unix(12100, 0).Add(time.Second))
	panel, ok := tt.Find("Original image preview")
	if !ok || panel.W != 560 || panel.H != 315 || panel.X != 40 || panel.Y != 106.5 {
		t.Fatal("lightbox fit", panel)
	}
	close, ok := tt.Find("Close original image preview")
	if !ok || close.W != 36 || close.H != 36 || close.X != 584 || close.Y != 68 {
		t.Fatal("caption-safe close", close)
	}
	if !tt.Focused("Close original image preview") {
		t.Fatal("modal did not focus close")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Close original image preview") {
		t.Fatal("Tab left image dialog")
	}
	tt.Key(0, ui.KeyEscape)
	if displayed != "" || tt.HasText("Original image preview") || !tt.Focused("Image opener") {
		t.Fatal("Escape/restore", displayed, tt.Texts())
	}
	if err := tt.Click("Image opener"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Close original image preview"); err != nil {
		t.Fatal(err)
	}
	if displayed != "" {
		t.Fatal("close control did not dismiss")
	}
	if err := tt.Click("Image opener"); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(10, 300)
	if displayed != "" {
		t.Fatal("backdrop did not dismiss")
	}
	bitmap := loadChatBitmap(path)
	for _, tc := range []struct{ width, height, w, h float32 }{{2200, 1400, 960, 540}, {400, 300, 320, 180}, {80, 80, 0, 0}} {
		w, h := imageLightboxSize(bitmap, tc.width, tc.height)
		if w != tc.w || h != tc.h {
			t.Fatal("natural limit", tc, w, h)
		}
	}
}
