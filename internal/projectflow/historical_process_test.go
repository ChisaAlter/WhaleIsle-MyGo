package projectflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type historicalFixtureTransport func(*http.Request) (*http.Response, error)

func (f historicalFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func historicalEvent(seq float64, kind, data string) ProcessRecord {
	r := ProcessRecord{Type: "event"}
	r.Event.Seq = seq
	r.Event.Type = kind
	r.Event.Data = json.RawMessage(data)
	return r
}
func TestNativeHistoricalAssignmentFixture(t *testing.T) {
	ctx := context.Background()
	mode := "ok"
	calls := 0
	report := Report{DelegationRef: "old-ref", CallID: "report-1", Summary: strings.Repeat("complete original report\n", 100), Request: json.RawMessage(`{"id":"request-1","revision":1}`)}
	target := HistoricalTarget{ProjectID: "project-1", WorkstreamID: "work-1", DelegationRef: "old-ref", ReportID: "report-1", RunID: "old-run", WorkerSessionID: "child-1", Report: report}
	c := Client{Origin: "https://synthetic.invalid", HTTP: &http.Client{Transport: historicalFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			return nil, err
		}
		if q.Payload["projectId"] != "project-1" {
			t.Fatalf("wrong Project: %#v", q)
		}
		var value any
		switch q.Method {
		case "workstreams/get":
			if len(q.Payload) != 2 || q.Payload["workstreamId"] != "work-1" {
				t.Fatalf("invented assignment query: %#v", q)
			}
			ref, run, revision, project := "old-ref", "old-run", 1, "project-1"
			if mode == "missing" {
				ref = "latest-ref"
			}
			if mode == "run" {
				run = "new-run"
			}
			if mode == "revision" {
				revision = 2
			}
			if mode == "project" {
				project = "other"
			}
			value = map[string]any{"workstream": map[string]any{"id": "work-1", "projectId": project, "title": "current title", "brief": "NEW BRIEF MUST NOT REPLACE OLD", "workerSessionId": "child-1", "currentDelegationRef": "latest-ref", "delegations": []any{map[string]any{"ref": ref, "messageId": "old-message", "runId": run, "requestId": "request-1", "requestRevision": revision, "scope": "readonly", "phase": "accepted", "createdAt": 50}, map[string]any{"ref": "new-ref", "messageId": "new-message", "runId": "new-run", "requestId": "request-1", "requestRevision": 2}}}, "worker": map[string]any{"sessionId": "child-1", "projectId": "project-1", "workstreamId": "work-1", "materialized": true}}
		case "detail":
			if len(q.Payload) != 1 {
				t.Fatal("wrong detail query")
			}
			value = map[string]any{"project": map[string]any{"id": "project-1", "coordinatorSessionId": "lead-1"}}
		default:
			t.Fatalf("unexpected endpoint / implicit follow: %s", q.Method)
		}
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})}}
	binding, err := c.HistoricalAssignment(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || binding.Delegation.Ref != "old-ref" || binding.Delegation.RequestRevision != 1 || binding.Delegation.RunID != "old-run" || binding.Target.Report.Summary != report.Summary {
		t.Fatal("historical identity/body replaced")
	}
	address, err := binding.Address()
	if err != nil || address["childSessionId"] != "child-1" || address["parentSessionId"] != "lead-1" || address["mode"] != "continuable" {
		t.Fatalf("wrong original address %#v %v", address, err)
	}
	for _, m := range []string{"missing", "run", "revision", "project"} {
		mode = m
		if _, err := c.HistoricalAssignment(ctx, target); err == nil {
			t.Fatalf("accepted %s substitution", m)
		}
	}
	mode = "ok"
	full := strings.Repeat("full historical tool body ", 1000)
	first := ProcessSnapshot{Type: "snapshot", Cursor: 30, HasMore: true, Records: []ProcessRecord{historicalEvent(20, "turn/start", `{}`), historicalEvent(21, "user/message", `{"id":"new-message","source":{"kind":"team-message","senderId":"lead-1"}}`), historicalEvent(22, "assistant/message", `{"text":"new assignment"}`)}}
	pages := 0
	window, err := LocateHistoricalAssignment(ctx, binding, first, func(ctx context.Context, through, before float64) (ProcessSnapshot, error) {
		pages++
		if through != 30 || before != 20 {
			t.Fatalf("wrong frozen cursor %v %v", through, before)
		}
		raw, _ := json.Marshal(map[string]any{"text": full})
		return ProcessSnapshot{Type: "snapshot", Cursor: 30, HasMore: false, Records: []ProcessRecord{historicalEvent(10, "turn/start", `{}`), historicalEvent(11, "user/message", `{"id":"saved","source":{"messageId":"old-message","kind":"team-message","senderId":"lead-1"}}`), historicalEvent(12, "tool/result", string(raw)), historicalEvent(13, "turn/end", `{}`)}}, nil
	})
	if err != nil || pages != 1 || len(window.Records) != 4 || window.Records[0].Event.Seq != 10 || window.Records[3].Event.Seq != 13 || !window.NextAssignmentBoundary || !strings.Contains(string(window.Records[2].Event.Data), full) {
		t.Fatalf("wrong complete historical slice %#v %v", window, err)
	}
	if _, err = LocateHistoricalAssignment(ctx, binding, first, func(context.Context, float64, float64) (ProcessSnapshot, error) { return first, nil }); err == nil {
		t.Fatal("nonprogress page accepted")
	}
	missing := first
	missing.HasMore = false
	if _, err = LocateHistoricalAssignment(ctx, binding, missing, nil); err == nil {
		t.Fatal("latest window substituted for missing old message")
	}
	bad := binding
	bad.Delegation.MessageID = ""
	if _, err = bad.Address(); err == nil {
		t.Fatal("missing original message accepted")
	}
	bad = binding
	bad.WorkerSessionID = "another"
	if _, err = bad.Address(); err == nil {
		t.Fatal("rebound member accepted")
	}
	b := NewHistoricalBrowser("host-a", 4, target)
	req, err := b.Begin("record")
	if err != nil {
		t.Fatal(err)
	}
	if !b.CompleteRecord(req, binding, nil) {
		t.Fatal("record completion rejected")
	}
	req, err = b.Begin("process")
	if err != nil {
		t.Fatal(err)
	}
	if !b.CompleteProcess(req, window, nil) {
		t.Fatal("process completion rejected")
	}
	req, _ = b.Begin("process")
	if !b.CompleteProcess(req, HistoricalWindow{}, fmt.Errorf("synthetic failure")) || len(b.Snapshot().Window.Records) != 4 {
		t.Fatal("failed reload erased readable original")
	}
	snap := b.Snapshot()
	snap.Window.Records[0].Event.Data[0] = 'x'
	if b.Snapshot().Window.Records[0].Event.Data[0] == 'x' {
		t.Fatal("snapshot aliases process records")
	}
	snap.Target.Report.Request[0] = 'x'
	if b.Snapshot().Target.Report.Request[0] == 'x' {
		t.Fatal("snapshot aliases report provenance")
	}
	req, _ = b.Begin("record")
	changed := binding
	changed.Delegation.MessageID = "latest-message"
	if !b.CompleteRecord(req, changed, nil) || b.Snapshot().Binding.Delegation.MessageID != "old-message" {
		t.Fatal("receipt reload rebound original message")
	}
	req, _ = b.Begin("process")
	wrong := window
	wrong.Target.ProjectID = "another"
	if !b.CompleteProcess(req, wrong, nil) || b.Snapshot().Window.Target.ProjectID != "project-1" {
		t.Fatal("wrong process source replaced old body")
	}
	req, _ = b.Begin("process")
	other := NewHistoricalBrowser("host-b", 4, target)
	if other.CompleteProcess(req, window, nil) {
		t.Fatal("cross-host reader accepted old response")
	}
	b.Close()
	if b.CompleteProcess(req, window, nil) {
		t.Fatal("closed reader accepted late process")
	}
	if _, err = b.Begin("process"); err == nil {
		t.Fatal("closed reader reopened")
	}
}
