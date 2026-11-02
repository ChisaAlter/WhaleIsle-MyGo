package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestConversationPluginsExplicitMenuAndOriginalActionsRemainReachable(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	active := false
	opened, closed, launcher := 0, 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		conversationMoreMenu(c, Model{SessionPluginsActive: active, OnOpenSessionPlugins: func() { opened++; active = true }, OnCloseSessionPlugins: func() { closed++; active = false }, OnOpenLauncher: func() { launcher++ }}, session.ViewState{SelectedID: "s"}, c.Theme().Text)
	}, 400, 100)
	if tt.HasText(i18n.T(i18n.KeyCmdSessionPluginTitle)) {
		t.Fatal("plugin label is a persistent header button")
	}
	for _, label := range []string{i18n.T(i18n.KeyCmdSessionPluginTitle), "Turn off Session plugins", i18n.T(i18n.KeyLauncherOpen)} {
		if err := tt.Click(i18n.T(i18n.KeyLauncherOpen)); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(label); err != nil {
			t.Fatal(err)
		}
	}
	if opened != 1 || closed != 1 || launcher != 1 || active {
		t.Fatalf("open/close/original=%d/%d/%d active=%v", opened, closed, launcher, active)
	}
}

func TestConversationMoreMenuFoldedViewsAndEntryCustomizationActions(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	tab := ""
	marketCalls := 0
	var draftRegion string
	tt := ui.NewTester(func(c *ui.Context) {
		conversationMoreMenu(c, Model{
			HideChatViewTabs:    true,
			ChatPaneTab:         &tab,
			OnOpenMarket:        func() { marketCalls++ },
			OnCreatePluginDraft: func(region string) { draftRegion = region },
		}, session.ViewState{}, c.Theme().Text)
	}, 400, 100)

	if err := tt.Click(i18n.T(i18n.KeyLauncherOpen)); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Trajectory"); err != nil {
		t.Fatal(err)
	}
	if tab != "trajectory" {
		t.Fatalf("tab=%q want trajectory", tab)
	}

	if err := tt.Click(i18n.T(i18n.KeyLauncherOpen)); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("✓ Trajectory"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyLauncherOpen)); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Chat"); err != nil {
		t.Fatal(err)
	}
	if tab != "chat" {
		t.Fatalf("tab=%q want chat", tab)
	}

	if err := tt.Click(i18n.T(i18n.KeyLauncherOpen)); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Browse other plugins"); err != nil {
		t.Fatal(err)
	}
	if marketCalls != 1 {
		t.Fatalf("marketCalls=%d want 1", marketCalls)
	}

	if err := tt.Click(i18n.T(i18n.KeyLauncherOpen)); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Create your plugin"); err != nil {
		t.Fatal(err)
	}
	if draftRegion != "conversation" {
		t.Fatalf("draftRegion=%q want conversation", draftRegion)
	}
}

func TestConversationHeaderRendersContributionsInTheirOwnZones(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Center = 800
	tt := ui.NewTester(func(c *ui.Context) {
		conversationHeaderTitleRow(c, Model{Sizes: &sizes, Store: session.New(), ConversationContributions: func(c *ui.Context, _ session.ViewState, slot string) { ui.Button(c, "").Label(slot).Size(28, 28) }}, session.ViewState{SelectedID: "s"}, theme.Resolve("deepseek.light").Light)
	}, 800, 100)
	var previous float32
	for _, slot := range []string{"conversation.session.header.leading", "conversation.session.header.actions", "conversation.session.header.utilities", "conversation.session.header.corner"} {
		r, ok := tt.Find(slot)
		if !ok || r.Y < 0 || r.Y+r.H > 40 || r.X < previous {
			t.Fatalf("slot %s misplaced: %+v", slot, r)
		}
		previous = r.X
	}
}
