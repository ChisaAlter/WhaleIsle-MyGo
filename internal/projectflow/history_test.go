package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecordedReportHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		json.NewDecoder(r.Body).Decode(&q)
		if r.URL.Path != "/dsh-project/"+q.Method || q.Payload["projectId"] != "p" {
			t.Error("wrong route/project")
		}
		report := func(ref string) map[string]any {
			return map[string]any{"delegationRef": ref, "summary": "recorded " + ref, "outcome": "completed", "at": 12, "request": map[string]any{"id": "request", "revision": 2}, "artifacts": []any{map[string]any{"path": "/project/result.txt"}}}
		}
		var value any
		switch q.Method {
		case "store/list":
			if q.Payload["resultsCursor"] == "list-cursor" {
				value = map[string]any{"results": []any{}, "resultsTotal": 1}
			} else {
				value = map[string]any{"results": []any{map[string]any{"workstreamId": "work", "report": report("new"), "reportCount": 2}}, "resultsTotal": 1, "nextResultsCursor": "list-cursor"}
			}
		case "store/results":
			if q.Payload["workstreamId"] != "work" {
				t.Error("work identity lost")
			}
			ref := "new"
			next := "new"
			if q.Payload["cursor"] == "new" || q.Payload["delegationRef"] == "old" {
				ref = "old"
				next = ""
			}
			if q.Payload["delegationRef"] == "missing" {
				ref = "other"
			}
			value = map[string]any{"items": []any{report(ref)}, "total": 2, "nextCursor": next}
		case "open":
			if q.Payload["workstreamId"] != "work" || q.Payload["delegationRef"] != "old" || q.Payload["path"] != "/project/result.txt" {
				t.Error("historical artifact identity lost")
			}
			value = map[string]any{"canonicalPath": "/project/result.txt", "modifiedSinceReport": true}
		default:
			t.Error("invented RPC", q.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	ctx := context.Background()
	c := Client{Origin: srv.URL}
	list, e := c.Reports(ctx, "p", "")
	if e != nil || list.Results[0].Report.DelegationRef != "new" {
		t.Fatal(list, e)
	}
	if _, e = c.Reports(ctx, "p", list.NextCursor); e != nil {
		t.Fatal(e)
	}
	h, e := c.ReportHistory(ctx, "p", "work", "", "")
	if e != nil {
		t.Fatal(e)
	}
	old, e := c.ReportHistory(ctx, "p", "work", h.NextCursor, "")
	if e != nil || old.Items[0].DelegationRef != "old" {
		t.Fatal(old, e)
	}
	exact, e := c.ReportHistory(ctx, "p", "work", "", "old")
	if e != nil || exact.Items[0].Summary != "recorded old" {
		t.Fatal(exact, e)
	}
	file, e := exact.Items[0].File("work", "/project/result.txt")
	if e != nil {
		t.Fatal(e)
	}
	resolved, e := c.ResolveArtifact(ctx, "p", file)
	if e != nil || !resolved.ModifiedSinceReport {
		t.Fatal(resolved, e)
	}
	if _, e = c.ReportHistory(ctx, "p", "work", "", "missing"); e == nil {
		t.Fatal("mismatched exact report accepted")
	}
	if _, e = exact.Items[0].File("work", "/not-reported"); e == nil {
		t.Fatal("unreported file accepted")
	}
}
