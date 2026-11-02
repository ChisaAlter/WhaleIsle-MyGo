package frame

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func TestConversationOfficialHeroUsesSourceAlphaAndCurrentInk(t *testing.T) {
	// Build the independent oracle from the original embedded asset's file,
	// using only its alpha, rather than calling the production tint helper.
	file, err := os.Open("../chrome/assets/whale-head.png")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	source, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, dark := range []bool{false, true} {
		colors := theme.Resolve("system").Active(dark)
		ink, background := chrome.Color(colors.FG), chrome.Color(colors.BG)
		oracle := image.NewNRGBA(source.Bounds())
		for y := 0; y < source.Bounds().Dy(); y++ {
			for x := 0; x < source.Bounds().Dx(); x++ {
				_, _, _, a := source.At(x, y).RGBA()
				oracle.SetNRGBA(x, y, color.NRGBA{R: ink.R, G: ink.G, B: ink.B, A: uint8(uint32(a) * uint32(ink.A) / 65535)})
			}
		}
		expectedBitmap := ui.NewBitmap(oracle)
		actual := ui.NewTester(func(c *ui.Context) {
			ui.Box(c).Fill().Background(background).Children(func() { conversationHeroFish(c, ink) })
		}, 80, 60)
		expected := ui.NewTester(func(c *ui.Context) {
			ui.Box(c).Fill().Background(background).Children(func() { ui.Image(c, expectedBitmap).Size(34, 34).Shrink(0).Fit(ui.Contain) })
		}, 80, 60)
		r, ok := actual.Find("Conversation hero fish")
		if !ok || r.W != 34 || r.H != 34 {
			t.Fatalf("dark%v official mark=%+v want34square", dark, r)
		}
		pixels := actual.Image()
		want := expected.Image()
		visiblePixels := 0
		for y := 0; y < pixels.Bounds().Dy(); y++ {
			for x := 0; x < pixels.Bounds().Dx(); x++ {
				got := pixels.RGBAAt(x, y)
				if got != want.RGBAAt(x, y) {
					t.Fatalf("dark%v official alpha silhouette differs at%d,%d got%+v want%+v", dark, x, y, got, want.RGBAAt(x, y))
				}
				if x < 34 && y < 34 && (got.R != background.R || got.G != background.G || got.B != background.B) {
					visiblePixels++
				}
				if x >= 34 || y >= 34 {
					if got.R != background.R || got.G != background.G || got.B != background.B {
						t.Fatal("mark painted beyond its34DIP slot")
					}
				}
			}
		}
		if visiblePixels < 20 {
			t.Fatalf("dark%v only%d visible mask pixels", dark, visiblePixels)
		}
		if dir := os.Getenv("OFFICIAL_HERO_SNAPSHOT_DIR"); dir != "" {
			name := "official-hero-light.png"
			if dark {
				name = "official-hero-dark.png"
			}
			out, err := os.Create(dir + string(os.PathSeparator) + name)
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(out, pixels); err != nil {
				out.Close()
				t.Fatal(err)
			}
			if err = out.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}
