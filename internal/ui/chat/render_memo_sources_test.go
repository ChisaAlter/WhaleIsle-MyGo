package chat

import (
	"encoding/json"
	"fmt"
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/workflowrun"
)

func TestMentionMemoEqualLengthPathReplacement(t *testing.T) {
	closing := session.Message{ID: "mention-closing", Seq: 20, Role: "assistant", Raw: json.RawMessage(`{"type":"assistant/message","seq":20,"data":{"turn":0}}`)}
	messages := []session.Message{{ID: "mention-files", Seq: 2, Raw: json.RawMessage(`{"type":"deliverables/presented","seq":2,"data":{"turn":0,"files":[{"path":"/a/x.txt"}]}}`)}}
	first := FileMentionResolver(messages, closing)
	if first == nil || first("x.txt") != "/a/x.txt" {
		t.Fatal("invalid file-mention fixture")
	}
	messages[0].Raw = json.RawMessage(`{"type":"deliverables/presented","seq":2,"data":{"turn":0,"files":[{"path":"/b/y.txt"}]}}`)
	current := FileMentionResolver(messages, closing)
	if current == nil || current("y.txt") != "/b/y.txt" || current("x.txt") != "" {
		t.Fatal("equal-length current source still resolved a file from the previous source")
	}
	copy(messages[0].Raw, `{"type":"deliverables/presented","seq":2,"data":{"turn":0,"files":[{"path":"/c/z.txt"}]}}`)
	current = FileMentionResolver(messages, closing)
	if current == nil || current("z.txt") != "/c/z.txt" {
		t.Fatal("mention source identity aliased a mutable input buffer")
	}
	if first("x.txt") != "/a/x.txt" {
		t.Fatal("rebuilding a memo changed an already-returned resolver")
	}
}

func TestChatProjectionMemoMetadataReplacement(t *testing.T) {
	for _, field := range []string{"Streaming", "Kind", "Role", "SourceMessageID", "TurnSeq", "ContextWaking"} {
		t.Run(field, func(t *testing.T) {
			message := session.Message{ID: "projection-metadata", Seq: 21, Kind: session.MessageKindText, Role: "assistant", Body: "stable", Raw: json.RawMessage(`{"data":{}}`)}
			snap := session.ViewState{SelectedID: "projection-" + field, Messages: []session.Message{message}}
			first := projectChatMessages(snap)
			if len(first) != 1 {
				t.Fatal("invalid metadata projection fixture")
			}
			switch field {
			case "Streaming":
				message.Streaming = true
			case "Kind":
				message.Kind = session.MessageKindThinking
			case "Role":
				message.Role = "user"
			case "SourceMessageID":
				message.SourceMessageID = "durable"
			case "TurnSeq":
				message.TurnSeq = 10
			case "ContextWaking":
				message.ContextWaking = true
			}
			snap.Messages = []session.Message{message}
			current := projectChatMessages(snap)
			if len(current) != 1 || current[0].Streaming != message.Streaming || current[0].Kind != message.Kind || current[0].Role != message.Role || current[0].SourceMessageID != message.SourceMessageID || current[0].TurnSeq != message.TurnSeq || current[0].ContextWaking != message.ContextWaking {
				t.Fatalf("%s metadata retained the previous projection: %+v", field, current)
			}
		})
	}
}

var chatProjectionSourceBenchmarkSink []session.Message

func BenchmarkChatProjectionSourceIdentity(b *testing.B) {
	for _, shape := range []string{"text", "tools", "timed-tools", "workflow"} {
		messages := longHistoryLinearFixture(500)
		for i := 1; i < len(messages) && shape != "text"; i += 4 {
			messages[i].Kind = session.MessageKindTool
			messages[i].Role = "tool"
			messages[i].EventType = "tool/call"
			messages[i].Raw = json.RawMessage(fmt.Sprintf(`{"type":"tool/call","seq":%g,"data":{"turn":%d,"step":0,"callId":"call-%d","name":"read","arguments":"{}"}}`, messages[i].Seq, i/4, i/4))
			if shape == "timed-tools" {
				messages[i].TrajectoryTime = &session.TrajectoryTiming{OperationID: fmt.Sprint(i)}
			}
		}
		snap := session.ViewState{SelectedID: "projection-bench-" + shape, Messages: messages}
		if shape == "workflow" {
			snap.WorkflowRuns = []workflowrun.Run{{ID: "bench", Name: "Synthetic workflow", Status: "completed", AnchorSeq: messages[1].Seq, EventSeqs: []float64{messages[1].Seq}}}
		}
		b.Run(shape, func(b *testing.B) {
			projectChatMessages(snap)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				chatProjectionSourceBenchmarkSink = projectChatMessages(snap)
			}
		})
	}
}

func TestChatProjectionMemoOwnsMutableSources(t *testing.T) {
	stamp := 1.0
	message := session.Message{ID: "owned-sources", Seq: 30, Role: "assistant", Kind: session.MessageKindText, Body: "body", Raw: json.RawMessage(`{"data":{"value":"a"}}`), ContentBlock: json.RawMessage(`{"text":"a"}`), TrajectoryTime: &session.TrajectoryTiming{OperationID: "op", StartedAt: &stamp}}
	snap := session.ViewState{SelectedID: "owned-sources", Messages: []session.Message{message}}
	first := projectChatMessages(snap)
	cached := projectChatMessages(snap)
	if &first[0] != &cached[0] {
		t.Fatal("unchanged plain/timed sources did not reuse projection")
	}
	copy(snap.Messages[0].Raw, `{"data":{"value":"b"}}`)
	current := projectChatMessages(snap)
	if &current[0] == &cached[0] {
		t.Fatal("owned raw identity failed to detect an in-place replacement")
	}
	cached = current
	copy(snap.Messages[0].ContentBlock, `{"text":"b"}`)
	current = projectChatMessages(snap)
	if &current[0] == &cached[0] {
		t.Fatal("owned content identity failed to detect an in-place replacement")
	}
	cached = current
	stamp = 2
	current = projectChatMessages(snap)
	if &current[0] == &cached[0] || *current[0].TrajectoryTime.StartedAt != 2 {
		t.Fatal("owned timing identity failed to detect mutable timestamp")
	}
}

func TestChatProjectionMemoComplexMetadataBypassesSafely(t *testing.T) {
	for _, field := range []string{"Tool", "Command", "Retry", "AutomaticCompaction", "Submission", "Web", "ConversationTurn", "ConversationStep", "Workflow"} {
		t.Run(field, func(t *testing.T) {
			msg := session.Message{ID: "complex", Role: "assistant", Kind: session.MessageKindText, Body: "safe", Raw: json.RawMessage(`{}`)}
			number := 1.0
			switch field {
			case "Tool":
				msg.Tool = &session.ToolCallView{}
			case "Command":
				msg.Command = &session.CommandView{}
			case "Retry":
				msg.Retry = &session.RetryStatus{}
			case "AutomaticCompaction":
				msg.AutomaticCompaction = &session.ManualCompactionView{}
			case "Submission":
				msg.Submission = &session.PendingSubmission{}
			case "Web":
				msg.Web = &session.WebResult{}
			case "ConversationTurn":
				msg.ConversationTurn = &number
			case "ConversationStep":
				msg.ConversationStep = &number
			case "Workflow":
				msg.Workflow = &workflowrun.Run{}
			}
			snap := session.ViewState{SelectedID: "complex-" + field, Messages: []session.Message{msg}}
			first, second := projectChatMessages(snap), projectChatMessages(snap)
			if len(first) != 1 || len(second) != 1 || &first[0] == &second[0] {
				t.Fatal("untracked complex metadata reused a projection")
			}
		})
	}
}

func TestMentionMemoTracksClosingState(t *testing.T) {
	messages := []session.Message{{Raw: json.RawMessage(`{"type":"deliverables/presented","seq":1,"data":{"turn":1,"files":[{"path":"/a.txt"}]}}`)}}
	closing := session.Message{Role: "assistant", Raw: json.RawMessage(`{"type":"assistant/message","seq":2,"data":{"turn":1}}`)}
	if FileMentionResolver(messages, closing) == nil {
		t.Fatal("invalid closing-state fixture")
	}
	closing.Streaming = true
	if FileMentionResolver(messages, closing) != nil {
		t.Fatal("streaming answer reused a completed resolver")
	}
	closing.Streaming = false
	closing.Role = "user"
	if FileMentionResolver(messages, closing) != nil {
		t.Fatal("non-assistant answer reused a completed resolver")
	}
	closing.Role = "assistant"
	copy(closing.Raw, `{"type":"assistant/message","seq":2,"data":{"turn":2}}`)
	if FileMentionResolver(messages, closing) != nil {
		t.Fatal("equal-length closing raw reused another turn's resolver")
	}
}

func TestWorkflowProjectionSourceOwnsAllFields(t *testing.T) {
	name := "phase"
	runs := []workflowrun.Run{{ID: "id", Name: "name", Status: "running", AnchorSeq: 1, Phases: []workflowrun.Phase{{Key: "phase", Name: &name, Members: []workflowrun.Member{{Seq: 2, Label: "member", ChildID: "child", Status: "running"}}}}, Events: []json.RawMessage{json.RawMessage(`{"a":1}`)}, EventSeqs: []float64{1}}}
	mutations := []struct {
		name  string
		apply func()
	}{
		{"ID", func() { runs[0].ID = "new" }}, {"Name", func() { runs[0].Name = "new" }}, {"Status", func() { runs[0].Status = "completed" }}, {"Anchor", func() { runs[0].AnchorSeq = 2 }},
		{"PhaseKey", func() { runs[0].Phases[0].Key = "new" }}, {"PhaseName", func() { *runs[0].Phases[0].Name = "new" }}, {"Member", func() { runs[0].Phases[0].Members[0].Status = "completed" }},
		{"EventSeqs", func() { runs[0].EventSeqs[0] = 2 }}, {"Events", func() { copy(runs[0].Events[0], `{"a":2}`) }},
	}
	for _, mutation := range mutations {
		owned := ownWorkflowSources(runs)
		if !sameWorkflowSources(owned, runs) {
			t.Fatal("new owned workflow identity did not match source")
		}
		mutation.apply()
		if sameWorkflowSources(owned, runs) {
			t.Fatalf("owned workflow identity missed %s mutation", mutation.name)
		}
	}
}

func TestChatProjectionMemoWorkflowReplacementAtSameCount(t *testing.T) {
	snap := session.ViewState{SelectedID: "projection-workflow", Messages: []session.Message{{ID: "workflow-boundary", Seq: 22, Raw: json.RawMessage(`{}`)}}, WorkflowRuns: []workflowrun.Run{{ID: "workflow", Name: "old", AnchorSeq: 22}}}
	first := projectChatMessages(snap)
	if len(first) != 1 || first[0].Body != "old" {
		t.Fatal("invalid workflow fixture")
	}
	snap.WorkflowRuns = []workflowrun.Run{{ID: "workflow", Name: "new", AnchorSeq: 22}}
	current := projectChatMessages(snap)
	if len(current) != 1 || current[0].Body != "new" {
		t.Fatal("workflow content changed at the same run count without rebuilding projection")
	}
}
