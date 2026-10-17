package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestToolInspectActionIsPresent(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ToolMessageView(c, session.Message{Tool: &session.ToolCallView{CallID: "c", Name: "shell"}}, ToolCardStyle{FontSize: 12}, ToolImageGalleryOptions{OnInspectCall: func(string) {}})
	}, 600, 500)
	if _, ok := tt.Find("检查此调用 / Inspect this call"); !ok {
		t.Fatal("tool card has no inspect-call action")
	}
}

func TestToolInspectActionDispatchesRealCallID(t *testing.T) {
	opened := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ToolMessageView(c, session.Message{Tool: &session.ToolCallView{CallID: "sub-opaque", Name: "same name"}}, ToolCardStyle{FontSize: 12}, ToolImageGalleryOptions{OnInspectCall: func(id string) { opened = id }})
	}, 600, 500)
	tt.Click("检查此调用 / Inspect this call")
	if opened != "sub-opaque" {
		t.Fatalf("inspect target %q", opened)
	}
	tt = ui.NewTester(func(c *ui.Context) {
		ToolMessageView(c, session.Message{Tool: &session.ToolCallView{CallID: "private", Name: "same name"}}, ToolCardStyle{FontSize: 12}, ToolImageGalleryOptions{TextOnly: true, OnInspectCall: func(id string) { opened = id }})
	}, 600, 500)
	if _, ok := tt.Find("检查此调用 / Inspect this call"); ok {
		t.Fatal("read-only addressed seat exposed main navigation")
	}
}

func TestToolInspectTargetUsesWireIdentity(t *testing.T) {
	msgs := []session.Message{
		{ID: "text", Body: "wanted", Kind: session.MessageKindText},
		{ID: "parent", Kind: session.MessageKindTool, Raw: []byte(`{"type":"tool/call","data":{"callId":"root","arguments":"wanted"}}`)},
		{ID: "child", Kind: session.MessageKindTool, Raw: []byte(`{"type":"tool/ptc-dispatch-start","data":{"subCallId":"wanted","parentCallId":"root"}}`)},
		{ID: "result", Kind: session.MessageKindTool, Raw: []byte(`{"type":"tool/result","data":{"message":{"toolCallId":"wanted"}}}`)},
	}
	id, index, ok := TrajectoryToolCallTarget(msgs, "wanted")
	if !ok || id != "child" || index != 2 {
		t.Fatalf("target %q %d %v", id, index, ok)
	}
	if _, _, ok := TrajectoryToolCallTarget(msgs, "want"); ok {
		t.Fatal("substring identity accepted")
	}
	conflict := session.Message{ID: "bad", Kind: session.MessageKindTool, Raw: []byte(`{"type":"tool/result","data":{"message":{"toolCallId":"wanted","source":{"callId":"other"}}}}`)}
	if _, _, ok := TrajectoryToolCallTarget([]session.Message{conflict}, "wanted"); ok {
		t.Fatal("conflicting identity accepted")
	}
	preparing := session.Message{Seq: 42, Kind: session.MessageKindTool, Tool: &session.ToolCallView{CallID: "preparing"}}
	if id, _, ok := TrajectoryToolCallTarget([]session.Message{preparing}, "preparing"); !ok || id != "seq-42" {
		t.Fatal("preparing target lost")
	}
}

func TestToolInspectTrajectoryRevealsSelectedRowOnMount(t *testing.T) {
	msgs := make([]session.Message, 100)
	for i := range msgs {
		msgs[i] = session.Message{ID: fmt.Sprint(i), Kind: session.MessageKindText, Body: "ordinary row"}
	}
	msgs[99].Body = "inspected last call"
	selected, query := "99", ""
	list := &ui.ListState{}
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: session.ViewState{Messages: msgs}, SelectedID: &selected, SearchQuery: &query, List: list})
	}, 900, 600)
	if _, ok := tt.Find("Trajectory row 99"); !ok {
		t.Fatal("trajectory did not reveal selected row on mount")
	}
}
