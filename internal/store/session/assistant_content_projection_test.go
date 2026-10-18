package session

import (
	"encoding/json"
	"testing"
)

func TestAssistantStructuredContentProjection(t *testing.T) {
	raw := json.RawMessage(`{"type":"assistant/message","seq":4,"data":{"message":{"id":"canonical","content":[{"type":"text","text":"before"},{"type":"image","attachment":{"attachmentId":"photo"}},{"type":"future-content","value":{"answer":42}},{"type":"reasoning","text":"why"},{"type":"text","text":"after"},{"type":"tool-call","id":"call"}]}}}`)
	msg, ok := classifyAssistantPersistence(raw)
	if !ok {
		t.Fatal("assistant event not classified")
	}
	rows := projectAssistantMessage(msg)
	if len(rows) != 5 {
		t.Fatalf("want text/image/unknown/reasoning/text, got %+v", rows)
	}
	if rows[1].ImageRef != "attachment:photo" {
		t.Fatalf("image missing: %+v", rows[1])
	}
	if string(rows[2].ContentBlock) != `{"type":"future-content","value":{"answer":42}}` {
		t.Fatalf("unknown block lost: %+v", rows[2])
	}
	ids := map[string]bool{}
	for _, row := range rows {
		if row.SourceMessageID != "canonical" || row.FeedbackMessageID() != "canonical" {
			t.Fatalf("feedback target lost: %+v", row)
		}
		if ids[row.ID] {
			t.Fatalf("duplicate render key: %q", row.ID)
		}
		ids[row.ID] = true
	}
	if rows[0].Body != "before" || rows[4].Body != "after" {
		t.Fatalf("text order: %+v", rows)
	}
}

func TestAssistantImageOnlyLiveAndDurableProjection(t *testing.T) {
	f := newAssistantFixture()
	f.start(t, "images", 1, 0)
	f.chunk(t, "images", 0, map[string]any{"type": "block-end", "index": 0, "block": map[string]any{"type": "image", "attachment": map[string]any{"attachmentId": "photo"}}})
	live := f.s.Snapshot().Messages
	if len(live) != 1 || live[0].ImageRef != "attachment:photo" || !live[0].Streaming {
		t.Fatalf("live image lost: %+v", live)
	}
	f.durable(t, assistantMessage(4, 1, 0, `[{"type":"image","attachment":{"attachmentId":"photo"}}]`, false))
	f.end(t, "images", 1, assistantCommit(4, "assistant/message"))
	settled := f.s.Snapshot().Messages
	if len(settled) != 1 || settled[0].ImageRef != "attachment:photo" || settled[0].Streaming || settled[0].SourceMessageID != "m4" {
		t.Fatalf("settled image lost or duplicated: %+v", settled)
	}
}
