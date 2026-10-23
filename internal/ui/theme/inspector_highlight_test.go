package theme

import "testing"

func TestInspectorHighlightFollowsCustomAccentContrastAndOverride(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, contrast := range []float64{0, 50, 100} {
			seed := Seed{Background: "#ffffff", Foreground: "#102030", Accent: "#40668f", Contrast: &contrast}
			strong := .18 + .1*contrast/100
			if dark {
				seed.Background, seed.Foreground = "#101820", "#eef4fa"
				strong = .26 + .12*contrast/100
			}
			got := palette(seed, dark)
			if got.BubbleHighlight != mix(seed.Background, seed.Accent, strong) {
				t.Fatalf("dark%v contrast%v highlight=%v", dark, contrast, got.BubbleHighlight)
			}
			seed.Overrides = map[string]string{"--dsw-specific-bubble-highlight": "#123456"}
			if got := palette(seed, dark); got.BubbleHighlight != "#123456" {
				t.Fatal("explicit highlight override lost", got.BubbleHighlight)
			}
		}
	}
}
