package frame

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

func TestConversationHeader400KeepsLongTitleAndEveryUtilityInside(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Center = 400
	snap := session.ViewState{SelectedID: "child", SelectedAgentPreset: "Extremely long session preset identity",
		Items: []session.Item{{SessionID: "root", Title: strings.Repeat("Long parent ", 12)}, {SessionID: "child", ParentID: "root", Title: strings.Repeat("Long current ", 12)}}}
	calls := map[string]int{}
	callback := func(name string) func() { return func() { calls[name]++ } }
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			conversationHeaderTitleRow(c, Model{Store: session.New(), Sizes: &sizes,
				OnOpenLauncher: callback("more"), OnOpenAgents: callback("agents"), OnToggleTools: callback("tools"),
				OnOpenHostTerminal: callback("host"), OnToggleTerminal: callback("terminal"), OnOpenSettings: callback("settings")}, snap, theme.Resolve("deepseek.light").Light)
		})
	}, 400, 100)
	title, ok := tt.Find("Conversation session title")
	if !ok || title.W < 40 {
		t.Fatalf("long ancestry has no readable seat: %+v", title)
	}
	preset, ok := tt.Find("Conversation agent preset")
	if !ok || preset.W > 18 {
		t.Fatalf("compact preset not icon-only: %+v", preset)
	}
	if tt.HasText(snap.SelectedAgentPreset) {
		t.Fatal("compact preset text did not collapse")
	}
	for _, tc := range []struct{ label, call string }{
		{i18n.T(i18n.KeyLauncherOpen), "more"}, {i18n.T(i18n.KeyFrameCaptionAgents), "agents"},
		{i18n.T(i18n.KeyFrameCaptionProjectDocs), "tools"}, {termui.HostTitle(), "host"},
		{i18n.T(i18n.KeyTerminalTitle), "terminal"}, {i18n.T(i18n.KeySettingsOpen), "settings"},
	} {
		r, ok := tt.Find(tc.label)
		if !ok || r.X < 20 || r.X+r.W > 388 || r.Y < 0 || r.Y+r.H > 40 {
			t.Fatalf("utility %q=%+v outside header", tc.label, r)
		}
		if title.X+title.W > r.X {
			t.Fatalf("ancestry=%+v overlaps %q=%+v", title, tc.label, r)
		}
		if err := tt.Click(tc.label); err != nil {
			t.Fatal(err)
		}
		if calls[tc.call] != 1 {
			t.Fatalf("%q dispatched %d times", tc.call, calls[tc.call])
		}
	}
	saveConversationEvidence(t, tt, "conversation-header-400")
}

func TestConversationHeaderShortTitleDoesNotStretchTo280AndShowsSubagentCatalog(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Center = 900
	catalog := make([]session.SubagentCatalogEntry, 7)
	for i := range catalog {
		catalog[i] = session.SubagentCatalogEntry{ID: fmt.Sprintf("sub-%d", i+1), Label: fmt.Sprintf("Sub %d", i+1)}
	}
	snap := session.ViewState{
		SelectedID:          "s1",
		SelectedAgentPreset: "standard",
		Items:               []session.Item{{SessionID: "s1", Title: "短标题"}},
		SubagentCatalog:     catalog,
	}
	agentsClicked := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			conversationHeaderTitleRow(c, Model{
				Store:            session.New(),
				Sizes:            &sizes,
				HideChatViewTabs: true,
				OnOpenAgents:     func() { agentsClicked++ },
			}, snap, theme.Resolve("deepseek.light").Light)
		})
	}, 900, 100)
	title, ok := tt.Find("Conversation session title")
	if !ok {
		t.Fatal("missing conversation session title")
	}
	if title.W >= 200 {
		t.Fatalf("short title stretched to %v, want tight fit well below 280", title.W)
	}
	agentsBtn, ok := tt.Find(i18n.T(i18n.KeyFrameCaptionAgents))
	if !ok {
		t.Fatal("missing agents button")
	}
	if agentsBtn.X-(title.X+title.W) > 16 {
		t.Fatalf("agents button too far from title: title=%+v agents=%+v", title, agentsBtn)
	}
	subBtn, ok := tt.Find("Subagents")
	if !ok {
		t.Fatal("missing subagents button from SubagentCatalog")
	}
	if !tt.HasText("7 个子智能体") && !tt.HasText("7 subagents") {
		t.Fatalf("expected 7 subagents label in header; texts=%q", tt.Texts())
	}
	if err := tt.Click("Subagents"); err != nil {
		t.Fatal(err)
	}
	if agentsClicked != 1 {
		t.Fatalf("clicking Subagents dispatched %d times, want 1", agentsClicked)
	}
	_ = subBtn
	saveConversationEvidence(t, tt, "conversation-header-900")
}

func TestRightDrawerExpandButtonRendersWhenRightPaneClosed(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Center = 800
	sizes.Right = 0
	toggled := 0
	rightWidth := float32(0)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			m := Model{
				Store:         session.New(),
				Sizes:         &sizes,
				RightPaneOpen: false,
				FilePaneWidth: &rightWidth,
				OnToggleTools: func() { toggled++ },
			}
			centerRightColumns(c, m,
				func() { ui.Box(c).Fill() },
				func() { ui.Box(c).Fill() },
			)
		})
	}, 800, 600)
	label := paneLabel("展开右侧栏", "Expand right sidebar")
	btn, ok := tt.Find(label)
	if !ok {
		t.Fatalf("missing right drawer expand button %q when RightPaneOpen is false", label)
	}
	if btn.X != 780 || btn.Y != 56 || btn.W != 20 || btn.H != 32 {
		t.Fatalf("unexpected right drawer expand button geometry: %+v, want X=780 Y=56 W=20 H=32", btn)
	}
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	if toggled != 1 {
		t.Fatalf("toggled=%d, want 1", toggled)
	}
}
