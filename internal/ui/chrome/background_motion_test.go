package chrome

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/renderpolicy"
)

func TestAmbientGradientAnimatesLiveAcrossAllVariants(t *testing.T) {
	prevGlass := renderpolicy.Glass()
	prevMinimal := renderpolicy.Minimal()
	renderpolicy.SetMinimal(false)
	renderpolicy.SetGlass(true)
	t.Cleanup(func() {
		renderpolicy.SetGlass(prevGlass)
		renderpolicy.SetMinimal(prevMinimal)
		resetBackdropPowerForTest()
		ConfigureBackdrop("", wallpapers.Defaults())
	})

	for _, variant := range []string{"orbs", "aurora", "chaos", "rays"} {
		t.Run(variant, func(t *testing.T) {
			resetBackdropPowerForTest()
			setBackdropWindowActiveHook(func() bool { return true })
			ambientGradientFrames.mu.Lock()
			rendersBefore := ambientGradientFrames.renders
			ambientGradientFrames.mu.Unlock()

			prefs := wallpapers.Defaults()
			prefs.Effect = "gradient"
			prefs.EffectExplicit = true
			prefs.Preset = "aurora"
			prefs.Variant = variant
			prefs.Speed = 190
			prefs.Count = 5

			now := time.Unix(2000, 0)
			NoteBackdropActivity(now)
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(ui.DarkTheme())
				paint := WallpaperBackgroundPainter("", prefs, false)
				ui.Box(c).Fill().Draw(paint)
			}, 160, 120)
			tt.SetTime(now)
			pix0 := bytes.Clone(tt.Image().Pix)

			t1 := now.Add(3 * time.Second)
			NoteBackdropActivity(t1)
			tt.SetTime(t1)
			pix1 := bytes.Clone(tt.Image().Pix)

			if bytes.Equal(pix0, pix1) {
				t.Fatalf("variant %s remained static across time phases", variant)
			}
			changed := 0
			for i := 0; i+3 < len(pix0); i += 4 {
				d := absInt(int(pix0[i])-int(pix1[i])) + absInt(int(pix0[i+1])-int(pix1[i+1])) + absInt(int(pix0[i+2])-int(pix1[i+2]))
				if d > 3 {
					changed++
				}
			}
			if changed < 200 {
				t.Fatalf("variant %s changed only %d pixels between t=0 and t=3s", variant, changed)
			}
			ambientGradientFrames.mu.Lock()
			rendersAfter := ambientGradientFrames.renders
			ambientGradientFrames.mu.Unlock()
			if rendersAfter != rendersBefore || gradientBackdrop.frame != nil {
				t.Fatalf("GPU ambient gradient triggered CPU render (renders=%d->%d, frame=%v)", rendersBefore, rendersAfter, gradientBackdrop.frame)
			}
		})
	}
}

func TestAmbientGradientFreezesAtZeroFPSWhenIdleReducedOrNoBackdrop(t *testing.T) {
	prevGlass := renderpolicy.Glass()
	prevMinimal := renderpolicy.Minimal()
	renderpolicy.SetMinimal(false)
	renderpolicy.SetGlass(true)
	t.Cleanup(func() {
		renderpolicy.SetGlass(prevGlass)
		renderpolicy.SetMinimal(prevMinimal)
		resetBackdropPowerForTest()
		ConfigureBackdrop("", wallpapers.Defaults())
	})

	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "sunset"
	prefs.Variant = "orbs"
	prefs.Speed = 200
	prefs.Count = 4

	resetBackdropPowerForTest()
	setBackdropWindowActiveHook(func() bool { return true })
	now := time.Unix(4000, 0)
	NoteBackdropActivity(now)
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		paint := WallpaperBackgroundPainter("", prefs, false)
		ui.Box(c).Fill().Draw(paint)
	}, 160, 120)
	tt.SetTime(now)

	// Active phase at +2s moves smoothly.
	tActive := now.Add(2 * time.Second)
	tt.SetTime(tActive)
	activePix := bytes.Clone(tt.Image().Pix)

	// Past 10s idle freeze delay: freezes in current phase at 0 fps.
	tIdle1 := now.Add(ambientIdleFreezeDelay + time.Second)
	tt.SetTime(tIdle1)
	idlePix1 := bytes.Clone(tt.Image().Pix)
	if !bytes.Equal(activePix, idlePix1) {
		t.Fatal("idle freeze jumped phase instead of freezing at current phase")
	}
	tIdle2 := now.Add(ambientIdleFreezeDelay + 6*time.Second)
	tt.SetTime(tIdle2)
	if !bytes.Equal(idlePix1, tt.Image().Pix) {
		t.Fatal("ambient gradient continued advancing past 10s idle freeze")
	}

	// Minimal mode / NoBackdrop also freezes phase.
	NoteBackdropActivity(tIdle2)
	renderpolicy.SetMinimal(true)
	tt.SetTime(tIdle2.Add(3 * time.Second))
	minPix1 := bytes.Clone(tt.Image().Pix)
	tt.SetTime(tIdle2.Add(6 * time.Second))
	if !bytes.Equal(minPix1, tt.Image().Pix) {
		t.Fatal("ambient gradient advanced while renderpolicy.NoBackdrop() was true")
	}
}

func TestBackgroundEffectPreviewAnimatesAndFreezes(t *testing.T) {
	prevGlass := renderpolicy.Glass()
	prevMinimal := renderpolicy.Minimal()
	renderpolicy.SetMinimal(false)
	renderpolicy.SetGlass(true)
	t.Cleanup(func() {
		renderpolicy.SetGlass(prevGlass)
		renderpolicy.SetMinimal(prevMinimal)
		resetBackdropPowerForTest()
	})

	resetBackdropPowerForTest()
	setBackdropWindowActiveHook(func() bool { return true })
	preview := &BackgroundEffectPreview{}
	t.Cleanup(preview.Close)

	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "ocean"
	prefs.Variant = "aurora"
	prefs.Speed = 240
	prefs.Count = 5

	now := time.Unix(6000, 0)
	NoteBackdropActivity(now)
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ui.Box(c).Fill().Draw(preview.Draw(prefs, false))
	}, 160, 100)
	tt.SetTime(now)
	p0 := bytes.Clone(tt.Image().Pix)

	t1 := now.Add(3 * time.Second)
	NoteBackdropActivity(t1)
	tt.SetTime(t1)
	p1 := bytes.Clone(tt.Image().Pix)
	if bytes.Equal(p0, p1) {
		t.Fatal("BackgroundEffectPreview remained static across time phases")
	}

	// Idle >= 10s freezes preview at current phase.
	tt.SetTime(t1.Add(ambientIdleFreezeDelay + 2*time.Second))
	frozen := bytes.Clone(tt.Image().Pix)
	tt.SetTime(t1.Add(ambientIdleFreezeDelay + 6*time.Second))
	if !bytes.Equal(frozen, tt.Image().Pix) {
		t.Fatal("BackgroundEffectPreview did not freeze after 10s idle")
	}
}
