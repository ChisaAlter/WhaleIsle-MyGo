package chat

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/store/session"
)

// A tool row keeps the transcript out of the source memo, so only the store
// revision lets frames reuse the projection.
func TestProjectChatMessagesReusesStoreRevision(t *testing.T) {
	messages := []session.Message{
		{ID: "u", Seq: 1, Kind: session.MessageKindText, Role: "user", Body: "hi"},
		{ID: "call", Seq: 2, Kind: session.MessageKindTool, Role: "tool", EventType: "tool/call", Raw: json.RawMessage(`{"type":"tool/call","seq":2,"data":{"callId":"c1","name":"read","arguments":"{}"}}`)},
	}
	s := session.New()
	view := s.Snapshot()
	view.Messages = messages
	first := projectChatMessages(view)
	if len(first) == 0 {
		t.Fatal("empty projection")
	}
	if again := projectChatMessages(view); &again[0] != &first[0] {
		t.Fatal("the same revision projected again")
	}
	if a, b := visibleProjectedMessages(view), visibleProjectedMessages(view); len(a) == 0 || &a[0] != &b[0] {
		t.Fatal("the same revision filtered again")
	}
	if want := todoBaselinesFor(view.Messages, view.HistoryHasMore); len(projectedTodoBaselines(view)) != len(want) {
		t.Fatal("todo baselines differ from the direct index")
	}
	shorter := view
	shorter.Messages = messages[:1]
	if got := projectChatMessages(shorter); len(got) == len(first) {
		t.Fatal("replaced messages reused a stale projection")
	}
	s.SetLastAction("changed")
	next := s.Snapshot()
	next.Messages = messages
	if got := projectChatMessages(next); &got[0] == &first[0] {
		t.Fatal("a new revision reused the old projection")
	}
	hand := session.ViewState{Messages: messages}
	if got := projectChatMessages(hand); len(got) != len(first) {
		t.Fatalf("hand-built view projection: %d rows, want %d", len(got), len(first))
	}
}

func TestStreamingTailProjectionParity(t *testing.T) {
	messages := buildStreamingChatMessages(20)
	var viewMemo transcriptViewMemo
	expanded := map[string]bool{}
	for step, chunk := range []string{"alpha", "alpha beta", "alpha beta gamma"} {
		messages[len(messages)-1].Body = chunk
		snap := session.New().Snapshot()
		snap.SelectedID = "parity-session"
		snap.PlanHistoryBinding = "parity-binding"
		snap.Messages = append([]session.Message(nil), messages...)
		gotProj := projectChatMessages(snap)
		wantProj := buildChatMessages(snap)
		if len(gotProj) != len(wantProj) || gotProj[len(gotProj)-1].Body != chunk {
			t.Fatalf("step %d: projected tail = %q (len %d), want %q (len %d)", step, gotProj[len(gotProj)-1].Body, len(gotProj), chunk, len(wantProj))
		}
		msgs, controls, targets, keys := viewMemo.resolve(Model{Snapshot: snap, ThinkingExpanded: &expanded})
		wantMsgs, wantControls := projectTranscript(visibleChatMessages(wantProj), Model{Snapshot: snap, ThinkingExpanded: &expanded})
		wantTargets := assistantActionTargets(wantMsgs)
		wantKeys := transcriptMessageKeys(Model{Snapshot: snap, ThinkingExpanded: &expanded}, wantMsgs)
		if len(msgs) != len(wantMsgs) || msgs[len(msgs)-1].Body != chunk {
			t.Fatalf("step %d: transcript tail = %q (len %d), want %q (len %d)", step, msgs[len(msgs)-1].Body, len(msgs), chunk, len(wantMsgs))
		}
		if len(controls) != len(wantControls) || len(targets) != len(wantTargets) || len(keys) != len(wantKeys) {
			t.Fatalf("step %d: derived counts mismatch", step)
		}
	}
}

func TestTurnChunkedIncrementalProjectionParity(t *testing.T) {
	base := buildStreamingChatMessages(20)
	// Strip the default trailing streaming text row so base ends at turn/end of turn 5.
	base = base[:len(base)-1]

	turn6 := float64(6)
	step1 := float64(1)
	step2 := float64(2)
	user6 := session.Message{
		ID:        "user-21",
		Seq:       210,
		TurnSeq:   600,
		Kind:      session.MessageKindText,
		Role:      "user",
		EventType: "user/message",
		Body:      "Analyze the repository structure",
		Raw:       json.RawMessage(`{"type":"user/message","seq":210,"time":1700000002100,"data":{"turn":6,"content":[{"type":"text","text":"Analyze the repository structure"}]}}`),
	}
	thinkingStream1 := session.Message{
		ID:               "think-stream-6",
		TurnSeq:          600,
		Kind:             session.MessageKindThinking,
		Role:             "assistant",
		Streaming:        true,
		ConversationTurn: &turn6,
		ConversationStep: &step1,
		Body:             "Checking files...",
	}
	thinkingStream2 := thinkingStream1
	thinkingStream2.Body = "Checking files and packages in detail..."

	prepTool1 := session.Message{
		ID:        "tool:call-6",
		TurnSeq:   600,
		Kind:      session.MessageKindTool,
		Role:      "tool",
		Body:      "read_file",
		Args:      `{"path":"main`,
		Streaming: true,
		Tool: &session.ToolCallView{
			CallID:    "call-6",
			Name:      "read_file",
			Arguments: `{"path":"main`,
			State:     "preparing",
		},
	}
	prepTool2 := prepTool1
	prepTool2.Args = `{"path":"main.go"}`
	prepTool2.Tool = &session.ToolCallView{
		CallID:    "call-6",
		Name:      "read_file",
		Arguments: `{"path":"main.go"}`,
		State:     "preparing",
	}

	durableCall := session.Message{
		ID:        "tool-call-22",
		Seq:       221,
		TurnSeq:   600,
		Kind:      session.MessageKindTool,
		Role:      "tool",
		EventType: "tool/call",
		Body:      "read_file",
		Raw:       json.RawMessage(`{"type":"tool/call","seq":221,"time":1700000002201,"data":{"turn":6,"step":1,"callId":"call-6","name":"read_file","arguments":"{\"path\":\"main.go\"}"}}`),
	}
	durableResult := session.Message{
		ID:        "tool-res-22",
		Seq:       222,
		TurnSeq:   600,
		Kind:      session.MessageKindTool,
		Role:      "tool",
		EventType: "tool/result",
		Body:      "package main",
		Raw:       json.RawMessage(`{"type":"tool/result","seq":222,"time":1700000002202,"data":{"turn":6,"step":1,"message":{"toolCallId":"call-6","isError":false,"content":[{"type":"text","text":"package main\n"}]}}}`),
	}
	answerStream := session.Message{
		ID:               "ans-stream-6",
		TurnSeq:          600,
		Kind:             session.MessageKindText,
		Role:             "assistant",
		Streaming:        true,
		ConversationTurn: &turn6,
		ConversationStep: &step2,
		Body:             "Here is the summary.",
	}
	durableAnswer := session.Message{
		ID:        "asst-23",
		Seq:       230,
		TurnSeq:   600,
		Kind:      session.MessageKindText,
		Role:      "assistant",
		EventType: "assistant/message",
		Body:      "Here is the summary.",
		Raw:       json.RawMessage(`{"type":"assistant/message","seq":230,"time":1700000002300,"data":{"turn":6,"step":2,"message":{"id":"msg-23","content":[{"type":"text","text":"Here is the summary."}]}}}`),
	}
	turnEnd6 := session.Message{
		ID:        "turn-end-24",
		Seq:       240,
		TurnSeq:   600,
		Kind:      session.MessageKindText,
		Role:      "status",
		EventType: "turn/end",
		Body:      "95",
		Raw:       json.RawMessage(`{"type":"turn/end","seq":240,"time":1700000002400,"data":{"turn":6}}`),
	}
	user7 := session.Message{
		ID:        "user-25",
		Seq:       250,
		TurnSeq:   700,
		Kind:      session.MessageKindText,
		Role:      "user",
		EventType: "user/message",
		Body:      "Next question in turn 7",
		Raw:       json.RawMessage(`{"type":"user/message","seq":250,"time":1700000002500,"data":{"turn":7,"content":[{"type":"text","text":"Next question in turn 7"}]}}`),
	}

	frames := [][]session.Message{
		append(append([]session.Message(nil), base...), user6),
		append(append([]session.Message(nil), base...), user6, thinkingStream1),
		append(append([]session.Message(nil), base...), user6, thinkingStream2),
		append(append([]session.Message(nil), base...), user6, thinkingStream2, prepTool1),
		append(append([]session.Message(nil), base...), user6, thinkingStream2, prepTool2),
		append(append([]session.Message(nil), base...), user6, durableCall, durableResult),
		append(append([]session.Message(nil), base...), user6, durableCall, durableResult, answerStream),
		append(append([]session.Message(nil), base...), user6, durableCall, durableResult, durableAnswer, turnEnd6),
		append(append([]session.Message(nil), base...), user6, durableCall, durableResult, durableAnswer, turnEnd6, user7),
	}

	var viewMemo transcriptViewMemo
	expanded := map[string]bool{}
	for idx, frameMsgs := range frames {
		snap := session.New().Snapshot()
		snap.SelectedID = "turn-chunked-session"
		snap.PlanHistoryBinding = "turn-chunked-binding"
		snap.Messages = frameMsgs

		gotProj := projectChatMessages(snap)
		wantProj := buildChatMessages(snap)
		if len(gotProj) != len(wantProj) {
			t.Fatalf("frame %d: len(gotProj) = %d, want %d", idx, len(gotProj), len(wantProj))
		}
		for i := range wantProj {
			if gotProj[i].ID != wantProj[i].ID || gotProj[i].Body != wantProj[i].Body || gotProj[i].Kind != wantProj[i].Kind {
				t.Fatalf("frame %d row %d: got (%s,%s,%q), want (%s,%s,%q)",
					idx, i, gotProj[i].ID, gotProj[i].Kind, gotProj[i].Body,
					wantProj[i].ID, wantProj[i].Kind, wantProj[i].Body)
			}
		}

		m := Model{Snapshot: snap, ThinkingExpanded: &expanded}
		msgs, controls, targets, keys := viewMemo.resolve(m)
		wantMsgs, wantControls := projectTranscript(visibleChatMessages(wantProj), m)
		wantTargets := assistantActionTargets(wantMsgs)
		wantKeys := transcriptMessageKeys(m, wantMsgs)

		if len(msgs) != len(wantMsgs) {
			t.Fatalf("frame %d: len(msgs) = %d, want %d", idx, len(msgs), len(wantMsgs))
		}
		for i := range wantMsgs {
			if msgs[i].ID != wantMsgs[i].ID || msgs[i].Body != wantMsgs[i].Body || msgs[i].Kind != wantMsgs[i].Kind {
				t.Fatalf("frame %d transcript row %d: got (%s,%s,%q), want (%s,%s,%q)",
					idx, i, msgs[i].ID, msgs[i].Kind, msgs[i].Body,
					wantMsgs[i].ID, wantMsgs[i].Kind, wantMsgs[i].Body)
			}
		}
		if len(controls) != len(wantControls) {
			t.Fatalf("frame %d: len(controls) = %d, want %d", idx, len(controls), len(wantControls))
		}
		for k, wantC := range wantControls {
			gotC, ok := controls[k]
			if !ok || gotC.label != wantC.label || gotC.open != wantC.open || gotC.running != wantC.running {
				t.Fatalf("frame %d control %q: got %+v, want %+v", idx, k, gotC, wantC)
			}
		}
		if len(targets) != len(wantTargets) {
			t.Fatalf("frame %d: len(targets) = %d, want %d", idx, len(targets), len(wantTargets))
		}
		for seat, wantT := range wantTargets {
			gotT, ok := targets[seat]
			if !ok || gotT.latest != wantT.latest || gotT.endSeq != wantT.endSeq || gotT.message.ID != wantT.message.ID {
				t.Fatalf("frame %d target %d: got %+v, want %+v", idx, seat, gotT, wantT)
			}
		}
		if len(keys) != len(wantKeys) {
			t.Fatalf("frame %d: len(keys) = %d, want %d", idx, len(keys), len(wantKeys))
		}
		for i := range wantKeys {
			if keys[i] != wantKeys[i] {
				t.Fatalf("frame %d key %d: got %+v, want %+v", idx, i, keys[i], wantKeys[i])
			}
		}
	}
}
