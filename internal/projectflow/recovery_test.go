package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func recoveryWork(t *testing.T) WorkDetail {
	t.Helper()
	var work WorkDetail
	err := json.Unmarshal([]byte(`{"workstream":{"id":"w","projectId":"p","title":"Saved work","workerSessionId":"member","currentDelegationRef":"assignment","request":{"id":"request","revision":2},"summaryFailure":{"delegationRef":"assignment","runId":"run","at":20,"error":{"code":"summary_error","message":"summary failed"}},"latestReport":{"delegationRef":"assignment","callId":"report","at":10},"recentSettlements":[{"runId":"run","delegationRefs":["assignment"],"at":12,"stopReason":"completed"}]},"worker":{"sessionId":"member"}}`), &work)
	if err != nil {
		t.Fatal(err)
	}
	return work
}

func recoveryProjectFixture() Project {
	return Project{ID: "p", CoordinatorSessionID: "lead", Lifecycle: "ready"}
}

func recoveryCard() RequestCard {
	return RequestCard{ID: "request", ProjectID: "p", Revision: 2, Title: "Saved request", Goal: "Existing authorized goal", HandoverError: "handover failed", Status: "summaryFailed"}
}

func TestSummaryRecoveryRequiresOriginalSavedEvidence(t *testing.T) {
	p := recoveryProjectFixture()
	selected, current := recoveryWork(t), recoveryWork(t)
	draft, err := SummaryRetryDraft(p, selected, current)
	if err != nil || !strings.Contains(draft, "saved report report") || !strings.Contains(draft, "without rerunning background work") || !strings.Contains(draft, "Original member session: member") {
		t.Fatal(draft, err)
	}
	current.Workstream.LatestReport = nil
	draft, err = SummaryRetryDraft(p, selected, current)
	if err != nil || !strings.Contains(draft, "saved terminal outcome run") {
		t.Fatal("saved terminal without a report must remain recoverable", draft, err)
	}
	cases := []struct {
		name string
		edit func(*WorkDetail)
	}{
		{"foreign project", func(d *WorkDetail) { d.Workstream.ProjectID = "other" }},
		{"replacement member", func(d *WorkDetail) { d.Workstream.WorkerSessionID = "replacement" }},
		{"new delegation", func(d *WorkDetail) { d.Workstream.CurrentDelegationRef = "new" }},
		{"changed failure", func(d *WorkDetail) { d.Workstream.SummaryFailure.At++ }},
		{"missing failure", func(d *WorkDetail) { d.Workstream.SummaryFailure = nil }},
		{"authority error", func(d *WorkDetail) { d.Workstream.Request.AuthorityError = "missing source" }},
		{"missing evidence", func(d *WorkDetail) { d.Workstream.LatestReport = nil; d.Workstream.RecentSettlements = nil }},
		{"foreign terminal", func(d *WorkDetail) {
			d.Workstream.LatestReport = nil
			d.Workstream.RecentSettlements[0].DelegationRefs = []string{"other"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := recoveryWork(t)
			tc.edit(&d)
			if _, err := SummaryRetryDraft(p, selected, d); err == nil {
				t.Fatal("stale or unbound saved result accepted")
			}
		})
	}
	p.Lifecycle = "archived"
	if _, err := SummaryRetryDraft(p, selected, recoveryWork(t)); err == nil {
		t.Fatal("archived Project accepted")
	}
}

func TestHandoverRecoveryRequiresOriginalRequestRevision(t *testing.T) {
	p, selected := recoveryProjectFixture(), recoveryCard()
	draft, err := HandoverRetryDraft(p, selected, selected)
	if err != nil || !strings.Contains(draft, "requestId: request") || !strings.Contains(draft, "Revision: 2") || !strings.Contains(draft, "without rerunning background work") {
		t.Fatal(draft, err)
	}
	cases := []struct {
		name string
		edit func(*RequestCard)
	}{
		{"foreign project", func(c *RequestCard) { c.ProjectID = "other" }},
		{"new revision", func(c *RequestCard) { c.Revision++ }},
		{"new failure", func(c *RequestCard) { c.HandoverError = "different failure" }},
		{"settled handover", func(c *RequestCard) { c.HandoverError = "" }},
		{"completed", func(c *RequestCard) { c.Status = "completed" }},
		{"authority error", func(c *RequestCard) { c.AuthorityError = "missing source" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			current := recoveryCard()
			tc.edit(&current)
			if _, err := HandoverRetryDraft(p, selected, current); err == nil {
				t.Fatal("stale or unauthorized handover accepted")
			}
		})
	}
}

func TestRecoveryPreparationUsesReadOnlyRPCsAndOriginalLead(t *testing.T) {
	project, work, card := recoveryProjectFixture(), recoveryWork(t), recoveryCard()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string
			Payload map[string]any
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Payload["projectId"] != "p" {
			t.Error("wrong Project identity", req.Payload)
		}
		calls = append(calls, r.URL.Path)
		var value any
		switch r.URL.Path {
		case "/dsh-project/detail":
			value = map[string]any{"project": project}
		case "/dsh-project/workstreams/get":
			if req.Payload["workstreamId"] != "w" {
				t.Error("wrong work identity", req.Payload)
			}
			value = work
		case "/dsh-project/requests/get":
			if req.Payload["requestId"] != "request" {
				t.Error("wrong request identity", req.Payload)
			}
			value = map[string]any{"card": card}
		default:
			t.Errorf("recovery must not send a prompt or retry RPC: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	client := Client{Origin: srv.URL}
	if _, err := client.PrepareSummaryRetry(context.Background(), project, work); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(calls, ","); got != "/dsh-project/detail,/dsh-project/workstreams/get,/dsh-project/requests/get" {
		t.Fatal("summary did not refresh its complete saved identity", got)
	}
	calls = nil
	if _, err := client.PrepareHandoverRetry(context.Background(), project, card); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(calls, ","); got != "/dsh-project/detail,/dsh-project/requests/get" {
		t.Fatal("handover did not refresh its saved identity", got)
	}
	selected := project
	project.CoordinatorSessionID = "replacement-lead"
	before := len(calls)
	if _, err := client.PrepareHandoverRetry(context.Background(), selected, card); err == nil || len(calls) != before+1 {
		t.Fatal("replaced Lead was accepted or followed")
	}
	project = selected
	card.Revision++
	if _, err := client.PrepareSummaryRetry(context.Background(), selected, work); err == nil {
		t.Fatal("new request revision was silently used for the old work")
	}
}
