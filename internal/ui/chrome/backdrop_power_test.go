package chrome

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
)

func testGradientPrefs() wallpapers.Preferences {
	p := wallpapers.Defaults()
	p.Effect = "gradient"
	p.EffectExplicit = true
	p.Preset = "aurora"
	p.Speed = 5
	return p
}

func waitGradientTickAdvance(t *testing.T, cache *gradientFrameCache, tt *ui.Tester, minTick int64, now *time.Time) {
	t.Helper()
	*now = now.Add(4 * gradientFrameInterval)
	tt.SetTime(*now)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		cache.mu.Lock()
		tick := int64(-1)
		rendering := cache.rendering
		if cache.frame != nil {
			tick = cache.frame.tick
		}
		cache.mu.Unlock()
		if !rendering && tick >= minTick {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("gradient tick did not advance to >= %d within timeout", minTick)
}

func waitGradientIdleSettled(t *testing.T, cache *gradientFrameCache) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cache.mu.Lock()
		rendering := cache.rendering
		cache.mu.Unlock()
		if !rendering {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("in-flight gradient worker did not settle")
}

func TestMetallicSheenFreezesToMidpointWhenInactiveOrIdle(t *testing.T) {
	t.Cleanup(resetBackdropPowerForTest)

	// Pick a timestamp whose 4.5s cosine phase is well away from 0.5 midpoint.
	now := time.Unix(2000, 0)

	staticTester := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(120, 40).Draw(func(p *ui.Painter, r ui.Rect) {
			MetallicSheen(p, r, 6, true, false)
		})
	}, 120, 40)
	staticTester.SetTime(now)
	midpointPix := bytes.Clone(staticTester.Image().Pix)

	liveTester := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(120, 40).Draw(func(p *ui.Painter, r ui.Rect) {
			MetallicSheen(p, r, 6, true, true)
		})
	}, 120, 40)
	liveTester.SetTime(now)
	if bytes.Equal(midpointPix, liveTester.Image().Pix) {
		t.Fatal("active animated sheen matched static midpoint unexpectedly")
	}

	SetBackdropFrozen(true)
	liveTester.Frame()
	if !bytes.Equal(midpointPix, liveTester.Image().Pix) {
		t.Fatal("manual backdrop freeze did not hold metallic sheen at static midpoint")
	}
	SetBackdropFrozen(false)

	SetLowPowerMode(true)
	liveTester.Frame()
	if !bytes.Equal(midpointPix, liveTester.Image().Pix) {
		t.Fatal("low-power mode did not hold metallic sheen at static midpoint")
	}
	SetLowPowerMode(false)

	setBackdropWindowActiveHook(func() bool { return false })
	liveTester.Frame()
	if !bytes.Equal(midpointPix, liveTester.Image().Pix) {
		t.Fatal("inactive window hook did not hold metallic sheen at static midpoint")
	}
	setBackdropWindowActiveHook(nil)

	NoteBackdropActivity(now)
	idleAt := now.Add(ambientIdleFreezeDelay + time.Second)
	liveTester.SetTime(idleAt)
	if !bytes.Equal(midpointPix, liveTester.Image().Pix) {
		t.Fatal("idle sheen did not hold static midpoint past idle timeout")
	}
}

func TestApplyCursorEffectsRespectsWindowActive(t *testing.T) {
	t.Cleanup(resetBackdropPowerForTest)

	p := wallpapers.Defaults()
	p.CursorEnabled = true
	p.CursorEffect = CursorEffectTrail
	accent := ui.RGB(20, 40, 200)

	var enabled bool
	now := time.Unix(5000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		enabled = ApplyCursorEffects(c, p, accent)
	}, 80, 60)
	// Start a new activity epoch after rewinding the headless clock.
	resetBackdropPowerForTest()
	tt.SetTime(now)
	if !enabled {
		t.Fatal("expected ApplyCursorEffects to enable pointer effects when active")
	}
	if !backdropActivityAt().Equal(now) {
		t.Fatalf("activity timestamp = %v, want %v", backdropActivityAt(), now)
	}
	tt.SetTime(now.Add(ambientIdleFreezeDelay + time.Second))
	tt.Frame()
	if !backdropActivityAt().Equal(now) {
		t.Fatal("a passive view rebuild renewed its own activity lease")
	}
	tt.Move(20, 20)
	if !backdropActivityAt().Equal(now.Add(ambientIdleFreezeDelay + time.Second)) {
		t.Fatal("actual pointer movement did not renew activity")
	}

	setBackdropWindowActiveHook(func() bool { return false })
	tt.Frame()
	if enabled {
		t.Fatal("expected ApplyCursorEffects to suppress pointer effects when window inactive")
	}

	setBackdropWindowActiveHook(nil)
	SetLowPowerMode(true)
	tt.Frame()
	if enabled {
		t.Fatal("expected ApplyCursorEffects to suppress pointer effects in low-power mode")
	}
}

func TestRecreatedPainterKeepsIdleEpoch(t *testing.T) {
	resetBackdropPowerForTest()
	t.Cleanup(resetBackdropPowerForTest)
	setBackdropWindowActiveHook(func() bool { return true })
	now := time.Unix(8000, 0)
	for i := 0; i < 15; i++ {
		var newPainterStart time.Time
		frozen := shouldFreezeAmbientAnimation(false, &newPainterStart, now.Add(time.Duration(i)*time.Second))
		if frozen != (i >= 10) {
			t.Fatalf("new painter at second %d frozen=%v", i, frozen)
		}
	}
	NoteBackdropActivity(now.Add(15 * time.Second))
	var newPainterStart time.Time
	if shouldFreezeAmbientAnimation(false, &newPainterStart, now.Add(15*time.Second)) {
		t.Fatal("real activity did not resume animation")
	}
}
