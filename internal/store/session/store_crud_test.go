package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func rpcOK(w http.ResponseWriter, rpcID any, value any) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":  "server-response",
		"rpcId": rpcID,
		"result": map[string]any{
			"ok":    true,
			"value": value,
		},
	})
}

func wait(t *testing.T, done <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func TestCreateSessionSelectsNewID(t *testing.T) {
	var listHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/create"):
			rpcOK(w, body["rpcId"], map[string]any{"sessionId": "new-sess"})
		case strings.HasSuffix(r.URL.Path, "/session/list"):
			listHits.Add(1)
			rpcOK(w, body["rpcId"], map[string]any{
				"items": []map[string]any{{
					"sessionId":      "new-sess",
					"updatedAt":      2.0,
					"running":        false,
					"blank":          true,
					"agentAvailable": true,
				}},
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)
	st.CreateSession(context.Background())

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snap := st.Snapshot()
		if snap.SelectedID == "new-sess" {
			if listHits.Load() == 0 {
				t.Fatal("expected list refresh after create")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("selected = %q last = %q", st.Snapshot().SelectedID, st.Snapshot().LastAction)
}

func TestRenameSessionUpdatesTitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasSuffix(r.URL.Path, "/session/rename") {
			rpcOK(w, body["rpcId"], map[string]any{})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{"title": "Renamed", "seq": 1.0})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.mu.Lock()
	st.items = []Item{{SessionID: "s1", Title: "Old"}}
	st.mu.Unlock()

	done := make(chan struct{})
	st.Attach(client, func() {
		snap := st.Snapshot()
		if len(snap.Items) == 1 && snap.Items[0].Title == "Renamed" {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	st.RenameSession(context.Background(), "s1", "Renamed")
	wait(t, done)
}

func TestCreateSessionReusesBlank(t *testing.T) {
	st := New()
	st.mu.Lock()
	st.cwd = `C:\ws`
	st.items = []Item{
		{SessionID: "blank-1", Title: "Blank", Cwd: `C:\ws`, Blank: true},
		{SessionID: "other", Title: "Busy", Cwd: `C:\ws`},
	}
	st.mu.Unlock()
	st.CreateSession(context.Background())
	if st.Snapshot().SelectedID != "blank-1" {
		t.Fatalf("selected = %q", st.Snapshot().SelectedID)
	}
	if st.Snapshot().LastAction != "blank session reused" {
		t.Fatalf("action = %q", st.Snapshot().LastAction)
	}
}

func TestCancelSessionAccepted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/cancel") {
			rpcOK(w, body["rpcId"], map[string]any{"accepted": true})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	done := make(chan struct{})
	st.Attach(client, func() {
		if st.Snapshot().LastAction == "cancel accepted" {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	st.CancelSession(context.Background(), "child-1")
	wait(t, done)
}

func TestOpenWorkspacePathAccepted(t *testing.T) {
	var gotPath, gotAction string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/canOpenWorkspacePath"):
			rpcOK(w, body["rpcId"], true)
		case strings.HasSuffix(r.URL.Path, "/session/openWorkspacePath"):
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				gotPath, _ = req["path"].(string)
				gotAction, _ = req["action"].(string)
			}
			rpcOK(w, body["rpcId"], map[string]any{"opened": true})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	done := make(chan struct{})
	st.Attach(client, func() {
		if st.Snapshot().LastAction == "workspace path opened" {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	st.OpenWorkspacePath(context.Background(), `C:\ws\a.md`, "reveal", "")
	wait(t, done)
	if gotPath != `C:\ws\a.md` || gotAction != "reveal" {
		t.Fatalf("path/action = %q/%q", gotPath, gotAction)
	}
}

func TestForkSessionSelectsNewID(t *testing.T) {
	var listHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/fork"):
			rpcOK(w, body["rpcId"], map[string]any{"sessionId": "forked-1"})
		case strings.HasSuffix(r.URL.Path, "/session/list"):
			listHits.Add(1)
			rpcOK(w, body["rpcId"], map[string]any{
				"items": []map[string]any{{
					"sessionId": "forked-1", "updatedAt": 1.0,
					"running": false, "blank": false, "agentAvailable": true,
				}},
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	done := make(chan struct{})
	st.Attach(client, func() {
		if st.Snapshot().SelectedID == "forked-1" && st.Snapshot().LastAction == "session forked" {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	st.ForkSession(context.Background(), "root-1")
	wait(t, done)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if listHits.Load() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected list refresh after fork")
}

func TestPromptChildAccepted(t *testing.T) {
	var gotParent, gotChild, gotMode, gotDelivery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/subagents/prompt") {
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				gotParent, _ = req["parentSessionId"].(string)
				gotChild, _ = req["childSessionId"].(string)
				gotMode, _ = req["mode"].(string)
				gotDelivery, _ = req["delivery"].(string)
			}
			rpcOK(w, body["rpcId"], map[string]any{"messageId": "m1"})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	done := make(chan struct{})
	st.Attach(client, func() {
		if st.Snapshot().LastAction == "subagent prompt sent" {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	st.PromptChild(context.Background(), "root", "child-1", "hello sub")
	wait(t, done)
	if gotParent != "root" || gotChild != "child-1" {
		t.Fatalf("parent/child = %q/%q", gotParent, gotChild)
	}
	if gotMode != "queue" || gotDelivery != "session" {
		t.Fatalf("mode/delivery = %q/%q", gotMode, gotDelivery)
	}
}

func TestArchiveSessionRefreshesList(t *testing.T) {
	var listHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/workspace/archiveSession"):
			rpcOK(w, body["rpcId"], map[string]any{"archivedSessionIds": []string{"s1"}})
		case strings.HasSuffix(r.URL.Path, "/session/list"):
			listHits.Add(1)
			rpcOK(w, body["rpcId"], map[string]any{"items": []map[string]any{}})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.mu.Lock()
	st.selectedID = "s1"
	st.mu.Unlock()

	done := make(chan struct{})
	st.Attach(client, func() {
		if st.Snapshot().LastAction == "session archived" && listHits.Load() > 0 {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	st.ArchiveSession(context.Background(), "s1")
	wait(t, done)
	if st.Snapshot().SelectedID != "" {
		t.Fatalf("selected = %q", st.Snapshot().SelectedID)
	}
}
