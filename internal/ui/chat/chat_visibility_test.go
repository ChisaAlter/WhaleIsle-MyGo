package chat

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func contextFixture(id, kind, blocks string) session.Message {
	return session.Message{ID: id, Kind: session.MessageKindContext, Role: "context", Seq: 2,
		Body: "retained context " + id, Raw: json.RawMessage(`{"type":"user/message","seq":2,"data":{"id":"` + id + `","source":{"kind":"` + kind + `","form":"snapshot"},"content":` + blocks + `}}`)}
}

func TestMainChatVisibilityRetainsRawInputsAndInspectorNodes(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	for _, mode := range []string{"compact", "standard", "detailed", "verbose"} {
		t.Run(mode, func(t *testing.T) {
			runtime := contextFixture("runtime", "runtime-context", `[{"type":"text","text":"private runtime"}]`)
			clock := contextFixture("clock", "time-context", `[{"type":"text","text":"private time"}]`)
			developer := contextFixture("developer", "unknown-extension", `[]`)
			developer.Raw = json.RawMessage(`{"type":"developer/message","seq":3,"data":{"message":{"source":{"kind":"unknown-extension"},"content":[{"type":"text","text":"private developer"}]}}}`)
			tool := contextFixture("tools", "tool-registry", `[{"type":"tool-removal","toolName":"old_tool"}]`)
			mixed := contextFixture("mixed", "tool-registry", `[{"type":"text","text":"mixed text"},{"type":"tool-addition","toolName":"new_tool"}]`)
			wake := contextFixture("wake", "schedule", `[{"type":"text","text":"claimed trigger detail"}]`)
			wake.ContextWaking = true
			messages := []session.Message{{ID: "human", Role: "user", Kind: session.MessageKindText, Body: "visible human"}, runtime, clock, developer,
				{ID: "prompt", Kind: session.MessageKindSystemPrompt, Body: "private system"},
				{ID: "permission", Kind: session.MessageKindCommand, Command: &session.CommandView{Name: "permission"}}, tool, mixed, wake}
			before := append([]session.Message(nil), messages...)
			visible := visibleChatMessages(messages)
			if got := inspectorMessageIDs(visible); !reflect.DeepEqual(got, []string{"human", "tools", "mixed", "wake"}) {
				t.Fatalf("chat order = %v", got)
			}
			inspector := &InspectorNavigation{SessionID: "s"}
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Snapshot: session.ViewState{SelectedID: "s", Messages: messages}, Preferences: settings.ChatPreferences{TranscriptView: mode}, Inspector: inspector})
			}, 900, 900)
			for _, unwanted := range []string{"runtime-context", "time-context", "private runtime", "private time", "private system", "private developer"} {
				if tt.HasText(unwanted) {
					t.Fatalf("hidden input %q still in chat: %q", unwanted, tt.Texts())
				}
			}
			for _, want := range []string{"visible human", "Tool removed: old_tool", "Context injection", "Automation task"} {
				if !tt.HasText(want) {
					t.Fatalf("visible notice %q lost: %q", want, tt.Texts())
				}
			}
			if !reflect.DeepEqual(messages, before) || inspectorMessageIndex(inspector.Nodes, InspectorTarget{ID: "runtime"}) < 0 || inspectorMessageIndex(inspector.Nodes, InspectorTarget{ID: "prompt"}) < 0 {
				t.Fatal("chat projection changed the snapshot or inspector inputs")
			}
			if tt.HasText("claimed trigger detail") {
				t.Fatal("trigger did not start collapsed")
			}
			if err := tt.Click("Automation task"); err != nil || !tt.HasText("claimed trigger detail") {
				t.Fatalf("trigger detail unavailable: %v, %q", err, tt.Texts())
			}
		})
	}
}

func TestHiddenContextStillSuppliesRecallAndSkillFacts(t *testing.T) {
	human := session.Message{ID: "user", Seq: 10, Kind: session.MessageKindText, Role: "user", Body: "use @skills/test and @session/ref"}
	recall := contextFixture("recall", "session-reference", `[]`)
	recall.Seq = 11
	recall.Raw = json.RawMessage(`{"type":"user/message","data":{"source":{"kind":"session-reference","references":[{"label":"Recorded session"}]}}}`)
	skill := contextFixture("skill", "skill-invocation", `[]`)
	skill.Seq = 12
	skill.Raw = json.RawMessage(`{"type":"user/message","data":{"source":{"kind":"skill-invocation","name":"Recorded skill"}}}`)
	messages := []session.Message{human, recall, skill}
	if len(visibleChatMessages(messages)) != 1 || strings.Join(userMessageReferenceLabels(messages, 10), ",") != "Recorded session" || strings.Join(userMessageSkillNames(messages, human), ",") != "Recorded skill" {
		t.Fatal("hidden context no longer supplies recorded user references")
	}
}
