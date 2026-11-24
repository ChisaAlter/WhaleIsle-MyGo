package openui

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/openui/card"
)

func TestParseCSSColor(t *testing.T) {
	for in, want := range map[string]ui.Color{
		"#fff":                {R: 255, G: 255, B: 255, A: 255},
		"#10203040":           {R: 16, G: 32, B: 48, A: 64},
		"rgb(1, 2, 3)":        {R: 1, G: 2, B: 3, A: 255},
		"rgba(255 0 0 / 50%)": {R: 255, A: 127},
		" RGBA(0,0,0,1) ":     {A: 255},
	} {
		got, ok := parseCSSColor(in)
		if !ok || got != want {
			t.Errorf("%q = %v %v, want %v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "#12", "red", "rgb(1)", "url(x)"} {
		if _, ok := parseCSSColor(bad); ok {
			t.Errorf("%q parsed", bad)
		}
	}
}

func TestCelebrationRespectsReducedMotion(t *testing.T) {
	for _, reduce := range []bool{false, true} {
		h := newHarness(t, "root = Col([Celebration($done), Button(\"Finish\", @Set($done, true))])\n$done = false", card.Config{})
		h.tt.SetPreferences(ui.Preferences{ReduceMotion: reduce})
		h.click("Finish")
		h.frame()
		if v := h.card.Frame().Root.Children()[0].Prop("when"); v != true {
			t.Fatalf("when = %v", v)
		}
	}
}
