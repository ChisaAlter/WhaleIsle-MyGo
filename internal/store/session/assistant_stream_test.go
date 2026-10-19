package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

type assistantFixture struct {
	s        *Store
	revision int
}

func newAssistantFixture() *assistantFixture { return &assistantFixture{s: New()} }
func (f *assistantFixture) frame(t *testing.T, value map[string]any) {
	t.Helper()
	f.revision++
	value["revision"] = f.revision
	raw, err := json.Marshal(map[string]any{"type": "assistant-stream", "frame": value})
	if err != nil {
		t.Fatal(err)
	}
	f.s.applyFollowFrame(raw)
}
func (f *assistantFixture) start(t *testing.T, id string, turn, step int) {
	f.frame(t, map[string]any{"type": "start", "attemptId": id, "turn": turn, "step": step, "startedAfterSeq": -1})
}
func (f *assistantFixture) chunk(t *testing.T, id string, index int, chunk any) {
	f.frame(t, map[string]any{"type": "chunk", "attemptId": id, "index": index, "time": 100 + index, "chunk": chunk})
}
func (f *assistantFixture) end(t *testing.T, id string, index int, outcome any) {
	f.frame(t, map[string]any{"type": "end", "attemptId": id, "index": index, "outcome": outcome})
}
func (f *assistantFixture) durable(t *testing.T, raw string) {
	t.Helper()
	f.s.applyFollowFrame(json.RawMessage(`{"type":"event","event":` + raw + `}`))
}
func assistantDelta(typ string, index int, text string) map[string]any {
	return map[string]any{"type": typ, "index": index, "text": text}
}
func assistantCommit(seq int, typ string) map[string]any {
	return map[string]any{"kind": "committed", "seq": seq, "eventType": typ}
}
func assistantMessage(seq, turn, step int, content string, interrupted bool) string {
	return fmt.Sprintf(`{"type":"assistant/message","seq":%d,"time":200,"surfaceOp":"append","data":{"turn":%d,"step":%d,"interrupted":%t,"message":{"id":"m%d","content":%s},"stream":[]}}`, seq, turn, step, interrupted, seq, content)
}

func TestAssistantSettlementKeepsOneCanonicalAnswerAndReasoning(t *testing.T) {
	f := newAssistantFixture()
	f.start(t, "a", 1, 0)
	f.chunk(t, "a", 0, assistantDelta("reasoning-delta", 0, "private reasoning"))
	f.chunk(t, "a", 1, assistantDelta("text-delta", 1, "draft answer"))
	messages := f.s.Snapshot().Messages
	if len(messages) != 2 || messages[0].Kind != MessageKindThinking || messages[1].Body != "draft answer" {
		t.Fatalf("live separation: %+v", messages)
	}
	durable := assistantMessage(4, 1, 0, `[{"type":"reasoning","text":"final reasoning"},{"type":"text","text":"canonical answer"}]`, false)
	f.durable(t, durable)
	if got := f.s.Snapshot().Messages; len(got) != 2 || got[1].Body != "draft answer" {
		t.Fatalf("settlement published before end: %+v", got)
	}
	f.end(t, "a", 2, assistantCommit(4, "assistant/message"))
	got := f.s.Snapshot().Messages
	if len(got) != 2 || got[0].Kind != MessageKindThinking || got[0].Body != "final reasoning" || got[1].Body != "canonical answer" || got[1].ID != "m4" || got[1].Streaming {
		t.Fatalf("settled projection: %+v", got)
	}
	if f.s.assistantStream.retained == nil {
		t.Fatal("successful prefix must survive internally until step/end")
	}
	f.durable(t, durable)
	if len(f.s.Snapshot().Messages) != 2 {
		t.Fatal("duplicate durable seq was appended")
	}
	f.durable(t, `{"type":"step/end","seq":5,"data":{"turn":1,"step":0}}`)
	if f.s.assistantStream.retained != nil || len(f.s.messages) != 1 {
		t.Fatalf("prefix not retired: %+v", f.s.messages)
	}
	f.start(t, "b", 1, 1)
	f.chunk(t, "b", 0, assistantDelta("text-delta", 0, "canonical answer"))
	f.durable(t, assistantMessage(6, 1, 1, `[{"type":"text","text":"canonical answer"}]`, false))
	f.end(t, "b", 1, assistantCommit(6, "assistant/message"))
	if got := f.s.Snapshot().Messages; len(got) != 3 || got[2].ID != "m6" {
		t.Fatalf("text equality swallowed a message: %+v", got)
	}
}

func TestAssistantAbandonmentAndFailedSettlementRetirePrefix(t *testing.T) {
	t.Run("abandoned", func(t *testing.T) {
		f := newAssistantFixture()
		f.start(t, "a", 0, 0)
		f.chunk(t, "a", 0, assistantDelta("text-delta", 0, "discard"))
		f.end(t, "a", 1, map[string]any{"kind": "abandoned"})
		if len(f.s.Snapshot().Messages) != 0 || f.s.assistantStream.active != nil {
			t.Fatal("abandoned text remained")
		}
	})
	t.Run("interrupted", func(t *testing.T) {
		f := newAssistantFixture()
		f.start(t, "a", 0, 0)
		f.chunk(t, "a", 0, assistantDelta("text-delta", 0, "prefix"))
		f.durable(t, assistantMessage(2, 0, 0, `[{"type":"text","text":"partial"}]`, true))
		f.end(t, "a", 1, assistantCommit(2, "assistant/message"))
		if got := f.s.Snapshot().Messages; len(got) != 1 || got[0].Body != "partial" || f.s.assistantStream.retained != nil {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("failed-attempt", func(t *testing.T) {
		f := newAssistantFixture()
		f.start(t, "a", 0, 0)
		f.chunk(t, "a", 0, assistantDelta("text-delta", 0, "prefix"))
		f.durable(t, `{"type":"assistant/attempt","seq":2,"data":{"turn":0,"step":0,"stream":[{"type":"reasoning-chunks","index":0,"time0":1,"dt":[],"texts":["reason"]},{"type":"text-chunks","index":1,"time0":2,"dt":[],"texts":["partial"]}]}}`)
		f.end(t, "a", 1, assistantCommit(2, "assistant/attempt"))
		if got := f.s.Snapshot().Messages; len(got) != 0 || len(f.s.messages) != 1 || f.s.messages[0].EventType != "assistant/attempt" || f.s.assistantStream.retained != nil {
			t.Fatalf("%+v", got)
		}
	})
}

func TestAssistantProtocolFaultsRequestRebaselineWithoutAppending(t *testing.T) {
	tests := []struct {
		name  string
		fault func(*testing.T, *assistantFixture)
	}{
		{"index-gap", func(t *testing.T, f *assistantFixture) { f.chunk(t, "a", 2, assistantDelta("text-delta", 0, "bad")) }},
		{"duplicate-index", func(t *testing.T, f *assistantFixture) { f.chunk(t, "a", 0, assistantDelta("text-delta", 0, "bad")) }},
		{"end-index", func(t *testing.T, f *assistantFixture) { f.end(t, "a", 3, map[string]any{"kind": "abandoned"}) }},
		{"missing-settlement", func(t *testing.T, f *assistantFixture) { f.end(t, "a", 1, assistantCommit(7, "assistant/message")) }},
		{"new-start-before-end", func(t *testing.T, f *assistantFixture) { f.start(t, "b", 0, 1) }},
		{"duplicate-revision", func(t *testing.T, f *assistantFixture) {
			f.revision--
			f.chunk(t, "a", 1, assistantDelta("text-delta", 0, "bad"))
		}},
		{"wrong-settlement-kind", func(t *testing.T, f *assistantFixture) {
			f.durable(t, assistantMessage(7, 0, 0, `[{"type":"text","text":"ok"}]`, false))
			f.end(t, "a", 1, assistantCommit(7, "assistant/attempt"))
		}},
		{"duplicate-pending", func(t *testing.T, f *assistantFixture) {
			raw := assistantMessage(7, 0, 0, `[{"type":"text","text":"ok"}]`, false)
			f.durable(t, raw)
			f.durable(t, raw)
		}},
		{"abandoned-with-pending", func(t *testing.T, f *assistantFixture) {
			f.durable(t, assistantMessage(7, 0, 0, `[{"type":"text","text":"ok"}]`, false))
			f.end(t, "a", 1, map[string]any{"kind": "abandoned"})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAssistantFixture()
			f.start(t, "a", 0, 0)
			f.chunk(t, "a", 0, assistantDelta("text-delta", 0, "prefix"))
			tt.fault(t, f)
			if !f.s.assistantStream.rebaseline {
				t.Fatal("protocol fault was not repaired")
			}
			if got := f.s.Snapshot().Messages; len(got) != 1 || got[0].Body != "prefix" {
				t.Fatalf("corrupted prefix: %+v", got)
			}
			f.chunk(t, "a", 1, assistantDelta("text-delta", 0, "ignored while repairing"))
			if f.s.Snapshot().Messages[0].Body != "prefix" {
				t.Fatal("accepted suffix after fault")
			}
		})
	}
}

func TestAssistantSnapshotRebuildsCompactPrefixAndResumesDenseIndex(t *testing.T) {
	f := newAssistantFixture()
	f.start(t, "old", 0, 0)
	f.chunk(t, "old", 0, assistantDelta("text-delta", 0, "old"))
	f.s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","cursor":20,"hasMore":true,"records":[{"type":"event","event":{"type":"user/message","seq":15,"data":{"content":[{"type":"text","text":"prompt"}]}}}],"assistantStream":{"revision":40,"activeAttempt":{"attemptId":"new","turn":2,"step":0,"startedAfterSeq":20,"nextIndex":5,"stream":[{"type":"reasoning-chunks","time0":100,"index":0,"dt":[2],"texts":["think","ing"]},{"type":"tool-call-chunks","time0":103,"index":1,"id":"tool","name":"read","dt":[],"args":["{}"]},{"type":"text-chunks","time0":104,"index":2,"dt":[1],"texts":["Hel","lo"]}]}}}`))
	f.revision = 40
	got := f.s.Snapshot()
	if len(got.Messages) != 3 || got.Messages[1].Body != "thinking" || got.Messages[1].Kind != MessageKindThinking || got.Messages[2].Body != "Hello" || !got.HistoryHasMore || f.s.historyThroughSeq != 20 {
		t.Fatalf("bad opening: %+v", got)
	}
	f.chunk(t, "new", 5, assistantDelta("text-delta", 2, "!"))
	if f.s.Snapshot().Messages[2].Body != "Hello!" || f.s.assistantStream.rebaseline {
		t.Fatal("did not resume at dense nextIndex")
	}
	f.chunk(t, "old", 99, assistantDelta("text-delta", 2, "stale"))
	if f.s.Snapshot().Messages[2].Body != "Hello!" || f.s.assistantStream.rebaseline {
		t.Fatal("late unrelated attempt mutated active prefix")
	}
	f.s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[],"assistantStream":{"revision":50}}`))
	if len(f.s.Snapshot().Messages) != 0 || f.s.assistantStream.active != nil {
		t.Fatal("snapshot did not retire previous attempt")
	}
	f.revision = 50
	f.chunk(t, "new", 6, assistantDelta("text-delta", 2, "late"))
	if len(f.s.Snapshot().Messages) != 0 {
		t.Fatal("suffix without known start was accepted")
	}
}

func TestAssistantBlockEndReplacesDeltaAndReasoningNeverExtractsAsText(t *testing.T) {
	f := newAssistantFixture()
	f.start(t, "a", 0, 0)
	f.chunk(t, "a", 0, assistantDelta("text-delta", 0, "He"))
	f.chunk(t, "a", 1, map[string]any{"type": "block-end", "index": 0, "block": map[string]any{"type": "text", "text": "Hello"}})
	if got := f.s.Snapshot().Messages; len(got) != 1 || got[0].Body != "Hello" {
		t.Fatalf("block-end appended instead of replacing: %+v", got)
	}
	for _, raw := range []string{`{"type":"reasoning-delta","text":"secret"}`, `{"type":"tool-call-delta","argumentsDelta":"secret"}`, `{"type":"usage","text":"secret"}`} {
		if extractChunkText(json.RawMessage(raw)) != "" {
			t.Fatal("non-answer chunk flattened into text")
		}
	}
}

func TestAssistantStreamStateResetsWithGeneration(t *testing.T) {
	f := newAssistantFixture()
	f.s.Select("A")
	generation := f.s.followGeneration
	f.start(t, "a", 0, 0)
	f.chunk(t, "a", 0, assistantDelta("text-delta", 0, "old"))
	f.s.Select("B")
	if f.s.assistantStream.active != nil {
		t.Fatal("attempt escaped selection generation")
	}
	if f.s.applyFollowFrameForTarget(context.Background(), "A", generation, json.RawMessage(`{"type":"assistant-stream","frame":{"type":"chunk","revision":3,"attemptId":"a","index":1,"chunk":{"type":"text-delta","index":0,"text":"late"}}}`)) {
		t.Fatal("stale generation accepted")
	}
}

func TestAssistantSuccessfulSettlementKeepsLiveAnchorUntilStepEnd(t *testing.T) {
	f := newAssistantFixture()
	f.start(t, "a", 0, 0)
	f.chunk(t, "a", 0, assistantDelta("text-delta", 0, "draft"))
	f.durable(t, `{"type":"tool/call","seq":2,"data":{"name":"read","arguments":"{}","callId":"c","turn":0,"step":0}}`)
	f.durable(t, assistantMessage(3, 0, 0, `[{"type":"text","text":"answer"}]`, false))
	f.end(t, "a", 1, assistantCommit(3, "assistant/message"))
	if got := f.s.Snapshot().Messages; len(got) != 2 || got[0].Body != "answer" || got[1].Kind != MessageKindTool {
		t.Fatalf("settlement jumped below pending tool: %+v", got)
	}
	f.durable(t, `{"type":"step/end","seq":4,"data":{"turn":0,"step":0}}`)
	if got := f.s.Snapshot().Messages; len(got) != 2 || got[0].Kind != MessageKindTool || got[1].Body != "answer" {
		t.Fatalf("durable ordering not restored: %+v", got)
	}
}

func TestAssistantProtocolRepairReopensFollowAndAdoptsAtomicBaseline(t *testing.T) {
	var opens atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		for {
			var frame dsh.ClientFrame
			if wsjson.Read(r.Context(), c, &frame) != nil {
				return
			}
			if frame.Type != dsh.FrameOpen {
				continue
			}
			number := opens.Add(1)
			send := func(raw string) bool {
				return wsjson.Write(r.Context(), c, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: json.RawMessage(raw)}) == nil
			}
			if number == 1 {
				if !send(`{"type":"snapshot","records":[],"assistantStream":{"revision":0}}`) {
					return
				}
				if !send(`{"type":"assistant-stream","frame":{"type":"start","revision":1,"attemptId":"a","startedAfterSeq":-1,"turn":0,"step":0}}`) {
					return
				}
				if !send(`{"type":"assistant-stream","frame":{"type":"chunk","revision":3,"attemptId":"a","index":1,"chunk":{"type":"text-delta","index":0,"text":"gap"}}}`) {
					return
				}
			} else {
				if !send(`{"type":"snapshot","records":[],"assistantStream":{"revision":3,"activeAttempt":{"attemptId":"a","startedAfterSeq":-1,"turn":0,"step":0,"nextIndex":1,"stream":[{"type":"text-chunks","time0":1,"index":0,"dt":[],"texts":["Hello"]}]}}}`) {
					return
				}
				if !send(`{"type":"assistant-stream","frame":{"type":"chunk","revision":4,"attemptId":"a","index":1,"time":2,"chunk":{"type":"text-delta","index":0,"text":"!"}}}`) {
					return
				}
				// Keep the repaired logical follow alive. An accepted EOF now
				// correctly means carrier loss and would request another baseline.
			}
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	defer mux.Close()
	s := New()
	s.selectedID = "a"
	repaired, finished := make(chan struct{}), make(chan struct{})
	var repairedOnce sync.Once
	s.notify = func() {
		messages := s.Snapshot().Messages
		if len(messages) == 1 && messages[0].Body == "Hello!" {
			repairedOnce.Do(func() { close(repaired) })
		}
	}
	go func() { s.runFollow(ctx, mux, "a", s.followGeneration); close(finished) }()
	select {
	case <-repaired:
	case <-ctx.Done():
		t.Fatal("repair did not adopt the complete atomic baseline")
	}
	cancel()
	wait(t, finished)
	if opens.Load() != 2 {
		t.Fatalf("repair opened %d streams, want 2", opens.Load())
	}
	if got := s.Snapshot().Messages; len(got) != 1 || got[0].Body != "Hello!" || s.assistantStream.rebaseline {
		t.Fatalf("repair did not adopt baseline: %+v", got)
	}
}
