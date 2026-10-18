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

func TestWorkspaceActionsRejectOldHostReceipts(t *testing.T) {
	actions := []struct {
		name string
		run  func(*Store)
	}{
		{"pin", func(s *Store) { s.PinSession(context.Background(), "same") }},
		{"unpin", func(s *Store) { s.UnpinSession(context.Background(), "same") }},
		{"unarchive", func(s *Store) { s.UnarchiveSession(context.Background(), "same") }},
		{"delete", func(s *Store) { s.DeleteSession(context.Background(), "same") }},
	}
	for _, action := range actions {
		for _, failed := range []bool{false, true} {
			for _, reuse := range []bool{false, true} {
				name := action.name
				if failed {
					name += "/error"
				}
				if reuse {
					name += "/reattach-original-client"
				}
				t.Run(name, func(t *testing.T) {
					started, release, responded := make(chan struct{}), make(chan struct{}), make(chan struct{})
					var extra atomic.Int32
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if strings.HasSuffix(r.URL.Path, "/session/list") {
							extra.Add(1)
							http.Error(w, "unexpected refresh", 500)
							return
						}
						var b map[string]any
						_ = json.NewDecoder(r.Body).Decode(&b)
						close(started)
						<-release
						if failed {
							http.Error(w, "old host failure", 500)
						} else {
							rpcOK(w, b["rpcId"], map[string]any{"pinnedSessionIds": []string{"old"}, "archivedSessionIds": []string{"old"}, "deletedSessionIds": []string{"same"}})
						}
						close(responded)
					}))
					defer srv.Close()
					s := New()
					old := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
					s.Attach(old, nil)
					action.run(s)
					wait(t, started)
					s.Attach(&dsh.Client{}, nil)
					if reuse {
						s.Attach(old, nil)
					}
					s.mu.Lock()
					s.items = []Item{{SessionID: "same", Pinned: true}}
					s.selectedID = "same"
					s.pinnedSessionIDs = []string{"same"}
					s.archivedSessionIDs = []string{"current"}
					s.lastAction = "current host"
					s.mu.Unlock()
					close(release)
					wait(t, responded)
					time.Sleep(20 * time.Millisecond)
					v := s.Snapshot()
					if v.LastAction != "current host" || len(v.Items) != 1 || !v.Items[0].Pinned || v.SelectedID != "same" || len(v.PinnedSessionIDs) != 1 || v.PinnedSessionIDs[0] != "same" || len(v.ArchivedSessionIDs) != 1 || v.ArchivedSessionIDs[0] != "current" || extra.Load() != 0 {
						t.Fatalf("old response changed current host: %+v refreshes=%d", v, extra.Load())
					}
				})
			}
		}
	}
}

func TestArchiveActionRefreshDoesNotRetargetAfterNotify(t *testing.T) {
	for _, deleting := range []bool{false, true} {
		name := "unarchive"
		if deleting {
			name = "delete"
		}
		t.Run(name, func(t *testing.T) {
			var replacementCalls atomic.Int32
			replacement := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				replacementCalls.Add(1)
				http.Error(w, "unexpected", 500)
			}))
			defer replacement.Close()
			old := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var b map[string]any
				_ = json.NewDecoder(r.Body).Decode(&b)
				rpcOK(w, b["rpcId"], map[string]any{"deletedSessionIds": []string{"same"}, "archivedSessionIds": []string{}})
			}))
			defer old.Close()
			s := New()
			done := make(chan struct{})
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: old.URL}}, func() {
				v := s.Snapshot()
				if v.LastAction == "session deleted" || v.LastAction == "session unarchived" {
					s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: replacement.URL}}, nil)
					close(done)
				}
			})
			if deleting {
				s.DeleteSession(context.Background(), "same")
			} else {
				s.UnarchiveSession(context.Background(), "same")
			}
			wait(t, done)
			time.Sleep(20 * time.Millisecond)
			if replacementCalls.Load() != 0 {
				t.Fatal("refresh sent to replacement host")
			}
		})
	}
}
