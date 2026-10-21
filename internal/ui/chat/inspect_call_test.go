package chat

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestInspectCallHandlerCarriesMessageAndSeq(t *testing.T) {
	var gotCall, gotMessage string
	var gotSeq float64
	handler := inspectCallHandler(func(callID, messageID string, seq float64) {
		gotCall, gotMessage, gotSeq = callID, messageID, seq
	}, "msg-7", 41)
	if handler == nil {
		t.Fatal("handler missing for a wired callback")
	}
	handler("call-9")
	if gotCall != "call-9" || gotMessage != "msg-7" || gotSeq != 41 {
		t.Fatalf("inspect args = %q %q %v", gotCall, gotMessage, gotSeq)
	}
	if inspectCallHandler(nil, "msg-7", 41) != nil {
		t.Fatal("nil callback must stay nil so the card hides the action")
	}
}

// The transcript's tool card offers the inspect action and reports which call and
// which log record it belongs to.
func TestToolCardInspectButtonReportsCallIdentity(t *testing.T) {
	result := &session.Message{Kind: session.MessageKindTool, Role: "tool", EventType: "tool/result",
		Raw: []byte(`{"type":"tool/result","seq":12,"data":{"message":{"content":[{"type":"text","text":"ok"}]}}}`)}
	msg := session.Message{ID: "msg-tool", Role: "assistant", Kind: session.MessageKindTool, Seq: 12,
		Tool: &session.ToolCallView{CallID: "call-1", Name: "bash", State: "ok",
			Arguments: `{"command":"go test ./internal/ui/chat"}`, Result: result}}
	var calls []string
	var messages []string
	expanded := map[string]bool{"msg-tool|tool:call-1": true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			ToolMessageView(c, msg, ToolCardStyle{FG: ui.RGB(20, 20, 20), Muted: ui.RGB(110, 110, 110), Field: ui.RGB(245, 246, 247)},
				ToolImageGalleryOptions{ID: msg.ID, Expanded: &expanded,
					OnInspectCall: inspectCallHandler(func(callID, messageID string, seq float64) {
						calls = append(calls, callID)
						messages = append(messages, messageID)
					}, msg.ID, msg.Seq)})
		})
	}, 900, 520)
	if err := tt.Click(i18n.T(i18n.KeyChatToolInspect)); err != nil {
		t.Fatalf("inspect button missing: %v (%q)", err, tt.Texts())
	}
	if len(calls) != 1 || calls[0] != "call-1" || messages[0] != "msg-tool" {
		t.Fatalf("inspect calls = %v %v", calls, messages)
	}
}
