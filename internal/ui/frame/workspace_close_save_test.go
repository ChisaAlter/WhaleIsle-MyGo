package frame

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/ui/files"
	remoteui "whaleisle/internal/ui/remote"
)

func waitWorkspaceClose(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatal("owned close fixture did not settle")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestMixedPaneCloseDrainsRemoteReceiptAfterLocalSaveFails(t *testing.T) {
	writeStarted, release := make(chan struct{}), make(chan struct{})
	var unblock sync.Once
	defer unblock.Do(func() { close(release) })
	var remoteWrites atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/dsh-remote/status":
			json.NewEncoder(w).Encode(map[string]any{"sessionBound": true, "sessionMode": "remote", "workspace": "/work"})
		case "/api/dsh-remote/ls":
			json.NewEncoder(w).Encode(map[string]any{"bound": true, "path": "/work", "items": []any{map[string]any{"path": "/work/a.txt", "name": "a.txt", "type": "file"}}})
		case "/api/dsh-remote/read":
			json.NewEncoder(w).Encode(map[string]any{"content": "remote baseline", "mtime": 10})
		case "/api/dsh-remote/write":
			var request struct {
				SessionID string  `json:"sessionId"`
				Content   string  `json:"content"`
				Mtime     float64 `json:"expectedMtime"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.SessionID != "owned-close" || request.Content != "remote draft" || request.Mtime != 10 {
				t.Errorf("wrong captured write: %+v err=%v", request, err)
			}
			if remoteWrites.Add(1) == 1 {
				close(writeStarted)
			}
			select {
			case <-release:
				json.NewEncoder(w).Encode(map[string]any{"ok": true, "mtime": 11})
			case <-r.Context().Done():
			}
		default:
			t.Errorf("unexpected owned route %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	remote := remoteui.NewFileWorkspaceState()
	defer remote.Suspend()
	remote.Files.Bind(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, "owned-close", nil)
	remote.Files.Browse("")
	waitWorkspaceClose(t, func() bool { return !remote.Files.Snapshot().Busy })
	remote.Files.Open("/work/a.txt")
	waitWorkspaceClose(t, func() bool { return !remote.Files.Snapshot().Busy })
	if !remote.Files.Snapshot().FileLoaded {
		t.Fatal("owned remote file not ready")
	}
	remote.Files.SetDraft("remote draft")

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("local baseline"), 0600); err != nil {
		t.Fatal(err)
	}
	tabs := files.Tabs{}
	tabs.Bind("owned-close", root)
	defer tabs.Dispose()
	selected, preview, draft, editing := "a.txt", "local baseline", "local draft", true
	tabs.Workspaces = map[string]*files.WorkspaceTabs{filepath.Clean(root): {Open: []*files.FileTab{{Path: selected, Preview: preview, Draft: draft, Editing: true}}, Active: selected}}
	var localWrites atomic.Int32
	model := files.Model{Tabs: &tabs, WorkspaceRoot: root, BindingID: "owned-close", SelectedFile: &selected, Preview: &preview, EditDraft: &draft, Editing: &editing,
		OnSaveEdit: func(string, string) error { localWrites.Add(1); return errors.New("local save denied") }}
	pane := &WorkspacePane{ID: "mixed", Kind: "files", Files: &WorkspaceFileState{}, RemoteFiles: remote}
	if pending, err := startWorkspacePaneCloseSave(pane, model); !pending || err != nil && !strings.Contains(err.Error(), "local save denied") {
		t.Fatalf("start pending=%v err=%v", pending, err)
	}
	select {
	case <-writeStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("remote save not started")
	}
	waitWorkspaceClose(t, func() bool {
		pending, err := workspacePaneCloseSaveStatus(pane, model)
		return pending && err != nil && strings.Contains(err.Error(), "local save denied")
	})
	for range 3 {
		pending, err := workspacePaneCloseSaveStatus(pane, model)
		if !pending || err == nil {
			t.Fatalf("local failure abandoned pending remote receipt: %v %v", pending, err)
		}
	}
	unblock.Do(func() { close(release) })
	waitWorkspaceClose(t, func() bool {
		pending, err := workspacePaneCloseSaveStatus(pane, model)
		return !pending && err != nil
	})
	if err := remote.PreflightDirtyForClose(); err != nil {
		t.Fatalf("settled remote cursor was stranded after local error: %v", err)
	}
	if remoteWrites.Load() != 1 || localWrites.Load() != 1 || remote.Dirty() || draft != "local draft" || preview != "local baseline" {
		t.Fatalf("mixed receipts repeated/lost draft: remote=%d local=%d remoteDirty=%v preview=%q draft=%q", remoteWrites.Load(), localWrites.Load(), remote.Dirty(), preview, draft)
	}
}
