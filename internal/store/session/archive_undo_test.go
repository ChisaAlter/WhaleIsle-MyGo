package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestArchiveUndoNoticeOwnership(t *testing.T) {
	s := New()
	client := &dsh.Client{}
	s.client = client
	s.offerArchiveUndoLocked(client, "id", "Title", false, time.Now())
	if got := s.ArchiveUndoNotice(); got == nil || got.SessionID != "id" || !got.Announce {
		t.Fatal("successful archive did not offer undo", got)
	}
	if got := s.ArchiveUndoNotice(); got == nil || got.Announce {
		t.Fatal("toast was announced twice")
	}
	s.client = &dsh.Client{}
	if s.ArchiveUndoNotice() != nil {
		t.Fatal("notice survived Host change")
	}
	s.client = client
	s.offerArchiveUndoLocked(client, "id", "Title", false, time.Now().Add(-9*time.Second))
	if s.ArchiveUndoNotice() != nil {
		t.Fatal("expired undo remained available")
	}
}

func TestArchiveUndoRestoreOnce(t *testing.T) {
	var restores atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/workspace/unarchiveSession"):
			restores.Add(1)
			rpcOK(w, body["rpcId"], map[string]any{"archivedSessionIds": []string{}})
		case strings.HasSuffix(r.URL.Path, "/session/list"):
			rpcOK(w, body["rpcId"], map[string]any{"items": []any{}})
		default:
			t.Errorf("unexpected action %s", r.URL.Path)
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.client = client
	s.archivedSessionIDs = []string{"id"}
	s.offerArchiveUndoLocked(client, "id", "Title", false, time.Now())
	notice := s.ArchiveUndoNotice()
	s.UndoArchive(notice.Token + 1)
	if restores.Load() != 0 {
		t.Fatal("wrong token dispatched")
	}
	s.UndoArchive(notice.Token)
	s.UndoArchive(notice.Token)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && s.Snapshot().LastAction != "session unarchived" {
		time.Sleep(time.Millisecond)
	}
	if restores.Load() != 1 || len(s.Snapshot().ArchivedSessionIDs) != 0 || s.ArchiveUndoNotice() != nil {
		t.Fatal("undo not exactly one real restore", restores.Load(), s.Snapshot().LastAction)
	}
}
