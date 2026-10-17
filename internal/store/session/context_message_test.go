package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInjectedContextIsNotHumanSpeech(t *testing.T) {
	raw := json.RawMessage(`{"type":"user/message","seq":7,"time":42,"data":{"source":{"kind":"agent-instructions","form":"instructions","changes":[{"path":"AGENTS.md","action":"add"}]},"content":[{"type":"text","text":"<system-reminder>instructions</system-reminder>"}]}}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Role == "user" || string(msg.Kind) != "context" {
		t.Fatalf("context misclassified: %+v", msg)
	}
	if msg.Body != "<system-reminder>instructions</system-reminder>" || string(msg.Raw) != string(raw) {
		t.Fatal("context lost model-facing content or original evidence")
	}
	if LatestEditableMessage([]Message{msg}) != 0 {
		t.Fatal("context is editable as human")
	}
}

func TestContextProducerMetadataAndUnknownFormsRemainReadable(t *testing.T) {
	for _, tc := range []struct{ source, producer, role, form string }{
		{`{"kind":"agent-instructions","form":"instructions","changes":[{"path":"AGENTS.md"},{"path":"AGENTS.md"}]}`, "AGENTS.md", "inject", "instructions"},
		{`{"kind":"session-reference","form":"recall","references":[{"label":"Old task","omitted":8}]}`, "Old task", "recall", "recall"},
		{`{"kind":"skill-invocation","name":"writer","form":"catalog","entries":[]}`, "writer", "inject", "catalog"},
		{`{"kind":"third-party","form":"future","custom":{"keep":"evidence"}}`, "third-party", "inject", ""},
		{`{"kind":"state","form":"snapshot","sections":[{"name":"state","text":"unchanged"}]}`, "state", "inject", "snapshot"},
		{`{"kind":"notice","form":"notice"}`, "notice", "inject", "notice"},
		{`{"kind":"relay","form":"relay","sender":"child"}`, "relay", "inject", "relay"},
		{`null`, "unknown", "inject", ""},
	} {
		raw := json.RawMessage(`{"type":"user/message","seq":7,"data":{"source":` + tc.source + `,"content":[{"type":"text","text":"literal text"},{"type":"future","payload":"kept"}]}}`)
		msg, ok := classifyPersistenceEvent(raw)
		if !ok || msg.Kind != MessageKindContext {
			t.Fatalf("source %s: %+v", tc.source, msg)
		}
		v := ContextMessageDetails(msg)
		if v.Producer != tc.producer || v.Role != tc.role || v.Form != tc.form || len(v.Fields) == 0 {
			t.Fatalf("source %s: %+v", tc.source, v)
		}
		if !strings.Contains(v.Body, "literal text") || !strings.Contains(v.Body, "kept") {
			t.Fatal("opaque block lost")
		}
	}
}

func TestContextClassificationPreservesHumansAndCompaction(t *testing.T) {
	for _, source := range []string{``, `,"source":{"kind":"user"}`} {
		raw := json.RawMessage(`{"type":"user/message","seq":8,"data":{"content":[{"type":"text","text":"human"}]` + source + `}}`)
		msg, ok := classifyPersistenceEvent(raw)
		if !ok || msg.Role != "user" || msg.Kind != MessageKindText {
			t.Fatalf("human changed: %+v", msg)
		}
	}
	raw := json.RawMessage(`{"type":"developer/message","seq":9,"data":{"message":{"id":"dev","source":{"kind":"developer","form":"notice"},"content":[{"type":"text","text":"developer context"}]}}}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Role != "context" || msg.SourceMessageID != "dev" || msg.Body != "developer context" {
		t.Fatalf("developer lost: %+v", msg)
	}
	// Existing manual compaction owns its checkpoint before generic context.
	msg, ok = classifyPersistenceEvent(json.RawMessage(compactCheckpoint))
	if !ok || msg.Kind != MessageKindCompaction {
		t.Fatalf("manual checkpoint changed: %+v", msg)
	}
}
