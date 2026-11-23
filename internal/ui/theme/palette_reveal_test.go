package theme

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// bgSteps reads the window background on every 16 ms frame from t.
func bgSteps(tt *ui.Tester, from time.Time, frames int, read func() Colors) []ui.Color {
	out := make([]ui.Color, frames)
	for k := range out {
		tt.SetTime(from.Add(time.Duration(k) * 16 * time.Millisecond))
		out[k] = Color(read().BG)
	}
	return out
}

// A switched palette reaches the window over the same 200 ms the rest of the
// interface moves in, so a theme switch does not jump every color in one frame.
func TestPaletteRevealMovesTheWindowOverTheSameTime(t *testing.T) {
	light, dark := Resolve(DefaultID).Light, Resolve(DefaultID).Dark
	if light.BG == dark.BG {
		t.Fatal("both schemes resolved to one background")
	}
	EnablePaletteReveal()
	t.Cleanup(func() { paletteReveal = paletteSwitch{} })
	darkScheme := false
	var shown Colors
	tt := ui.NewTester(func(c *ui.Context) {
		RevealFrame(c)
		shown = Resolve(DefaultID).Active(darkScheme)
		ui.Box(c).Fill().Background(Color(shown.BG))
	}, 200, 200)
	if got := Color(shown.BG); got != Color(light.BG) {
		t.Fatalf("first palette arrived as %v, want the light one", got)
	}
	base := time.Unix(9000, 0)
	tt.SetTime(base)
	darkScheme = true
	steps := bgSteps(tt, base, 14, func() Colors { return shown })
	if steps[0] != Color(light.BG) {
		t.Fatalf("switch started at %v, want the light background", steps[0])
	}
	between := 0
	for k := 1; k < len(steps); k++ {
		if steps[k] != steps[k-1] && steps[k] != Color(dark.BG) {
			between++
		}
	}
	if between < 4 {
		t.Fatalf("switch took %d frames between the two palettes: %v", between, steps)
	}
	if steps[len(steps)-1] != Color(dark.BG) {
		t.Fatalf("switch ended at %v, want the dark background", steps[len(steps)-1])
	}
}

// Without the app asking for it, a palette switch stays in one frame, so a
// process that reads exact palette values is unaffected.
func TestPaletteSwitchStaysInOneFrameByDefault(t *testing.T) {
	paletteReveal = paletteSwitch{}
	resolved := Resolve(DefaultID)
	if got := resolved.Active(false); got != resolved.Light {
		t.Fatal("light scheme did not resolve to the light palette")
	}
	if got := resolved.Active(true); got != resolved.Dark {
		t.Fatal("dark scheme did not resolve to the dark palette")
	}
}

// A palette mixed part way is the color part way between the two, alpha
// included, so a switch through a transparent token keeps its hue.
func TestMixTokenMovesEachChannelAndAlpha(t *testing.T) {
	if got := Color(mixToken("#000000", "#ffffff", .5)); got != Color("#808080") {
		t.Fatalf("half way to white is %v", got)
	}
	if got := Color(mixToken("rgba(0, 0, 0, 0)", "rgba(255, 0, 0, 1)", .5)); got != Color("rgba(255, 0, 0, 0.5)") {
		t.Fatalf("half way from transparent is %v", got)
	}
	if got := mixToken("#123456", "#123456", .4); got != "#123456" {
		t.Fatalf("an unchanged token became %q", got)
	}
}

// A landed switch stays landed: frames that keep asking for the palette they
// show start no switch, so they ask for no animation frames. The defect this
// pins restarted a switch from a palette to itself after every landing, and
// any frame within 200 ms of the last kept the window animating.
func TestPaletteRevealDoesNotRestartOnTheSamePalette(t *testing.T) {
	EnablePaletteReveal()
	t.Cleanup(func() { paletteReveal = paletteSwitch{} })
	paletteReveal = paletteSwitch{enabled: true}
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		RevealFrame(c)
		ui.Box(c).Fill().Background(Color(Resolve(DefaultID).Active(false).BG))
	}, 100, 100)
	base := time.Unix(9000, 0)
	for k := 0; k < 20; k++ {
		builds = 0
		// A frame every 40 ms, as the boot page's tick asks for.
		tt.SetTime(base.Add(time.Duration(k) * 40 * time.Millisecond))
		if builds > 1 || paletteReveal.on {
			t.Fatalf("frame %d: an unchanged palette is switching (on=%v, %d builds)", k, paletteReveal.on, builds)
		}
	}
}
