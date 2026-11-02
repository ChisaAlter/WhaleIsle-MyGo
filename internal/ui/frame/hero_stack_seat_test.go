package frame

import (
	"math"
	"testing"

	"whaleisle/internal/ui/i18n"
)

// Installed0.3.10 defines a natural flex item, not the older guessed 250-DIP
// seat: headline34+gaps20+preset40+card114, plus32 bottom padding. A taller
// viewport moves the entire item by half the height change.
func TestHeroStackKeepsTheNativeSeat(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	var previousTop float32
	for _, height := range []int{864, 920} {
		tt := chatChromeSeatTester(t, height)
		head := chatChromeSeatFind(t, tt, "Conversation hero headline")
		preset := chatChromeSeatFind(t, tt, "Composer hero preset row")
		card := chatChromeSeatFind(t, tt, "Composer card")
		body := chatChromeSeatFind(t, tt, "Conversation body")
		close := func(name string, got, want float32) {
			t.Helper()
			if math.Abs(float64(got-want)) > .1 {
				t.Fatalf("height%d %s=%.2f want%.2f", height, name, got, want)
			}
		}
		close("preset top", preset.Y, head.Y+34+20+4)
		close("preset height", preset.H, 28)
		close("card top", card.Y, preset.Y+preset.H+8)
		close("card height", card.H, 114)
		close("visual stack height", card.Y+card.H-head.Y, 208)
		close("source flex center", head.Y+(208+32)/2, body.Y+body.H/2)
		if previousTop != 0 {
			close("height reflow", head.Y-previousTop, float32(920-864)/2)
		}
		previousTop = head.Y
		if _, ok := tt.Find("conversation-hero-top-block"); ok {
			t.Fatal("obsolete hero top compensation still mounted")
		}
	}
}
