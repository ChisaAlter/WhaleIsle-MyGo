package chat

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

func TestInspectorRevealTargetInstalledReference(t *testing.T) {
	raw, err := os.ReadFile("testdata/inspector-targets.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Name   string
			Target InspectorTarget
			Nodes  []struct {
				ID, Kind, RootCallID string
				Seq                  float64
				Turn, Step           *float64
				Visible              bool
			}
			Expected []InspectorTarget
		}
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, item := range fixture.Cases {
		t.Run(item.Name, func(t *testing.T) {
			nodes := make([]inspectorRevealNode, 0, len(item.Nodes))
			for _, node := range item.Nodes {
				nodes = append(nodes, inspectorRevealNode{ID: node.ID, Kind: node.Kind, RootCallID: node.RootCallID, Seq: node.Seq, Turn: node.Turn, Step: node.Step, Visible: node.Visible})
			}
			got := inspectorRevealTargets(item.Target, nodes)
			if len(got) != len(item.Expected) {
				t.Fatalf("candidate count %d, want %d: %+v", len(got), len(item.Expected), got)
			}
			for i := range got {
				if inspectorTargetIdentity(got[i]) != inspectorTargetIdentity(item.Expected[i]) {
					a, _ := json.Marshal(got)
					b, _ := json.Marshal(item.Expected)
					t.Fatalf("candidate %d differs from installed function\ngot  %s\nwant %s", i, a, b)
				}
			}
		})
	}
}

func TestInspectorExactOccurrencePrecedesSharedAssistantSource(t *testing.T) {
	for _, kind := range []session.MessageKind{session.MessageKindThinking, session.MessageKindImage} {
		messages := []session.Message{
			{ID: "m/first/0", SourceMessageID: "m", Kind: kind, Role: "assistant", Seq: 40},
			{ID: "m", SourceMessageID: "m", Kind: session.MessageKindText, Role: "assistant", Seq: 40},
			{ID: "m/text/2", SourceMessageID: "m", Kind: session.MessageKindText, Role: "assistant", Seq: 40},
		}
		if got := inspectorMessageIndex(messages, InspectorTarget{ID: "m", Seq: 40}); got != 1 {
			t.Fatalf("%s shadowed exact text occurrence: %d", kind, got)
		}
		if got := inspectorMessageIndex(messages, messages[2].InspectorTarget()); got != 2 {
			t.Fatalf("earlier response shadowed second exact text occurrence: %d", got)
		}
		if kind == session.MessageKindThinking {
			if got := inspectorMessageIndex(messages, InspectorTarget{ID: "m/first/0", GroupPart: "response"}); got != 1 {
				t.Fatalf("response selected reasoning occurrence: %d", got)
			}
		}
	}
}

func TestInspectorToolResultOwnerSurvivesFoldingAndCallPrepend(t *testing.T) {
	call := session.Message{ID: "call", Kind: session.MessageKindTool, Seq: 10, EventType: "tool/call", Raw: json.RawMessage(`{"type":"tool/call","seq":10,"data":{"turn":1,"step":0,"callId":"root","name":"functions.pwsh","arguments":"{}"}}`)}
	result := session.Message{ID: "result", Kind: session.MessageKindTool, Seq: 20, EventType: "tool/result", Raw: json.RawMessage(`{"type":"tool/result","seq":20,"data":{"turn":1,"step":0,"message":{"source":{"callId":"root"},"toolCallId":"root","content":[]}}}`)}
	answer := session.Message{ID: "answer", Kind: session.MessageKindText, Role: "assistant", Seq: 21, Body: "Final answer", Raw: json.RawMessage(`{"data":{"turn":1,"step":1}}`)}
	end := session.Message{ID: "end", Seq: 22, EventType: "turn/end", Raw: json.RawMessage(`{"data":{"turn":1}}`)}
	for _, mode := range []string{"verbose", "standard"} {
		for _, prepend := range []bool{false, true} {
			messages := []session.Message{result, answer, end}
			if prepend {
				messages = append([]session.Message{call}, messages...)
			}
			n := &InspectorNavigation{SessionID: "s", Target: &InspectorTarget{Seq: 20, CallID: "root"}}
			expanded := map[string]bool{}
			m := Model{Inspector: n, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h", Messages: messages}, ThinkingExpanded: &expanded, Preferences: settings.ChatPreferences{TranscriptView: mode}}
			for range 8 {
				projected, controls := projectTranscript(session.ProjectToolMessages(messages), m)
				prepareSessionInspector(m, projected, controls, &ui.ListState{})
				if n.Target == nil {
					break
				}
			}
			if n.Target != nil || n.Highlight != "tool:root" {
				t.Fatalf("mode=%s prepend=%v target=%+v highlight=%q expanded=%v", mode, prepend, n.Target, n.Highlight, expanded)
			}
			if mode == "standard" && !expanded["transcript-turn:h:s:1"] {
				t.Fatal("tool result did not open its actual projected ancestor")
			}
		}
	}
}

func TestInspectorExactFinalAnswerDoesNotOpenReasoningGroup(t *testing.T) {
	n := &InspectorNavigation{SessionID: "s", Target: &InspectorTarget{ID: "m", Seq: 40}}
	expanded := map[string]bool{}
	coordinate := json.RawMessage(`{"data":{"turn":1,"step":1}}`)
	messages := []session.Message{
		{ID: "m/reasoning/0", SourceMessageID: "m", Seq: 40, Role: "thinking", Kind: session.MessageKindThinking, Body: "Hidden", Raw: coordinate},
		{ID: "m", SourceMessageID: "m", Seq: 40, Role: "assistant", Kind: session.MessageKindText, Body: "Answer", Raw: coordinate},
		{ID: "end", Seq: 41, EventType: "turn/end", Raw: coordinate},
	}
	m := Model{Inspector: n, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h", Messages: messages}, ThinkingExpanded: &expanded}
	projected, controls := projectTranscript(messages, m)
	prepareSessionInspector(m, projected, controls, &ui.ListState{})
	if n.Highlight != "m" || n.Target != nil || expanded["transcript-turn:h:s:1"] || expanded["m/reasoning/0"] {
		t.Fatalf("visible exact answer unfolded unrelated reasoning: highlight=%q choices=%v", n.Highlight, expanded)
	}
}

func TestInspectorResponsePartUsesPaintedAnswerAfterSettlement(t *testing.T) {
	for _, part := range []string{"response", "reasoning"} {
		n := &InspectorNavigation{SessionID: "s"}
		expanded := map[string]bool{}
		coordinate := json.RawMessage(`{"data":{"turn":1,"step":1}}`)
		m := Model{Inspector: n, List: &ui.ListState{}, ThinkingExpanded: &expanded, ThemeID: theme.IDDeepseekLight, SuppressStatus: true, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h", Messages: []session.Message{
			{ID: "m/reasoning/0", SourceMessageID: "m", Seq: 40, Kind: session.MessageKindThinking, Role: "thinking", Body: "Reasoning details.", Raw: coordinate},
			{ID: "m", SourceMessageID: "m", Seq: 40, Kind: session.MessageKindText, Role: "assistant", Body: "Canonical answer.", Raw: coordinate},
			{ID: "end", Seq: 41, EventType: "turn/end", Raw: coordinate},
		}}}
		tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 700, 500)
		tt.SetTime(time.Unix(7000, 0))
		n.Target = &InspectorTarget{Seq: 40, GroupPart: part, NodeKind: "assistant-step"}
		for range 5 {
			tt.Frame()
		}
		want := "m"
		if part == "reasoning" {
			want = "m/reasoning/0"
		}
		if n.Highlight != want || n.reveal.started.IsZero() || n.reveal.rect.W <= 0 || n.reveal.rect.H <= 0 {
			t.Fatalf("part=%s highlighted=%q reveal=%+v", part, n.Highlight, n.reveal)
		}
		if n.reveal.rect != inspectorChatIntersect(n.reveal.rows[want], n.reveal.viewport) {
			t.Fatal("part highlight did not use its actual painted occurrence")
		}
		if part == "response" && expanded["transcript-turn:h:s:1"] {
			t.Fatal("response flash opened unrelated reasoning ancestor")
		}
		if part == "reasoning" && !tt.HasText("Reasoning details.") {
			t.Fatal("reasoning part did not reveal its body")
		}
		n.CancelReveal()
	}
}

func TestInspectorQueuedLivePartRebindsWhenSettlementRetiresNativeID(t *testing.T) {
	zero := float64(0)
	for _, part := range []string{"response", "reasoning"} {
		n := &InspectorNavigation{SessionID: "s", Target: &InspectorTarget{ID: "retired-live-block", LiveAssistant: true, NodeKind: "assistant-step", Turn: &zero, Step: &zero, GroupPart: part}}
		expanded := map[string]bool{}
		coordinate := json.RawMessage(`{"data":{"turn":0,"step":0}}`)
		messages := []session.Message{
			{ID: "m/reasoning/0", SourceMessageID: "m", Seq: 40, Kind: session.MessageKindThinking, Role: "thinking", Body: "reason", Raw: coordinate},
			{ID: "m", SourceMessageID: "m", Seq: 40, Kind: session.MessageKindText, Role: "assistant", Body: "answer", Raw: coordinate},
			{ID: "end", Seq: 41, EventType: "turn/end", Raw: coordinate},
		}
		m := Model{Inspector: n, ThinkingExpanded: &expanded, Snapshot: session.ViewState{SelectedID: "s", Messages: messages}}
		for range 8 {
			projected, controls := projectTranscript(messages, m)
			prepareSessionInspector(m, projected, controls, &ui.ListState{})
			if n.Target == nil {
				break
			}
		}
		want := "m"
		if part == "reasoning" {
			want = "m/reasoning/0"
		}
		if n.Highlight != want {
			t.Fatalf("queued %s lost the surviving Assistant Step: %q", part, n.Highlight)
		}
	}
}

func TestInspectorLivePartRebindsDuringQueuedAndRunningScroll(t *testing.T) {
	for _, running := range []bool{false, true} {
		tt, n, scroll, list, m := inspectorRevealFixture(t)
		zero := float64(0)
		m.Snapshot.Messages[25].Streaming = true
		m.Snapshot.Messages[25].Role = "assistant"
		m.Snapshot.Messages[25].ConversationTurn = &zero
		m.Snapshot.Messages[25].ConversationStep = &zero
		list.ScrollToOffset(0)
		tt.Frame()
		target := m.Snapshot.Messages[25].InspectorTarget()
		n.Target = &target
		tt.Frame()
		if running {
			tt.Frame()
			tt.SetTime(time.Unix(7000, 0).Add(80 * time.Millisecond))
			if !list.ScrollAnimating() {
				t.Fatal("fixture did not start actual smooth scrolling")
			}
		}
		m.Snapshot.Messages[25].ID = "canonical"
		m.Snapshot.Messages[25].Streaming = false
		for range 3 {
			tt.Frame()
		}
		tt.SetTime(time.Unix(7000, 0).Add(3 * time.Second))
		tt.SetTime(time.Unix(7000, 0).Add(3*time.Second + time.Millisecond))
		tt.SetTime(time.Unix(7000, 0).Add(3*time.Second + 2*time.Millisecond))
		tt.SetTime(time.Unix(7000, 0).Add(3*time.Second + 3*time.Millisecond))
		if n.Highlight != "canonical" || n.reveal.started.IsZero() || scroll.Y <= 1000 || list.ScrollAnimating() {
			t.Fatalf("running=%v settlement lost pending part: highlight=%q scroll=%v reveal=%+v", running, n.Highlight, scroll.Y, n.reveal)
		}
		n.CancelReveal()
	}
}

func TestInspectorEmptyPaintedCandidateTriesRankedNeighborBeforeColumn(t *testing.T) {
	n := &InspectorNavigation{SessionID: "s"}
	m := Model{Inspector: n, List: &ui.ListState{}, ThemeID: theme.IDDeepseekLight, SuppressStatus: true, Snapshot: session.ViewState{SelectedID: "s", Messages: []session.Message{
		{ID: "empty", Seq: 10, Kind: session.MessageKindTurnDelivery, TurnDelivery: &session.TurnDelivery{Turn: 1, Changes: &session.TurnFileTarget{Kind: "changes", Turn: 1, Seq: 10}}},
		{ID: "neighbor", Seq: 11, Kind: session.MessageKindText, Role: "user", Body: "Visible ranked neighbor."},
	}}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 700, 500)
	tt.SetTime(time.Unix(7000, 0))
	n.Target = &InspectorTarget{ID: "empty"}
	for range 6 {
		tt.Frame()
	}
	tt.SetTime(time.Unix(7000, 0).Add(200 * time.Millisecond))
	if n.Highlight != "neighbor" || n.reveal.started.IsZero() || n.reveal.rect != inspectorChatIntersect(n.reveal.rows["neighbor"], n.reveal.viewport) {
		t.Fatalf("empty candidate skipped nearby painted box: highlight=%q reveal=%+v", n.Highlight, n.reveal)
	}
}

func TestInspectorReducedMotionRejectsPaintedEmptyRowBeforeCentering(t *testing.T) {
	for _, reduced := range []bool{false, true} {
		n := &InspectorNavigation{SessionID: "s"}
		list, scroll := &ui.ListState{}, &ui.ScrollState{}
		m := Model{Inspector: n, Snapshot: session.ViewState{SelectedID: "s"}}
		for i := range 40 {
			m.Snapshot.Messages = append(m.Snapshot.Messages, session.Message{ID: string(rune('A' + i)), Seq: float64(i + 1)})
		}
		targetID := "E"
		tt := ui.NewTester(func(c *ui.Context) {
			prepareSessionInspector(m, m.Snapshot.Messages, nil, list)
			view := ui.List(c, list, len(m.Snapshot.Messages)).Fill().TrackScroll(scroll)
			inspectorChatViewport(c, m, view, list)
			view.Rows(func(row ui.ListRow) {
				id := m.Snapshot.Messages[row.Index].ID
				box := ui.Box(row.Context).Height(60).FillWidth()
				if id == targetID {
					box.Height(0)
				}
				inspectorChatRowGeometry(row.Context, m, id, box, 0, 0, 0, 0)
			})
			inspectorChatRevealOverlay(c, m, theme.Default().Light)
		}, 300, 360)
		tt.SetTime(time.Unix(7000, 0))
		tt.SetPreferences(ui.Preferences{ReduceMotion: reduced})
		list.ScrollToOffset(0)
		tt.Frame()
		n.Target = &InspectorTarget{ID: targetID}
		for range 5 {
			tt.Frame()
		}
		if scroll.Y != 0 || list.ScrollAnimating() || n.reveal.centerQueued {
			t.Fatalf("reduced=%v empty row queued a scroll before neighbor lookup: y=%v reveal=%+v", reduced, scroll.Y, n.reveal)
		}
		n.CancelReveal()
	}
}

func TestInspectorHiddenExplicitNodeCanUseRankedVisibleNeighbor(t *testing.T) {
	n := &InspectorNavigation{SessionID: "s", Target: &InspectorTarget{ID: "hidden"}}
	m := Model{Inspector: n, Snapshot: session.ViewState{SelectedID: "s", Messages: []session.Message{
		{ID: "hidden", Seq: 10, Kind: session.MessageKindSystemPrompt},
		{ID: "near", Seq: 11, Kind: session.MessageKindText, Role: "user", Body: "visible"},
	}}}
	prepareSessionInspector(m, visibleChatMessages(m.Snapshot.Messages), nil, &ui.ListState{})
	if n.Highlight != "near" {
		t.Fatalf("hidden candidate prevented rendered neighbor fallback: %q", n.Highlight)
	}
}

func TestInspectorNestedCallAndGroupOccurrenceOrder(t *testing.T) {
	messages := []session.Message{
		{ID: "tool:root", Tool: &session.ToolCallView{CallID: "root", Children: []*session.ToolCallView{{CallID: "child"}}}},
		{ID: "answer", SourceMessageID: "m", Kind: session.MessageKindText, Role: "assistant"},
		{ID: "group", Kind: transcriptControlKind},
	}
	got := inspectorRevealOccurrences(messages, []InspectorTarget{{ID: "answer", CallID: "child", RootCallID: "root", GroupKey: "group"}})
	if !reflect.DeepEqual(got, []string{"tool:root", "answer", "group"}) {
		t.Fatalf("call specificity/owner/group order lost: %v", got)
	}
	if got := inspectorMessageIndex(messages, InspectorTarget{ID: "group"}); got != 2 {
		t.Fatalf("explicit group header was retargeted: %d", got)
	}
}
