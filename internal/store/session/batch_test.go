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

func TestSessionBatchSelectionScopeAndFrozenTargets(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.EditSessionBatch("toggle", nil)
	s.EditSessionBatch("visible", []SessionBatchTarget{{"a", "A"}, {"a", "A"}, {"b", "B"}})
	s.RequestSessionBatch("delete")
	b := s.SessionBatch()
	if !b.Confirming || len(b.Targets) != 2 {
		t.Fatalf("batch=%+v", b)
	}
	s.EditSessionBatch("clear", nil)
	if len(s.SessionBatch().Selected) != 2 {
		t.Fatal("confirmation targets mutated")
	}
	b.Selected["injected"] = "bad"
	if len(s.SessionBatch().Selected) != 2 {
		t.Fatal("snapshot aliases selection")
	}
	s.CancelSessionBatch(b.Token)
	s.SetWorkspaceCwd("/other")
	if got := s.SessionBatch(); len(got.Selected) != 0 || got.Confirming {
		t.Fatalf("scope leaked: %+v", got)
	}
	s.ConfirmSessionBatch(context.Background(), b.Token)
	if s.SessionBatch().Busy {
		t.Fatal("stale confirmation accepted")
	}
}

func TestSessionBatchArchiveProgressRetainsFailure(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args struct {
					Request struct {
						SessionID    string `json:"sessionId"`
						StopActivity bool   `json:"stopActivity"`
					} `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/workspace/archiveSession") {
			calls.Add(1)
			if !body.Payload.Args.Request.StopActivity {
				t.Error("confirmed batch did not include stopActivity")
			}
			if body.Payload.Args.Request.SessionID == "b" {
				http.Error(w, "fixture failure", http.StatusInternalServerError)
				return
			}
			rpcOK(w, body.RPCID, map[string]any{"archivedSessionIds": []string{"a"}})
			return
		}
		rpcOK(w, body.RPCID, map[string]any{"items": []any{}})
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.EditSessionBatch("toggle", nil)
	s.EditSessionBatch("visible", []SessionBatchTarget{{"a", "A"}, {"b", "B"}})
	s.RequestSessionBatch("archive")
	if calls.Load() != 0 {
		t.Fatal("RPC before confirmation")
	}
	s.ConfirmSessionBatch(context.Background(), s.SessionBatch().Token)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !s.SessionBatch().Busy {
			break
		}
		time.Sleep(time.Millisecond)
	}
	b := s.SessionBatch()
	if b.Busy || b.Attempted != 2 || len(b.Results) != 2 || b.Results[0].Error != "" || b.Results[1].Error == "" || len(b.Selected) != 1 || b.Selected["b"] != "B" {
		t.Fatalf("batch=%+v", b)
	}
}
