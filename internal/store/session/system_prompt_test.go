package session

import (
	"encoding/json"
	"strings"
	"testing"
)

// One focused B10 fixture: the original request/header + system/message
// contract across append, replacement, explicit series, partial windows and
// replay. Headers and diagnostic fields must never become prompt text.
func TestSystemPromptDisclosureContract(t *testing.T) {
	events := []string{
		`{"seq":1,"type":"turn/start","data":{"turn":1}}`,
		`{"seq":2,"type":"step/start","data":{"turn":1,"step":1}}`,
		`{"seq":3,"type":"system/message","surfaceOp":"append","data":{"turn":1,"step":1,"diagnostic":"PRIVATE_DIAGNOSTIC","message":{"content":[{"type":"text","text":"A\n"},{"type":"image","text":"NOT_TEXT"},{"type":"text","text":"B"}]}}}`,
		`{"seq":4,"type":"request/header","data":{"reason":"initial","header":{"config":{"authorization":"PRIVATE_AUTH"},"tools":[{"description":"PRIVATE_TOOL"}]},"text":"PRIVATE_FALLBACK"}}`,
		`{"seq":5,"type":"request/header","data":{"reason":"change","header":{"tools":[{"name":"changed"}]}}}`,
		`{"seq":6,"type":"step/end","data":{"turn":1,"step":1}}`,
		`{"seq":7,"type":"step/start","data":{"turn":1,"step":2}}`,
		`{"seq":8,"type":"system/message","surfaceOp":"append","data":{"turn":1,"step":2,"message":{"content":[{"type":"text","text":"Updated"}]}}}`,
		`{"seq":9,"type":"request/header","data":{"reason":"series","startsSeries":true}}`,
		`{"seq":10,"type":"step/end","data":{"turn":1,"step":2}}`,
		`{"seq":11,"type":"step/start","data":{"turn":1,"step":3}}`,
		`{"seq":12,"type":"user/message","surfaceOp":{"op":"replace","startSeq":7,"endSeq":9},"data":{"content":[{"type":"text","text":"summary"}]}}`,
		`{"seq":13,"type":"request/header","data":{"reason":"series"}}`,
		`{"seq":14,"type":"system/message","surfaceOp":{"op":"replace","startSeq":3,"endSeq":3},"data":{"turn":1,"step":3,"message":{"content":[{"type":"text","text":"Replacement"}]}}}`,
		`{"seq":15,"type":"request/header","data":{"reason":"change"}}`,
		`{"seq":16,"type":"system/message","surfaceOp":{"op":"replace","startSeq":14,"endSeq":14},"data":{"turn":1,"step":3,"message":{"content":[{"type":"text","text":"Chained"}]}}}`,
		`{"seq":17,"type":"request/header","data":{"reason":"resume"}}`,
		`{"seq":18,"type":"system/message","surfaceOp":{"op":"replace","startSeq":16,"endSeq":16},"data":{"turn":1,"step":3,"message":{"content":[]}}}`,
		`{"seq":19,"type":"request/header","data":{"reason":"series"}}`,
	}
	s := New()
	s.Select("fixture-session")
	// Live events start after the authoritative session/follow baseline.
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[]}`))
	for _, raw := range events {
		s.applyPersistenceEvent(json.RawMessage(raw))
		// Live snapshots must be side-effect free and must not expose raw headers.
		for _, m := range s.projectSystemPromptsLocked(nil) {
			if strings.Contains(m.Body, "PRIVATE_") || len(m.Raw) != 0 {
				t.Fatalf("leaked private request fields: %+v", m)
			}
		}
	}
	check := func(s *Store) {
		t.Helper()
		cards := s.projectSystemPromptsLocked(nil)
		want := []struct {
			text   string
			seq    float64
			update bool
		}{{"A\nB", 1, false}, {"Updated", 8, true}, {"A\nB", 11, false}, {"Replacement", 15, false}, {"Chained", 17, false}}
		if len(cards) != len(want) {
			t.Fatalf("cards=%+v", cards)
		}
		for i, w := range want {
			if cards[i].Body != w.text || cards[i].Seq != w.seq || (cards[i].EventType == "system-prompt-update") != w.update {
				t.Fatalf("card %d=%+v want=%+v", i, cards[i], w)
			}
		}
		for _, raw := range s.trajectoryBoundaries {
			if strings.Contains(string(raw), "PRIVATE_") || strings.Contains(string(raw), "NOT_TEXT") {
				t.Fatalf("unsanitized boundary: %s", raw)
			}
		}
	}
	check(s)
	// A replacement referring outside a loaded window is unavailable, even if
	// another append follows. Prepending the original events resolves ordering.
	window := New()
	for _, raw := range events[13:] {
		window.captureTrajectoryBoundaryLocked(json.RawMessage(raw))
	}
	if got := window.projectSystemPromptsLocked(nil); len(got) != 0 {
		t.Fatalf("unknown endpoints produced prompt: %+v", got)
	}
	for _, raw := range events[:13] {
		window.captureTrajectoryBoundaryLocked(json.RawMessage(raw))
	}
	check(window)
	for _, raw := range events {
		var e systemPromptEvent
		_ = json.Unmarshal([]byte(raw), &e)
		if e.Type == "system/message" || e.Type == "request/header" {
			if m, ok := classifyPersistenceEvent([]byte(raw)); ok {
				t.Fatalf("prompt leaked as generic transcript: %+v", m)
			}
		}
	}
	// Equal loaded history produces identical card identities and ordering.
	a, _ := json.Marshal(s.projectSystemPromptsLocked(nil))
	b, _ := json.Marshal(window.projectSystemPromptsLocked(nil))
	if string(a) != string(b) {
		t.Fatalf("live/prepend mismatch: %s vs %s", a, b)
	}
	// A visible request card keeps its first presentation anchor after prepend.
	stable := New()
	for _, raw := range []string{
		`{"seq":3,"type":"system/message","surfaceOp":"append","data":{"turn":1,"step":1,"message":{"content":[{"type":"text","text":"first"}]}}}`,
		`{"seq":4,"type":"system/message","surfaceOp":{"op":"replace","startSeq":3,"endSeq":3},"data":{"turn":1,"step":1,"message":{"content":[{"type":"text","text":"second"}]}}}`,
		`{"seq":5,"type":"request/header","data":{"reason":"initial"}}`,
	} {
		stable.captureTrajectoryBoundaryLocked(json.RawMessage(raw))
	}
	before := stable.projectSystemPromptsLocked(nil)
	if len(before) != 2 || before[1].Seq != 5 {
		t.Fatalf("partial anchors=%+v", before)
	}
	for _, raw := range events[:2] {
		stable.captureTrajectoryBoundaryLocked(json.RawMessage(raw))
	}
	after := stable.projectSystemPromptsLocked(nil)
	if len(after) != 2 || after[0].Seq != 1 || after[1].Seq != 5 {
		t.Fatalf("prepend moved request anchor: %+v", after)
	}
	stable.invalidateFollowLocked()
	if stable.systemPromptAnchors != nil {
		t.Fatal("request anchors survived binding reset")
	}

}
