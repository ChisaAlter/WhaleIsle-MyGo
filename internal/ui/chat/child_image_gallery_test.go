package chat

import (
	"github.com/egoist/mygo/ui"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestChildImageGalleryScopedNavigation(t *testing.T) {
	root := t.TempDir()
	paths := []string{}
	for _, name := range []string{"one.png", "two.png"} {
		p := filepath.Join(root, name)
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		im := image.NewRGBA(image.Rect(0, 0, 8, 6))
		im.Set(0, 0, color.White)
		if err = png.Encode(f, im); err != nil {
			t.Fatal(err)
		}
		f.Close()
		paths = append(paths, p)
	}
	reads := map[string]session.AgentResourceRead{"attachment:a": {Path: paths[0], EOF: true}, "attachment:b": {Path: paths[1], EOF: true}, "remote": {Path: "https://invalid.test/image.png", EOF: true}}
	scope := "host/child/token1"
	tt := ui.NewTester(func(c *ui.Context) {
		ReadOnlyResourceResultsView(c, ReadOnlyResourceOptions{Scope: scope, Reads: reads}, ToolCardStyle{})
	}, 1100, 900)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	click(i18n.T(i18n.KeyChatImageEnlarge) + ": attachment:a")
	if !tt.HasText("1 / 2 · attachment:a") {
		t.Fatal("gallery did not open only local images")
	}
	click(i18n.T(i18n.KeyChatImageNext))
	if !tt.HasText("2 / 2 · attachment:b") {
		t.Fatal("next image failed")
	}
	click("1:1")
	click(i18n.T(i18n.KeyChatImageFit))
	click(i18n.T(i18n.KeyChatImagePrev))
	scope = "host/other-child/token2"
	tt.Frame()
	if tt.HasText(i18n.T(i18n.KeyChatImageCloseOne)) {
		t.Fatal("viewer crossed child ownership")
	}
	click(i18n.T(i18n.KeyChatImageEnlarge) + ": attachment:a")
	delete(reads, "attachment:a")
	tt.Frame()
	if tt.HasText(i18n.T(i18n.KeyChatImageCloseOne)) {
		t.Fatal("revoked image remained visible")
	}
	click(i18n.T(i18n.KeyChatImageEnlarge) + ": attachment:b")
	tt.Key(0, ui.KeyEscape)
	if tt.HasText(i18n.T(i18n.KeyChatImageCloseOne)) {
		t.Fatal("Escape did not close modal")
	}
}
