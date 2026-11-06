package office

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
)

type fixtureTransport struct {
	mu          sync.Mutex
	calls       []string
	payloads    []map[string]any
	status      string
	response    map[string]string
	errResponse map[string]string
	gate        <-chan struct{}
	arrived     chan struct{}
}

func (f *fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var q struct {
		RPCID   string         `json:"rpcId"`
		Method  string         `json:"method"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.calls = append(f.calls, r.URL.Path)
	f.payloads = append(f.payloads, q.Payload)
	value := f.status
	if v, ok := f.response[q.Method]; ok {
		value = v
	}
	errValue := f.errResponse[q.Method]
	gate, arrived := f.gate, f.arrived
	f.gate = nil
	f.arrived = nil
	f.mu.Unlock()
	if arrived != nil {
		close(arrived)
	}
	if gate != nil {
		<-gate
	}
	body := `{"type":"server-response","rpcId":"` + q.RPCID + `","result":{"ok":true,"value":` + value + `}}`
	if errValue != "" {
		body = `{"type":"server-response","rpcId":"` + q.RPCID + `","result":{"ok":false,"error":` + errValue + `}}`
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

const configured = `{"schemaVersion":1,"configured":true,"connected":false,"state":"reconnecting","tokenConfigured":true,"config":{"protocolVersion":"office-harness.v1","baseUrl":"https://office.example","deviceId":"device-one","maxConcurrency":1,"heartbeatSeconds":30,"workspaces":{"project":"C:\\project"},"instructionPresets":{}},"health":{"lastHeartbeatAt":"2026-10-15T07:00:00Z","lastEventType":"job.available","reconnects":2,"error":{"code":"office-hook-unavailable","message":"must-not-display-untrusted-secret"}}}`

func setup(t *testing.T) (*Store, *fixtureTransport) {
	t.Helper()
	f := &fixtureTransport{status: configured, response: map[string]string{}, errResponse: map[string]string{}}
	s := &Store{}
	s.Bind(&pluginrpc.Client{BaseURL: "http://127.0.0.1", Cookie: "synthetic-account", HTTPClient: &http.Client{Transport: f}}, 1, nil)
	s.Show(context.Background())
	wait(t, s)
	return s, f
}
func wait(t *testing.T, s *Store) {
	t.Helper()
	deadline := time.Now().Add(time.Second * 3)
	for s.Snapshot().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.Snapshot().Busy {
		t.Fatal("operation stuck")
	}
}
func TestOfficeStatusAndSecretBoundary(t *testing.T) {
	s, f := setup(t)
	v := s.Snapshot()
	if !v.Loaded || v.Status.Connected || !v.Status.Configured || v.Draft.DeviceToken != "" {
		t.Fatal("bad normalized status")
	}
	b, _ := json.Marshal(v)
	if strings.Contains(string(b), "must-not-display") || strings.Contains(string(b), "synthetic-account") {
		t.Fatal("secret/error leak")
	}
	if len(f.calls) != 1 || f.calls[0] != "/office/connection.status" {
		t.Fatal(f.calls)
	}
}
func TestOfficeExplicitSavePreflightAndBlankTokenOmission(t *testing.T) {
	s, f := setup(t)
	v := s.Snapshot()
	d := v.Draft
	d.HeartbeatSeconds = "31"
	if !s.Edit(v.Scope, d) || !s.Review(v.Scope, "configure") {
		t.Fatal("review rejected")
	}
	if len(f.calls) != 1 {
		t.Fatal("review wrote")
	}
	f.response["connector.configure"] = strings.Replace(configured, `"heartbeatSeconds":30`, `"heartbeatSeconds":31`, 1)
	if !s.Confirm(context.Background(), v.Scope) {
		t.Fatal("confirm rejected")
	}
	wait(t, s)
	if len(f.calls) != 3 || f.calls[1] != "/office/connection.status" || f.calls[2] != "/office/connector.configure" {
		t.Fatal(f.calls)
	}
	p := f.payloads[2]
	if len(p) != 6 || p["deviceToken"] != nil || p["expectedRevision"] != nil {
		t.Fatal("invented or blank credential payload")
	}
	v = s.Snapshot()
	if !strings.Contains(v.Notice, "尚未连接") || v.Dirty || v.Status.Connected {
		t.Fatal(v.Notice)
	}
}
func TestOfficeDraftConflictBlocksWrites(t *testing.T) {
	s, f := setup(t)
	v := s.Snapshot()
	d := v.Draft
	d.HeartbeatSeconds = "42"
	s.Edit(v.Scope, d)
	s.Review(v.Scope, "configure")
	f.status = strings.Replace(configured, "device-one", "device-other", 1)
	s.Confirm(context.Background(), v.Scope)
	wait(t, s)
	v = s.Snapshot()
	if len(f.calls) != 2 || !v.Conflict || v.Draft.HeartbeatSeconds != "42" {
		t.Fatal("conflict overwritten")
	}
	if s.Review(v.Scope, "remove") {
		t.Fatal("conflict remained writable")
	}
	s.Discard(v.Scope)
	if s.Snapshot().Draft.DeviceID != "device-other" || s.Snapshot().Conflict {
		t.Fatal("explicit discard failed")
	}
}
func TestOfficeHostEpochDropsLateResultsAndToken(t *testing.T) {
	s, f := setup(t)
	v := s.Snapshot()
	d := v.Draft
	d.DeviceToken = strings.Repeat("x", 40)
	s.Edit(v.Scope, d)
	gate, arrived := make(chan struct{}), make(chan struct{})
	f.gate = gate
	f.arrived = arrived
	s.Refresh(context.Background())
	<-arrived
	s.Bind(nil, 2, nil)
	close(gate)
	wait(t, s)
	v = s.Snapshot()
	if v.Loaded || v.Draft.DeviceToken != "" || v.Host != "" {
		t.Fatal("old host data survived")
	}
	if s.Edit(1, d) || s.Review(1, "configure") {
		t.Fatal("stale UI accepted")
	}
}
func TestOfficeNoFalseTestOrRemoveSuccess(t *testing.T) {
	for _, action := range []string{"test", "remove"} {
		t.Run(action, func(t *testing.T) {
			s, f := setup(t)
			v := s.Snapshot()
			f.response["connector."+action] = configured
			if !s.Review(v.Scope, action) {
				t.Fatal("review")
			}
			s.Confirm(context.Background(), v.Scope)
			wait(t, s)
			v = s.Snapshot()
			if v.Notice != "" || v.Fresh || v.Error == "" {
				t.Fatal("false success")
			}
		})
	}
}
func TestOfficeWriteOnlyAndNoAutomaticRetry(t *testing.T) {
	s, f := setup(t)
	v := s.Snapshot()
	d := v.Draft
	d.DeviceToken = strings.Repeat("fixture-token-", 4)
	s.Edit(v.Scope, d)
	s.Review(v.Scope, "configure")
	f.response["connector.configure"] = `null`
	s.Confirm(context.Background(), v.Scope)
	wait(t, s)
	v = s.Snapshot()
	b, _ := json.Marshal(v)
	if v.Draft.DeviceToken != "" || strings.Contains(string(b), "fixture-token") || len(f.calls) != 3 || v.Fresh {
		t.Fatal("write error leaked or retried")
	}
}
func TestOfficePostSaveRuntimeFailureRequiresRefreshAndReconcilesSavedDraft(t *testing.T) {
	s, f := setup(t)
	v := s.Snapshot()
	d := v.Draft
	d.HeartbeatSeconds = "31"
	if !s.Edit(v.Scope, d) || !s.Review(v.Scope, "configure") {
		t.Fatal("review rejected")
	}
	f.errResponse["connector.configure"] = `{"code":"office-operation-failed","message":"runtime restart failed after config save"}`
	if !s.Confirm(context.Background(), v.Scope) {
		t.Fatal("confirm rejected")
	}
	wait(t, s)
	v = s.Snapshot()
	if v.Fresh || !v.Dirty || v.Draft.HeartbeatSeconds != "31" || !strings.Contains(v.Error, "office-operation-failed") {
		t.Fatalf("post-save failure treated as pre-write refusal: %+v", v)
	}
	f.mu.Lock()
	f.status = strings.Replace(configured, `"heartbeatSeconds":30`, `"heartbeatSeconds":31`, 1)
	f.mu.Unlock()
	s.Refresh(context.Background())
	wait(t, s)
	v = s.Snapshot()
	if !v.Fresh || v.Dirty || v.Conflict || v.Status.Config.HeartbeatSeconds != 31 {
		t.Fatalf("refresh after post-save failure did not reconcile saved draft: %+v", v)
	}
	f.errResponse["connector.test"] = `{"code":"office-hook-unavailable","message":"hook offline"}`
	if !s.Review(v.Scope, "test") || !s.Confirm(context.Background(), v.Scope) {
		t.Fatal("test review/confirm rejected")
	}
	wait(t, s)
	if v = s.Snapshot(); !v.Fresh || !strings.Contains(v.Error, "office-hook-unavailable") {
		t.Fatalf("read-only test probe failure invalidated fresh status: %+v", v)
	}
}
func TestOfficeValidationAndEndpointIdentity(t *testing.T) {
	s, _ := setup(t)
	v := s.Snapshot()
	cases := []Draft{v.Draft, v.Draft, v.Draft, v.Draft}
	cases[0].BaseURL = "http://public.example"
	cases[1].Workspaces = "bad=relative"
	cases[2].HeartbeatSeconds = "9"
	cases[3].BaseURL = "https://new-office.example"
	for _, d := range cases {
		if _, err := configurePayload(d, v.Status); err == nil {
			t.Fatal("unsafe draft accepted")
		}
	}
	d := v.Draft
	d.BaseURL = "http://127.0.0.1:1234/path"
	d.DeviceToken = strings.Repeat("a", 32)
	p, err := configurePayload(d, v.Status)
	if err != nil || p["baseUrl"] != "http://127.0.0.1:1234" {
		t.Fatal("loopback normalization")
	}
}
