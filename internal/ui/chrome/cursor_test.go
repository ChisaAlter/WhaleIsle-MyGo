package chrome

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
)

func cursorPrefs() wallpapers.Preferences {
	p := wallpapers.Defaults()
	p.CursorEnabled = true
	p.CursorEffect = CursorEffectTrail
	p.CursorColors = [6]string{"#ff0000", "not-a-color", "", "#00ff00", "#12345", "#0a0b0c"}
	p.CursorSpeed = 5
	p.CursorSize = 999
	return p
}

// TestCursorEffectConfigStates covers the three states the pointer effect can
// be in: on, off, and suppressed by reduce motion.
func TestCursorEffectConfigStates(t *testing.T) {
	accent := ui.RGB(10, 20, 30)
	p := cursorPrefs()
	config, on := CursorEffectConfig(p, accent, false)
	if !on || !config.PointerEnabled {
		t.Fatal("an enabled trail did not turn the pointer effect on")
	}
	if config.PointerEffect != CursorEffectTrail {
		t.Fatalf("pointer effect = %q, want %q", config.PointerEffect, CursorEffectTrail)
	}
	if config.Speed != MinCursorSpeed || config.Size != MaxCursorSize {
		t.Fatalf("speed/size not clamped to the settings bounds: %v %v", config.Speed, config.Size)
	}
	if config.ColorCount != 3 || config.Colors[0] != ui.RGB(255, 0, 0) || config.Colors[1] != ui.RGB(0, 255, 0) || config.Colors[2] != ui.RGB(0x0a, 0x0b, 0x0c) {
		t.Fatalf("stored palette not kept, or invalid slots not dropped: %+v", config.Colors)
	}

	p.CursorEnabled = false
	if _, on := CursorEffectConfig(p, accent, false); on {
		t.Fatal("a disabled preference kept the pointer effect on")
	}

	p = cursorPrefs()
	p.MetallicButtons = true
	reduced, on := CursorEffectConfig(p, accent, true)
	if on || reduced.PointerEnabled {
		t.Fatal("reduce motion did not suppress the pointer effect")
	}
	if !reduced.MetallicButtons {
		t.Fatal("reduce motion should leave the stored metallic sheen alone")
	}
}

func TestCursorEffectConfigDefaults(t *testing.T) {
	accent := ui.RGB(200, 10, 20)
	p := wallpapers.Defaults()
	p.CursorEnabled = true
	config, on := CursorEffectConfig(p, accent, false)
	if !on || config.PointerEffect != CursorEffectTrail {
		t.Fatalf("missing effect id should default to the trail: %+v", config)
	}
	if config.Speed != 100 || config.Size != 100 {
		t.Fatalf("missing speed/size should default to 100: %v %v", config.Speed, config.Size)
	}
	if config.ColorCount != 1 || config.Colors[0] != accent {
		t.Fatalf("empty palette should follow the accent: %+v", config.Colors)
	}

	splash := p
	splash.CursorEffect = CursorEffectSplash
	if config, _ := CursorEffectConfig(splash, accent, false); config.PointerEffect != CursorEffectSplash {
		t.Fatalf("splash was not kept: %q", config.PointerEffect)
	}
	unknown := p
	unknown.CursorEffect = "sparkles"
	if config, _ := CursorEffectConfig(unknown, accent, false); config.PointerEffect != CursorEffectTrail {
		t.Fatalf("unknown effect should fall back to the trail: %q", config.PointerEffect)
	}

	// The renderer keeps its own grey when no theme accent resolved.
	config, _ = CursorEffectConfig(p, ui.Color{}, false)
	if config.ColorCount != 1 || config.Colors[0].A != 255 {
		t.Fatalf("no accent should still give one opaque color: %+v", config.Colors)
	}
}

// TestTrailPaintsUnderThePointer drives the trail through the renderer: moving
// the pointer leaves colored cells, and a disabled or reduced-motion effect
// leaves the window untouched.
func TestTrailPaintsUnderThePointer(t *testing.T) {
	prefs := cursorPrefs()
	prefs.CursorColors = [6]string{"#ff0000"}
	run := func(enabled, reduced bool) []byte {
		t.Helper()
		tt := ui.NewTester(func(c *ui.Context) {
			config, _ := CursorEffectConfig(prefs, ui.RGB(255, 0, 0), c.Preferences().ReduceMotion)
			config.PointerEnabled = config.PointerEnabled && enabled
			ui.SetDecorativeEffects(c, config)
			ui.Box(c).Fill().Background(ui.RGB(0, 0, 0))
		}, 80, 40)
		tt.SetPreferences(ui.Preferences{ReduceMotion: reduced})
		clock := time.Unix(1000, 0)
		tt.SetTime(clock)
		for i := 0; i < 6; i++ {
			tt.Move(float32(10+i*10), 20)
		}
		tt.SetTime(clock.Add(16 * time.Millisecond))
		return bytes.Clone(tt.Image().Pix)
	}
	trail := run(true, false)
	if !hasRedPixel(trail) {
		t.Fatal("the trail painted nothing after the pointer moved")
	}
	if off := run(false, false); hasRedPixel(off) {
		t.Fatal("a disabled pointer effect still painted")
	}
	if reduced := run(true, true); hasRedPixel(reduced) {
		t.Fatal("reduce motion still painted the trail")
	}
}

// hasRedPixel reports a pixel the trail's red palette painted.
func hasRedPixel(pix []byte) bool {
	for i := 0; i+3 < len(pix); i += 4 {
		if pix[i] > 100 && pix[i+1] < 60 && pix[i+2] < 60 {
			return true
		}
	}
	return false
}
