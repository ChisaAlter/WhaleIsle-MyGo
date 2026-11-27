package settings

import (
	"encoding/json"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestBodyHostValueAndBusyEnterCachedZeroAlloc(t *testing.T) {
	access := &BodySettingsAccess{
		Owner: 1,
		Snapshot: store.ViewState{
			ClientReady: true,
			Writable:    true,
			Namespaces: []store.Namespace{{
				NS:    "ui-conversation",
				Value: json.RawMessage(`{"busyEnter":"steer","nextPrompt":true,"count":3,"unset":null}`),
				Fields: []store.Field{
					{Key: "busyEnter", Kind: store.FieldEnum},
					{Key: "nextPrompt", Kind: store.FieldBool},
					{Key: "count", Kind: store.FieldNumber},
					{Key: "unset", Kind: store.FieldString},
				},
			}},
		},
	}

	if _, got, ok := bodyHostValue(access, "ui-conversation", "busyEnter"); !ok || got != "steer" {
		t.Fatalf("busyEnter = (%q, %v), want (steer, true)", got, ok)
	}
	if _, got, ok := bodyHostValue(access, "ui-conversation", "nextPrompt"); !ok || got != "true" {
		t.Fatalf("nextPrompt = (%q, %v), want (true, true)", got, ok)
	}
	if _, got, ok := bodyHostValue(access, "ui-conversation", "count"); !ok || got != "3" {
		t.Fatalf("count = (%q, %v), want (3, true)", got, ok)
	}
	if _, got, ok := bodyHostValue(access, "ui-conversation", "unset"); !ok || got != "" {
		t.Fatalf("unset = (%q, %v), want (\"\", true)", got, ok)
	}
	if allocs := testing.AllocsPerRun(200, func() {
		_, _, _ = bodyHostValue(access, "ui-conversation", "busyEnter")
		_, _, _ = bodyHostValue(access, "ui-conversation", "nextPrompt")
	}); allocs != 0 {
		t.Fatalf("cached bodyHostValue allocated %g times/run, want 0", allocs)
	}

	client := &dsh.Client{}
	st := store.New()
	st.Attach(client, nil)
	if got := st.BusyEnterFor(client); got != "queue" {
		t.Fatalf("undescribed BusyEnterFor = %q, want queue", got)
	}
	if allocs := testing.AllocsPerRun(200, func() {
		_ = st.BusyEnterFor(client)
	}); allocs != 0 {
		t.Fatalf("BusyEnterFor allocated %g times/run, want 0", allocs)
	}
}

func TestPluginInventoryFirstFrameTwoColumnsAndExpandedPeerNotStretched(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	rows := []plugininventory.Row{
		{
			Key:         "p/instructions",
			EntryID:     "include:ctx-instructions",
			Module:      "ctx-instructions",
			Enabled:     "enabled",
			Phase:       "active",
			Title:       json.RawMessage(`{"zh":"工作区指令加载","en":"Workspace instructions"}`),
			Description: json.RawMessage(`{"zh":"将工作区级和项目级的指令文件（如 AGENTS.md、CLAUDE.md、MEMORY.md）自动注入到每轮对话上下文中。"}`),
		},
		{
			Key:         "p/persona",
			EntryID:     "include:ctx-persona",
			Module:      "ctx-persona",
			Enabled:     "enabled",
			Phase:       "active",
			Title:       json.RawMessage(`{"zh":"智能体角色设定","en":"Agent persona"}`),
			Description: json.RawMessage(`{"zh":"为智能体注入基础角色与行为规范。"}`),
		},
		{
			Key:         "p/schedule",
			EntryID:     "include:tool-schedule",
			Module:      "tool-schedule",
			Enabled:     "enabled",
			Phase:       "active",
			Title:       json.RawMessage(`"tool-schedule"`),
			Description: json.RawMessage(`"Model-facing reminder management tools"`),
		},
		{
			Key:         "p/files",
			EntryID:     "include:tool-files",
			Module:      "tool-files",
			Enabled:     "enabled",
			Phase:       "active",
			Title:       json.RawMessage(`{"zh":"文件读写工具","en":"File tools"}`),
			Description: json.RawMessage(`{"zh":"提供工作区内文件读取、编辑与搜索能力。"}`),
		},
	}

	expanded := ""
	firstPassCols := 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := fullRailModel()
		m.DesktopActive = false
		m.PluginsActive = true
		m.Plugins = &PluginsAccess{
			Inventory: &PluginInventoryAccess{
				Snapshot: plugininventory.Snapshot{
					Host:       1,
					Revision:   1,
					Connected:  true,
					Ready:      true,
					SelectedID: "default",
					Expanded:   expanded,
					Presets:    []plugininventory.Preset{{ID: "default", Name: "Default", Default: true, Rows: rows}},
				},
				OnToggleRow: func(key string) {
					if expanded == key {
						expanded = ""
					} else {
						expanded = key
					}
				},
			},
		}
		if firstPassCols == 0 {
			w, _ := settingsModalSizeFor(1400, 900, c.TitleBar())
			firstPassCols = pluginInventoryColumns(w - design.Sp(52) - 2*design.Sp(6))
		}
		ModalView(c, m)
	}, 1400, 900)
	t0 := time.Unix(1000, 0)
	tt.SetTime(t0)
	tt.SetTime(t0.Add(time.Second))

	if firstPassCols != 2 {
		t.Fatalf("first-pass predicted columns = %d, want 2", firstPassCols)
	}

	cardRect := func(label string) ui.Rect {
		t.Helper()
		var best ui.Rect
		for _, r := range tt.FindAll(label) {
			if r.W*r.H > best.W*best.H {
				best = r
			}
		}
		if best.W == 0 {
			t.Fatalf("card %q not found: %v", label, tt.Texts())
		}
		return best
	}

	c0 := cardRect("工作区指令加载")
	c1 := cardRect("智能体角色设定")
	c2 := cardRect("定时提醒工具")
	c3 := cardRect("文件读写工具")

	if c1.X <= c0.X || math.Abs(float64(c0.Y-c1.Y)) >= 0.5 {
		t.Fatalf("row 1 not laid out in 2 columns: c0=%+v c1=%+v", c0, c1)
	}
	if math.Abs(float64(c0.H-c1.H)) >= 0.5 || math.Abs(float64(c0.H-c2.H)) >= 0.5 || math.Abs(float64(c2.H-c3.H)) >= 0.5 {
		t.Fatalf("collapsed cards not uniform height: c0=%+v c1=%+v c2=%+v c3=%+v", c0, c1, c2, c3)
	}

	// Expand c0 and confirm its sibling c1 stays top-aligned and unstretched.
	if err := tt.Click("工作区指令加载"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if expanded != "p/instructions" {
		t.Fatalf("click did not expand p/instructions: expanded=%q", expanded)
	}
	c0Exp := cardRect("工作区指令加载")
	c1Peer := cardRect("智能体角色设定")
	if c0Exp.H <= c1Peer.H+20 {
		t.Fatalf("expanded card did not grow above peer: expanded=%+v peer=%+v", c0Exp, c1Peer)
	}
	if math.Abs(float64(c0Exp.Y-c1Peer.Y)) >= 0.5 {
		t.Fatalf("expanded row peer not top-aligned: expanded=%+v peer=%+v", c0Exp, c1Peer)
	}
	if c1Peer.H > c0.H+1 {
		t.Fatalf("peer card was stretched when sibling expanded: before=%v after=%v", c0.H, c1Peer.H)
	}
}

func TestThemeGridFirstFrameZeroExtraReflow(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	editor := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		colors := theme.ResolveWithSystem("system", c.Theme().Dark).Active(c.Theme().Dark)
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Children(func() {
			themeLibraryPane(c, editor, colors, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 760, 1200)
	if builds != 1 {
		t.Fatalf("themeLibraryPane triggered %d builds on initial mount (expected 1 without width=0 double reflow)", builds)
	}
	families := editor.Saved.Families()
	c0, ok0 := tt.Find("Theme card " + families[0].LocalizedName())
	c1, ok1 := tt.Find("Theme card " + families[1].LocalizedName())
	c2, ok2 := tt.Find("Theme card " + families[2].LocalizedName())
	if !ok0 || !ok1 || !ok2 || c1.X <= c0.X || c2.X <= c1.X || math.Abs(float64(c0.Y-c2.Y)) >= 0.5 {
		t.Fatalf("theme grid did not lay out 3 columns on frame 1: c0=%+v c1=%+v c2=%+v", c0, c1, c2)
	}
}

func TestSettingsGlassRailHoverAndSelectedPillsNoLeftBar(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	design.SetGlass(true, func(s design.Scheme, p *design.Palette) {
		if s == design.Dark {
			p.SidebarAccent = ui.RGBA(255, 255, 255, 0.08)
		} else {
			p.SidebarAccent = ui.RGBA(0, 0, 0, 0.055)
		}
	})
	t.Cleanup(func() { design.SetGlass(false, nil) })

	for _, tc := range []struct {
		name     string
		dark     bool
		backdrop ui.Color
	}{
		{"light-glass", false, ui.RGB(210, 225, 245)},
		{"dark-glass", true, ui.RGB(24, 30, 44)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := fullRailModel()
			m.ActiveSection = SectionModels
			m.SectionContent = func(c *ui.Context) { ui.Text(c, "body") }

			tt := ui.NewTester(func(c *ui.Context) {
				if tc.dark {
					c.SetTheme(ui.DarkTheme())
					design.Bind(c, design.Dark, ui.RGB(77, 107, 254), ui.RGB(255, 255, 255))
				} else {
					c.SetTheme(ui.LightTheme())
					design.Bind(c, design.Light, ui.RGB(77, 107, 254), ui.RGB(255, 255, 255))
				}
				ui.Column(c).Fill().Background(tc.backdrop).Children(func() {
					View(c, m)
				})
			}, 900, 800)
			tt.SetDark(tc.dark)
			tt.SetTime(time.Unix(1000, 0))
			tt.SetTime(time.Unix(1001, 0))

			findCell := func(label string) ui.Rect {
				t.Helper()
				var best ui.Rect
				for _, r := range tt.FindAll(label) {
					if r.X < 208 && r.W*r.H > best.W*best.H {
						best = r
					}
				}
				if best.W == 0 {
					t.Fatalf("rail cell %q missing", label)
				}
				return best
			}

			sel := findCell("模型")
			hov := findCell("Agent 预设")
			idle := findCell("技能")

			tt.Move(hov.X+hov.W/2, hov.Y+hov.H/2)
			tt.Frame()
			img := tt.Image()
			sample := func(r ui.Rect) color.RGBA {
				return img.RGBAAt(int(r.X+r.W)-6, int(r.Y+r.H/2))
			}
			dist := func(a, b color.RGBA) int {
				return absDiff(a.R, b.R) + absDiff(a.G, b.G) + absDiff(a.B, b.B)
			}

			selFill := sample(sel)
			hovFill := sample(hov)
			idleFill := sample(idle)
			if hovFill == idleFill || hovFill == selFill {
				t.Fatalf("%s glass rail hover not distinct: idle=%v hov=%v sel=%v", tc.name, idleFill, hovFill, selFill)
			}
			if dist(hovFill, idleFill) >= dist(selFill, idleFill) {
				t.Fatalf("%s glass rail hover should be subtler than selected: idle=%v hov=%v sel=%v", tc.name, idleFill, hovFill, selFill)
			}
			// No left blue/accent bar on selected pill.
			for dx := 2; dx <= 4; dx++ {
				leftPx := img.RGBAAt(int(sel.X)+dx, int(sel.Y+sel.H/2))
				if dist(leftPx, selFill) > 4 {
					t.Fatalf("%s selected pill painted left bar at dx=%d: left=%v fill=%v", tc.name, dx, leftPx, selFill)
				}
			}
		})
	}
}

func TestSettingsFormPaneMemoKeepsScrollHitAndLiveUpdates(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	assistant, remote := false, false
	var toggled []string
	m := fullRailModel()
	m.DesktopActive = false
	m.FeaturesActive = true
	m.Features = &FeaturesAccess{
		AssistantEnabled:       &assistant,
		OnSetAssistant:         func(v bool) { assistant = v; toggled = append(toggled, "assistant") },
		RemoteWorkspaceEnabled: &remote,
		OnSetRemoteWorkspace:   func(v bool) { remote = v; toggled = append(toggled, "remote") },
		Project:                &ProjectPreferenceAccess{OnSet: func(bool) { toggled = append(toggled, "project") }},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 860, 260)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))

	// Scroll down inside the memoized options body so the bottom row enters view,
	// then click its switch to confirm post-scroll hit-testing stays accurate.
	tt.Scroll(560, 180, 0, 400)
	tt.Frame()

	var sshSwitch ui.Rect
	for _, r := range tt.FindAll("远程工作区（SSH）") {
		if math.Abs(float64(r.W-toggleWidth)) < 1 && math.Abs(float64(r.H-toggleHeight)) < 1 {
			sshSwitch = r
			break
		}
	}
	if sshSwitch.W == 0 {
		t.Fatalf("scrolled SSH switch not found: %v", tt.FindAll("远程工作区（SSH）"))
	}
	tt.ClickAt(sshSwitch.X+sshSwitch.W/2, sshSwitch.Y+sshSwitch.H/2)
	tt.Frame()
	if !remote || len(toggled) != 1 || toggled[0] != "remote" {
		t.Fatalf("clicking scrolled switch inside memoized formPane failed: remote=%v toggled=%v", remote, toggled)
	}
}

func TestSettingsFeaturesMemoReflectsExternalDshbotRollbackAndThemeRadiusMemo(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	assistant, remote, dshbot := false, false, true
	var gotKey uint64
	m := fullRailModel()
	m.DesktopActive = false
	m.FeaturesActive = true
	m.Features = &FeaturesAccess{
		AssistantEnabled:       &assistant,
		OnSetAssistant:         func(v bool) { assistant = v },
		RemoteWorkspaceEnabled: &remote,
		OnSetRemoteWorkspace:   func(v bool) { remote = v },
		Project:                &ProjectPreferenceAccess{OnSet: func(bool) {}},
	}
	m.Interface = &InterfaceAccess{
		DshbotEnabled: &dshbot,
		OnSetDshbot:   func(v bool) { dshbot = v },
	}

	tt := ui.NewTester(func(c *ui.Context) {
		gotKey, _ = formPaneMemoKey(c, m, m.snapshot(), SectionFeatures, "")
		View(c, m)
	}, 900, 640)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))

	findBotSwitch := func() ui.Rect {
		t.Helper()
		for _, r := range tt.FindAll("机器人（Bots）") {
			if math.Abs(float64(r.W-toggleWidth)) < 1 && math.Abs(float64(r.H-toggleHeight)) < 1 {
				return r
			}
		}
		t.Fatalf("Bots switch not found: %v", tt.FindAll("机器人（Bots）"))
		return ui.Rect{}
	}

	keyBefore := gotKey
	swRect := findBotSwitch()
	// Sample the left side of the 30×18 switch track: when ON, the knob rests
	// on the right so x+4 paints the dark primary track; when OFF, the knob
	// slides left so x+4 paints the bright white thumb.
	sampleSwitchLeft := func() color.RGBA {
		return tt.Image().RGBAAt(int(swRect.X)+4, int(swRect.Y+swRect.H/2))
	}
	pxBefore := sampleSwitchLeft()

	// Simulate an async config rollback (e.g. cancelled Harness restart) without
	// bumping settings.Store rev: memo key and painted switch must update immediately.
	dshbot = false
	tt.SetTime(time.Unix(1002, 0))
	tt.SetTime(time.Unix(1003, 0))
	if gotKey == keyBefore {
		t.Fatalf("SectionFeatures formPaneMemoKey did not change when m.Interface.DshbotEnabled rolled back (%d)", gotKey)
	}
	pxAfter := sampleSwitchLeft()
	if pxAfter == pxBefore {
		t.Fatalf("Bots switch pixels stayed frozen after external DshbotEnabled rollback: before=%v after=%v", pxBefore, pxAfter)
	}

	// Verify themeGridMemoKey folds custom --dsw-radius-button-sm override.
	ed := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	custom := ed.Saved.Family(theme.SkinDefault)
	custom.ID = "custom-radius-test"
	custom.Light.Overrides = map[string]string{"--dsw-radius-button-sm": "8px"}
	ed.Draft.Custom = []theme.Family{custom}
	ed.Draft.Light = custom.ID
	var gridKey1, gridKey2 uint64
	ui.NewTester(func(c *ui.Context) {
		colors := theme.ResolveWithSystem("system", false).Active(false)
		fams := ed.Draft.Families()
		gridKey1 = themeGridMemoKey(c, ed, colors, 3, fams)
		ed.Draft.Custom[0].Light.Overrides["--dsw-radius-button-sm"] = "24px"
		gridKey2 = themeGridMemoKey(c, ed, colors, 3, ed.Draft.Families())
	}, 600, 400)
	if gridKey1 == gridKey2 {
		t.Fatalf("themeGridMemoKey did not change when custom radius override changed (%d)", gridKey1)
	}
}
