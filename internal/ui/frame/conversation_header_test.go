package frame

import (
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
