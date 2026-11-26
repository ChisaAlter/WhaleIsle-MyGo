package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestRemoteFilesOwnerLossNormalizedBrowseAndReloadFailure(t *testing.T) {
	var failRead atomic.Bool
	entered := make(chan struct{}, 4)
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/dsh-remote/status":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionBound": true,
				"sessionMode":  "remote",
				"workspace":    "/workspace",
				"host":         "example",
				"username":     "dev",
				"port":         22,
			})
		case "/api/dsh-remote/ls":
			select {
			case entered <- struct{}{}:
			default:
			}
			<-release
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path":  "/workspace",
				"bound": true,
				"items": []map[string]any{{"name": "hello.txt", "path": "/workspace/hello.txt", "kind": "file"}},
			})
		case "/api/dsh-remote/read":
			if failRead.Load() {
				http.Error(w, "transient read error", http.StatusBadGateway)
				return
			}
			mtime := 1234.0
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path":    "/workspace/hello.txt",
				"content": "initial-content",
				"mtime":   mtime,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var owned atomic.Bool
	owned.Store(true)

	s := NewFiles()
	s.BindSession(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, "s1", "s1", func() bool { return owned.Load() }, nil)

	// 1. Browse("") while owner drops mid-flight must clear Busy/Operation/Tree[""].Loading.
	s.Browse("")
	<-entered
	owned.Store(false)
	close(release)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot().Value
		if !v.Busy {
			break
		}
		time.Sleep(time.Millisecond)
	}
	v := s.Snapshot().Value
	if v.Busy || v.Operation != "" || v.Tree[""].Loading {
		t.Fatalf("expected !owner() to clear Busy/Operation/Tree Loading, got busy=%v op=%q tree=%+v", v.Busy, v.Operation, v.Tree)
	}

	// 2. Successful Browse("") normalizing to "/workspace" must clear Tree[""].Loading.
	owned.Store(true)
	s.Browse("")
	for time.Now().Before(deadline) {
		v = s.Snapshot().Value
		if !v.Busy && v.Loaded {
			break
		}
		time.Sleep(time.Millisecond)
	}
	v = s.Snapshot().Value
	if v.Tree[""].Loading {
		t.Fatalf("normalized Browse(\"\") left Tree[\"\"].Loading stuck true: %+v", v.Tree)
	}
	if !v.Tree["/workspace"].Loaded {
		t.Fatalf("expected /workspace loaded in Tree: %+v", v.Tree)
	}

	// 3. Load file, then fail Reload transiently: FileLoaded must stay true.
	s.Open("/workspace/hello.txt")
	for time.Now().Before(deadline) {
		v = s.Snapshot().Value
		if !v.Busy && v.FileLoaded {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !s.Snapshot().Value.FileLoaded || s.Snapshot().Value.File.Content != "initial-content" {
		t.Fatalf("expected initial file loaded: %+v", s.Snapshot().Value)
	}
	failRead.Store(true)
	s.Reload()
	for time.Now().Before(deadline) {
		v = s.Snapshot().Value
		if !v.Busy && v.Error != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	v = s.Snapshot().Value
	if !v.FileLoaded || v.File.Content != "initial-content" {
		t.Fatalf("transient reload failure discarded loaded file: loaded=%v content=%q", v.FileLoaded, v.File.Content)
	}
}
