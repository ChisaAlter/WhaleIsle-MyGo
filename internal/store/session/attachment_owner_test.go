package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"whaleisle/internal/dsh"
)

func TestAttachmentReadKeepsSessionHostAndLateWorkerOwners(t *testing.T) {
	SetAttachmentCacheDir(t.TempDir())
	t.Cleanup(func() { SetAttachmentCacheDir("") })
	started, release := make(chan struct{}), make(chan struct{})
	server := func(host string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				RPCID   string `json:"rpcId"`
				Payload struct {
					Args struct {
						Request struct{ SessionID, AttachmentID string } `json:"request"`
					} `json:"args"`
				} `json:"payload"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			if r.URL.Path != "/api/session/attachment" {
				rpcOK(w, body.RPCID, map[string]any{})
				return
			}
			request := body.Payload.Args.Request
			if request.AttachmentID == "delayed" {
				close(started)
				<-release
			}
			rpcOK(w, body.RPCID, map[string]any{"attachment": map[string]any{"mediaType": "image/png"}, "data": base64.StdEncoding.EncodeToString([]byte(host + "/" + request.SessionID + "/" + request.AttachmentID))})
		}))
	}
	a, b := server("host-a"), server("host-b")
	t.Cleanup(a.Close)
	t.Cleanup(b.Close)
	s := New()
	t.Cleanup(s.Detach)
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: a.URL}}, nil)
	s.Select("session-a")
	first := s.CaptureAttachmentRead(s.PlanHistoryBinding(), "session-a")
	get := func(read AttachmentRead, want string) string {
		t.Helper()
		path, err := read.Fetch(context.Background(), "same-opaque-id")
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil || string(raw) != want {
			t.Fatalf("attachment=%q err=%v", raw, err)
		}
		if read.Cached("same-opaque-id") != path {
			t.Fatal("captured cache missed")
		}
		return path
	}
	firstPath := get(first, "host-a/session-a/same-opaque-id")
	done := make(chan error, 1)
	go func() { _, err := first.Fetch(context.Background(), "delayed"); done <- err }()
	<-started
	s.Select("session-b")
	second := s.CaptureAttachmentRead(s.PlanHistoryBinding(), "session-b")
	close(release)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("late worker result=%v", err)
	}
	if second.Cached("delayed") != "" || first.Key == second.Key {
		t.Fatal("old worker entered replacement session cache")
	}
	secondPath := get(second, "host-a/session-b/same-opaque-id")
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: b.URL}}, nil)
	third := s.CaptureAttachmentRead(s.PlanHistoryBinding(), "session-b")
	thirdPath := get(third, "host-b/session-b/same-opaque-id")
	if firstPath == secondPath || secondPath == thirdPath || first.Key == third.Key {
		t.Fatal("Host/Session aliases share an attachment cache")
	}
	if _, err := second.Fetch(context.Background(), "same-opaque-id"); !errors.Is(err, context.Canceled) {
		t.Fatalf("stale cached read accepted=%v", err)
	}
}

func TestAgentAttachmentReadUsesChildAddressAndNavigationToken(t *testing.T) {
	SetAttachmentCacheDir(t.TempDir())
	t.Cleanup(func() { SetAttachmentCacheDir("") })
	requested := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		req := b["payload"].(map[string]any)["args"].(map[string]any)["request"].(map[string]any)
		requested = req["sessionId"].(string)
		rpcOK(w, b["rpcId"], map[string]any{"attachment": map[string]any{"mediaType": "image/png"}, "data": base64.StdEncoding.EncodeToString([]byte("child bytes"))})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "parent"
	s.planHistoryBinding = fmt.Sprintf("%p\x00%s", s.client, s.selectedID)
	address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent", ChildSessionID: "child", Mode: "continuable"}
	s.agentNavigation = agentNavigationState{client: s.client, mux: s.mux, generation: s.followGeneration, binding: s.planHistoryBinding, root: s.selectedID, view: AgentNavigationView{Token: 7, Address: address, SessionID: "child"}}
	read := s.CaptureAgentAttachmentRead(7, address)
	if read.Fetch == nil {
		t.Fatal("current child has no canonical reader")
	}
	if _, err := read.Fetch(context.Background(), "queue-image"); err != nil || requested != "child" {
		t.Fatalf("child queue image borrowed root: session=%q err=%v", requested, err)
	}
	s.agentNavigation.view.Token++
	if _, err := read.Fetch(context.Background(), "queue-image"); !errors.Is(err, context.Canceled) {
		t.Fatalf("old child token accepted cached image: %v", err)
	}
	next := s.CaptureAgentAttachmentRead(8, address)
	if next.Key == read.Key || next.Cached("queue-image") != "" {
		t.Fatal("new child occurrence inherited old image cache")
	}
}
