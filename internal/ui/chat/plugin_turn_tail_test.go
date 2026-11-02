package chat

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

func TestPluginTurnTailEntryPrecedesNormalAndFailureFooterAfterFolding(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	for _, reason := range []string{`{"kind":"complete"}`, `{"kind":"error","error":{"code":"FAILED","message":"Fixture failure"}}`} {
		t.Run(reason, func(t *testing.T) {
			messages := []session.Message{
				{ID: "thinking", Kind: session.MessageKindThinking, Role: "assistant", Body: "Earlier reasoning", Seq: 2, Raw: json.RawMessage(`{"type":"assistant/message","seq":2,"data":{"turn":0,"step":0}}`)},
				{ID: "answer", Kind: session.MessageKindText, Role: "assistant", Body: "Closing prose", Seq: 3, Raw: json.RawMessage(`{"type":"assistant/message","seq":3,"data":{"turn":0,"step":1}}`)},
				{ID: "end", Kind: session.MessageKindText, Role: "status", EventType: "turn/end", Seq: 4, Raw: json.RawMessage(`{"type":"turn/end","seq":4,"data":{"turn":0,"reason":` + reason + `}}`)},
			}
			m := Model{Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "owned", Messages: messages}, Preferences: settings.ChatPreferences{TranscriptView: "standard"}, ThemeID: theme.IDDeepseekLight, SuppressStatus: true, FullColumnViewport: true, List: &ui.ListState{FollowEnd: true}}
			clicked := 0
			m.TurnContributions = func(c *ui.Context, snapshot session.ViewState, msg session.Message) {
				if msg.ID != "end" || len(snapshot.Messages) != 3 {
					t.Fatal("entry called with synthetic/group row")
				}
				if ui.Button(c, "Open fixture turn").Clicked() {
					clicked++
				}
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 390, 500)
			entry, ok := tt.Find("Open fixture turn")
			if !ok {
				t.Fatalf("missing completed turn entry: %v", tt.Texts())
			}
			answer, _ := tt.Find("Closing prose")
			if entry.Y < answer.Y+answer.H {
				t.Fatalf("entry before closing prose %+v %+v", answer, entry)
			}
			if err := tt.Click("Open fixture turn"); err != nil {
				t.Fatal(err)
			}
			if clicked != 1 {
				t.Fatal("footer did not forward gesture")
			}
			// The production bridge, not chat, must validate raw; chat forwards it intact.
			m.Snapshot.Messages = m.Snapshot.Messages[:2]
			tt.Frame()
			if _, ok := tt.Find("Open fixture turn"); ok {
				t.Fatal("missing closing row retained plugin entry")
			}
		})
	}
}
