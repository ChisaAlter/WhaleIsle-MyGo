package settings

import (
	"bytes"
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// TestSettingsModalDrawsItsElevationHairline pins --dsw-elevation-stroke: the
// panel paints a 0.5px hairline after its glass, so the edge stays visible where
// an opaque fill would otherwise cover the element's border.
func TestSettingsModalDrawsItsElevationHairline(t *testing.T) {
	dark := Model{Desktop: &DesktopAccess{}, DesktopActive: true, ThemeID: theme.IDDeepseekDark, SystemDark: true, OnClose: func() {}}
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ModalView(c, dark)
	}, 1400, 900)
	// The overlay fades in; a settled clock shows the paint it settles on.
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))
	// settingsModalSize(1400, 900) is 800×800, centered: (300, 50).
	image := tt.Image()
	const panelLeft, panelTop = 300, 50
	inside := modalPixel(image.RGBAAt(panelLeft+400, panelTop+8))
	edge := modalPixel(image.RGBAAt(panelLeft+400, panelTop))
	contrast := chrome.ContrastRatio(edge, inside)
	t.Logf("settings hairline %v on fill %v = %.2f:1", edge, inside, contrast)
	if contrast < 1.15 {
		t.Fatalf("settings modal hairline invisible in the dark: %.2f:1 (edge %v on fill %v)", contrast, edge, inside)
	}
	if edge.R <= inside.R {
		t.Fatalf("settings modal hairline is darker than its fill: %v on %v", edge, inside)
	}
	// The hairline belongs to the panel, not to a stray pixel: it must run along
	// the whole top edge.
	for _, x := range []int{panelLeft + 120, panelLeft + 400, panelLeft + 660} {
		got := modalPixel(image.RGBAAt(x, panelTop))
		if chrome.ContrastRatio(got, inside) < 1.15 {
			t.Fatalf("hairline missing at x=%d: %v on %v", x, got, inside)
		}
	}
}

// modalPixel reads a rendered pixel as a theme color.
func modalPixel(c color.RGBA) ui.Color { return ui.RGBA(c.R, c.G, c.B, float32(c.A)/255) }

// TestSettingsModalGlassClarityAndSingleHairline pins the macOS-style layered
// glass contract: low-mud glass scrims (0.12 light / 0.32 dark), solid-mode
// scrim compatibility (0.24 / 0.48), left rail more translucent than right
// options pane, restrained inner specular top rim, and translucent inner cards.
func TestSettingsModalGlassClarityAndSingleHairline(t *testing.T) {
	if got := modalScrimAlpha(true); got != 0.48 {
		t.Fatalf("solid dark scrim alpha = %v, want 0.48", got)
	}
	if got := modalScrimAlpha(false); got != 0.24 {
		t.Fatalf("solid light scrim alpha = %v, want 0.24", got)
	}
	if got := modalScrim(true, true); got.A != design.TW.Black.Alpha(0.32).A {
		t.Fatalf("glass dark scrim alpha = %v, want 0.32", got)
	}
	if got := modalScrim(false, true); got.A != design.TW.Gray900.Alpha(0.12).A {
		t.Fatalf("glass light scrim alpha = %v, want 0.12", got)
	}

	design.SetGlass(true, func(s design.Scheme, p *design.Palette) {
		if s == design.Dark {
			p.Card = ui.RGBA(32, 32, 35, 0.80)
			p.Popover = ui.RGBA(38, 39, 43, 0.48)
			p.Border = ui.RGBA(255, 255, 255, 0.07)
			p.SidebarBorder = ui.RGBA(255, 255, 255, 0.055)
		} else {
			p.Card = ui.RGBA(252, 253, 255, 0.58)
			p.Popover = ui.RGBA(252, 253, 254, 0.44)
			p.Border = ui.RGBA(0, 0, 0, 0.075)
			p.SidebarBorder = ui.RGBA(0, 0, 0, 0.055)
		}
	})
	t.Cleanup(func() { design.SetGlass(false, nil) })

	for _, tc := range []struct {
		name     string
		dark     bool
		backdrop ui.Color
	}{
		{"light-layered", false, ui.RGB(40, 140, 240)},
		{"dark-layered", true, ui.RGB(70, 120, 210)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{
				ActiveSection: SectionGeneral,
				SectionContent: func(c *ui.Context) {
					kSurface(c, ui.Box(c).Label("Glass Inner Card").FillWidth().Height(64))
					modelRowCard(c, "glass-model-card", func() {
						ui.Box(c).Label("Glass Model Card").FillWidth().Height(48)
					})
					modelSetupCard(c, "glass-setup-card", func() {
						ui.Box(c).Label("Glass Setup Card").FillWidth().Height(48)
					})
				},
				OnClose: func() {},
			}
			tt := ui.NewTester(func(c *ui.Context) {
				if tc.dark {
					c.SetTheme(ui.DarkTheme())
					design.Bind(c, design.Dark, ui.Color{}, ui.Color{})
				} else {
					c.SetTheme(ui.LightTheme())
					design.Bind(c, design.Light, ui.Color{}, ui.Color{})
				}
				ui.Box(c).Fill().Background(tc.backdrop)
				ModalView(c, m)
			}, 1400, 900)
			tt.SetDark(tc.dark)
			start := time.Unix(1000, 0)
			tt.SetTime(start)
			tt.SetTime(start.Add(time.Second))

			img := tt.Image()
			const panelLeft, panelTop = 260, 50 // settingsModalSize(1400, 900) is 880×800 -> (260, 50)
			topRim := img.RGBAAt(panelLeft+440, panelTop+1)
			inside := img.RGBAAt(panelLeft+440, panelTop+8)
			railFill := img.RGBAAt(panelLeft+12, panelTop+8)

			// Specular top rim inside glass perimeter must be brighter than the pane fill.
			rimSum := int(topRim.R) + int(topRim.G) + int(topRim.B)
			insideSum := int(inside.R) + int(inside.G) + int(inside.B)
			if topRim.R <= inside.R || rimSum <= insideSum+12 {
				t.Fatalf("%s specular top rim missing: rim=%v inside=%v", tc.name, topRim, inside)
			}

			// macOS layered translucency: left rail lets noticeably more blue
			// backdrop through than the slightly more solid right options pane.
			if tc.dark {
				if railFill.B <= inside.B+4 {
					t.Fatalf("%s left rail should be more translucent than right pane: rail=%v options=%v", tc.name, railFill, inside)
				}
			} else {
				// Over blue (40,140,240), more white fill raises R closer to 255;
				// a more translucent rail keeps R lower than the right pane.
				if railFill.R >= inside.R-8 {
					t.Fatalf("%s left rail should be more translucent than right pane: rail=%v options=%v", tc.name, railFill, inside)
				}
			}

			// Right pane itself must stay genuinely translucent over the blue backdrop.
			if int(inside.B)-int(inside.R) < 18 {
				t.Fatalf("%s right pane opaque over colored backdrop: inside=%v", tc.name, inside)
			}

			// Inner kSurface, modelRowCard and modelSetupCard must stay translucent (not opaque t.Card / SelectorFill).
			for _, label := range []string{"Glass Inner Card", "Glass Model Card", "Glass Setup Card"} {
				cardRect, ok := tt.Find(label)
				if !ok {
					t.Fatalf("%s missing %q", tc.name, label)
				}
				px := img.RGBAAt(int(cardRect.X+cardRect.W/2), int(cardRect.Y+cardRect.H/2))
				if int(px.B)-int(px.R) < 12 {
					t.Fatalf("%s %q lost backdrop translucency (stacked opaque t.Card?): px=%v pane=%v", tc.name, label, px, inside)
				}
			}
		})
	}
}

func TestSettingsModalAndConfirmDialogFreezeAmbientBackdrop(t *testing.T) {
	prevGlass := renderpolicy.Glass()
	prevMinimal := renderpolicy.Minimal()
	renderpolicy.SetMinimal(false)
	renderpolicy.SetGlass(true)
	t.Cleanup(func() {
		renderpolicy.SetGlass(prevGlass)
		renderpolicy.SetMinimal(prevMinimal)
		chrome.SetBackdropOverlayActive(false)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})

	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "aurora"
	prefs.Variant = "chaos"
	prefs.Speed = 240
	prefs.Count = 5

	for _, tc := range []struct {
		name    string
		confirm bool
	}{
		{"settings-modal", false},
		{"confirm-dialog", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			show := false
			now := time.Unix(30000, 0)
			chrome.NoteBackdropActivity(now)
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(ui.DarkTheme())
				design.Bind(c, design.Dark, ui.Color{}, ui.Color{})
				paint := chrome.WallpaperBackgroundPainter("", prefs, false)
				ui.Box(c).Fill().Draw(paint).Children(func() {
					if !show {
						return
					}
					if tc.confirm {
						ConfirmDialog(c, ConfirmOptions{
							Title:        "Confirm",
							Body:         "Proceed?",
							ConfirmLabel: "OK",
							CancelLabel:  "Cancel",
						})
					} else {
						ModalView(c, Model{
							ActiveSection: SectionGeneral,
							SectionContent: func(c *ui.Context) {
								ui.Box(c).Size(20, 20)
							},
							OnClose: func() {},
						})
					}
				})
			}, 360, 240)
			tt.SetTime(now)

			// Strip along the very top edge outside the centered dialog scrim/panel.
			cropStrip := func(pix []byte) []byte { return bytes.Clone(pix[:12*360*4]) }

			t1 := now.Add(2 * time.Second)
			chrome.NoteBackdropActivity(t1)
			tt.SetTime(t1)

			show = true
			tOpen := t1.Add(100 * time.Millisecond)
			chrome.NoteBackdropActivity(tOpen)
			tt.SetTime(tOpen)
			// Let the dialog reveal animation finish, then verify the ambient
			// gradient behind it stays completely frozen across further frames.
			tSettled := tOpen.Add(600 * time.Millisecond)
			chrome.NoteBackdropActivity(tSettled)
			tt.SetTime(tSettled)
			frozen := cropStrip(tt.Image().Pix)

			for _, dt := range []time.Duration{500 * time.Millisecond, 2 * time.Second, 4 * time.Second} {
				ts := tSettled.Add(dt)
				chrome.NoteBackdropActivity(ts)
				tt.SetTime(ts)
				if !bytes.Equal(frozen, cropStrip(tt.Image().Pix)) {
					t.Fatalf("%s failed to freeze ambient gradient at +%v", tc.name, dt)
				}
			}

			// Closing resumes ambient backdrop progression cleanly.
			show = false
			tClose := tSettled.Add(5 * time.Second)
			chrome.NoteBackdropActivity(tClose)
			tt.SetTime(tClose)
			resumedStart := cropStrip(tt.Image().Pix)
			tResume := tClose.Add(3 * time.Second)
			chrome.NoteBackdropActivity(tResume)
			tt.SetTime(tResume)
			if bytes.Equal(resumedStart, cropStrip(tt.Image().Pix)) {
				t.Fatalf("%s failed to resume ambient gradient after closing", tc.name)
			}
		})
	}
}
