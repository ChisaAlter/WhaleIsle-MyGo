package interaction

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestApprovalRequesterArgumentsAreComplete(t *testing.T) {
	var req approvalRequest
	raw := `{"toolName":"exec_command","callId":"member-call","requester":{"sessionId":"child","label":"Builder","cwd":"/repo","call":{"callId":"member-call","name":"exec_command","arguments":"{\"command\":\"git diff\",\"workdir\":\"src\"}"}}}`
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	got := approvalArgsPreview(req)
	if !strings.Contains(got, "git diff") || !strings.Contains(got, "workdir") {
		t.Fatalf("requester arguments omitted: %q", got)
	}
	long := strings.Repeat("x", 5000)
	rawArgs, _ := json.Marshal(map[string]string{"command": long})
	got = approvalArgsPreview(approvalRequest{Input: rawArgs})
	if !strings.Contains(got, long) {
		t.Fatalf("approval arguments clipped to %d bytes", len(got))
	}
}

func TestApprovalRequesterContextNeverBorrowsParentArguments(t *testing.T) {
	var req approvalRequest
	if err := json.Unmarshal([]byte(`{"callId":"child-call","input":{"command":"parent-secret"},"requester":{"sessionId":"child","label":"Builder","cwd":"/repo/","call":{"callId":"child-call","arguments":"{\"command\":\"child-command\",\"workdir\":\"src\"}"}}}`), &req); err != nil {
		t.Fatal(err)
	}
	if got := approvalArgsPreview(req); !strings.Contains(got, "child-command") || strings.Contains(got, "parent-secret") {
		t.Fatalf("arguments %q", got)
	}
	if got := approvalRequesterContext(req); !strings.Contains(got, "Builder") || !strings.Contains(got, "Session: child") || !strings.Contains(got, "/repo/src") {
		t.Fatalf("context %q", got)
	}
	req.Requester.Call.CallID = "another-call"
	if got := approvalArgsPreview(req); got != "" {
		t.Fatalf("mismatched requester borrowed arguments: %q", got)
	}
	if !strings.Contains(approvalRequesterContext(req), "does not match") {
		t.Fatal("missing identity warning")
	}
}

func TestApprovalWorkingDirectoryIsPortableDisplayOnly(t *testing.T) {
	for _, tc := range []struct{ cwd, workdir, want string }{
		{"/repo/", "src", "/repo/src"}, {"/repo", "/elsewhere", "/elsewhere"},
		{`C:\repo\`, `src`, `C:\repo\src`}, {`C:/repo`, "src", "C:/repo/src"},
		{`\\host\repo\`, `src`, `\\host\repo\src`}, {"/repo", `D:\src`, `D:\src`},
		{"/repo", "../src", "/repo/../src"}, {"", "src", "src"},
	} {
		if got := approvalWorkingDirectory(tc.cwd, tc.workdir); got != tc.want {
			t.Errorf("%q + %q = %q, want %q", tc.cwd, tc.workdir, got, tc.want)
		}
	}
}

func TestApprovalCallsAreSessionScopedStartRecords(t *testing.T) {
	s := New()
	s.pending = []Pending{
		{AgentID: "A", EventID: "root", Event: EventApproval, CallID: "call", approvalCallID: "call"},
		{AgentID: "A", EventID: "nested", Event: EventApproval, approvalCallID: "nested"},
		{AgentID: "B", EventID: "other", Event: EventApproval, approvalCallID: "call"},
		{AgentID: "A", EventID: "delegated", Event: EventApproval, approvalCallID: "call", approvalRequester: true, Args: "child original"},
		{AgentID: "A", EventID: "request-only", Event: EventApproval, CallID: "call"},
	}
	events := []json.RawMessage{
		json.RawMessage(`{"type":"tool/result","data":{"callId":"call","arguments":{"command":"result injection"}}}`),
		json.RawMessage(`{"type":"tool/call","data":{"callId":"call","arguments":"{\"command\":\"root original\"}"}}`),
		json.RawMessage(`{"type":"tool/ptc-dispatch-start","data":{"callId":"call","subCallId":"nested","arguments":{"command":"nested original"}}}`),
		json.RawMessage(`{"type":"tool/ptc-dispatch","data":{"subCallId":"nested","arguments":{"command":"settled replacement"}}}`),
		json.RawMessage(`{"type":"tool/call","data":{"callId":"call","arguments":{"command":"duplicate replacement"}}}`),
	}
	s.SyncApprovalCalls("A", events)
	pending := s.Snapshot().Pending
	byID := make(map[string]Pending, len(pending))
	for _, p := range pending {
		byID[p.EventID] = p
	}
	if len(byID) != 5 || !strings.Contains(byID["root"].Args, "root original") || !strings.Contains(byID["nested"].Args, "nested original") {
		t.Fatalf("missing originals: %+v", pending)
	}
	if byID["other"].Args != "" || byID["delegated"].Args != "child original" || byID["request-only"].Args != "" {
		t.Fatalf("cross-session/requester/requestId leak: %+v", pending)
	}
	s.SyncApprovalCalls("A", []json.RawMessage{json.RawMessage(`{"type":"tool/call","data":{"callId":"call","arguments":{"command":"late replacement"}}}`)})
	var frozen string
	for _, p := range s.Snapshot().Pending {
		if p.EventID == "root" {
			frozen = p.Args
		}
	}
	if !strings.Contains(frozen, "root original") {
		t.Fatalf("frozen original missing or changed: %q", frozen)
	}
	if s.Snapshot().AutoAllow || len(s.responses) != 0 {
		t.Fatal("presentation changed approval policy or decisions")
	}
}

func TestApprovalContextWaterfallWiring(t *testing.T) {
	s := New()
	defer s.Detach()
	done := make(chan *dsh.EventResultOutcome, 1)
	go func() {
		out, _ := s.handleApproval(dsh.EventsWaterfall{AgentID: "A", EventID: "approval", Event: EventApproval, Request: json.RawMessage(`{"toolName":"exec_command","callId":"call"}`)})
		done <- out
	}()
	awaitQuestionFixture(t, func() bool { return len(s.Snapshot().Pending) == 1 })
	s.SyncApprovalCalls("B", []json.RawMessage{json.RawMessage(`{"type":"tool/call","data":{"callId":"call","arguments":{"command":"foreign"}}}`)})
	if s.Snapshot().Pending[0].Args != "" {
		t.Fatal("foreign session borrowed")
	}
	s.SyncApprovalCalls("A", []json.RawMessage{json.RawMessage(`{"type":"tool/call","data":{"callId":"call","arguments":{"command":"expected"}}}`)})
	if !strings.Contains(s.Snapshot().Pending[0].Args, "expected") {
		t.Fatal("real waterfall not correlated")
	}
	s.ResolveAllow("approval")
	select {
	case out := <-done:
		if string(out.Value) != `"allowed-once"` {
			t.Fatalf("outcome %+v", out)
		}
	case <-time.After(time.Second):
		t.Fatal("original request no longer answerable")
	}
}
