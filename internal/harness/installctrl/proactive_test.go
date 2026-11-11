package installctrl

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"whaleisle/internal/desktop/proactive"
)

func TestProactiveControlRoundTripAndUserDecision(t *testing.T) {
	dir := t.TempDir()
	service, err := proactive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(Options{Desktop: DesktopOps{Proactive: service.Dispatch}})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()
	post := func(body string, auth bool, status int, out any) {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, srv.URL()+"/desktop/proactive", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		if auth {
			req.Header.Set("Authorization", "Bearer "+srv.Token())
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != status {
			t.Fatalf("status=%d want=%d", res.StatusCode, status)
		}
		if out != nil {
			if err := json.NewDecoder(res.Body).Decode(out); err != nil {
				t.Fatal(err)
			}
		}
	}
	post(`{"action":"task/create","payload":{"title":"不能匿名写入"}}`, false, 401, nil)
	if len(service.Snapshot().Tasks) != 0 {
		t.Fatal("unauthorized mutation")
	}
	var task proactive.Task
	post(`{"action":"task/create","payload":{"title":"核对发布说明","autonomy":2}}`, true, 200, &task)
	var approval proactive.Approval
	body, _ := json.Marshal(map[string]any{"action": "approval/request", "payload": map[string]any{"taskId": task.ID, "action": "发布核对结果", "target": "named-channel"}})
	post(string(body), true, 200, &approval)
	post(`{"action":"approval/decide","payload":{"approvalId":"`+approval.ID+`","decision":"approved"}}`, true, 403, nil)
	if _, err := service.Decide(approval.ID, "approved", "UI decision"); err != nil {
		t.Fatal(err)
	}
	var snapshot proactive.Snapshot
	post(`{"action":"snapshot","payload": {}}`, true, 200, &snapshot)
	if len(snapshot.Tasks) != 1 || snapshot.Tasks[0].State != "resumed" || snapshot.Outbox[0].Status != "approved" {
		t.Fatal(snapshot)
	}
	post(`{"action":"task/create","payload":[]}`, true, 400, nil)
	post(`{"payload":{}}`, true, 400, nil)
	large := strings.Repeat("file edit line\n", 12000)
	largeBody, _ := json.Marshal(map[string]any{"action": "task/create", "payload": map[string]any{"title": "large tool preparation", "description": large}})
	var largeTask proactive.Task
	post(string(largeBody), true, 200, &largeTask)
	if largeTask.Description != large {
		t.Fatal("proactive content was truncated at old install envelope limit")
	}
	post(`{"action":"snapshot"} {"action":"task/create"}`, true, 400, nil)
	reopened, err := proactive.Open(dir)
	if err != nil || reopened.Snapshot().Outbox[0].Note != "UI decision" {
		t.Fatal(err)
	}
}
