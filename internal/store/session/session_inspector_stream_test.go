package session

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func inspectorRowKeys(rows []InspectorRecord) []string {
	var keys []string
	for _, row := range rows {
		keys = append(keys, row.Key)
	}
	return keys
}

func inspectorFind(t *testing.T, rows []InspectorRecord, key string) InspectorRecord {
	t.Helper()
	for _, row := range rows {
		if row.Key == key {
			return row
		}
	}
	t.Fatalf("missing %s in %v", key, inspectorRowKeys(rows))
	return InspectorRecord{}
}

func TestSessionInspectorStreamHierarchyLifecycleAndSummaries(t *testing.T) {
	root := inspectorRecord("event:3", "event:2", json.RawMessage(`{"type":"assistant/attempt","seq":3,"data":{"turn":1,"step":2,"stream":[
		{"type":"chunk","time":100,"chunk":{"type":"block-start","index":0,"blockType":"reasoning"}},
		{"type":"reasoning-chunks","index":0,"time0":101,"dt":[],"texts":["first "]},
		{"type":"chunk","time":102,"chunk":{"type":"block-start","index":1,"blockType":"reasoning"}},
		{"type":"reasoning-chunks","index":1,"time0":103,"dt":[],"texts":["other"]},
		{"type":"reasoning-chunks","index":0,"time0":104,"dt":[],"texts":["second"]},
		{"type":"chunk","time":105,"chunk":{"type":"block-end","index":0,"block":{"type":"reasoning","text":"not a delta"}}},
		{"type":"chunk","time":106,"chunk":{"type":"finish","reason":{"kind":"stop"}}},
		{"type":"reasoning-chunks","index":9,"time0":107,"dt":[],"texts":["no loaded start"]}
	]}}`))
	rows := inspectorChildren(root)
	want := []string{"event:3", "event:3/chunk:0", "event:3/chunk:1", "event:3/chunk:4", "event:3/chunk:5", "event:3/chunk:2", "event:3/chunk:3", "event:3/chunk:6", "event:3/chunk:7"}
	if !reflect.DeepEqual(inspectorRowKeys(rows), want) {
		t.Fatalf("interleaved blocks lost depth-first order: %v", inspectorRowKeys(rows))
	}
	first, second := rows[1], rows[5]
	if first.Disclosure != "closed" || second.Disclosure != "open" || root.Disclosure != "" || rows[0].Disclosure != "" {
		t.Fatalf("wrong block or root lifecycle: %+v %+v", first, second)
	}
	if first.CollapsedSummary != "blockType: reasoning · first second" || second.CollapsedSummary != "blockType: reasoning · other" {
		t.Fatalf("reasoning summaries crossed blocks or used final text: %+v %+v", first, second)
	}
	for _, key := range []string{"event:3/chunk:6", "event:3/chunk:7"} {
		if got := inspectorFind(t, rows, key); got.Parent != root.Key || got.CollapsedSummary != "" {
			t.Fatalf("unindexed finish or orphan delta became block 0's child: %+v", got)
		}
	}
	child := inspectorFind(t, rows, "event:3/chunk:4")
	if child.Identity != "4" || child.Time != 104 || child.Seq != 3 || child.Location != "1/2" || child.Parent != first.Key {
		t.Fatalf("chunk metadata or original index lost: %+v", child)
	}
}

func TestSessionInspectorReasoningSummaryBoundAndRaw(t *testing.T) {
	text := strings.Repeat("海🌊", 180)
	value := map[string]any{"type": "assistant/message", "data": map[string]any{"stream": []any{
		map[string]any{"type": "chunk", "time": 0, "chunk": map[string]any{"type": "block-start", "index": 4, "blockType": "reasoning"}},
		map[string]any{"type": "reasoning-chunks", "index": 4, "time0": 1, "dt": []int{1}, "texts": []string{text, "later"}},
	}}}
	raw, _ := json.Marshal(value)
	rows := inspectorChildren(inspectorRecord("assistant", "", raw))
	want := inspectorStreamPrefix("blockType: reasoning · "+text, 240)
	if rows[1].CollapsedSummary != want || len(utf16.Encode([]rune(want))) != 240 || len(rows) != 4 {
		t.Fatalf("bounded summary: %q, rows %d", rows[1].CollapsedSummary, len(rows))
	}
	var original struct{ Text string }
	_ = json.Unmarshal(rows[2].Raw, &original)
	if original.Text != text || string(rows[0].Raw) != string(raw) {
		t.Fatal("summary truncation mutated the raw stream")
	}
}

func TestSessionInspectorLiveChunksAnchorAtFirstChunkAndInvalidate(t *testing.T) {
	f := newAssistantFixture()
	f.s.selectedID = "s"
	f.durable(t, `{"type":"turn/start","seq":0,"data":{"turn":1}}`)
	f.durable(t, `{"type":"step/start","seq":1,"data":{"turn":1,"step":0}}`)
	f.start(t, "live", 1, 0)
	if len(f.s.InspectorSnapshot().Records) != 2 {
		t.Fatal("start without chunks invented a live row")
	}
	f.durable(t, `{"type":"future/before-first-chunk","seq":2}`)
	f.s.mu.Lock()
	changed := f.s.applyFollowFrameLocked(json.RawMessage(`{"type":"assistant-stream","frame":{"type":"chunk","revision":2,"attemptId":"live","index":0,"time":100,"chunk":{"type":"block-start","index":0,"blockType":"reasoning"}}}`))
	f.s.mu.Unlock()
	if !changed {
		t.Fatal("raw-only start chunk did not invalidate the inspector")
	}
	f.revision = 2
	f.durable(t, `{"type":"future/after-first-chunk","seq":3}`)
	f.chunk(t, "live", 1, assistantDelta("reasoning-delta", 0, "thinking"))
	rows := f.s.InspectorSnapshot().Records
	want := []string{"event:0", "event:1", "event:2", "attempt:live", "attempt:live/chunk:0", "attempt:live/chunk:1", "event:3"}
	if !reflect.DeepEqual(inspectorRowKeys(rows), want) {
		t.Fatalf("live root moved out of its first insertion gap: %v", inspectorRowKeys(rows))
	}
	root := rows[3]
	if root.Type != "assistant/live" || root.Identity != "live" || root.Parent != "event:1" || root.Location != "1/0" || root.Time != 101 || root.ChatID == "" || rows[4].ChatID != root.ChatID || rows[5].ChatID != root.ChatID {
		t.Fatalf("live metadata: %+v", root)
	}
	var metadata map[string]any
	_ = json.Unmarshal(root.Raw, &metadata)
	if len(metadata) != 4 || metadata["chunks"] != float64(2) || metadata["attemptId"] != "live" {
		t.Fatalf("live value exposed internal fold state: %s", root.Raw)
	}
	f.chunk(t, "live", 2, map[string]any{"type": "block-end", "index": 0, "block": map[string]any{"type": "reasoning", "text": "thinking"}})
	if inspectorFind(t, f.s.InspectorSnapshot().Records, "attempt:live/chunk:0").Disclosure != "closed" {
		t.Fatal("live block-end did not update disclosure")
	}
	f.chunk(t, "live", 3, map[string]any{"type": "finish", "reason": map[string]any{"kind": "stop"}})
	if inspectorFind(t, f.s.InspectorSnapshot().Records, "attempt:live/chunk:3").ChatID != root.ChatID {
		t.Fatal("unindexed finish lost the owning native Chat identity")
	}
	before := f.s.InspectorSnapshot()
	last := len(before.Records)
	f.chunk(t, "live", 5, assistantDelta("text-delta", 0, "bad gap"))
	if !f.s.assistantStream.rebaseline || len(f.s.InspectorSnapshot().Records) != last || before.Records[3].Time != 103 {
		t.Fatal("protocol gap mutated last good inspector prefix")
	}
}

func TestSessionInspectorSuccessfulLiveRetiresAtMatchingStepEnd(t *testing.T) {
	f := newAssistantFixture()
	f.s.selectedID = "s"
	f.start(t, "a", 1, 0)
	f.chunk(t, "a", 0, assistantDelta("text-delta", 0, "answer"))
	f.durable(t, assistantMessage(4, 1, 0, `[{"type":"text","text":"answer"}]`, false))
	f.end(t, "a", 1, assistantCommit(4, "assistant/message"))
	rows := f.s.InspectorSnapshot().Records
	if got := inspectorRowKeys(rows); !reflect.DeepEqual(got, []string{"attempt:a", "attempt:a/chunk:0", "event:4"}) {
		t.Fatalf("success should retain live and a separate durable record: %v", got)
	}
	if rows[0].Seq != 4 || rows[1].Seq != 4 || rows[0].ChatID != "" || rows[1].ChatID != "" {
		t.Fatal("retained successful raw rows target retired native prefix IDs")
	}
	f.durable(t, `{"type":"step/end","seq":5,"data":{"turn":9,"step":9}}`)
	if f.s.assistantStream.retained == nil {
		t.Fatal("unrelated step retired the live attempt")
	}
	f.durable(t, `{"type":"step/end","seq":6,"data":{"turn":1,"step":0}}`)
	if f.s.assistantStream.retained != nil || inspectorFind(t, f.s.InspectorSnapshot().Records, "event:4").Type != "assistant/message" {
		t.Fatal("matching end lost durable row or retained live prefix")
	}
	for _, row := range f.s.InspectorSnapshot().Records {
		if strings.HasPrefix(row.Key, "attempt:") {
			t.Fatal("successful live prefix remained after matching step end")
		}
	}
}

func TestSessionInspectorFailedSettlementIdentityAndWindowPruning(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		f := newAssistantFixture()
		f.s.selectedID = "s"
		f.start(t, "a", 1, 0)
		f.chunk(t, "a", 0, map[string]any{"type": "block-start", "index": 0, "blockType": "text"})
		f.chunk(t, "a", 1, assistantDelta("text-delta", 0, "partial"))
		typ := "assistant/attempt"
		if interrupted {
			typ = "assistant/message"
		}
		raw := `{"type":"` + typ + `","seq":4,"surfaceOp":"append","data":{"turn":1,"step":0,"interrupted":true,"stream":[{"type":"chunk","time":100,"chunk":{"type":"block-start","index":0,"blockType":"text"}},{"type":"text-chunks","index":0,"time0":101,"dt":[],"texts":["partial"]}]}}`
		f.durable(t, raw)
		f.end(t, "a", 2, assistantCommit(4, typ))
		check := func(key string) {
			t.Helper()
			rows := f.s.InspectorSnapshot().Records
			if len(rows) != 3 || rows[0].Key != key || rows[0].Type != typ || rows[0].Identity != "4" || rows[1].Key != key+"/chunk:0" || rows[1].Disclosure != "open" {
				t.Fatalf("failed/interrupted settlement changed stable row or closed unfinished block: %+v", rows)
			}
		}
		check("attempt:a")
		f.s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":` + raw + `}]}`))
		check("attempt:a")
		// Replacement evicts the alias immediately, even before another inspector snapshot.
		f.s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[]}`))
		f.durable(t, raw)
		check("event:4")
	}
}

func TestSessionInspectorBaselineSuffixCapturedOnceAndAbandonment(t *testing.T) {
	f := newAssistantFixture()
	f.s.selectedID = "s"
	f.s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"turn/start","seq":4}}],"assistantStream":{"revision":10,"activeAttempt":{"attemptId":"reconnect","startedAfterSeq":-1,"turn":2,"step":0,"nextIndex":2,"stream":[{"type":"chunk","time":100,"chunk":{"type":"block-start","index":0,"blockType":"reasoning"}},{"type":"reasoning-chunks","index":0,"time0":101,"dt":[],"texts":["prefix"]}]}}}`))
	f.revision = 10
	before := f.s.InspectorSnapshot()
	if len(before.Records) != 4 || before.Records[1].Key != "attempt:reconnect" || before.Records[1].Parent != "event:4" {
		t.Fatalf("baseline must append the reconstructed prefix after loaded events: %+v", before.Records)
	}
	f.chunk(t, "reconnect", 2, assistantDelta("reasoning-delta", 0, " suffix"))
	rows := f.s.InspectorSnapshot().Records
	if len(rows) != 5 || rows[4].Identity != "2" || rows[2].CollapsedSummary != "blockType: reasoning · prefix suffix" || before.Records[2].CollapsedSummary != "blockType: reasoning · prefix" {
		t.Fatal("baseline replay duplicated raw chunks or mutated an earlier snapshot")
	}
	f.chunk(t, "other-attempt", 50, assistantDelta("text-delta", 0, "ignored"))
	if len(f.s.InspectorSnapshot().Records) != 5 {
		t.Fatal("unknown attempt suffix entered inspector")
	}
	f.end(t, "reconnect", 3, map[string]any{"kind": "abandoned"})
	if got := f.s.InspectorSnapshot().Records; len(got) != 1 || got[0].Key != "event:4" {
		t.Fatal("abandonment retained a live row or invented a durable record")
	}
}

func TestSessionInspectorEmptyBaselineWaitsForFirstActualChunk(t *testing.T) {
	f := newAssistantFixture()
	f.s.selectedID = "s"
	f.s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"private/loaded","seq":10}}],"assistantStream":{"revision":10,"activeAttempt":{"attemptId":"empty","startedAfterSeq":10,"turn":2,"step":0,"nextIndex":0,"stream":[]}}}`))
	f.revision = 10
	f.durable(t, `{"type":"private/before-first","seq":11}`)
	f.chunk(t, "empty", 0, map[string]any{"type": "block-start", "index": 0, "blockType": "reasoning"})
	if got := inspectorRowKeys(f.s.InspectorSnapshot().Records); !reflect.DeepEqual(got, []string{"event:10", "event:11", "attempt:empty", "attempt:empty/chunk:0"}) {
		t.Fatalf("empty reconnect baseline anchored before first actual chunk: %v", got)
	}
	f.chunk(t, "empty", 1, assistantDelta("text-delta", 1, "visible response"))
	f.chunk(t, "empty", 2, map[string]any{"type": "finish", "reason": map[string]any{"kind": "stop"}})
	rows := f.s.InspectorSnapshot().Records
	owner := inspectorFind(t, rows, "attempt:empty")
	visible := inspectorFind(t, rows, "attempt:empty/chunk:1")
	if owner.ChatID == "" || owner.ChatID != visible.ChatID || visible.ChatPickID != visible.ChatID || inspectorFind(t, rows, "attempt:empty/chunk:0").ChatPickID != "" || inspectorFind(t, rows, "attempt:empty/chunk:0").ChatID != visible.ChatID || inspectorFind(t, rows, "attempt:empty/chunk:2").ChatID != visible.ChatID {
		t.Fatal("empty first block swallowed owner fallback to visible native message")
	}
}

func TestSessionInspectorTurnEndAndMissingRangeHeaders(t *testing.T) {
	f := newAssistantFixture()
	f.durable(t, `{"type":"private/first","seq":0}`)
	f.durable(t, `{"type":"step/start","seq":1,"data":{"turn":1,"step":1}}`)
	f.durable(t, `{"type":"turn/end","seq":2,"data":{"turn":1}}`)
	rows := f.s.InspectorSnapshot().Records
	if rows[0].Parent != "" || rows[1].Parent != "" || rows[2].Parent != "" || rows[1].Disclosure != "open" {
		t.Fatalf("missing start invented ancestry or end inherited Step: %+v", rows)
	}
	f.durable(t, `{"type":"turn/start","seq":3,"data":{"turn":2}}`)
	f.durable(t, `{"type":"step/start","seq":4,"data":{"turn":2,"step":0}}`)
	f.durable(t, `{"type":"turn/end","seq":5,"data":{"turn":2}}`)
	rows = f.s.InspectorSnapshot().Records
	if rows[5].Parent != "event:3" || rows[3].Disclosure != "open" || rows[4].Disclosure != "open" {
		t.Fatal("Turn end changed start defaults or inherited the still-open Step")
	}
}
