package projectflow

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
)

type activityFixtureTransport func(*http.Request) (*http.Response, error)

func (f activityFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func activityTime(v float64) *float64                                                { return &v }
func TestNativeRequestActivityFixture(t *testing.T) {
	fullGoal := strings.Repeat("Original requirement body\n", 30)
	card := RequestActivityCard{ID: "request-a", ProjectID: "project-a", Revision: 2, Title: "Request A", Versions: []ActivityVersion{{Revision: 1, Title: "Original", Goal: fullGoal, At: activityTime(100)}, {Revision: 2, Title: "Revised", Goal: "Complete revised body", At: activityTime(200)}}, Work: []ActivityWork{{WorkstreamID: "work-a", DelegationRef: "assignment-a", RunID: "run-a", Title: "Work A", MemberName: "Member A", WorkerSessionID: "child-a", Status: "completed", Phase: "settled", AssignmentAt: activityTime(300), ReportAt: activityTime(400), TerminalAt: activityTime(500), ReportID: "report-a", Summary: "Complete work report body", TerminalStopReason: "completed"}}, Delivery: &ActivityDelivery{ID: "delivery-a", Summary: "Complete delivery body", Outcome: "completed", Applicable: true, At: activityTime(600), Evidence: []string{"first", "second", "third"}, ReportRefs: []DeliveryReportRef{{WorkstreamID: "work-a", DelegationRef: "assignment-a", ReportID: "report-a"}}}}
	mode := "ok"
	calls := 0
	c := Client{Origin: "http://127.0.0.1", HTTP: &http.Client{Transport: activityFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			return nil, err
		}
		if r.URL.Path != "/dsh-project/"+q.Method || q.Payload["projectId"] != "project-a" {
			t.Fatalf("wrong source: %#v", q)
		}
		var value any
		switch q.Method {
		case "requests/get":
			if q.Payload["requestId"] != "request-a" || q.Payload["cursor"] != nil || len(q.Payload) != 2 {
				t.Fatalf("invented timeline query: %#v", q.Payload)
			}
			out := card
			if mode == "wrong-request" {
				out.ID = "request-b"
			}
			if mode == "wrong-project" {
				out.ProjectID = "project-b"
			}
			value = map[string]any{"card": out}
		case "store/results":
			if q.Payload["workstreamId"] != "work-a" || q.Payload["delegationRef"] != "assignment-a" || q.Payload["cursor"] != nil {
				t.Fatalf("not exact report provenance: %#v", q.Payload)
			}
			report := Report{DelegationRef: "assignment-a", CallID: "report-a", Summary: "Full source report", Evidence: []string{"source evidence"}}
			if mode == "wrong-report" {
				report.CallID = "report-latest"
			}
			value = ReportHistory{Items: []Report{report}, Request: json.RawMessage(`{"id":"request-a","revision":1,"goal":"Original"}`)}
		default:
			t.Fatalf("unexpected timeline endpoint or mutation: %s", q.Method)
		}
		result := map[string]any{"ok": true, "value": value}
		if mode == "fail" {
			result = map[string]any{"ok": false, "error": map[string]any{"message": "synthetic disconnected"}}
		}
		raw, _ := json.Marshal(map[string]any{"result": result, "rpcId": q.RPCID, "type": "server-response"})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})}}
	ctx := context.Background()
	scope := ActivityScope{Host: "host-a", ProjectID: "project-a", RequestID: "request-a", Generation: 5}
	b := NewActivityBrowser(scope)
	req, err := b.BeginRefresh()
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.RequestActivity(ctx, scope.ProjectID, scope.RequestID)
	if !b.CompleteRefresh(req, got, err) {
		t.Fatal("timeline not applied")
	}
	snap := b.Snapshot()
	if len(snap.Events) != 6 || snap.Events[0].Kind != "delivery" || snap.Events[5].Version.Goal != fullGoal {
		t.Fatal("real chronological events/full body missing")
	}
	if snap.Events[1].Work.RunID != "run-a" || snap.Events[2].Work.ReportID != "report-a" {
		t.Fatal("run/report identity dropped")
	}
	if err = b.Filter("version"); err != nil {
		t.Fatal(err)
	}
	if len(b.Snapshot().Visible) != 2 {
		t.Fatal("type filter did not filter events")
	}
	versionKey := b.Snapshot().Visible[1].Key
	if err = b.Select(versionKey); err != nil {
		t.Fatal(err)
	}
	if b.Snapshot().Selected.Version.Goal != fullGoal || b.Snapshot().Selected.Revision != 1 {
		t.Fatal("old version replaced with current goal")
	}
	b.BackList()
	if b.Snapshot().Filter != "version" {
		t.Fatal("Back lost filter")
	}
	_ = b.Filter("report")
	event := b.Snapshot().Visible[0]
	if err = b.Select(event.Key); err != nil {
		t.Fatal(err)
	}
	ref := ActivityEventReports(event)[0]
	req, err = b.BeginReport(event.Key, ref)
	if err != nil {
		t.Fatal(err)
	}
	report, err := c.ActivityReport(ctx, req.Event(), req.Reference())
	b.CompleteReport(req, report, err)
	if b.Snapshot().Report.CallID != "report-a" || !strings.Contains(string(b.Snapshot().Report.Request), `"revision":1`) {
		t.Fatal("source report or original version missing")
	}
	mode = "wrong-report"
	req, _ = b.BeginReport(event.Key, ref)
	report, err = c.ActivityReport(ctx, req.Event(), req.Reference())
	b.CompleteReport(req, report, err)
	if err == nil || b.Snapshot().Report.CallID != "report-a" {
		t.Fatal("latest report substituted or old body erased")
	}
	mode = "ok"
	before := calls
	bad := ref
	bad.ReportID = "foreign"
	if _, err = c.ActivityReport(ctx, event, bad); err == nil {
		t.Fatal("unlisted reference allowed")
	}
	if calls != before {
		t.Fatal("invalid reference reached transport")
	}
	b.BackReport()
	req, _ = b.BeginReport(event.Key, ref)
	b.BackList()
	if b.CompleteReport(req, Report{CallID: "late"}, nil) {
		t.Fatal("late report crossed Back")
	}
	_ = b.Filter("all")
	req, _ = b.BeginRefresh()
	mode = "fail"
	got, err = c.RequestActivity(ctx, scope.ProjectID, scope.RequestID)
	b.CompleteRefresh(req, got, err)
	if len(b.Snapshot().Events) != 6 || b.Snapshot().Error == "" {
		t.Fatal("failed refresh erased old timeline")
	}
	for _, m := range []string{"wrong-request", "wrong-project"} {
		mode = m
		if _, err = c.RequestActivity(ctx, scope.ProjectID, scope.RequestID); err == nil {
			t.Fatalf("%s accepted", m)
		}
	}
	mode = "ok"
	req, _ = b.BeginRefresh()
	other := NewActivityBrowser(scope)
	otherReq, _ := other.BeginRefresh()
	if other.CompleteRefresh(req, card, nil) {
		t.Fatal("cross-reader completion accepted")
	}
	other.CompleteRefresh(otherReq, card, nil)
	b.Close()
	if b.CompleteRefresh(req, card, nil) {
		t.Fatal("closed timeline accepted result")
	}
	// Local pagination does not send invented cursors or fabricate old runs.
	many := card
	many.Work = []ActivityWork{}
	many.Delivery = nil
	many.Versions = make([]ActivityVersion, 45)
	for i := range many.Versions {
		many.Versions[i] = ActivityVersion{Revision: i + 1, Title: "version", Goal: fullGoal, At: activityTime(float64(i + 1))}
	}
	many.Revision = 45
	b = NewActivityBrowser(scope)
	req, _ = b.BeginRefresh()
	b.CompleteRefresh(req, many, nil)
	if len(b.Snapshot().Visible) != ActivityPageSize || b.Snapshot().Page != 0 {
		t.Fatal("first local page incorrect")
	}
	if err = b.MovePage(1); err != nil {
		t.Fatal(err)
	}
	if b.Snapshot().Visible[0].Revision != 25 {
		t.Fatal("local page order changed")
	}
	if err = b.Select(b.Snapshot().Visible[0].Key); err != nil {
		t.Fatal(err)
	}
	b.BackList()
	if b.Snapshot().Page != 1 {
		t.Fatal("Back lost page")
	}
	if err = b.MovePage(-1); err != nil {
		t.Fatal(err)
	}
	if err = b.Filter("unknown"); err == nil {
		t.Fatal("unknown type accepted")
	}
	if err = b.Select("not-an-event"); err == nil {
		t.Fatal("unowned event accepted")
	}
	broken := card
	broken.Versions = append([]ActivityVersion(nil), card.Versions...)
	broken.Versions[0].At = nil
	broken.Versions[1].At = activityTime(math.Inf(1))
	events, err := ProjectActivityEvents(broken)
	if err != nil || len(events) != 4 {
		t.Fatal("missing/invalid timestamps fabricated events", err)
	}
	dup := card
	dup.Versions = []ActivityVersion{card.Versions[0], card.Versions[0]}
	if _, err = ProjectActivityEvents(dup); err == nil {
		t.Fatal("ambiguous version identity accepted")
	}
	snap = b.Snapshot()
	snap.Visible[0].Version.Goal = "tampered"
	if b.Snapshot().Visible[0].Version.Goal == "tampered" {
		t.Fatal("UI aliased authoritative body")
	}
}
