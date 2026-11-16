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

func TestWallpaperFreezeStatesStopPhaseAndScheduling(t *testing.T) {
	t.Cleanup(func() {
		resetBackdropPowerForTest()
		ConfigureBackdrop("", wallpapers.Defaults())
	})

	prefs := testGradientPrefs()
	paint := WallpaperBackgroundPainter("", prefs, false)
	now := time.Now()

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(paint)
	}, 160, 100)
	tt.SetTime(now)

	waitGradientTickAdvance(t, &ambientGradientFrames, tt, 1, &now)
	waitGradientIdleSettled(t, &ambientGradientFrames)

	assertFrozenHolds := func(name string, activate func()) {
		t.Helper()
		activate()
		tt.Frame()
		ambientGradientFrames.mu.Lock()
		frozenTick := ambientGradientFrames.frame.tick
		ambientGradientFrames.mu.Unlock()
		frozenPix := bytes.Clone(tt.Image().Pix)

		now = now.Add(200 * time.Millisecond)
		tt.SetTime(now)
		time.Sleep(25 * time.Millisecond)
		tt.Frame()

		ambientGradientFrames.mu.Lock()
		gotTick := ambientGradientFrames.frame.tick
		rendering := ambientGradientFrames.rendering
		ambientGradientFrames.mu.Unlock()
		if rendering {
			t.Fatalf("%s: frozen gradient unexpectedly spawned render worker", name)
		}
		if gotTick != frozenTick {
			t.Fatalf("%s: frozen gradient tick changed from %d to %d", name, frozenTick, gotTick)
		}
		if !bytes.Equal(frozenPix, tt.Image().Pix) {
			t.Fatalf("%s: frozen gradient pixels changed while frozen", name)
		}
	}

	assertFrozenHolds("manual-freeze", func() { SetBackdropFrozen(true) })
	SetBackdropFrozen(false)

	assertFrozenHolds("low-power", func() { SetLowPowerMode(true) })
	SetLowPowerMode(false)

	assertFrozenHolds("unfocused-or-minimized", func() {
		setBackdropWindowActiveHook(func() bool { return false })
	})
	setBackdropWindowActiveHook(nil)

	ambientGradientFrames.mu.Lock()
	beforeResumeTick := ambientGradientFrames.frame.tick
	ambientGradientFrames.mu.Unlock()
	waitGradientTickAdvance(t, &ambientGradientFrames, tt, beforeResumeTick+1, &now)
}

func TestWallpaperAndPreviewIdleFreezeAt10sAndResumeOnActivity(t *testing.T) {
	t.Cleanup(func() {
		resetBackdropPowerForTest()
		ConfigureBackdrop("", wallpapers.Defaults())
	})

	prefs := testGradientPrefs()
	paint := WallpaperBackgroundPainter("", prefs, false)
	now := time.Now()

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(paint)
	}, 160, 100)
	tt.SetTime(now)
	waitGradientTickAdvance(t, &ambientGradientFrames, tt, 1, &now)
	waitGradientIdleSettled(t, &ambientGradientFrames)

	// Jump past idle threshold (10s).
	now = now.Add(ambientIdleFreezeDelay + 50*time.Millisecond)
	tt.SetTime(now)
	waitGradientIdleSettled(t, &ambientGradientFrames)

	ambientGradientFrames.mu.Lock()
	idleTick := ambientGradientFrames.frame.tick
	ambientGradientFrames.mu.Unlock()
	idlePix := bytes.Clone(tt.Image().Pix)

	now = now.Add(300 * time.Millisecond)
	tt.SetTime(now)
	time.Sleep(25 * time.Millisecond)
	tt.Frame()

	ambientGradientFrames.mu.Lock()
	gotTick := ambientGradientFrames.frame.tick
	ambientGradientFrames.mu.Unlock()
	if gotTick != idleTick || !bytes.Equal(idlePix, tt.Image().Pix) {
		t.Fatalf("idle wallpaper kept advancing after %v (tick %d -> %d)", ambientIdleFreezeDelay, idleTick, gotTick)
	}

	// Activity resumes animation cleanly.
	NoteBackdropActivity(now)
	waitGradientTickAdvance(t, &ambientGradientFrames, tt, idleTick+1, &now)

	// Verify BackgroundEffectPreview obeys freeze, low-power, zero bounds, and idle.
	resetBackdropPowerForTest()
	preview := &BackgroundEffectPreview{}
	t.Cleanup(preview.Close)
	previewPaint := preview.Draw(prefs, false)
	pNow := time.Now()
	pt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(previewPaint)
	}, 160, 100)
	pt.SetTime(pNow)
	waitGradientTickAdvance(t, &preview.frames, pt, 1, &pNow)
	waitGradientIdleSettled(t, &preview.frames)

	SetLowPowerMode(true)
	pt.Frame()
	preview.frames.mu.Lock()
	pFrozenTick := preview.frames.frame.tick
	preview.frames.mu.Unlock()
	pFrozenPix := bytes.Clone(pt.Image().Pix)
	pNow = pNow.Add(200 * time.Millisecond)
	pt.SetTime(pNow)
	time.Sleep(25 * time.Millisecond)
	pt.Frame()
	preview.frames.mu.Lock()
	pAfterTick := preview.frames.frame.tick
	preview.frames.mu.Unlock()
	if pAfterTick != pFrozenTick || !bytes.Equal(pFrozenPix, pt.Image().Pix) {
		t.Fatalf("preview advanced during low-power freeze: tick %d -> %d", pFrozenTick, pAfterTick)
	}
	SetLowPowerMode(false)

	// Zero-sized bounds must be a clean no-op without panicking.
	zeroPaint := WallpaperBackgroundPainter("", prefs, false)
	zt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(0, 0).Draw(zeroPaint)
	}, 80, 60)
	_ = zt.Image()
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
	tt.SetTime(now)
	if !enabled {
		t.Fatal("expected ApplyCursorEffects to enable pointer effects when active")
	}
	if !backdropActivityAt().Equal(now) {
		t.Fatalf("activity timestamp = %v, want %v", backdropActivityAt(), now)
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
