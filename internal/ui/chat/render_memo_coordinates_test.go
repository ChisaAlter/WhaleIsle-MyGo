package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

func assertMemoCoordinates(t *testing.T, msg session.Message, wantTurn, wantStep float64, wantKnown bool) {
	t.Helper()
	turn, step, known := memoizedTranscriptCoordinates(msg)
	if turn != wantTurn || step != wantStep || known != wantKnown {
		t.Fatalf("coordinate memo=(%g,%g,%v), want (%g,%g,%v) for %s", turn, step, known, wantTurn, wantStep, wantKnown, msg.Raw)
	}
}

func TestCoordinateMemoSameLengthRawReplacement(t *testing.T) {
	msg := session.Message{ID: "coordinate-same-length", Seq: 27, Raw: json.RawMessage(`{"data":{"turn":1,"step":2}}`)}
	assertMemoCoordinates(t, msg, 1, 2, true)
	msg.Raw = json.RawMessage(`{"data":{"turn":3,"step":4}}`)
	assertMemoCoordinates(t, msg, 3, 4, true)
	// The cache must own the bytes it compares, even when an input buffer is
	// changed in place and keeps its identity, length and capacity.
	copy(msg.Raw, `{"data":{"turn":5,"step":6}}`)
	assertMemoCoordinates(t, msg, 5, 6, true)
}

func TestCoordinateMemoFollowsCurrentToolCall(t *testing.T) {
	call := &session.Message{ID: "coordinate-call", Seq: 28, Raw: json.RawMessage(`{"data":{"turn":1,"step":2}}`)}
	msg := session.Message{ID: "coordinate-result", Seq: 29, Raw: json.RawMessage(`{"data":{}}`), Tool: &session.ToolCallView{Call: call}}
	assertMemoCoordinates(t, msg, 1, 2, true)
	call.Raw = json.RawMessage(`{"data":{"turn":3,"step":4}}`)
	assertMemoCoordinates(t, msg, 3, 4, true)
	// Missing/invalid coordinates fall back through the current chain, not the
	// one whose result was last cached for this outer message.
	msg.Tool.Call = &session.Message{ID: "coordinate-nested", Tool: &session.ToolCallView{Call: &session.Message{Raw: json.RawMessage(`{"data":{"turn":0,"step":0}}`)}}}
	assertMemoCoordinates(t, msg, 0, 0, true)
	msg.Tool = nil
	assertMemoCoordinates(t, msg, 0, 0, false)
	msg.Tool = &session.ToolCallView{Call: call}
	msg.Raw = json.RawMessage(`{"data":{"turn":0,"step":7}}`)
	assertMemoCoordinates(t, msg, 0, 7, true)
}

func TestCoordinateMemoIsolatesSequenceBasedSessionRows(t *testing.T) {
	// Legacy assistant-event IDs are sequence-derived, and projected tool IDs
	// use only callId. Their identity does not include the selected session.
	for _, id := range []string{"assistant-event/2", "tool:call", ""} {
		first := session.Message{ID: id, Seq: 2, Raw: json.RawMessage(`{"data":{"turn":1,"step":0}}`)}
		second := first
		second.Raw = json.RawMessage(`{"data":{"turn":8,"step":9}}`)
		assertMemoCoordinates(t, first, 1, 0, true)
		assertMemoCoordinates(t, second, 8, 9, true)
		assertMemoCoordinates(t, first, 1, 0, true)
	}
}

func TestCoordinateMemoUnknownBecomesKnownAtSameLength(t *testing.T) {
	msg := session.Message{ID: "coordinate-unknown", Seq: 30, Raw: json.RawMessage(`{"data":{"turn":null,"step":2}}`)}
	assertMemoCoordinates(t, msg, 0, 0, false)
	msg.Raw = json.RawMessage(`{"data":{"turn":1000,"step":2}}`)
	assertMemoCoordinates(t, msg, 1000, 2, true)
}

func TestCoordinateMemoRetainsBoundedOwnedBytes(t *testing.T) {
	memo := coordinateMemoStore{m: make(map[coordinateKey]coordinateMemoEntry)}
	raw := json.RawMessage(`{"data":{"turn":1,"step":2}}`)
	msg := session.Message{ID: "bounded", Seq: 1, Raw: raw}
	memo.rawCoordinates(msg)
	copy(raw, `{"data":{"turn":3,"step":4}}`)
	if got := memo.rawCoordinates(msg); got.turn != 3 || got.step != 4 || !got.known {
		t.Fatal("cached bytes aliased the mutable input")
	}
	if memo.rawBytes != len(raw) || len(memo.m) != 1 {
		t.Fatal("replacing a candidate did not subtract its old owned bytes")
	}
	atLimit := session.Message{ID: "at-limit", Raw: json.RawMessage(`{"data":{"turn":0}}` + strings.Repeat(" ", coordinateMemoByteLimit-len(`{"data":{"turn":0}}`)))}
	memo.rawCoordinates(atLimit)
	if memo.rawBytes != coordinateMemoByteLimit || len(memo.m) != 1 {
		t.Fatal("exact byte-limit entry was not retained within the bounded cache")
	}
	oversized := atLimit
	oversized.ID = "oversized"
	oversized.Raw = append(append(json.RawMessage(nil), atLimit.Raw...), ' ')
	if got := memo.rawCoordinates(oversized); !got.known || got.turn != 0 {
		t.Fatal("oversized uncached record lost its coordinates")
	}
	if memo.rawBytes != coordinateMemoByteLimit || len(memo.m) != 1 {
		t.Fatal("oversized record evicted useful entries or exceeded the byte budget")
	}
	memo.rawCoordinates(session.Message{ID: "small", Raw: json.RawMessage(`{}`)})
	if memo.rawBytes != 2 || len(memo.m) != 1 {
		t.Fatal("new cache entry failed to reset an exhausted byte budget")
	}
	memo = coordinateMemoStore{m: make(map[coordinateKey]coordinateMemoEntry)}
	for i := 0; i < memoLimit; i++ {
		memo.rawCoordinates(session.Message{ID: fmt.Sprint(i)})
	}
	if len(memo.m) != memoLimit {
		t.Fatal("memo reset before reaching its count limit")
	}
	memo.rawCoordinates(session.Message{ID: "next"})
	if len(memo.m) != 1 || memo.rawBytes != 0 {
		t.Fatal("memo did not reset at its count limit")
	}
}

var coordinateBenchmarkSink coordinateValue

func BenchmarkCoordinateMemoIdentity(b *testing.B) {
	for _, padding := range []int{0, 65536} {
		b.Run(fmt.Sprintf("hit_padding_%d", padding), func(b *testing.B) {
			msg := session.Message{ID: fmt.Sprintf("coordinate-bench-%d", padding), Seq: 31, Raw: json.RawMessage(`{"data":{"turn":1,"step":2}}` + strings.Repeat(" ", padding))}
			memoizedTranscriptCoordinates(msg)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				coordinateBenchmarkSink.turn, coordinateBenchmarkSink.step, coordinateBenchmarkSink.known = memoizedTranscriptCoordinates(msg)
			}
		})
	}
}
