package settings

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestSettingsSwitchSurvivesDialogExitGhost covers the crash the settings
// close hit: DialogBase keeps a fading copy of its subtree for the exit
// transition (applyMinimalMode installs ui.SetDialogExit at startup), and the
// copy still runs each element's paint callbacks a pass after they were
// built. Querying the element itself inside Draw then hits the expired-slot
// diagnostic — the query must be captured while the element is built.
func TestSettingsSwitchSurvivesDialogExitGhost(t *testing.T) {
	ui.SetDialogExit(ui.DialogExit{Duration: 100 * time.Millisecond})
	defer ui.SetDialogExit(ui.DialogExit{})
	open := true
	value := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.DialogBase(c.Key("settings-modal"), &open, func(backdrop, panel ui.Element) {
			panel.Children(func() {
				settingsPreferenceSwitch(c, &value, settingsSwitchLabelled).Label("Probe switch")
			})
		})
	}, 800, 600)
	if _, ok := tt.Find("Probe switch"); !ok {
		t.Fatalf("switch is not reachable: %v", tt.Texts())
	}
	open = false
	tt.Frame()
}

// TestSettingsSwitchAndToggleFocusRingHasTransparentOffset verifies that both
// settingsPreferenceSwitch and settingsToggle stroke their keyboard focus ring
// outside the 1 DIP offset without painting an opaque t.Background oval patch
// that would muddy translucent glass backdrops.
func TestSettingsSwitchAndToggleFocusRingHasTransparentOffset(t *testing.T) {
	backdrop := ui.RGB(30, 180, 70)
	ringColor := ui.RGB(220, 40, 40)

	for _, tc := range []struct {
		name  string
		build func(c *ui.Context, value *bool)
	}{
		{
			name: "settingsPreferenceSwitch",
			build: func(c *ui.Context, value *bool) {
				settingsPreferenceSwitch(c, value, settingsSwitchLabelled).Label("Focus probe")
			},
		},
		{
			name: "settingsToggle",
			build: func(c *ui.Context, value *bool) {
				settingsToggle(c, "Focus probe", value, false)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := false
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(ui.LightTheme())
				design.Bind(c, design.Light, ringColor, ui.RGB(255, 255, 255))
				ui.Column(c).Fill().Padding(24).Background(backdrop).Children(func() {
					tc.build(c, &value)
				})
			}, 160, 100)
			tt.Key(0, ui.KeyTab)
			tt.Frame()
			if !tt.Focused("Focus probe") {
				t.Fatalf("%s did not receive keyboard focus", tc.name)
			}
			rect, ok := tt.Find("Focus probe")
			if !ok {
				t.Fatalf("%s bounds missing", tc.name)
			}
			img := tt.Image()
			cx := int(rect.X + rect.W/2)
			// Outer stroked ring sits at track.Y - RingOffset - RingWidth/2 (= track.Y - 2).
			ringPx := img.RGBAAt(cx, int(rect.Y-design.RingOffset-design.RingWidth/2))
			if int(ringPx.R) < int(backdrop.R)+60 {
				t.Fatalf("%s missing focus ring stroke above track: ringPx=%v backdrop=%v", tc.name, ringPx, backdrop)
			}
			// 1 DIP offset gap immediately above the track must keep the real green
			// backdrop instead of an opaque white t.Background fill patch.
			gapPx := img.RGBAAt(cx, int(rect.Y)-1)
			if int(gapPx.R) > 95 || int(gapPx.G) < 130 {
				t.Fatalf("%s focus offset painted opaque background patch over glass backdrop: gapPx=%v backdrop=%v", tc.name, gapPx, backdrop)
			}
		})
	}
}

// TestFontSizePillArrowsStayGhostTransparentOnFrost pins the removal of the
// opaque tk(c).Background.Alpha(.75) patch behind the font-size stepper arrows:
// when the pill is hovered (revealing the arrows) without hovering the arrow
// button itself, the arrow cell blends seamlessly into the FrostSurface pill.
func TestFontSizePillArrowsStayGhostTransparentOnFrost(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	editor := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	backdrop := ui.RGB(40, 140, 240)
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.LightTheme())
		design.Bind(c, design.Light, ui.RGB(77, 107, 254), ui.RGB(255, 255, 255))
		ui.Column(c).Fill().Padding(16).Background(backdrop).Children(func() {
			generalFontSizeRow(c, editor, "text", "Font size", "Hint", 14, 10, 22, nil, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 520, 140)

	incRect, ok := tt.Find("Increase Font size")
	if !ok {
		t.Fatalf("font size increase arrow not found: %v", tt.Texts())
	}
	// Hover inside the left edge of the 88 DIP pill (away from the right-aligned
	// arrow buttons) so arrows fade in at Opacity(1) while staying unhovered.
	tt.Move(incRect.X-50, incRect.Y+incRect.H/2)
	tt.Frame()

	img := tt.Image()
	arrowCorner := img.RGBAAt(int(incRect.X)+1, int(incRect.Y)+1)
	pillNeighbor := img.RGBAAt(int(incRect.X)-6, int(incRect.Y)+1)
	diff := absDiff(arrowCorner.R, pillNeighbor.R) + absDiff(arrowCorner.G, pillNeighbor.G) + absDiff(arrowCorner.B, pillNeighbor.B)
	if diff > 4 {
		t.Fatalf("font size arrow painted background patch on FrostSurface: arrowCorner=%v pillNeighbor=%v (diff=%d)", arrowCorner, pillNeighbor, diff)
	}
}

// TestSettingsGlassSwitchThumbAndFocusContrast verifies that in frosted glass
// mode (where t.Background is translucent ~24–30%), switch knobs stay crisp
// opaque with >= 3.5:1 contrast on the ON track, keyboard focus rings reach
// WCAG >= 3:1 on light/dark glass panels, and selected Appearance color-scheme
// tiles render a >= 3:1 1px border.
func TestSettingsGlassSwitchThumbAndFocusContrast(t *testing.T) {
	t.Cleanup(func() { design.SetGlass(false, nil) })
	design.SetGlass(true, func(s design.Scheme, p *design.Palette) {
		if s == design.Dark {
			p.Background = ui.RGBA(21, 21, 23, 0.30)
		} else {
			p.Background = ui.RGBA(255, 255, 255, 0.24)
		}
	})

	for _, tc := range []struct {
		name     string
		dark     bool
		accent   ui.Color
		backdrop ui.Color
	}{
		{"light-glass", false, ui.Hex("#4176e6"), ui.RGB(208, 220, 229)},
		{"dark-glass", true, ui.Hex("#7aaaff"), ui.RGB(32, 34, 40)},
		{"dark-neutral-gray", true, ui.Hex("#52525b"), ui.RGB(32, 34, 40)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []string{"settingsPreferenceSwitch", "settingsToggle"} {
				t.Run(kind, func(t *testing.T) {
					on := true
					tt := ui.NewTester(func(c *ui.Context) {
						if tc.dark {
							c.SetTheme(ui.DarkTheme())
							design.Bind(c, design.Dark, tc.accent, ui.Color{})
						} else {
							c.SetTheme(ui.LightTheme())
							design.Bind(c, design.Light, tc.accent, ui.Color{})
						}
						ui.Column(c).Fill().Padding(24).Background(tc.backdrop).Children(func() {
							if kind == "settingsPreferenceSwitch" {
								settingsPreferenceSwitch(c, &on, settingsSwitchLabelled).Label("Glass switch")
							} else {
								settingsToggle(c, "Glass switch", &on, false)
							}
						})
					}, 160, 100)
					tt.Key(0, ui.KeyTab)
					tt.Frame()
					rect, ok := tt.Find("Glass switch")
					if !ok {
						t.Fatalf("%s bounds missing", kind)
					}
					img := tt.Image()
					// 1. Opaque switch knob center (x+21, y+9) must achieve >= 3.5:1 on ON track.
					knobPx := img.RGBAAt(int(rect.X)+21, int(rect.Y)+9)
					knobCol := ui.RGBA(knobPx.R, knobPx.G, knobPx.B, 1)
					if c := design.Contrast(knobCol, tc.accent); c < 3.5 {
						t.Fatalf("%s glass ON thumb %v on track %v contrast %.2f < 3.5", kind, knobPx, tc.accent, c)
					}
					// 2. Stroked keyboard focus ring above track must achieve WCAG >= 3.0:1 on glass backdrop.
					ringPx := img.RGBAAt(int(rect.X+rect.W/2), int(rect.Y-design.RingOffset-design.RingWidth/2))
					ringCol := ui.RGBA(ringPx.R, ringPx.G, ringPx.B, 1)
					if c := design.Contrast(ringCol, tc.backdrop); c < 3.0 {
						t.Fatalf("%s glass focus ring %v on backdrop %v contrast %.2f < 3.0", kind, ringPx, tc.backdrop, c)
					}
				})
			}

			// 3. Appearance selected color-scheme card 1px border must reach WCAG >= 3.0:1.
			selectedTheme := theme.IDDeepseekLight
			if tc.dark {
				selectedTheme = theme.IDDeepseekDark
			}
			ttApp := ui.NewTester(func(c *ui.Context) {
				if tc.dark {
					c.SetTheme(ui.DarkTheme())
					design.Bind(c, design.Dark, tc.accent, ui.Color{})
				} else {
					c.SetTheme(ui.LightTheme())
					design.Bind(c, design.Light, tc.accent, ui.Color{})
				}
				ui.Column(c).Fill().Padding(16).Background(tc.backdrop).Children(func() {
					appearancePane(c, Model{
						ThemeID:    selectedTheme,
						SystemDark: tc.dark,
						Appearance: &AppearanceAccess{
							Theme:      &selectedTheme,
							OnSetTheme: func(string) error { return nil },
						},
					}, tk(c).Foreground, tk(c).MutedForeground)
				})
			}, 720, 260)
			ttApp.Frame()
			cardLabel := providerCopy("浅色", "Light")
			if tc.dark {
				cardLabel = providerCopy("深色", "Dark")
			}
			cardRect, ok := ttApp.Find(cardLabel)
			if !ok {
				t.Fatalf("%s missing selected scheme tile %q: %v", tc.name, cardLabel, ttApp.Texts())
			}
			borderPx := ttApp.Image().RGBAAt(int(cardRect.X+cardRect.W/2), int(cardRect.Y))
			borderCol := ui.RGBA(borderPx.R, borderPx.G, borderPx.B, 1)
			if c := design.Contrast(borderCol, tc.backdrop); c < 3.0 {
				t.Fatalf("%s selected appearance card 1px border %v on backdrop %v contrast %.2f < 3.0", tc.name, borderPx, tc.backdrop, c)
			}
		})
	}
}
