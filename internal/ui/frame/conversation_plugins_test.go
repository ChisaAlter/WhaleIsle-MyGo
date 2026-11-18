package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestConversationMoreMenuOnlyOffersCurrentSessionActions(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	store := session.New()
	store.Select("s")
	var downloaded string
	more := Model{
		Store: store, HideChatViewTabs: true,
		OnOpenMarket:          func() { t.Fatal("global market action ran") },
		OnCreatePluginDraft:   func(string) { t.Fatal("global create action ran") },
		OnOpenDesktopSettings: func() { t.Fatal("global settings action ran") },
		OnDownloadSessionLog:  func(id string) { downloaded = id },
	}
	selected := "s"
	tt := ui.NewTester(func(c *ui.Context) {
		conversationMoreMenu(c, more, session.ViewState{SelectedID: selected}, c.Theme().Text)
	}, 400, 140)
	if err := tt.Click("More actions"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Download session log"); err != nil {
		t.Fatal(err)
	}
	if downloaded != "s" {
		t.Fatalf("downloaded = %q", downloaded)
	}
	for _, label := range []string{"Chat", "Trajectory", "Browse other plugins", "Create your plugin", "Desktop settings"} {
		if err := tt.Click("More actions"); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(label); err == nil {
			t.Fatalf("unexpected global menu item %q", label)
		}
		tt.Key(0, ui.KeyEscape)
	}
	selected = ""
	tt.Frame()
	if _, ok := tt.Find("More actions"); ok {
		t.Fatal("more menu with no selected session")
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
