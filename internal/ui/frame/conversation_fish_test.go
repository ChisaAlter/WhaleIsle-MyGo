package frame

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func TestConversationFishUsesOriginalGeometryAndPrimaryInk(t *testing.T) {
	for _, dark := range []bool{false, true} {
		colors := theme.Resolve("system").Active(dark)
		ink := chrome.Color(colors.FG)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Box(c).Fill().Background(chrome.Color(colors.BG)).Children(func() { conversationHeroFish(c, ink) })
		}, 80, 60)
		r, ok := tt.Find("Conversation hero fish")
		if !ok || r.W != 34 || math.Abs(float64(r.H-34*17.04/23.16)) > 0.001 {
			t.Fatalf("dark%v fish=%+v does not match source viewBox", dark, r)
		}
		primaryPixels := 0
		for y := 0; y < int(r.H); y++ {
			for x := 0; x < int(r.W); x++ {
				pixel := tt.Image().RGBAAt(x, y)
				if pixel.R == ink.R && pixel.G == ink.G && pixel.B == ink.B {
					primaryPixels++
				}
			}
		}
		if primaryPixels < 20 {
			t.Fatalf("dark%v fish painted only %d primary-ink pixels", dark, primaryPixels)
		}
	}
}
