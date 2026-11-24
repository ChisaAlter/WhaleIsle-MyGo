package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestSessionFileReadConflictAndDownload(t *testing.T) {
	writes := 0
	conflict := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		if r.Method == "POST" {
			json.NewDecoder(r.Body).Decode(&b)
			if b["sessionId"] != "session-fixture" {
				t.Error("missing explicit session binding")
			}
		} else if r.URL.Query().Get("sessionId") != "session-fixture" {
			t.Error("missing GET binding")
		}
		switch r.URL.Path {
		case "/api/dsh-remote/status":
			json.NewEncoder(w).Encode(map[string]any{"sessionBound": true, "sessionMode": "remote"})
		case "/api/dsh-remote/ls":
			json.NewEncoder(w).Encode(map[string]any{"bound": true, "path": "/work", "items": []any{map[string]any{"path": "/work/a.txt", "name": "a.txt", "type": "file"}}})
		case "/api/dsh-remote/read":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "content": "original", "mtime": 10, "size": 8})
		case "/api/dsh-remote/write":
			writes++
			if b["expectedMtime"] != float64(10) {
				t.Error("mtime not preserved")
			}
			if conflict {
				w.WriteHeader(409)
				w.Write([]byte(`{"error":"mtime conflict"}`))
			} else {
				w.Write([]byte(`{"ok":true,"mtime":11}`))
			}
		case "/api/dsh-remote/fs":
			if b["op"] != "download" {
				t.Error("invented sync operation")
			}
			w.Write([]byte(`{"ok":true,"local":"/mirror/a.txt","bytes":8}`))
		default:
			t.Error("unexpected route", r.URL.Path)
		}
	}))
	defer srv.Close()
	s := NewFiles()
	s.Bind(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, "session-fixture", nil)
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(time.Second)
		for s.Snapshot().Value.Busy && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if s.Snapshot().Value.Busy {
			t.Fatal("request did not settle")
		}
	}
	s.Browse("")
	wait()
	s.Open("/work/a.txt")
	wait()
	s.SetDraft("edited")
	s.Save()
	wait()
	v := s.Snapshot().Value
	if writes != 1 || v.Draft != "edited" || !v.Stale || v.Error == nil {
		t.Fatal(v)
	}
	s.Open("/work/a.txt")
	wait()
	if s.Snapshot().Value.Draft != "edited" {
		t.Fatal("reload discarded dirty draft")
	}
	s.Discard()
	s.Reload() // Opening a loaded tab retains its cache; explicit Reload reconciles stale mtime.
	wait()
	conflict = false
	s.SetDraft("saved")
	s.Save()
	wait()
	if s.Snapshot().Value.Dirty() || s.Snapshot().Value.Stale {
		t.Fatal(s.Snapshot().Value)
	}
	s.Download()
	wait()
	if s.Snapshot().Value.Downloaded != "/mirror/a.txt" {
		t.Fatal(s.Snapshot().Value)
	}
	s.SetDraft("keep me")
	s.Bind(nil, "", nil)
	if s.Snapshot().Value.SessionID != "" {
		t.Fatal("leaked prior session")
	}
}
