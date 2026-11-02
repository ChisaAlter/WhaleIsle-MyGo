package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func batchAwaitIdle(t *testing.T, s *Store) SessionBatch {
	t.Helper()
	until := time.Now().Add(3 * time.Second)
	for time.Now().Before(until) {
		b := s.SessionBatch()
		if !b.Busy {
			return b
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("batch did not settle")
	return SessionBatch{}
}
func TestSessionBatchRenderedScopeClearsChoicesAndUnconfirmedTargets(t *testing.T) {
	base := NewSessionBatchScope(true, "workspace", "default", "query")
	for _, scope := range []SessionBatchScope{NewSessionBatchScope(false, "workspace", "default", "query"), NewSessionBatchScope(true, "flat", "default", "query"), NewSessionBatchScope(true, "workspace", "only", "query"), NewSessionBatchScope(true, "workspace", "default", "other")} {
		s := New()
		s.client = &dsh.Client{}
		s.SetSessionBatchScope(base)
		s.EditSessionBatch("toggle", nil)
		s.EditSessionBatch("select", []SessionBatchTarget{{"a", "A"}})
		s.RequestSessionBatch("delete")
		token := s.SessionBatch().Token
		s.SetSessionBatchScope(scope)
		b := s.SessionBatch()
		if !b.Enabled || len(b.Selected) != 0 || b.Confirming || len(b.Targets) != 0 || b.Token == token {
			t.Fatalf("new scope retained old selection: %+v", b)
		}
		s.ConfirmSessionBatch(context.Background(), token)
		if s.SessionBatch().Busy {
			t.Fatal("old confirmation token accepted")
		}
	}
	same := NewSessionBatchScope(true, "", "", " \x00query\t")
	if same != base {
		t.Fatalf("normalized scope=%+v want%+v", same, base)
	}
	s := New()
	s.SetSessionBatchScope(base)
	s.EditSessionBatch("toggle", nil)
	s.EditSessionBatch("select", []SessionBatchTarget{{"a", "A"}})
	s.SetSessionBatchScope(same)
	if len(s.SessionBatch().Selected) != 1 {
		t.Fatal("equivalent scope cleared choices")
	}
	if got := NewSessionBatchScope(true, "workspace", "default", strings.Repeat("a", 499)+"😀x").Query; got != strings.Repeat("a", 499) {
		t.Fatal("UTF16 scope truncation splits surrogate pair")
	}
}
func TestSessionBatchDeleteArchivesFirstAndRetriesOnlyFailures(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	failed := false
	archiveSet := map[string]bool{"b": true}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args struct {
					Request struct {
						ID   string `json:"sessionId"`
						Stop *bool  `json:"stopActivity"`
					} `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		id := body.Payload.Args.Request.ID
		mu.Lock()
		defer mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/workspace/archiveSession"):
			calls = append(calls, "archive:"+id)
			if body.Payload.Args.Request.Stop != nil {
				t.Error("delete preparatory archive must not stop activity")
			}
			if id == "c" {
				http.Error(w, "running fixture", 500)
				return
			}
			archiveSet[id] = true
			ids := []string{}
			for key := range archiveSet {
				ids = append(ids, key)
			}
			rpcOK(w, body.RPCID, map[string]any{"archivedSessionIds": ids})
		case strings.HasSuffix(r.URL.Path, "/session/delete"):
			calls = append(calls, "delete:"+id)
			if !archiveSet[id] {
				t.Error("delete before successful archive")
			}
			if id == "a" && !failed {
				failed = true
				http.Error(w, "delete fixture", 500)
				return
			}
			delete(archiveSet, id)
			ids := []string{}
			for key := range archiveSet {
				ids = append(ids, key)
			}
			rpcOK(w, body.RPCID, map[string]any{"archivedSessionIds": ids, "deletedSessionIds": []string{id}})
		default:
			rpcOK(w, body.RPCID, map[string]any{"items": []any{}})
		}
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.archivedSessionIDs = []string{"b"}
	s.SetSessionBatchScope(NewSessionBatchScope(true, "workspace", "default", ""))
	s.EditSessionBatch("toggle", nil)
	s.EditSessionBatch("visible", []SessionBatchTarget{{"a", "A"}, {"b", "B"}, {"c", "C"}})
	s.RequestSessionBatch("delete")
	s.ConfirmSessionBatch(context.Background(), s.SessionBatch().Token)
	b := batchAwaitIdle(t, s)
	if b.Attempted != 3 || len(b.Selected) != 2 || b.Selected["a"] != "A" || b.Selected["c"] != "C" || len(b.Results) != 3 || b.Results[0].Error == "" || b.Results[1].Error != "" || b.Results[2].Error == "" {
		t.Fatalf("partial result=%+v", b)
	}
	s.RequestSessionBatch("delete")
	s.ConfirmSessionBatch(context.Background(), s.SessionBatch().Token)
	b = batchAwaitIdle(t, s)
	if len(b.Selected) != 1 || b.Selected["c"] != "C" || b.Attempted != 2 {
		t.Fatalf("retry=%+v", b)
	}
	mu.Lock()
	got := strings.Join(calls, ",")
	mu.Unlock()
	if got != "archive:a,delete:a,delete:b,archive:c,delete:a,archive:c" {
		t.Fatalf("RPC order/retry=%s", got)
	}
}
func TestSessionBatchBusyScopeChangeDoesNotRetargetOrRestoreHiddenFailures(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/workspace/archiveSession") {
			close(started)
			<-release
			http.Error(w, "fixture failure", 500)
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{"items": []any{}})
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.SetSessionBatchScope(NewSessionBatchScope(true, "workspace", "default", ""))
	s.EditSessionBatch("toggle", nil)
	s.EditSessionBatch("select", []SessionBatchTarget{{"a", "A"}})
	s.RequestSessionBatch("delete")
	token := s.SessionBatch().Token
	s.ConfirmSessionBatch(context.Background(), token)
	wait(t, started)
	s.SetSessionBatchScope(NewSessionBatchScope(true, "workspace", "default", "new"))
	during := s.SessionBatch()
	if !during.Busy || during.Token != token || len(during.Selected) != 0 || len(during.Targets) != 1 || during.Targets[0].ID != "a" {
		t.Fatalf("scope retargeted confirmed batch=%+v", during)
	}
	close(release)
	done := batchAwaitIdle(t, s)
	if len(done.Selected) != 0 || len(done.Results) != 1 || done.Results[0].Error == "" {
		t.Fatalf("old scope failure reselected hidden row=%+v", done)
	}
}
func TestSessionBatchDeleteRechecksAuthorityAndCancellationBetweenSteps(t *testing.T) {
	for _, mode := range []string{"host", "host-reuse", "cwd", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			deletes := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if strings.HasSuffix(r.URL.Path, "/workspace/archiveSession") {
					rpcOK(w, body["rpcId"], map[string]any{"archivedSessionIds": []string{"a"}})
					return
				}
				if strings.HasSuffix(r.URL.Path, "/session/delete") {
					mu.Lock()
					deletes++
					mu.Unlock()
				}
				rpcOK(w, body["rpcId"], map[string]any{"items": []any{}})
			}))
			defer srv.Close()
			s := New()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			changed := make(chan struct{})
			once := sync.Once{}
			s.Attach(client, func() {
				if len(s.Snapshot().ArchivedSessionIDs) > 0 {
					once.Do(func() {
						switch mode {
						case "host":
							s.Attach(&dsh.Client{}, nil)
						case "host-reuse":
							s.Attach(&dsh.Client{}, nil)
							s.Attach(client, nil)
						case "cwd":
							s.SetWorkspaceCwd("/new")
						case "cancel":
							cancel()
						}
						close(changed)
					})
				}
			})
			s.EditSessionBatch("toggle", nil)
			s.EditSessionBatch("select", []SessionBatchTarget{{"a", "A"}})
			s.RequestSessionBatch("delete")
			s.ConfirmSessionBatch(ctx, s.SessionBatch().Token)
			wait(t, changed)
			batchAwaitIdle(t, s)
			mu.Lock()
			n := deletes
			mu.Unlock()
			if n != 0 {
				t.Fatalf("%s sent delete after lost authority/cancellation", mode)
			}
		})
	}
}
