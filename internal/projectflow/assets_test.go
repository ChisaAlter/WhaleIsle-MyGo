package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeliveryAssetKeepsReportScope(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		json.NewDecoder(r.Body).Decode(&q)
		if q.Payload["projectId"] != "p" {
			t.Error("wrong project")
		}
		var value any
		switch q.Method {
		case "open":
			if q.Payload["path"] != nil {
				if q.Payload["workstreamId"] != "w" || q.Payload["delegationRef"] != "old-report" || q.Payload["path"] != "/report/result.txt" {
					t.Error("lost report-scoped path")
				}
			}
			value = map[string]any{"canonicalPath": "/checked/result.txt", "modifiedSinceReport": true}
		case "diff":
			if q.Payload["workstreamId"] != "w" {
				t.Error("wrong diff work")
			}
			value = map[string]any{"snapshot": map[string]any{"isGit": true, "diff": "fixture diff", "truncated": true}, "worker": map[string]any{"cwd": "/checked", "mode": "worktree"}}
		default:
			t.Fatal("invented file route")
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	c := Client{Origin: srv.URL}
	ctx := context.Background()
	out, err := c.ResolveArtifact(ctx, "p", Artifact{WorkstreamID: "w", DelegationRef: "old-report", Path: "/report/result.txt"})
	if err != nil || out.CanonicalPath != "/checked/result.txt" || !out.ModifiedSinceReport {
		t.Fatal(out, err)
	}
	if _, err = c.ResolveArtifact(ctx, "p", Artifact{WorkstreamID: "w", Path: "/report/result.txt"}); err == nil || calls != 1 {
		t.Fatal("missing report identity transmitted")
	}
	diff, err := c.WorkDiff(ctx, "p", "w")
	if err != nil || diff.Snapshot.Diff != "fixture diff" || !diff.Snapshot.Truncated {
		t.Fatal(diff, err)
	}
	if err = c.OpenProjectDirectory(ctx, "p"); err != nil {
		t.Fatal(err)
	}
}
