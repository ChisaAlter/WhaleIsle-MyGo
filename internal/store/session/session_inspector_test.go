package session

import (
	"context"
	"encoding/json"
	"testing"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func TestSessionInspectorRetainsExplicitZeroSequence(t *testing.T) {
	s := New()
	s.selectedID = "s"
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"agent/inbox/spliced","seq":0,"data":{"target":"next-turn","start":0,"inserted":[{"id":"wake","source":{"kind":"schedule"}}]}}},{"type":"event","event":{"type":"missing-sequence"}}]}`))
	rows := s.InspectorSnapshot().Records
	if len(rows) != 1 || rows[0].Seq != 0 || rows[0].Type != "agent/inbox/spliced" {
		t.Fatalf("zero-based durable sequence was discarded: %+v", rows)
	}
}

func TestSessionInspectorWindowOwnershipAndUnknownEvents(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "one"
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"turn/start","seq":4}},{"type":"event","event":{"type":"unrecognized/private","seq":5,"data":{"value":"fixture"}}}]}`))
	v := s.InspectorSnapshot()
	if len(v.Records) != 2 || v.Records[1].Parent != "event:4" {
		t.Fatalf("lost raw window: %+v", v)
	}
	v.Records[1].Raw[0] = '!'
	if s.InspectorSnapshot().Records[1].Raw[0] != '{' {
		t.Fatal("snapshot aliases source")
	}
	s.applyFollowFrame(json.RawMessage(`{"type":"event","event":{"type":"unrecognized/private","seq":5,"data":{"value":"fixture"}}}`))
	if len(s.InspectorSnapshot().Records) != 2 {
		t.Fatal("replay duplicate")
	}
	generation := s.followGeneration
	s.selectedID = "two"
	s.followGeneration++
	if s.applyFollowFrameForTarget(context.Background(), "one", generation, json.RawMessage(`{"type":"event","event":{"type":"secret","seq":6}}`)) {
		t.Fatal("stale accepted")
	}
	if len(s.InspectorSnapshot().Records) != 0 {
		t.Fatal("cross-session raw leak")
	}
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[]}`))
	if len(s.InspectorSnapshot().Records) != 0 {
		t.Fatal("snapshot did not replace window")
	}
}
func TestSessionInspectorOlderPageAndPackedStream(t *testing.T) {
	s := New()
	s.selectedID = "s"
	var page api.SessionPage
	if err := json.Unmarshal([]byte(`{"records":[{"type":"event","event":{"type":"private/history","seq":2}}]}`), &page); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.prependPageRecordsLocked(page.Records)
	s.mu.Unlock()
	if rows := s.InspectorSnapshot().Records; len(rows) != 1 || rows[0].Type != "private/history" {
		t.Fatalf("missing older raw: %+v", rows)
	}
	root := inspectorRecord("event:3", "", json.RawMessage(`{"type":"assistant/message","seq":3,"data":{"stream":[{"type":"chunk","time":1,"chunk":{"type":"block-start","index":0,"blockType":"text"}},{"type":"text-chunks","index":0,"time0":2,"dt":[3],"texts":["a","b"]}]}}`))
	rows := inspectorChildren(root)
	if len(rows) != 4 || rows[3].Type != "text-delta" || rows[3].Time != 5 || rows[3].Parent != rows[1].Key || rows[3].Identity != "2" {
		t.Fatalf("lost chunk/time: %+v", rows)
	}
}
