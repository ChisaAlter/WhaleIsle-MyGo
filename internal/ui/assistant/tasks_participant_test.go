package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func TestTaskParticipantConversationNavigation(t *testing.T) {
	for _, tc := range []struct {
		name, kind, session string
		loaded, want        bool
	}{
		{"current bot", "bot", "session-a", true, true},
		{"deleted bot", "", "", true, false},
		{"missing session", "bot", " ", true, false},
		{"room is not a participant", "room", "room-session", true, false},
		{"stale catalog", "bot", "session-a", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opened := ""
			catalog := pluginrpc.DshbotContactCatalog{DshbotBotCatalog: pluginrpc.DshbotBotCatalog{Items: []pluginrpc.DshbotBotItem{{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "a", Kind: tc.kind, SessionID: tc.session, Name: "Current A"}}}}}
			tt := ui.NewTester(func(c *ui.Context) {
				m := Model{Mode: assiststore.ModeDshbot, HarnessReady: true, DshbotEnabled: true,
					Contacts:                  ContactsModel{State: dshbotstore.ContactSnapshot{Loaded: tc.loaded, Catalog: catalog}},
					OnDshbotOpenRosterSession: func(id string) { opened = id },
					Tasks:                     TaskModel{OnRefresh: func() {}, State: dshbotstore.TaskControlSnapshot{Scope: dshbotstore.TaskScope{BotID: "b"}, Loaded: true, SelectedID: "t", Tasks: []pluginrpc.DshbotTask{{ID: "t", FromID: "a", FromName: "Historical A", ToID: "deleted", ToName: "Historical B", Task: "Task", Status: "completed"}}}},
				}
				taskControlBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(80, 130, 220))
			}, 1100, 1800)
			label := i18n.T(i18n.KeyMiscAssistantOpenSenderConversation)
			if tt.HasText(label) != tc.want {
				t.Fatalf("navigation availability=%v want %v: %q", tt.HasText(label), tc.want, tt.Texts())
			}
			if tc.want {
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
				if opened != "a" {
					t.Fatalf("opened %q", opened)
				}
			}
			if tt.HasText(i18n.T(i18n.KeyMiscAssistantOpenRecipientConversation)) {
				t.Fatal("deleted recipient exposed navigation")
			}
		})
	}
}
