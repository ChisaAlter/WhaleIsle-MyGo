package session

import (
	"encoding/json"
	"testing"
)

func TestSavedQuestionReplyCurrentAndLegacyAppendMarkers(t *testing.T) {
	for _, tt := range []struct {
		name, marker string
		want         bool
	}{
		{"current append", `"append"`, true},
		{"legacy append", `{"op":"append"}`, true},
		{"current replacement", `{"op":"replace","startSeq":1,"endSeq":2}`, false},
		{"legacy replace", `"replace"`, false},
		{"missing", `null`, false},
		{"unknown", `"future"`, false},
		{"case differs", `"APPEND"`, false},
		{"number", `42`, false},
		{"array", `["append"]`, false},
		{"malformed object", `{"op":42}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			raw := json.RawMessage(`{"type":"user/message","seq":30,"time":1000,"surfaceOp":` + tt.marker + `,"data":{"id":"owned-reply","role":"user","source":{"kind":"user-question-reply","callId":"owned-call","outcome":"answered"},"content":[{"type":"text","text":"owned saved answer"}]}}`)
			message, ok := classifyQuestionReply(raw)
			if ok != tt.want {
				t.Fatalf("claimed=%v want=%v", ok, tt.want)
			}
			if tt.want && (message.Kind != MessageKindQuestionReply || message.ID != "owned-reply" || message.Body != "owned saved answer" || string(message.Raw) != string(raw)) {
				t.Fatalf("reply identity/content lost: %+v", message)
			}
			projected, _ := classifyPersistenceEvent(raw)
			if (projected.Kind == MessageKindQuestionReply) != tt.want {
				t.Fatalf("classification chain returned %s", projected.Kind)
			}
		})
	}
}
