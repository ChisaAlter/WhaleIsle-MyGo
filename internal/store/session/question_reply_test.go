package session

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/store/storelocale"

)

func TestQuestionReplyClassifiesBeforeContext(t *testing.T) {
	raw := json.RawMessage(`{"type":"user/message","seq":4,"time":1000,"surfaceOp":{"op":"append"},"data":{"id":"reply","source":{"kind":"user-question-reply","callId":"call","outcome":"answered"},"content":[{"type":"text","text":"fallback reply"}]}}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || string(msg.Kind) != "question-reply" || msg.Body != "fallback reply" || string(msg.Raw) != string(raw) {
		t.Fatalf("reply lost or generic context: %+v", msg)
	}
}

func TestQuestionReplyBoundaryAndStructuredValues(t *testing.T) {
	body := `{"questions":[{"id":"q","header":"Heading","question":"Pick","detail":"Details","options":[{"label":"yes","description":"Why"}]},{"id":"skip","question":"Skip?"}],"answers":[{"id":"q","selected":["yes"],"custom":" custom "},{"id":"skip","selected":[]}]}`
	event := map[string]any{"type": "user/message", "seq": 4, "surfaceOp": map[string]string{"op": "append"}, "data": map[string]any{"source": map[string]string{"kind": "user-question-reply", "callId": "call", "outcome": "answered"}, "content": []map[string]string{{"type": "text", "text": body}}}}
	raw, _ := json.Marshal(event)
	msg, _ := classifyPersistenceEvent(raw)
	details := QuestionReplyDetails(msg)
	if len(details.Questions) != 2 || details.Questions[0].Options[0].Description != "Why" || len(details.AnswerValues("q")) != 2 {
		t.Fatalf("lost context %+v", details)
	}
	want := "Heading — Pick\n" + storelocale.T(storelocale.KeyStoreSessionAnswer) + "yes, custom\n\nSkip?\n" + storelocale.T(storelocale.KeyStoreSessionSkipped)
	if details.ClipboardText() != want {
		t.Fatal(details.ClipboardText())
	}
	event["surfaceOp"] = map[string]string{"op": "replace"}
	raw, _ = json.Marshal(event)
	msg, _ = classifyPersistenceEvent(raw)
	if msg.Kind == MessageKindQuestionReply {
		t.Fatal("replace event claimed reply")
	}
	if msg.Kind != MessageKindContext {
		t.Fatal("non-reply context classification changed")
	}
	bad := QuestionReplyDetails(Message{Body: "not-json"})
	if bad.ClipboardText() != "not-json" {
		t.Fatal("fallback lost")
	}
}
