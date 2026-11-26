package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSidebarDrawer(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(oldLocale)

	t.Run("modal escape, focus trap, and focus restore", func(t *testing.T) {
		sizes := DefaultLayout()
		tt := ui.NewTester(func(c *ui.Context) {
			sidebarColumns(c, &sizes, func() {
				ui.Column(c).Fill().Children(func() {
					ui.Button(c, "Drawer action A").Focusable()
					ui.Button(c, "Drawer action B").Focusable()
				})
			}, func() {
				ui.Column(c).Fill().Children(func() {
					ui.Button(c, "Center chat button").Focusable()
				})
			})
		}, 640, 800)

		if !sizes.drawerMode() {
			t.Fatal("expected drawerMode at 640x800")
		}
		openLabel := i18n.T(i18n.KeySidebarExpand)
		if err := tt.Click(openLabel); err != nil {
			t.Fatalf("could not open drawer via %q: %v", openLabel, err)
		}
		if _, found := tt.Find("Drawer action A"); !found {
			t.Fatal("drawer content did not mount after clicking menu button")
		}

		// Tabbing while modal drawer is open must stay trapped inside the drawer and never reach Center chat button.
		for i := 0; i < 6; i++ {
			tt.Key(0, ui.KeyTab)
			if tt.Focused("Center chat button") {
				t.Fatalf("Tab leaked through open drawer into center content on iteration %d", i)
			}
		}

		// Escape must close the drawer and restore focus to the open-drawer button.
		tt.Key(0, ui.KeyEscape)
		if _, found := tt.Find("Drawer action A"); found {
			t.Fatal("Escape did not close the narrow sidebar drawer")
		}
		if !tt.Focused(openLabel) {
			t.Fatalf("closing drawer did not restore focus to %q", openLabel)
		}
	})

	t.Run("blank and subagent headers clear floating drawer button", func(t *testing.T) {
		sizes := DefaultLayout()
		sizes.observeSidebarOrientation(640, 800, false, true)
		if !sizes.drawerMode() {
			t.Fatal("expected drawerMode at 640x800")
		}

		// Blank conversation header leading contribution must start at X >= 46.
		ttBlank := ui.NewTester(func(c *ui.Context) {
			m := Model{
				Sizes: &sizes,
				ConversationContributions: func(c *ui.Context, _ session.ViewState, slot string) {
					if slot == "conversation.session.header.leading" {
						ui.Text(c, "Blank Leading")
					}
				},
			}
			conversationBlankHeader(c, m, session.ViewState{})
		}, 640, 120)
		blankLead, ok := ttBlank.Find("Blank Leading")
		if !ok {
			t.Fatalf("blank header leading content missing: %v", ttBlank.Texts())
		}
		if blankLead.X < 46 {
			t.Fatalf("blank header leading content X=%.1f overlaps floating drawer button (want >= 46)", blankLead.X)
		}

		// Subagent conversation header back button must start at X >= 46.
		store := session.New()
		store.Select("parent-1")
		w := &WorkspacePanes{}
		leaf := w.leaf("subagentchat")
		leaf.SubagentChat = &SubagentChatPaneState{
			Root:    "parent-1",
			Address: session.AgentSessionAddress{ParentSessionID: "parent-1", ChildSessionID: "child-1"},
			Title:   "Subagent",
		}
		w.conversation = &mainChildConversation{Pane: leaf, Owner: store, Binding: store.PlanHistoryBinding()}
		sizes.Workspace = w

		ttSub := ui.NewTester(func(c *ui.Context) {
			m := Model{Store: store, Sizes: &sizes}
			mainSubagentConversationHeader(c, m)
		}, 640, 120)
		backBtn, ok := ttSub.Find("Back to parent conversation")
		if !ok {
			t.Fatalf("subagent header back button missing: %v", ttSub.Texts())
		}
		if backBtn.X < 46 {
			t.Fatalf("subagent header back button X=%.1f overlaps floating drawer button (want >= 46)", backBtn.X)
		}
	})
}
