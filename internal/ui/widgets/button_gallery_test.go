package widgets

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// TestButtonGallery writes the button recipes for review. It runs only when
// BUTTON_GALLERY_DIR names a directory, so the suite stays headless otherwise.
func TestButtonGallery(t *testing.T) {
	dir := os.Getenv("BUTTON_GALLERY_DIR")
	if dir == "" {
		t.Skip("set BUTTON_GALLERY_DIR to render the button gallery")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, half := range []struct {
		name   string
		colors theme.Colors
		dark   bool
	}{
		{"light", theme.Resolve(theme.IDDeepseekLight).Light, false},
		{"dark", theme.Resolve(theme.IDDeepseekDark).Dark, true},
	} {
		colors, dark := half.colors, half.dark
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(24).Gap(16).Children(func() {
				ui.Text(c, "Primary / Ghost / Outline / Toolbar / Info").FontSize(12).TextColor(theme.Color(colors.Muted))
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					for _, variant := range []ButtonVariant{ButtonPrimary, ButtonGhost, ButtonOutline, ButtonToolbar, ButtonInfo} {
						Button(c, colors, "Continue", ButtonSpec{Variant: variant})
					}
				})
				ui.Text(c, "Small and disabled").FontSize(12).TextColor(theme.Color(colors.Muted))
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					Button(c, colors, "Cancel", ButtonSpec{Variant: ButtonGhost, Size: ButtonSmall})
					Button(c, colors, "Save", ButtonSpec{Variant: ButtonPrimary, Size: ButtonSmall})
					Button(c, colors, "Remove", ButtonSpec{Variant: ButtonOutline, Size: ButtonSmall})
					Button(c, colors, "Disabled", ButtonSpec{Variant: ButtonPrimary, Disabled: true})
				})
			})
		}, 560, 240)
		if dark {
			tt.SetDark(true)
		}
		file, err := os.Create(filepath.Join(dir, "buttons-"+half.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, tt.Image()); err != nil {
			file.Close()
			t.Fatal(err)
		}
		file.Close()
	}
}
