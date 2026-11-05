package settings

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func TestSettingsSelectedRailUsesExactSourceTokensAndOverrides(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	prefs := theme.Defaults()
	theme.SetPreferences(prefs)
	for _, tc := range []struct {
		id          string
		fill, brand ui.Color
	}{
		// 0.3.13: a neutral wash that keeps the label colour.
		{theme.IDDeepseekLight, ui.RGBA(38, 49, 72, .10), ui.RGB(65, 118, 230)},
		{theme.IDDeepseekDark, ui.RGBA(255, 255, 255, .14), ui.RGB(86, 134, 254)},
	} {
		fill, ink := settingsSelectedRailColors(Model{ThemeID: tc.id})
		fg := theme.Color(theme.Resolve(tc.id).Active(false).FG)
		if fill != tc.fill || ink != fg {
			t.Fatalf("source token %s: fill%v ink%v", tc.id, fill, ink)
		}
	}
	prefs.Light = "codex"
	theme.SetPreferences(prefs)
	fill, _ := settingsSelectedRailColors(Model{ThemeID: theme.IDDeepseekLight})
	if fill != ui.Hex("#dce7ff") {
		t.Fatalf("explicit installed Codex active-accent override lost: %v", fill)
	}
}

func TestSettingsSwitchSourceBranchesKeepNativeInteractionAndColors(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	prefs := theme.Defaults()
	theme.SetPreferences(prefs)
	for _, dark := range []bool{false, true} {
		for _, style := range []settingsSwitchStyle{settingsSwitchLabelled, settingsSwitchInput} {
			value, disabled, changes := true, false, 0
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Padding(20).Children(func() {
					if settingsPreferenceSwitch(c, &value, style).Label("Source switch branch").Disabled(disabled).Changed() {
						changes++
					}
				})
			}, 200, 100)
			tt.SetDark(dark)
			tt.SetTime(time.Unix(1000, 0))
			tt.SetTime(time.Unix(1001, 0))
			bounds, found := tt.Find("Source switch branch")
			wantWidth, wantHeight := float32(36), float32(20)
			if style == settingsSwitchInput {
				wantWidth, wantHeight = 44, 28
			}
			if !found || bounds.W != wantWidth || bounds.H != wantHeight {
				t.Fatalf("style%v bounds%+v dark%v", style, bounds, dark)
			}
			colors := theme.ResolveWithSystem(theme.IDSystem, dark).Active(dark)
			want := theme.Color(colors.BrandPrimary)
			if style == settingsSwitchInput {
				want = theme.Color(colors.Accent)
			}
			pixel := tt.Image().RGBAAt(int(bounds.X+(bounds.W-36)/2+8), int(bounds.Y+bounds.H/2))
			if pixel.R != want.R || pixel.G != want.G || pixel.B != want.B {
				t.Fatalf("style%v dark%v on track%v want%v", style, dark, pixel, want)
			}
			tt.Key(0, ui.KeyTab)
			if !tt.Focused("Source switch branch") {
				t.Fatal("source switch lost Tab focus")
			}
			tt.Key(0, ui.KeySpace)
			if value || changes != 1 {
				t.Fatalf("keyboard value%v changes%d", value, changes)
			}
			disabled = true
			tt.Frame()
			tt.Key(0, ui.KeySpace)
			tt.ClickAt(bounds.X+1, bounds.Y+1)
			if value || changes != 1 {
				t.Fatal("disabled switch accepted pointer or keyboard input")
			}
			disabled = false
			tt.Frame()
			tt.ClickAt(bounds.X+1, bounds.Y+1) // input branch's extra hit area must toggle
			if !value || changes != 2 {
				t.Fatalf("source hit area did not toggle: %v/%d", value, changes)
			}
		}
	}
}

func TestSettingsSwitchHostSourceStylesPreserveCallbacks(t *testing.T) {
	host := bodyFixtureHost()
	calls := []string{}
	host.OnChange = func(namespace, key, value string) {
		calls = append(calls, namespace+"/"+key+"="+value)
		setBodyFixtureValue(host, namespace, key, value)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			bodyHostPreference(c, host, "ui-settings", "enabled", "Coding switch", "", c.Theme().Text, c.Theme().TextMuted)
			bodyHostPreference(c, host, "ui-git", "titlebarGit", "Git switch", "", c.Theme().Text, c.Theme().TextMuted)
		})
	}, 600, 300)
	coding, _ := tt.Find("Coding switch")
	git, _ := tt.Find("Git switch")
	// Titles now own only their intrinsic text width; switch geometry itself is
	// pinned by TestSettingsSwitchSourceBranchesKeepNativeInteractionAndColors.
	if coding.W <= 0 || git.W <= 0 || coding.H != 22 || git.H != 22 || coding.Y >= git.Y {
		t.Fatalf("source row title geometry coding%+v git%+v", coding, git)
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	if len(calls) != 2 || calls[0] != "ui-settings/enabled=true" || calls[1] != "ui-git/titlebarGit=false" {
		t.Fatalf("Host callbacks changed: %v", calls)
	}
	host.Snapshot.Writable = false
	tt.Frame()
	tt.Key(0, ui.KeySpace)
	tt.ClickAt(582, coding.Y+coding.H/2)
	tt.ClickAt(578, git.Y+git.H/2)
	if len(calls) != 2 {
		t.Fatal("read-only Host switch accepted a write")
	}
}

func TestSettingsPillSelectPreservesNativeKeyboardAndDisabledContract(t *testing.T) {
	selected := "Alpha"
	disabled := false
	changes := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			control := settingsPreferenceSelect(c, &selected, []string{"Alpha", "Beta"}).Label("Pill probe").Disabled(disabled)
			if control.Changed() {
				changes++
			}
		})
	}, 400, 240)
	bounds, ok := tt.Find("Pill probe")
	if !ok || bounds.H != 36 {
		t.Fatalf("pill trigger=%+v found=%v want36px", bounds, ok)
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if selected != "Beta" || changes != 1 {
		t.Fatalf("native selection changed=%q count=%d", selected, changes)
	}
	disabled = true
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyEnter)
	if selected != "Beta" || changes != 1 {
		t.Fatal("disabled pill accepted a change")
	}
}

func TestSettingsSourceRowHeightAndAdjacentGearToggle(t *testing.T) {
	value := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			bodyPreferenceRow(c, "source-row", "Source title", "Source description", c.Theme().Text, c.Theme().TextMuted, func() { settingsPreferenceButton(c, "Action").Label("Action") })
			withRowGear(rowGear{label: "Source gear"}, func() {
				bodyPreferenceRow(c, "source-gear-row", "Other title", "Other description", c.Theme().Text, c.Theme().TextMuted, func() { ui.Switch(c, &value).Label("Source switch") })
			})
		})
	}, 560, 300)
	first, _ := tt.Find("Source title")
	second, _ := tt.Find("Other title")
	if math.Abs(float64(second.Y-first.Y-76.5)) > .1 {
		t.Fatalf("source22+4+18+16*2+.5 row pitch=%g", second.Y-first.Y)
	}
	gear, g := tt.Find("Source gear")
	toggle, s := tt.Find("Source switch")
	if !g || !s || gear.W != 28 || math.Abs(float64(toggle.X-gear.X-gear.W-8)) > .1 {
		t.Fatalf("gear and toggle are not one trailing8px group: gear%+v switch%+v", gear, toggle)
	}
	if math.Abs(float64(toggle.X+toggle.W-560)) > .1 {
		t.Fatalf("control not aligned to pane right: %+v", toggle)
	}
	tt.SetSize(300, 400)
	gear, g = tt.Find("Source gear")
	toggle, s = tt.Find("Source switch")
	if !g || !s || toggle.X+toggle.W > 300 || gear.X < 0 || gear.Y <= second.Y {
		t.Fatalf("narrow group clipped: gear%+v switch%+v", gear, toggle)
	}
}
