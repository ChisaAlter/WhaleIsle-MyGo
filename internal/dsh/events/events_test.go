package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRegistryHasKeyEventKinds(t *testing.T) {
	if len(Roots) < 50 {
		t.Fatalf("Roots = %d, want >= 50", len(Roots))
	}
	want := []struct {
		typ     string
		goType  string
		surface bool
	}{
		{TypeUserMessage, "UserMessage", true},
		{TypeAssistantMessage, "AssistantMessage", true},
		{TypeToolCall, "ToolCall", false},
		{TypeToolResult, "ToolResult", true},
		{TypeTurnStart, "TurnStart", false},
		{TypeApprovalAsked, "ApprovalAsked", false},
	}
	for _, w := range want {
		info, ok := EventsByType[w.typ]
		if !ok {
			t.Fatalf("EventsByType missing %q", w.typ)
		}
		if info.GoType != w.goType {
			t.Fatalf("%s GoType = %q, want %q", w.typ, info.GoType, w.goType)
		}
		if info.Surface != w.surface {
			t.Fatalf("%s Surface = %v, want %v", w.typ, info.Surface, w.surface)
		}
		if info.Kind != KindEvent {
			t.Fatalf("%s Kind = %q", w.typ, info.Kind)
		}
	}
	// Compile-time presence of generated structs.
	_ = SessionHeader{}
	_ = UserMessage{}
	_ = AssistantMessage{}
	_ = ToolCall{}
	_ = ToolResult{}
	_ = TurnStart{}
	_ = ApprovalAsked{}
	_ = SessionEventEnvelope(nil)
}

func TestToolCallRoundTrip(t *testing.T) {
	in := ToolCall{
		Type: TypeToolCall,
		Seq:  7,
		Time: 1_700_000_000_000,
		Data: ToolCallData{
			Arguments: `{"path":"README.md"}`,
			CallId:    "call_1",
			Name:      "read",
			Step:      1,
			Turn:      2,
		},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	if env.Type != TypeToolCall || env.Seq != 7 {
		t.Fatalf("envelope = %+v", env)
	}
	var out ToolCall
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestUserMessageRoundTrip(t *testing.T) {
	in := UserMessage{
		Type:      TypeUserMessage,
		Seq:       3,
		Time:      42,
		SurfaceOp: json.RawMessage(`"append"`),
		Data: UserMessageData{
			Id:      "msg_1",
			Role:    "user",
			Content: []json.RawMessage{json.RawMessage(`{"type":"text","text":"hello"}`)},
			Source:  json.RawMessage(`{"kind":"typed"}`),
		},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if env.Type != TypeUserMessage {
		t.Fatalf("type = %q", env.Type)
	}
	var out UserMessage
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Type != in.Type || out.Seq != in.Seq || out.Data.Id != in.Data.Id || out.Data.Role != in.Data.Role {
		t.Fatalf("round-trip mismatch: %+v vs %+v", in, out)
	}
	if string(out.SurfaceOp) != `"append"` {
		t.Fatalf("surfaceOp = %s", out.SurfaceOp)
	}
}

func TestTurnStartAndSessionHeaderRoundTrip(t *testing.T) {
	turn := TurnStart{
		Type: TypeTurnStart,
		Seq:  1,
		Time: 10,
		Data: TurnStartData{Turn: 1},
	}
	raw, err := json.Marshal(turn)
	if err != nil {
		t.Fatal(err)
	}
	var turnOut TurnStart
	if err := json.Unmarshal(raw, &turnOut); err != nil {
		t.Fatal(err)
	}
	if turnOut != turn {
		t.Fatalf("turn: %+v vs %+v", turn, turnOut)
	}

	hdr := SessionHeader{
		Id:        "sess_1",
		CreatedAt: 99,
		IsSeeded:  false,
		Version:   4,
	}
	raw, err = json.Marshal(hdr)
	if err != nil {
		t.Fatal(err)
	}
	var hdrOut SessionHeader
	if err := json.Unmarshal(raw, &hdrOut); err != nil {
		t.Fatal(err)
	}
	if hdrOut != hdr {
		t.Fatalf("header: %+v vs %+v", hdr, hdrOut)
	}
}

func TestSchemaFileStillParses(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	candidates := []string{
		filepath.Join(root, "runtime", "vendor", "deepseek-harness", "docs", "persistence-schema.json"),
		filepath.Join(root, "reference", "electron", "vendor", "deepseek-harness", "docs", "persistence-schema.json"),
	}
	var data []byte
	var err error
	var path string
	for _, c := range candidates {
		data, err = os.ReadFile(c)
		if err == nil {
			path = c
			break
		}
	}
	if err != nil {
		t.Skipf("persistence-schema.json not found: %v", err)
	}
	var inv struct {
		FormatVersion int `json:"formatVersion"`
		Roots         []struct {
			Key   string `json:"key"`
			Kind  string `json:"kind"`
			Event string `json:"event"`
		} `json:"roots"`
	}
	if err := json.Unmarshal(data, &inv); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if inv.FormatVersion != 2 {
		t.Fatalf("formatVersion = %d", inv.FormatVersion)
	}
	if len(inv.Roots) != len(Roots) {
		t.Fatalf("schema roots %d != generated Roots %d (regenerate with tools/gen-events)", len(inv.Roots), len(Roots))
	}
	for _, r := range inv.Roots {
		if r.Kind != "event" {
			continue
		}
		if _, ok := EventsByType[r.Event]; !ok {
			t.Fatalf("generated EventsByType missing %q from %s", r.Event, path)
		}
	}
}
