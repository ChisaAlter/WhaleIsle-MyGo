package projectflow

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type deliveryFixtureTransport func(*http.Request) (*http.Response, error)

func (f deliveryFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNativeDeliveryHistoryFixture(t *testing.T) {
	ref := DeliveryReportRef{WorkstreamID: "work-a", DelegationRef: "assignment-old", ReportID: "report-old"}
	newest := HistoricalDelivery{ID: "delivery-new", ProjectID: "project-a", RequestID: "request-a", RequestRevision: 2, Title: "New", Status: "delivered", Outcome: "completed", Applicable: true, Summary: "New body", ReportRefs: []DeliveryReportRef{ref}}
	older := HistoricalDelivery{ID: "delivery-old", ProjectID: "project-a", RequestID: "request-a", RequestRevision: 1, Title: "Old", Status: "historical", Outcome: "partial", Summary: strings.Repeat("Complete historical body\n", 100), Evidence: []string{"one", "two", "third evidence"}, RemainingIssues: []string{"unmet original condition"}, ReportRefs: []DeliveryReportRef{ref}, TerminalRefs: []DeliveryTerminalRef{{WorkstreamID: "work-a", DelegationRef: "assignment-old", RunID: "run-old"}}}
	calls := 0
	mode := "ok"
	lastCursor := ""
	c := Client{Origin: "http://127.0.0.1", Cookie: "fixture-only", HTTP: &http.Client{Transport: deliveryFixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			return nil, err
		}
		if r.URL.Host != "127.0.0.1" || r.URL.Path != "/dsh-project/"+q.Method || q.Payload["projectId"] != "project-a" || r.Header.Get("Cookie") != "fixture-only" {
			t.Fatalf("source scope changed: %s %#v", r.URL, q)
		}
		var value any
		switch q.Method {
		case "store/list":
			if q.Payload["kind"] != "deliveries" || q.Payload["resultsCursor"] != nil {
				t.Fatalf("wrong original query: %#v", q.Payload)
			}
			lastCursor, _ = q.Payload["cursor"].(string)
			page := DeliveryHistoryPage{Items: []HistoricalDelivery{newest, older}, Total: 3, NextCursor: older.ID}
			if lastCursor != "" {
				if lastCursor != older.ID {
					t.Fatal("invented cursor")
				}
				last := older
				last.ID = "delivery-earliest"
				page = DeliveryHistoryPage{Items: []HistoricalDelivery{last}, Total: 3}
			}
			if mode == "wrong-project" {
				page.Items[0].ProjectID = "project-b"
			}
			value = map[string]any{"deliveries": page}
		case "store/results":
			if q.Payload["workstreamId"] != ref.WorkstreamID || q.Payload["delegationRef"] != ref.DelegationRef || q.Payload["cursor"] != nil {
				t.Fatalf("exact historical ref not requested: %#v", q.Payload)
			}
			report := Report{DelegationRef: ref.DelegationRef, CallID: ref.ReportID, Summary: "Complete original report", Evidence: []string{"report evidence"}}
			if mode == "wrong-report" {
				report.CallID = "latest-report"
			}
			if mode == "wrong-assignment" {
				report.DelegationRef = "latest-assignment"
			}
			value = ReportHistory{Items: []Report{report}, Total: 1, Request: json.RawMessage(`{"id":"request-a","revision":1,"goal":"Original goal"}`)}
		default:
			t.Fatalf("unexpected mutation/open/tool RPC: %s", q.Method)
		}
		result := map[string]any{"ok": true, "value": value}
		if mode == "fail" {
			result = map[string]any{"ok": false, "error": map[string]any{"message": "synthetic disconnected"}}
		}
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": result})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})}}
	ctx := context.Background()
	scope := DeliveryScope{Host: "host-a", ProjectID: "project-a", Generation: 4}
	b := NewDeliveryBrowser(scope, "Project A")
	pageReq, err := b.BeginPage("refresh")
	if err != nil {
		t.Fatal(err)
	}
	page, err := c.DeliveryHistory(ctx, pageReq.ProjectID(), pageReq.Cursor())
	if !b.CompletePage(pageReq, page, err) || b.Snapshot().Error != "" {
		t.Fatal("initial history missing")
	}
	if err = b.Select(older.ID); err != nil {
		t.Fatal(err)
	}
	if b.Snapshot().Selected.Summary != older.Summary || len(b.Snapshot().Selected.Evidence) != 3 || b.Snapshot().Selected.RequestRevision != 1 {
		t.Fatal("historical full body/version lost")
	}
	if _, err = b.BeginPage("next"); err == nil {
		t.Fatal("cursor moved behind selected version")
	}
	reportReq, err := b.BeginReport(older.ID, ref)
	if err != nil {
		t.Fatal(err)
	}
	report, err := c.DeliveryReport(ctx, reportReq.ProjectID(), reportReq.Delivery(), reportReq.Reference())
	if !b.CompleteReport(reportReq, report, err) {
		t.Fatal("evidence did not open")
	}
	if b.Snapshot().Report.CallID != ref.ReportID || !strings.Contains(string(b.Snapshot().Report.Request), `"revision":1`) {
		t.Fatal("replaced historical source version")
	}
	// The source-reference triple is immutable; never substitute the latest report.
	mode = "wrong-report"
	reportReq, _ = b.BeginReport(older.ID, ref)
	report, err = c.DeliveryReport(ctx, reportReq.ProjectID(), reportReq.Delivery(), reportReq.Reference())
	b.CompleteReport(reportReq, report, err)
	if err == nil || b.Snapshot().Report.CallID != ref.ReportID || b.Snapshot().Error == "" {
		t.Fatal("wrong report ID accepted or readable body erased")
	}
	mode = "wrong-assignment"
	if _, err = c.DeliveryReport(ctx, "project-a", older, ref); err == nil {
		t.Fatal("wrong assignment accepted")
	}
	mode = "ok"
	before := calls
	badRef := ref
	badRef.ReportID = "unlisted"
	if _, err = c.DeliveryReport(ctx, "project-a", older, badRef); err == nil {
		t.Fatal("unlisted evidence accepted")
	}
	if _, err = c.DeliveryReport(ctx, "project-b", older, ref); err == nil {
		t.Fatal("foreign project accepted")
	}
	if calls != before {
		t.Fatal("invalid evidence contacted transport")
	}
	b.BackReport()
	reportReq, _ = b.BeginReport(older.ID, ref)
	b.BackList()
	if b.CompleteReport(reportReq, Report{CallID: "late"}, nil) {
		t.Fatal("late evidence crossed Back")
	}
	pageReq, _ = b.BeginPage("next")
	mode = "fail"
	page, err = c.DeliveryHistory(ctx, pageReq.ProjectID(), pageReq.Cursor())
	b.CompletePage(pageReq, page, err)
	if len(b.Snapshot().Cursors) != 1 || b.Snapshot().Page.Items[1].ID != older.ID {
		t.Fatal("failed page advanced cursor or erased page")
	}
	mode = "ok"
	pageReq, _ = b.BeginPage("next")
	page, err = c.DeliveryHistory(ctx, pageReq.ProjectID(), pageReq.Cursor())
	b.CompletePage(pageReq, page, err)
	if lastCursor != older.ID || len(b.Snapshot().Cursors) != 2 {
		t.Fatal("Host cursor not committed")
	}
	pageReq, _ = b.BeginPage("previous")
	page, err = c.DeliveryHistory(ctx, pageReq.ProjectID(), pageReq.Cursor())
	b.CompletePage(pageReq, page, err)
	if lastCursor != "" || len(b.Snapshot().Cursors) != 1 {
		t.Fatal("previous page not restored")
	}
	mode = "wrong-project"
	pageReq, _ = b.BeginPage("refresh")
	page, err = c.DeliveryHistory(ctx, pageReq.ProjectID(), pageReq.Cursor())
	b.CompletePage(pageReq, page, err)
	if err == nil || b.Snapshot().Page.Items[0].ProjectID != "project-a" {
		t.Fatal("cross-project history accepted")
	}
	mode = "ok"
	pageReq, _ = b.BeginPage("refresh")
	other := NewDeliveryBrowser(scope, "Other reader")
	otherReq, _ := other.BeginPage("refresh")
	if other.CompletePage(pageReq, DeliveryHistoryPage{Items: []HistoricalDelivery{}}, nil) {
		t.Fatal("old reader completed new reader")
	}
	other.CompletePage(otherReq, DeliveryHistoryPage{Items: []HistoricalDelivery{}}, nil)
	b.Close()
	if b.CompletePage(pageReq, DeliveryHistoryPage{Items: []HistoricalDelivery{}}, nil) {
		t.Fatal("closed reader accepted history")
	}
	// Missing pages, duplicate IDs and cursor loops cannot masquerade as history.
	for _, p := range []DeliveryHistoryPage{{}, {Items: []HistoricalDelivery{older, older}}, {Items: []HistoricalDelivery{older}, NextCursor: "not-the-last-delivery"}} {
		if err := validateDeliveryPage("project-a", "", p); err == nil {
			t.Fatalf("invalid page accepted: %#v", p)
		}
	}
	if err := validateDeliveryPage("project-a", older.ID, DeliveryHistoryPage{Items: []HistoricalDelivery{older}, NextCursor: older.ID}); err == nil {
		t.Fatal("looping history cursor accepted")
	}
	if err := validateDeliveryPage("project-a", "", DeliveryHistoryPage{Items: []HistoricalDelivery{}}); err != nil {
		t.Fatal("empty history rejected", err)
	}
	b = NewDeliveryBrowser(scope, "A")
	pageReq, _ = b.BeginPage("refresh")
	b.CompletePage(pageReq, DeliveryHistoryPage{Items: []HistoricalDelivery{older}}, nil)
	snap := b.Snapshot()
	snap.Page.Items[0].ReportRefs[0].ReportID = "tampered"
	if b.Snapshot().Page.Items[0].ReportRefs[0].ReportID != ref.ReportID {
		t.Fatal("UI aliased authoritative reference")
	}
	if err := b.Select("unknown"); err == nil {
		t.Fatal("unknown delivery selected")
	}
}
