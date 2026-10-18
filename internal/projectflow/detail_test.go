package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemberDetailsKeepOriginalIdentity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			RPCID   string
			Payload map[string]any
		}
		json.NewDecoder(r.Body).Decode(&q)
		if r.URL.Path != "/dsh-project/workstreams/get" || q.Payload["projectId"] != "p" || q.Payload["workerSessionId"] != "member" {
			t.Error("wrong member lookup")
		}
		if _, ok := q.Payload["workstreamId"]; ok {
			t.Error("must use exactly one lookup identity")
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"workstream": map[string]any{"id": "w", "projectId": "p", "workerSessionId": "member", "title": "Build", "brief": "Original work"}, "worker": map[string]any{"sessionId": "member", "phase": "failed", "materialized": false}}}})
	}))
	defer srv.Close()
	c := Client{Origin: srv.URL}
	d, err := c.WorkDetail(context.Background(), "p", "", "member")
	if err != nil {
		t.Fatal(err)
	}
	p := Project{ID: "p", CoordinatorSessionID: "lead"}
	draft, err := RetryDraft(p, d)
	if err != nil || !strings.Contains(draft, "w") || !strings.Contains(draft, "member") {
		t.Fatal(draft, err)
	}
	if _, err = d.ProcessAddress(p); err == nil {
		t.Fatal("unmaterialized member must not open a replacement")
	}
	d.Worker.Materialized = true
	addr, err := d.ProcessAddress(p)
	if err != nil || addr["mode"] != "continuable" || addr["parentSessionId"] != "lead" || addr["childSessionId"] != "member" {
		t.Fatal(addr, err)
	}
	d.Workstream.WorkerSessionID = "other"
	if _, err = d.ProcessAddress(p); err == nil {
		t.Fatal("mismatched child accepted")
	}
	snap, err := ParseProcessSnapshot([]byte(`{"type":"snapshot","cursor":7,"records":[{"type":"event","event":{"type":"user/message","seq":4,"data":{"text":"fixture"}}}],"hasMore":true}`))
	if err != nil || snap.Cursor != 7 || len(snap.Records) != 1 || !snap.HasMore {
		t.Fatal(snap, err)
	}
}
