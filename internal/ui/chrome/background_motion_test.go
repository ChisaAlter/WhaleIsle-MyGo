package chrome

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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

func TestAmbientGradientZeroHzWhenWindowInactiveOrReduced(t *testing.T) {
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

	light := GradientDefaultColors(false)
	dark := GradientDefaultColors(true)
	if light == dark || light[0] != "#edf3fe" || dark[0] != "#283142" {
		t.Fatalf("unexpected theme default palettes: light=%+v dark=%+v", light, dark)
	}

	resetBackdropPowerForTest()
	active := true
	setBackdropWindowActiveHook(func() bool { return active })

	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "ocean"
	prefs.Variant = "chaos"
	prefs.Speed = 220
	prefs.Count = 5

	now := time.Unix(8000, 0)
	NoteBackdropActivity(now)
	reduced := false
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ui.Box(c).Fill().Draw(WallpaperBackgroundPainter("", prefs, reduced))
	}, 160, 120)
	tt.SetTime(now)
	t1 := now.Add(time.Second)
	NoteBackdropActivity(t1)
	tt.SetTime(t1)
	pixActive := bytes.Clone(tt.Image().Pix)

	// Window hidden / minimized / unfocused: freezes cleanly at 0 Hz.
	active = false
	t2 := t1.Add(2 * time.Second)
	NoteBackdropActivity(t2)
	tt.SetTime(t2)
	pixInactive := bytes.Clone(tt.Image().Pix)
	if !bytes.Equal(pixActive, pixInactive) {
		t.Fatal("ambient gradient jumped phase when window became inactive")
	}
	t3 := t2.Add(2 * time.Second)
	NoteBackdropActivity(t3)
	tt.SetTime(t3)
	if !bytes.Equal(pixInactive, tt.Image().Pix) {
		t.Fatal("ambient gradient kept advancing while window was inactive")
	}

	// Resuming active window does not jump forward over the frozen gap on its
	// first frame, then advances smoothly from the frozen phase on subsequent frames.
	active = true
	t4 := t3.Add(2 * time.Second)
	NoteBackdropActivity(t4)
	tt.SetTime(t4)
	if !bytes.Equal(pixInactive, tt.Image().Pix) {
		t.Fatal("ambient gradient jumped across frozen interval on resume")
	}
	t5 := t4.Add(2 * time.Second)
	NoteBackdropActivity(t5)
	tt.SetTime(t5)
	if bytes.Equal(pixInactive, tt.Image().Pix) {
		t.Fatal("ambient gradient failed to resume after window became active")
	}

	// Reduced motion locks to static phase regardless of time progression.
	reduced = true
	tt.Frame()
	pixReduced := bytes.Clone(tt.Image().Pix)
	t6 := t5.Add(5 * time.Second)
	NoteBackdropActivity(t6)
	tt.SetTime(t6)
	if !bytes.Equal(pixReduced, tt.Image().Pix) {
		t.Fatal("ambient gradient animated under prefers-reduced-motion")
	}
}

func TestAmbientGradientPacedAt24FPSAndFreezesDuringOverlay(t *testing.T) {
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

	resetBackdropPowerForTest()
	setBackdropWindowActiveHook(func() bool { return true })
	mainAmbientClock.reset()

	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "aurora"
	prefs.Variant = "chaos"
	prefs.Speed = 240
	prefs.Count = 5

	showPopup := false
	now := time.Unix(12000, 0)
	NoteBackdropActivity(now)
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		paint := WallpaperBackgroundPainter("", prefs, false)
		ui.Box(c).Fill().Draw(paint).Children(func() {
			if showPopup {
				MenuSurface(ui.Box(c).Absolute().Left(10).Top(10).Width(40).Height(30),
					darkGlassPopoverBase, GlassOuterBorder(true), 8)
			}
		})
	}, 160, 120)
	tt.SetTime(now)
	pix0 := bytes.Clone(tt.Image().Pix)

	// High-frequency pointer/hover repaints inside the 42ms cadence window
	// keep identical ambient gradient output so bdCache hits stay high.
	for _, dt := range []time.Duration{6 * time.Millisecond, 12 * time.Millisecond, 24 * time.Millisecond, 36 * time.Millisecond} {
		ts := now.Add(dt)
		NoteBackdropActivity(ts)
		tt.SetTime(ts)
		if !bytes.Equal(pix0, tt.Image().Pix) {
			t.Fatalf("ambient gradient changed within 42ms cadence at +%v (would thrash bdCache at 156Hz)", dt)
		}
	}

	// Advancing well past the 42ms cadence smoothly animates the backdrop.
	tMoved := now.Add(2 * time.Second)
	NoteBackdropActivity(tMoved)
	tt.SetTime(tMoved)
	if bytes.Equal(pix0, tt.Image().Pix) {
		t.Fatal("ambient gradient did not advance past 42ms cadence")
	}

	// Opening a frosted popup automatically freezes the ambient backdrop time
	// so repeated hover/scroll repaints hit bdCache instead of invalidating it.
	showPopup = true
	tOpen := tMoved.Add(100 * time.Millisecond)
	NoteBackdropActivity(tOpen)
	tt.SetTime(tOpen)
	// Sample backdrop region outside the popup (bottom half y=60..120).
	cropBottom := func(pix []byte) []byte { return bytes.Clone(pix[60*160*4:]) }
	openBg := cropBottom(tt.Image().Pix)
	for _, dt := range []time.Duration{500 * time.Millisecond, 2 * time.Second, 4 * time.Second} {
		ts := tOpen.Add(dt)
		NoteBackdropActivity(ts)
		tt.SetTime(ts)
		if !bytes.Equal(openBg, cropBottom(tt.Image().Pix)) {
			t.Fatalf("ambient gradient advanced while frosted overlay was open (+%v)", dt)
		}
	}

	// Closing the popup resumes ambient progression cleanly on subsequent frames.
	showPopup = false
	tClose := tOpen.Add(5 * time.Second)
	NoteBackdropActivity(tClose)
	tt.SetTime(tClose)
	tResume := tClose.Add(3 * time.Second)
	NoteBackdropActivity(tResume)
	tt.SetTime(tResume)
	if bytes.Equal(openBg, cropBottom(tt.Image().Pix)) {
		t.Fatal("ambient gradient failed to resume after frosted overlay closed")
	}

	// Explicit SetBackdropOverlayActive(true) also freezes the backdrop.
	SetBackdropOverlayActive(true)
	tGate := tResume.Add(100 * time.Millisecond)
	NoteBackdropActivity(tGate)
	tt.SetTime(tGate)
	gateBg := cropBottom(tt.Image().Pix)
	tGateLater := tGate.Add(3 * time.Second)
	NoteBackdropActivity(tGateLater)
	tt.SetTime(tGateLater)
	if !bytes.Equal(gateBg, cropBottom(tt.Image().Pix)) {
		t.Fatal("SetBackdropOverlayActive(true) did not freeze ambient gradient")
	}
	SetBackdropOverlayActive(false)
}

func TestAmbientGradientFreezesDuringCossOverlays(t *testing.T) {
	prevGlass := renderpolicy.Glass()
	prevMinimal := renderpolicy.Minimal()
	renderpolicy.SetMinimal(false)
	renderpolicy.SetGlass(true)
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Popover = p.Popover.Alpha(0.48)
	})
	t.Cleanup(func() {
		design.SetGlass(false, nil)
		renderpolicy.SetGlass(prevGlass)
		renderpolicy.SetMinimal(prevMinimal)
		resetBackdropPowerForTest()
		ConfigureBackdrop("", wallpapers.Defaults())
	})

	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "aurora"
	prefs.Variant = "chaos"
	prefs.Speed = 240
	prefs.Count = 5

	for _, tc := range []struct {
		name string
		kind string
	}{
		{"coss-dialog", "dialog"},
		{"coss-menu", "menu"},
		{"coss-select", "select"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetBackdropPowerForTest()
			setBackdropWindowActiveHook(func() bool { return true })
			mainAmbientClock.reset()

			open := false
			selected := "a"
			now := time.Unix(20000, 0)
			NoteBackdropActivity(now)
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(ui.DarkTheme())
				design.Bind(c, design.Dark, ui.Color{}, ui.Color{})
				paint := WallpaperBackgroundPainter("", prefs, false)
				ui.Box(c).Fill().Draw(paint).Children(func() {
					switch tc.kind {
					case "dialog":
						coss.Dialog(c, &open, coss.DialogOptions{Title: "Test", NoClose: true}, func() {
							ui.Box(c).Size(60, 24)
						})
					case "menu":
						trigger := coss.Button(c, "MenuTrigger", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM})
						coss.Menu(c, trigger, &open, []coss.MenuItem{{ID: "a", Label: "Alpha"}}, coss.MenuOptions{})
					case "select":
						coss.Select(c, &selected, []coss.SelectOption{{Value: "a", Label: "Alpha"}, {Value: "b", Label: "Beta"}}, coss.SelectOptions{Label: "SelectTrigger"})
					}
				})
			}, 240, 180)
			tt.SetTime(now)

			// Advance without overlay: gradient animates normally.
			t1 := now.Add(2 * time.Second)
			NoteBackdropActivity(t1)
			tt.SetTime(t1)
			cropEdge := func(pix []byte) []byte { return bytes.Clone(pix[140*240*4:]) }

			// Open overlay.
			if tc.kind == "select" {
				if err := tt.Click("SelectTrigger"); err != nil {
					t.Fatal(err)
				}
			} else {
				open = true
			}
			tOpen := t1.Add(100 * time.Millisecond)
			NoteBackdropActivity(tOpen)
			tt.SetTime(tOpen)
			// Let any dialog backdrop scrim enter transition settle before
			// asserting pixel-identical frozen ambient output across frames.
			tSettled := tOpen.Add(600 * time.Millisecond)
			NoteBackdropActivity(tSettled)
			tt.SetTime(tSettled)
			frozenBg := cropEdge(tt.Image().Pix)

			for _, dt := range []time.Duration{500 * time.Millisecond, 2 * time.Second, 4 * time.Second} {
				ts := tSettled.Add(dt)
				NoteBackdropActivity(ts)
				tt.SetTime(ts)
				if !bytes.Equal(frozenBg, cropEdge(tt.Image().Pix)) {
					t.Fatalf("%s did not freeze ambient gradient at +%v", tc.name, dt)
				}
			}

			// Close overlay and confirm gradient resumes.
			if tc.kind == "select" {
				tt.Key(0, ui.KeyEscape)
			} else {
				open = false
			}
			tClose := tSettled.Add(5 * time.Second)
			NoteBackdropActivity(tClose)
			tt.SetTime(tClose)
			// Let any exit transition finish before comparing post-close progression.
			tCloseSettled := tClose.Add(600 * time.Millisecond)
			NoteBackdropActivity(tCloseSettled)
			tt.SetTime(tCloseSettled)
			resumedStart := cropEdge(tt.Image().Pix)
			tResume := tCloseSettled.Add(3 * time.Second)
			NoteBackdropActivity(tResume)
			tt.SetTime(tResume)
			if bytes.Equal(resumedStart, cropEdge(tt.Image().Pix)) {
				t.Fatalf("%s did not resume ambient gradient after closing", tc.name)
			}
		})
	}
}
