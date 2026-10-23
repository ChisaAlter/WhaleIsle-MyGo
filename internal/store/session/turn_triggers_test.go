package session

import (
	"encoding/json"
	"reflect"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func contextTriggerEvent(t *testing.T, seq float64, typ string, data any) json.RawMessage {
	t.Helper()
	event := map[string]any{"seq": seq, "type": typ, "data": data}
	if typ == "user/message" {
		event["surfaceOp"] = "append"
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func contextTriggerInput(id, source string) map[string]any {
	return map[string]any{
		"id": id, "source": map[string]any{"kind": source},
		"content": []map[string]any{{"type": "text", "text": "input " + id}},
	}
}

func contextTriggerSplice(t *testing.T, seq float64, target string, start int, removedCount *int, outcome string, inserted ...map[string]any) json.RawMessage {
	t.Helper()
	if inserted == nil {
		inserted = []map[string]any{}
	}
	data := map[string]any{"target": target, "start": start, "inserted": inserted}
	if removedCount != nil {
		data["removedCount"] = *removedCount
	}
	if outcome != "" {
		data["outcome"] = outcome
	}
	return contextTriggerEvent(t, seq, "agent/inbox/spliced", data)
}

func contextTriggerCount(value int) *int { return &value }

func contextTriggerSnapshot(t *testing.T, s *Store, events ...json.RawMessage) {
	t.Helper()
	records := make([]map[string]any, 0, len(events))
	for _, event := range events {
		records = append(records, map[string]any{"type": "event", "event": event})
	}
	raw, err := json.Marshal(map[string]any{"type": "snapshot", "records": records})
	if err != nil {
		t.Fatal(err)
	}
	s.applyFollowFrame(raw)
}

func contextTriggerPrepend(t *testing.T, s *Store, events ...json.RawMessage) {
	t.Helper()
	records := make([]map[string]any, 0, len(events))
	for _, event := range events {
		records = append(records, map[string]any{"type": "event", "event": event})
	}
	raw, err := json.Marshal(map[string]any{"records": records})
	if err != nil {
		t.Fatal(err)
	}
	var page api.SessionPage
	if err := json.Unmarshal(raw, &page); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.prependPageRecordsLocked(page.Records)
	s.mu.Unlock()
}

func contextTriggerMessage(t *testing.T, s *Store, seq float64) Message {
	t.Helper()
	for _, message := range s.Snapshot().Messages {
		if message.Seq == seq && (message.Kind == MessageKindContext || message.Role == "user") {
			return message
		}
	}
	t.Fatalf("message at seq %g missing", seq)
	return Message{}
}

func contextTriggerLoad(t *testing.T, mode string, before []json.RawMessage, message json.RawMessage) *Store {
	t.Helper()
	s := New()
	s.selectedID = "context-trigger"
	switch mode {
	case "replay":
		contextTriggerSnapshot(t, s, append(append([]json.RawMessage(nil), before...), message)...)
	case "live":
		contextTriggerSnapshot(t, s)
		for _, event := range append(append([]json.RawMessage(nil), before...), message) {
			s.applyFollowFrame(append(append(json.RawMessage(`{"type":"event","event":`), event...), '}'))
			_ = s.Snapshot() // Exercise publication between admissions, not only the final fold.
		}
	case "prepend":
		contextTriggerSnapshot(t, s, message)
		if got := contextTriggerMessage(t, s, 11); got.ContextWaking {
			t.Fatal("unloaded claim already woke input")
		}
		contextTriggerPrepend(t, s, before...)
	default:
		t.Fatalf("unknown mode %q", mode)
	}
	return s
}

func TestContextTriggerReferenceCases(t *testing.T) {
	notice := contextTriggerInput("notice", "schedule")
	human := contextTriggerInput("human", "user")
	insertion := func(target string, inputs ...map[string]any) json.RawMessage {
		if len(inputs) == 0 {
			inputs = []map[string]any{notice}
		}
		return contextTriggerSplice(t, 1, target, 0, nil, "", inputs...)
	}
	claim := func(target string, count int) json.RawMessage {
		return contextTriggerSplice(t, 3, target, 0, contextTriggerCount(count), "")
	}
	start := contextTriggerEvent(t, 2, "turn/start", map[string]any{"turn": 1})
	step := contextTriggerEvent(t, 8, "step/start", map[string]any{"turn": 1, "step": 1})
	cases := []struct {
		name   string
		before []json.RawMessage
		waking bool
	}{
		{"next-turn claim", []json.RawMessage{insertion("next-turn"), start, claim("next-turn", 1), step}, true},
		{"idle non-human steer", []json.RawMessage{insertion("next-step"), start, claim("next-step", 1), step}, true},
		{"unclaimed input", []json.RawMessage{insertion("next-turn"), start, step}, false},
		{"canceled input", []json.RawMessage{insertion("next-turn"), start, contextTriggerSplice(t, 3, "next-turn", 0, contextTriggerCount(1), "canceled"), step}, false},
		{"requeued claim", []json.RawMessage{insertion("next-turn"), start, claim("next-turn", 1), contextTriggerSplice(t, 4, "next-turn", 0, nil, "", notice), step}, false},
		{"human steer in same batch", []json.RawMessage{insertion("next-step", notice, human), start, claim("next-step", 2), step}, false},
		{"queued message starts turn", []json.RawMessage{insertion("next-step"), start, claim("next-step", 1), contextTriggerSplice(t, 4, "next-turn", 0, nil, "", human), contextTriggerSplice(t, 5, "next-turn", 0, contextTriggerCount(1), ""), step}, false},
		{"later step", []json.RawMessage{insertion("next-step"), start, claim("next-step", 1), step, contextTriggerEvent(t, 9, "step/end", map[string]any{"turn": 1, "step": 1}), contextTriggerEvent(t, 10, "step/start", map[string]any{"turn": 1, "step": 2})}, false},
		{"missing turn start", []json.RawMessage{insertion("next-step"), claim("next-step", 1), step}, false},
		{"missing step", []json.RawMessage{insertion("next-step"), start, claim("next-step", 1)}, false},
		{"claim before turn", []json.RawMessage{insertion("next-step"), claim("next-step", 1), contextTriggerEvent(t, 4, "turn/start", map[string]any{"turn": 1}), step}, false},
		{"not in claim", []json.RawMessage{insertion("next-step", contextTriggerInput("other", "schedule")), start, claim("next-step", 1), step}, false},
	}
	for _, mode := range []string{"replay", "live", "prepend"} {
		for _, test := range cases {
			t.Run(mode+"/"+test.name, func(t *testing.T) {
				raw := contextTriggerEvent(t, 11, "user/message", notice)
				s := contextTriggerLoad(t, mode, test.before, raw)
				message := contextTriggerMessage(t, s, 11)
				if message.ContextWaking != test.waking || message.Kind != MessageKindContext || message.Role != "context" {
					t.Fatalf("context projection: %+v; want waking %t", message, test.waking)
				}
				if string(message.Raw) != string(raw) || message.SourceMessageID != "notice" || message.Body != "input notice" {
					t.Fatalf("changed source input: %+v", message)
				}
				if len(s.InspectorSnapshot().Records) != len(test.before)+1 {
					t.Fatal("trigger projection changed retained window")
				}
			})
		}
	}
}

func TestContextTriggerSpliceSemantics(t *testing.T) {
	a, b, c := contextTriggerInput("a", "schedule"), contextTriggerInput("b", "schedule"), contextTriggerInput("c", "schedule")
	for _, target := range []string{"next-turn", "next-step"} {
		cases := []struct {
			name    string
			splices []json.RawMessage
			claimed map[string]bool
		}{
			{"omitted count keeps pending", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a, c),
				contextTriggerSplice(t, 2, target, 1, nil, "", b),
				contextTriggerSplice(t, 5, target, 0, contextTriggerCount(3), ""),
			}, map[string]bool{"a": true, "b": true, "c": true}},
			{"negative start and capped count", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a, b, c),
				contextTriggerSplice(t, 5, target, -2, contextTriggerCount(99), ""),
			}, map[string]bool{"b": true, "c": true}},
			{"negative start caps at zero", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a, b),
				contextTriggerSplice(t, 5, target, -99, contextTriggerCount(1), ""),
			}, map[string]bool{"a": true}},
			{"large insertion start caps at end", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a),
				contextTriggerSplice(t, 2, target, 99, nil, "", b),
				contextTriggerSplice(t, 5, target, 1, contextTriggerCount(1), ""),
			}, map[string]bool{"b": true}},
			{"empty claim replaces prior claim", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a),
				contextTriggerSplice(t, 2, target, 0, contextTriggerCount(1), ""),
				contextTriggerSplice(t, 5, target, 99, contextTriggerCount(1), ""),
			}, map[string]bool{}},
			{"later claim replaces prior claim", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a, b),
				contextTriggerSplice(t, 2, target, 0, contextTriggerCount(1), ""),
				contextTriggerSplice(t, 5, target, 0, contextTriggerCount(1), ""),
			}, map[string]bool{"b": true}},
			{"negative count only removes reinserted claim IDs", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a, b),
				contextTriggerSplice(t, 2, target, 0, contextTriggerCount(2), ""),
				contextTriggerSplice(t, 5, target, 0, contextTriggerCount(-2), "", a),
			}, map[string]bool{"b": true}},
			{"canceled removal preserves prior claim", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a, b),
				contextTriggerSplice(t, 2, target, 0, contextTriggerCount(1), ""),
				contextTriggerSplice(t, 5, target, 0, contextTriggerCount(1), "canceled"),
			}, map[string]bool{"a": true}},
			{"canceled splice removes reinserted claim IDs", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a, b, c),
				contextTriggerSplice(t, 2, target, 0, contextTriggerCount(2), ""),
				contextTriggerSplice(t, 5, target, 0, contextTriggerCount(1), "canceled", a),
			}, map[string]bool{"b": true}},
			{"claim keeps its own inserted ID", []json.RawMessage{
				contextTriggerSplice(t, 1, target, 0, nil, "", a),
				contextTriggerSplice(t, 5, target, 0, contextTriggerCount(1), "", a),
			}, map[string]bool{"a": true}},
		}
		for _, test := range cases {
			t.Run(target+"/"+test.name, func(t *testing.T) {
				s := New()
				s.applyPersistenceEvent(contextTriggerEvent(t, 0, "turn/start", map[string]any{"turn": 1}))
				s.applyPersistenceEvent(contextTriggerEvent(t, 4, "step/start", map[string]any{"turn": 1, "step": 1}))
				for _, raw := range test.splices {
					s.applyPersistenceEvent(raw)
				}
				for i, input := range []map[string]any{a, b, c} {
					seq := float64(10 + i)
					s.applyPersistenceEvent(contextTriggerEvent(t, seq, "user/message", input))
					if got := contextTriggerMessage(t, s, seq).ContextWaking; got != test.claimed[input["id"].(string)] {
						t.Fatalf("input %s waking = %t; claimed %v", input["id"], got, test.claimed)
					}
				}
			})
		}
		t.Run(target+"/fractional and huge splice arguments", func(t *testing.T) {
			s := New()
			contextTriggerSnapshot(t, s,
				contextTriggerEvent(t, 0, "turn/start", map[string]any{"turn": 1}),
				contextTriggerSplice(t, 1, target, 0, nil, "", a, b),
				contextTriggerEvent(t, 2, "agent/inbox/spliced", map[string]any{"target": target, "start": 1e40, "inserted": []any{c}}),
				contextTriggerEvent(t, 3, "agent/inbox/spliced", map[string]any{"target": target, "start": -2.9, "removedCount": 1.9, "inserted": []any{}}),
				contextTriggerEvent(t, 4, "step/start", map[string]any{"turn": 1, "step": 1}),
				contextTriggerEvent(t, 5, "user/message", b),
			)
			if !contextTriggerMessage(t, s, 5).ContextWaking {
				t.Fatal("splice did not truncate fractional or cap huge start")
			}
			s.applyPersistenceEvent(contextTriggerEvent(t, 6, "agent/inbox/spliced", map[string]any{"target": target, "start": 0, "removedCount": 0.5, "inserted": []any{}}))
			s.applyPersistenceEvent(contextTriggerEvent(t, 7, "user/message", b))
			if contextTriggerMessage(t, s, 7).ContextWaking {
				t.Fatal("positive fractional count did not replace claim with its empty removal")
			}
		})
	}
}

func TestContextTriggerAvailableWindowAndMessageTime(t *testing.T) {
	notice := contextTriggerInput("notice", "schedule")
	insert := contextTriggerSplice(t, 1, "next-turn", 0, nil, "", notice)
	claim := contextTriggerSplice(t, 3, "next-turn", 0, contextTriggerCount(1), "")
	message := contextTriggerEvent(t, 4, "user/message", notice)
	s := New()
	contextTriggerSnapshot(t, s, claim, message)
	if contextTriggerMessage(t, s, 4).ContextWaking {
		t.Fatal("reconstructed an unavailable claim identity")
	}
	contextTriggerPrepend(t, s, insert)
	if !contextTriggerMessage(t, s, 4).ContextWaking {
		t.Fatal("older pending identity did not hydrate claim")
	}
	// A later requeue changes future messages only; the entered message keeps
	// the current claim observed at its own sequence.
	s.applyPersistenceEvent(contextTriggerSplice(t, 5, "next-turn", 0, nil, "", notice))
	s.applyPersistenceEvent(contextTriggerEvent(t, 6, "user/message", notice))
	if !contextTriggerMessage(t, s, 4).ContextWaking || contextTriggerMessage(t, s, 6).ContextWaking {
		t.Fatal("classified against final or ever-claimed IDs instead of message-time claim")
	}
	contextTriggerSnapshot(t, s, claim, message)
	if contextTriggerMessage(t, s, 4).ContextWaking {
		t.Fatal("authoritative snapshot retained an omitted insertion")
	}
	// A positive removedCount with no available identities still advances the
	// next-turn claim; that claim suppresses an idle steer in the same Turn.
	stepNotice := contextTriggerInput("steer", "schedule")
	contextTriggerSnapshot(t, s,
		contextTriggerSplice(t, 1, "next-step", 0, nil, "", stepNotice),
		contextTriggerEvent(t, 2, "turn/start", map[string]any{"turn": 1}),
		contextTriggerSplice(t, 3, "next-step", 0, contextTriggerCount(1), ""),
		contextTriggerSplice(t, 4, "next-turn", 0, contextTriggerCount(1), ""),
		contextTriggerEvent(t, 5, "step/start", map[string]any{"turn": 1, "step": 1}),
		contextTriggerEvent(t, 6, "user/message", stepNotice),
	)
	if contextTriggerMessage(t, s, 6).ContextWaking {
		t.Fatal("empty available-window claim did not advance claim sequence")
	}
}

func TestContextTriggerRawRetentionAndSnapshotIsolation(t *testing.T) {
	s := New()
	notice := contextTriggerInput("same-id", "future-producer")
	human := contextTriggerInput("human", "user")
	raw := contextTriggerEvent(t, 3, "user/message", notice)
	contextTriggerSnapshot(t, s,
		contextTriggerSplice(t, 1, "next-turn", 0, nil, "", notice, human),
		contextTriggerSplice(t, 2, "next-turn", 0, contextTriggerCount(2), ""),
		raw,
		contextTriggerEvent(t, 4, "developer/message", map[string]any{"message": notice}),
		contextTriggerEvent(t, 5, "user/message", human),
	)
	before, err := json.Marshal(s.messages)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		message := contextTriggerMessage(t, s, 3)
		if !message.ContextWaking || string(message.Raw) != string(raw) || message.Role != "context" || message.Kind != MessageKindContext {
			t.Fatalf("unknown-source trigger or source evidence changed: %+v", message)
		}
		if contextTriggerMessage(t, s, 4).ContextWaking || contextTriggerMessage(t, s, 5).ContextWaking {
			t.Fatal("developer or human message marked as context wake")
		}
	}
	after, _ := json.Marshal(s.messages)
	if string(before) != string(after) {
		t.Fatal("Snapshot mutated retained Messages")
	}
	s.mu.Lock()
	projected := s.projectContextTriggersLocked(s.messages)
	s.mu.Unlock()
	if !projected[0].ContextWaking || s.messages[0].ContextWaking {
		t.Fatal("projection writes into its input slice")
	}
	projected[0].Body = "snapshot-only change"
	if s.messages[0].Body != "input same-id" {
		t.Fatal("projection shares mutable Message structs")
	}
	inspector := s.InspectorSnapshot()
	if len(inspector.Records) != 5 || string(inspector.Records[2].Raw) != string(raw) {
		t.Fatal("original Inspector evidence changed")
	}
	inspector.Records[2].Raw[0] = '!'
	if string(s.InspectorSnapshot().Records[2].Raw) != string(raw) {
		t.Fatal("Inspector snapshot aliases retained raw")
	}
}

func TestContextTriggerLocationAndSurface(t *testing.T) {
	notice := contextTriggerInput("notice", "schedule")
	before := []json.RawMessage{
		contextTriggerSplice(t, 1, "next-step", 0, nil, "", notice),
		contextTriggerEvent(t, 2, "turn/start", map[string]any{"turn": 1}),
		contextTriggerSplice(t, 3, "next-step", 0, contextTriggerCount(1), ""),
		contextTriggerEvent(t, 4, "step/start", map[string]any{"turn": 1, "step": 1}),
	}
	cases := []struct {
		name   string
		extra  []json.RawMessage
		input  map[string]any
		waking bool
	}{
		{"session event preserves running location", []json.RawMessage{contextTriggerEvent(t, 5, "runtime/context", map[string]any{"turn": nil, "step": 2})}, notice, true},
		{"explicit later step", []json.RawMessage{contextTriggerEvent(t, 5, "runtime/context", map[string]any{"turn": 1, "step": 2})}, notice, false},
		{"explicit different turn", []json.RawMessage{contextTriggerEvent(t, 5, "runtime/context", map[string]any{"turn": 2, "step": 1})}, notice, false},
		{"ended step", []json.RawMessage{contextTriggerEvent(t, 5, "step/end", map[string]any{"turn": 1, "step": 1})}, notice, false},
		{"ended turn", []json.RawMessage{contextTriggerEvent(t, 5, "turn/end", map[string]any{"turn": 1})}, notice, false},
		{"session-scoped input", nil, map[string]any{"id": "notice", "source": map[string]any{"kind": "schedule"}, "turn": nil, "content": []any{}}, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			s := New()
			events := append(append([]json.RawMessage(nil), before...), test.extra...)
			events = append(events, contextTriggerEvent(t, 6, "user/message", test.input))
			contextTriggerSnapshot(t, s, events...)
			if got := contextTriggerMessage(t, s, 6).ContextWaking; got != test.waking {
				t.Fatalf("waking = %t, want %t", got, test.waking)
			}
		})
	}
	for _, marker := range []string{`null`, `{"op":"append"}`, `{"op":"replace","startSeq":0,"endSeq":0}`} {
		t.Run("non-append surface "+marker, func(t *testing.T) {
			s := New()
			var event map[string]any
			if err := json.Unmarshal(contextTriggerEvent(t, 6, "user/message", notice), &event); err != nil {
				t.Fatal(err)
			}
			event["surfaceOp"] = json.RawMessage(marker)
			raw, _ := json.Marshal(event)
			contextTriggerSnapshot(t, s, append(append([]json.RawMessage(nil), before...), raw)...)
			if contextTriggerMessage(t, s, 6).ContextWaking {
				t.Fatal("non-append message entered trigger projection")
			}
		})
	}
}

func TestContextTriggerZeroSequenceAndChangedRetainedClaim(t *testing.T) {
	s := New()
	s.selectedID = "zero"
	notice := contextTriggerInput("wake", "schedule")
	insert := contextTriggerSplice(t, 0, "next-turn", 0, nil, "", notice)
	claim := contextTriggerSplice(t, 2, "next-turn", 0, contextTriggerCount(1), "")
	message := contextTriggerEvent(t, 4, "user/message", notice)
	contextTriggerSnapshot(t, s, insert,
		contextTriggerEvent(t, 1, "turn/start", map[string]any{"turn": 1}), claim,
		contextTriggerEvent(t, 3, "step/start", map[string]any{"turn": 1, "step": 1}), message)
	for i := 0; i < 3; i++ {
		if !contextTriggerMessage(t, s, 4).ContextWaking {
			t.Fatal("sequence-zero insertion did not supply the claimed identity")
		}
	}
	// A corrected retained claim at the same sequence must replace the cached
	// drawing fact even when event count and the user message are unchanged.
	s.applyPersistenceEvent(contextTriggerSplice(t, 2, "next-turn", 0, contextTriggerCount(1), "canceled"))
	if contextTriggerMessage(t, s, 4).ContextWaking {
		t.Fatal("changed retained claim kept the prior waking fact")
	}
	s.applyPersistenceEvent(claim)
	if !contextTriggerMessage(t, s, 4).ContextWaking {
		t.Fatal("restored claim did not republish its drawing fact")
	}
}

func TestContextTriggerSortedFactsAndBindingReset(t *testing.T) {
	notice := contextTriggerInput("notice", "schedule")
	insert := contextTriggerSplice(t, 1, "next-turn", 0, nil, "", notice)
	claim := contextTriggerSplice(t, 2, "next-turn", 0, contextTriggerCount(1), "")
	message := contextTriggerEvent(t, 3, "user/message", notice)
	for _, reset := range []string{"session", "host", "generation"} {
		t.Run(reset, func(t *testing.T) {
			s := New()
			s.selectedID = "one"
			contextTriggerSnapshot(t, s, message)
			// Capture in reverse order to pin sequence sorting independent of
			// both ingestion order and map iteration order.
			s.mu.Lock()
			s.captureInspectorEventLocked(claim)
			s.captureInspectorEventLocked(insert)
			s.mu.Unlock()
			if !contextTriggerMessage(t, s, 3).ContextWaking {
				t.Fatal("unsorted retained facts did not replay by seq")
			}
			s.mu.Lock()
			switch reset {
			case "session":
				s.selectSessionLocked("two")
			case "host":
				s.mu.Unlock()
				s.Attach(&dsh.Client{}, nil)
				s.mu.Lock()
			case "generation":
				s.invalidateFollowLocked()
			}
			// Exercise the ownership check even before the new binding sends
			// its first event; Attach can retain old source Messages briefly.
			projected := s.projectContextTriggersLocked([]Message{{Seq: 3, Kind: MessageKindContext, EventType: "user/message", ContextWaking: true, Raw: message}})
			s.mu.Unlock()
			if projected[0].ContextWaking {
				t.Fatal("old binding supplied trigger facts")
			}
			s.applyPersistenceEvent(message)
			if contextTriggerMessage(t, s, 3).ContextWaking {
				t.Fatal("new binding reused old Inbox claim")
			}
			contextTriggerSnapshot(t, s, insert, claim, message)
			if !contextTriggerMessage(t, s, 3).ContextWaking {
				t.Fatal("new binding did not recover with its own facts")
			}
			if !reflect.DeepEqual(s.inspector.events[3], message) {
				t.Fatal("trigger projection rewrote retained event")
			}
		})
	}
}
