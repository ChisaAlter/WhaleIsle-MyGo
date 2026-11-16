package chat

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

func buildChunkedTurnMessages(turns int) []session.Message {
	msgs := make([]session.Message, 0, turns*5+2)
	seq := 1
	for t := 0; t < turns; t++ {
		turnID := t + 1
		turnSeq := seq
		msgs = append(msgs,
			session.Message{
				ID: fmt.Sprintf("u-%d", t), Seq: float64(seq), TurnSeq: float64(turnSeq),
				Kind: session.MessageKindText, Role: "user", Body: fmt.Sprintf("prompt %d", t),
				Raw: json.RawMessage(fmt.Sprintf(`{"type":"user","seq":%d,"data":{"turn":%d,"step":1}}`, seq, turnID)),
			},
		)
		seq++
		msgs = append(msgs,
			session.Message{
				ID: fmt.Sprintf("th-%d", t), Seq: float64(seq), TurnSeq: float64(turnSeq),
				Kind: session.MessageKindThinking, Role: "assistant", Body: fmt.Sprintf("reasoning %d", t),
				Raw: json.RawMessage(fmt.Sprintf(`{"type":"thinking","seq":%d,"data":{"turn":%d,"step":1}}`, seq, turnID)),
			},
		)
		seq++
		msgs = append(msgs,
			session.Message{
				ID: fmt.Sprintf("tool-%d", t), Seq: float64(seq), TurnSeq: float64(turnSeq),
				Kind: session.MessageKindTool, Role: "tool", EventType: "tool/call", Body: "read",
				Raw:  json.RawMessage(fmt.Sprintf(`{"type":"tool/call","seq":%d,"data":{"turn":%d,"step":2,"callId":"c-%d","name":"read","arguments":"{}"}}`, seq, turnID, t)),
				Tool: &session.ToolCallView{CallID: fmt.Sprintf("c-%d", t), Name: "read", State: "done"},
			},
		)
		seq++
		msgs = append(msgs,
			session.Message{
				ID: fmt.Sprintf("asst-%d", t), Seq: float64(seq), TurnSeq: float64(turnSeq),
				Kind: session.MessageKindText, Role: "assistant", SourceMessageID: fmt.Sprintf("src-%d", t),
				Body: fmt.Sprintf("answer %d", t),
				Raw:  json.RawMessage(fmt.Sprintf(`{"type":"assistant","seq":%d,"data":{"turn":%d,"step":3}}`, seq, turnID)),
			},
		)
		seq++
		msgs = append(msgs,
			session.Message{
				ID: fmt.Sprintf("end-%d", t), Seq: float64(seq), TurnSeq: float64(turnSeq),
				Kind: session.MessageKindUnknown, Role: "status", EventType: "turn/end", Body: "1200",
				Raw: json.RawMessage(fmt.Sprintf(`{"type":"turn/end","seq":%d,"data":{"turn":%d,"step":3,"status":"done","elapsedMs":1200}}`, seq, turnID)),
			},
		)
		seq++
	}
	tailTurn := turns + 1
	tailSeq := seq
	msgs = append(msgs,
		session.Message{
			ID: "u-tail", Seq: float64(seq), TurnSeq: float64(tailSeq),
			Kind: session.MessageKindText, Role: "user", Body: "tail prompt",
			Raw: json.RawMessage(fmt.Sprintf(`{"type":"user","seq":%d,"data":{"turn":%d,"step":1}}`, seq, tailTurn)),
		},
		session.Message{
			ID: "asst-tail", Seq: float64(seq + 1), TurnSeq: float64(tailSeq),
			Kind: session.MessageKindText, Role: "assistant", Streaming: true, Body: "working",
			Raw: json.RawMessage(fmt.Sprintf(`{"type":"assistant","seq":%d,"data":{"turn":%d,"step":1}}`, seq+1, tailTurn)),
		},
	)
	return msgs
}

func TestTurnPrefixChunkCacheHitsUntouchedTurnsOnSingleFoldToggle(t *testing.T) {
	messages := buildChunkedTurnMessages(12)
	store := session.New()
	expanded := map[string]bool{}
	var memo transcriptViewMemo

	assertFullParity := func(step string, snap session.ViewState) ([]session.Message, map[string]transcriptControl, map[int]assistantActionTarget, []transcriptMessageKey) {
		t.Helper()
		m := Model{Snapshot: snap, ThinkingExpanded: &expanded}
		msgs, controls, targets, keys := memo.resolve(m)
		wantMsgs, wantControls := projectTranscript(visibleProjectedMessages(snap), m)
		wantTargets := assistantActionTargets(wantMsgs)
		wantKeys := transcriptMessageKeys(m, wantMsgs)
		if !reflect.DeepEqual(msgs, wantMsgs) {
			t.Fatalf("%s: chunked msgs mismatch\ngot=%d want=%d", step, len(msgs), len(wantMsgs))
		}
		if !reflect.DeepEqual(controls, wantControls) {
			t.Fatalf("%s: chunked controls mismatch\ngot=%+v\nwant=%+v", step, controls, wantControls)
		}
		if !reflect.DeepEqual(targets, wantTargets) {
			t.Fatalf("%s: chunked action targets mismatch\ngot=%+v\nwant=%+v", step, targets, wantTargets)
		}
		if !reflect.DeepEqual(keys, wantKeys) {
			t.Fatalf("%s: chunked message keys mismatch", step)
		}
		return msgs, controls, targets, keys
	}

	snap := store.Snapshot()
	snap.SelectedID = "chunk-sess"
	snap.PlanHistoryBinding = "chunk-bind"
	snap.Messages = append([]session.Message(nil), messages...)

	_, controls, _, _ := assertFullParity("initial", snap)
	chunkCount := len(memo.turnPrefix.chunks)
	if chunkCount < 10 {
		t.Fatalf("expected multi-turn prefix chunks, got %d", chunkCount)
	}
	if memo.turnPrefix.chunkRebuilds != chunkCount {
		t.Fatalf("initial chunkRebuilds = %d, want %d", memo.turnPrefix.chunkRebuilds, chunkCount)
	}

	// Pick one middle turn fold key (turn 4 -> turnID 4).
	turnKey := "transcript-turn:chunk-bind:chunk-sess:4"
	if _, ok := controls[turnKey]; !ok {
		t.Fatalf("expected turn control key %q in %+v", turnKey, controls)
	}

	rebuildsBefore := memo.turnPrefix.chunkRebuilds
	hitsBefore := memo.turnPrefix.chunkHits
	expanded[turnKey] = true
	_, afterExpandControls, _, _ := assertFullParity("expand-turn-4", snap)

	if got := memo.turnPrefix.chunkRebuilds - rebuildsBefore; got != 1 {
		t.Fatalf("expanding single turn rebuilt %d chunks, want 1", got)
	}
	if got := memo.turnPrefix.chunkHits - hitsBefore; got != chunkCount-1 {
		t.Fatalf("expanding single turn hit %d chunks, want %d", got, chunkCount-1)
	}

	// Pick a tool-group control key exposed inside turn 4 and toggle it too.
	var groupKey string
	for k := range afterExpandControls {
		if strings.HasPrefix(k, "transcript-group:") {
			groupKey = k
			break
		}
	}
	if groupKey == "" {
		t.Fatalf("expected exposed group control after expanding turn 4")
	}

	rebuildsBefore = memo.turnPrefix.chunkRebuilds
	hitsBefore = memo.turnPrefix.chunkHits
	expanded[groupKey] = true
	assertFullParity("toggle-group-in-turn-4", snap)
	if got := memo.turnPrefix.chunkRebuilds - rebuildsBefore; got != 1 {
		t.Fatalf("toggling group rebuilt %d chunks, want 1", got)
	}
	if got := memo.turnPrefix.chunkHits - hitsBefore; got != chunkCount-1 {
		t.Fatalf("toggling group hit %d chunks, want %d", got, chunkCount-1)
	}

	// Append a tool call to the tail turn (bypassing streaming-text tail fast path)
	// and verify turnPrefix hits all 12 prefix chunks without rebuilding any.
	messages = append(messages[:len(messages)-1],
		session.Message{
			ID: "tool-tail", Seq: float64(len(messages)), TurnSeq: float64(len(messages) - 1),
			Kind: session.MessageKindTool, Role: "tool", EventType: "tool/call", Body: "search",
			Raw:  json.RawMessage(fmt.Sprintf(`{"type":"tool/call","seq":%d,"data":{"turn":13,"step":1,"callId":"c-tail","name":"search","arguments":"{}"}}`, len(messages))),
			Tool: &session.ToolCallView{CallID: "c-tail", Name: "search", State: "running"},
		},
		messages[len(messages)-1],
	)
	store.SetLastAction("tail-tool")
	nextSnap := store.Snapshot()
	nextSnap.SelectedID = "chunk-sess"
	nextSnap.PlanHistoryBinding = "chunk-bind"
	nextSnap.Messages = append([]session.Message(nil), messages...)

	rebuildsBefore = memo.turnPrefix.chunkRebuilds
	hitsBefore = memo.turnPrefix.chunkHits
	assertFullParity("tail-tool-append-with-folded-prefix", nextSnap)
	if got := memo.turnPrefix.chunkRebuilds - rebuildsBefore; got != 0 {
		t.Fatalf("tail tool update rebuilt %d prefix chunks, want 0", got)
	}
	if got := memo.turnPrefix.chunkHits - hitsBefore; got != chunkCount {
		t.Fatalf("tail tool update hit %d prefix chunks, want %d", got, chunkCount)
	}
}
