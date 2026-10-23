package session

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestInspectorLogTargetInstalledReference(t *testing.T) {
	raw, err := os.ReadFile("../../ui/chat/testdata/inspector-targets.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Logs []struct {
			Name                   string
			Event, Chunk, Expected json.RawMessage
		}
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, item := range fixture.Logs {
		t.Run(item.Name, func(t *testing.T) {
			target := inspectorEventChatTarget(item.Event)
			if len(item.Chunk) > 0 {
				target = inspectorChunkChatTarget(target, item.Chunk)
			}
			got, _ := json.Marshal(target)
			var a, b any
			_ = json.Unmarshal(got, &a)
			_ = json.Unmarshal(item.Expected, &b)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("installed log target differs\ngot %s\nwant %s", got, item.Expected)
			}
		})
	}
}

func TestInspectorAssistantPartsRemainAddressableThroughSettlement(t *testing.T) {
	f := newAssistantFixture()
	f.s.selectedID = "s"
	f.start(t, "a", 0, 0)
	f.chunk(t, "a", 0, map[string]any{"type": "block-start", "index": 4, "blockType": "reasoning"})
	f.chunk(t, "a", 1, assistantDelta("reasoning-delta", 4, "reason"))
	f.chunk(t, "a", 2, map[string]any{"type": "block-start", "index": 9, "blockType": "text"})
	f.chunk(t, "a", 3, assistantDelta("text-delta", 9, "answer"))
	before := f.s.InspectorSnapshot()
	if target := before.Records[3].ChatTarget; target.GroupPart != "response" || target.Turn == nil || *target.Turn != 0 || target.Step == nil || *target.Step != 0 {
		t.Fatalf("live source coordinates or response part lost: %+v", target)
	}
	for _, msg := range f.s.Snapshot().Messages {
		target := msg.InspectorTarget()
		if target.Turn == nil || *target.Turn != 0 || target.Step == nil || *target.Step != 0 {
			t.Fatal("native live row lost zero Turn/Step")
		}
	}
	f.durable(t, assistantMessage(20, 0, 0, `[{"type":"reasoning","text":"canonical reason"},{"type":"text","text":"canonical answer"}]`, false))
	f.end(t, "a", 4, assistantCommit(20, "assistant/message"))
	after := f.s.InspectorSnapshot()
	for _, row := range after.Records {
		if row.Key == "attempt:a/chunk:2" || row.Key == "attempt:a/chunk:3" {
			target := row.ChatTarget
			if target.GroupPart != "response" || target.AnchorSeq == nil || *target.AnchorSeq != 20 || target.NodeKind != "assistant-step" {
				t.Fatalf("settlement collapsed response coordinates: %+v", target)
			}
		}
	}
	if before.Records[3].ChatTarget.AnchorSeq != nil {
		t.Fatal("later settlement mutated an earlier snapshot")
	}
}
