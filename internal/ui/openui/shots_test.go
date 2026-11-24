package openui

import (
	"image/png"
	"os"
	"testing"

	"whaleisle/internal/openui/card"
)

// TestShots writes the gallery in light and dark to $OPENUI_SHOTS for review.
func TestShots(t *testing.T) {
	dir := os.Getenv("OPENUI_SHOTS")
	if dir == "" {
		t.Skip()
	}
	for name, g := range gallery {
		for _, dark := range []bool{false, true} {
			h := newHarness(t, g.program, card.Config{})
			h.dark = dark
			h.tt.SetSize(600, 1400)
			h.frame()
			h.frame()
			suffix := "-light"
			if dark {
				suffix = "-dark"
			}
			f, _ := os.Create(dir + "/" + name + suffix + ".png")
			png.Encode(f, h.tt.Image())
			f.Close()
		}
	}
}
