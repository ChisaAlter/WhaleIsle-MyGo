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

func TestRenameDraftFailureThenAcknowledgedRetry(t *testing.T) {
	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		if strings.HasSuffix(r.URL.Path, "/session/rename") && count.Add(1) == 1 {
			http.Error(w, "unavailable", 503)
			return
		}
		rpcOK(w, b["rpcId"], map[string]any{"title": "New title", "seq": 1})
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.items = []Item{{SessionID: "s", Title: "Old"}}
	s.BeginSessionRename("s", "Old")
	s.EditSessionRename("  New title  ")
	s.SubmitSessionRename("s", s.SessionRename().Draft)
	deadline := time.Now().Add(time.Second)
	for s.SessionRename().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	v := s.SessionRename()
	if v.SessionID != "s" || v.Draft != "  New title  " || v.Error == "" {
		t.Fatalf("lost failed draft: %+v", v)
	}
	s.SubmitSessionRename("s", v.Draft)
	deadline = time.Now().Add(time.Second)
	for s.SessionRename().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.SessionRename().SessionID != "" || s.Snapshot().Items[0].Title != "New title" {
		t.Fatal("ack did not close editor")
	}
}
func TestRenameLateOldHostCannotCloseReplacementDraft(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		close(started)
		<-release
		rpcOK(w, b["rpcId"], map[string]any{"title": "Late"})
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.BeginSessionRename("s", "Old")
	s.RenameSession(context.Background(), "s", "Late")
	<-started
	s.Attach(&dsh.Client{}, nil)
	s.BeginSessionRename("s", "Replacement")
	close(release)
	time.Sleep(15 * time.Millisecond)
	if v := s.SessionRename(); v.Draft != "Replacement" || v.Busy {
		t.Fatalf("stale acknowledgement: %+v", v)
	}
}

func TestRenamePresentationCannotCrossHostAfterAcknowledgement(t *testing.T) {
	var replacementCalls, oldPresentationCalls atomic.Int32
	replacement := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replacementCalls.Add(1)
		http.Error(w, "must not be called", 500)
	}))
	defer replacement.Close()
	old := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		if strings.HasSuffix(r.URL.Path, "/session/setPresentation") {
			oldPresentationCalls.Add(1)
		}
		rpcOK(w, b["rpcId"], map[string]any{"title": "Renamed", "seq": 1})
	}))
	defer old.Close()
	s := New()
	done := make(chan struct{})
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: old.URL}}, func() {
		if s.Snapshot().LastAction == "session renamed" {
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: replacement.URL}}, nil)
			s.BeginSessionRename("s", "Replacement draft")
			close(done)
		}
	})
	s.BeginSessionRename("s", "Old")
	s.SubmitSessionRename("s", "Renamed")
	wait(t, done)
	time.Sleep(15 * time.Millisecond)
	if replacementCalls.Load() != 0 || oldPresentationCalls.Load() != 0 || s.SessionRename().Draft != "Replacement draft" {
		t.Fatal("presentation crossed the Host replacement boundary")
	}
}
