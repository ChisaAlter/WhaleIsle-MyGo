package settings

import (
	"math"
	"testing"
	"time"

	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// buttonRect picks a labelled element's own box out of the nodes that carry
// its name: the element registers first, its copy paints inside it.
func buttonRect(rs []ui.Rect) ui.Rect {
	out := ui.Rect{}
	for _, r := range rs {
		if r.W*r.H > out.W*out.H {
			out = r
		}
	}
	return out
}

// TestSettingsSelectedRailPaintsTheSidebarAccentWash pins the selected rail
// cell's fill to the coss sidebar-accent token in both schemes; an inactive
// cell keeps the bare sidebar surface.
func TestSettingsSelectedRailPaintsTheSidebarAccentWash(t *testing.T) {
	for _, dark := range []bool{false, true} {
		query := ""
		var sidebar, accent ui.Color
		tt := ui.NewTester(func(c *ui.Context) {
			tokens := tk(c)
			sidebar, accent = tokens.Sidebar, tokens.SidebarAccent
			ui.Column(c).Fill().Background(sidebar).Children(func() {
				railButton(c, Model{}, navigationEntry{id: SectionGeneral, icon: "SettingsOutline"}, true, &query)
				railButton(c, Model{}, navigationEntry{id: SectionAppearance, icon: "LightOutline"}, false, &query)
			})
		}, 240, 120)
		tt.SetDark(dark)
		sel := buttonRect(tt.FindAll(navigationLabel(Model{}, SectionGeneral)))
		idle := buttonRect(tt.FindAll(navigationLabel(Model{}, SectionAppearance)))
		if sel.W == 0 || idle.W == 0 {
			t.Fatalf("dark%v rail cells not found: %v", dark, tt.Texts())
		}
		px := func(r ui.Rect) (uint8, uint8, uint8) {
			p := tt.Image().RGBAAt(int(r.X+r.W)-6, int(r.Y+r.H/2))
			return p.R, p.G, p.B
		}
		blend := func(base, over uint8, alpha uint8) int {
			a := float64(alpha) / 255
			return int(float64(base)*(1-a) + float64(over)*a + .5)
		}
		want := func(base, over ui.Color) (int, int, int) {
			return blend(base.R, over.R, over.A), blend(base.G, over.G, over.A), blend(base.B, over.B, over.A)
		}
		near := func(got uint8, want int) bool { d := int(got) - want; return d > -4 && d < 4 }
		r, g, b := px(sel)
		wr, wg, wb := want(sidebar, accent)
		if !near(r, wr) || !near(g, wg) || !near(b, wb) {
			t.Fatalf("dark%v selected fill rgba(%d,%d,%d), want sidebar-accent over sidebar ~(%d,%d,%d)", dark, r, g, b, wr, wg, wb)
		}
		r, g, b = px(idle)
		if !near(r, int(sidebar.R)) || !near(g, int(sidebar.G)) || !near(b, int(sidebar.B)) {
			t.Fatalf("dark%v inactive fill rgba(%d,%d,%d), want bare sidebar %v", dark, r, g, b, sidebar)
		}
	}
}

// TestSettingsSwitchSourceBranchesKeepInteractionAndColors pins the coss
// switch contract: the labelled row switch is the 30×18 track itself while the
// input branch keeps a wider 44×28 hit box with the track drawn centered, the
// on track paints the primary token in both schemes, and keyboard, pointer and
// the disabled state behave identically on both branches.
func TestSettingsSwitchSourceBranchesKeepInteractionAndColors(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	prefs := theme.Defaults()
	theme.SetPreferences(prefs)
	for _, dark := range []bool{false, true} {
		for _, style := range []settingsSwitchStyle{settingsSwitchLabelled, settingsSwitchInput} {
			value, disabled, changes := true, false, 0
			var want ui.Color
			tt := ui.NewTester(func(c *ui.Context) {
				want = tk(c).Primary
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
			wantWidth, wantHeight := toggleWidth, toggleHeight
			if style == settingsSwitchInput {
				wantWidth, wantHeight = 11*design.Unit, 7*design.Unit
			}
			if !found || bounds.W != wantWidth || bounds.H != wantHeight {
				t.Fatalf("style%v bounds%+v dark%v", style, bounds, dark)
			}
			pixel := tt.Image().RGBAAt(int(bounds.X+(bounds.W-toggleWidth)/2+8), int(bounds.Y+bounds.H/2))
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

// titleRect picks the title text out of the same-named row group and its
// labeled controls: the row registers first, the title copy right after it.
func titleRect(rs []ui.Rect) ui.Rect {
	if len(rs) < 2 {
		return ui.Rect{}
	}
	return rs[1]
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
	coding := titleRect(tt.FindAll("Coding switch"))
	git := titleRect(tt.FindAll("Git switch"))
	// Titles now own only their intrinsic text width; switch geometry itself is
	// pinned by TestSettingsSwitchSourceBranchesKeepInteractionAndColors.
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
	if !ok || bounds.H != design.Sp(8) {
		t.Fatalf("pill trigger=%+v found=%v want h-8", bounds, ok)
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
	if math.Abs(float64(second.Y-first.Y-74.5)) > .1 {
		t.Fatalf("source row pitch=%g want 74.5 (rows tile with 0.5 DIP hairline and no gap)", second.Y-first.Y)
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
