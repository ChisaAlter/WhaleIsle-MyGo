package im

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func TestParityRefreshOwnershipAndAuthoritativeEmpty(t *testing.T) {
	s := New()
	old := &dsh.Client{}
	s.Attach(old, nil)
	s.rows = []Entry{{ID: "feishu:old", BotID: "old"}}
	s.selectedID = "feishu:old"
	epoch, generation := s.connectionEpoch, s.refreshGeneration
	s.Detach()
	s.Attach(&dsh.Client{}, nil)
	s.finishRefresh(finishArgs{client: old, epoch: epoch, generation: generation, replaceRows: true, rows: []Entry{{ID: "stale"}}, hasRPC: true, bridgeMode: "cordis"})
	if snap := s.Snapshot(); snap.HasIMRPC || snap.BridgeMode == "cordis" || snap.Rows[0].ID == "stale" {
		t.Fatal("old Host refresh republished facts", snap)
	}
	s.finishRefresh(finishArgs{client: s.client, epoch: s.connectionEpoch, generation: s.refreshGeneration, replaceRows: true, rows: []Entry{}, hasRPC: true, bridgeMode: "cordis"})
	if snap := s.Snapshot(); len(snap.Rows) != 0 || snap.SelectedID != "" || !snap.HasIMRPC {
		t.Fatal("authoritative empty roster retained deleted Bot", snap)
	}
}

func TestDraftPersistenceFailureRetiresUnsentMessage(t *testing.T) {
	old, had := api.Registry["im/send"]
	api.Registry["im/send"] = api.Endpoint{Method: "im/send"}
	defer func() {
		if had {
			api.Registry["im/send"] = old
		} else {
			delete(api.Registry, "im/send")
		}
	}()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// History can be committed, but the draft filename cannot be replaced.
	if err := os.Mkdir(filepath.Join(s.root, draftsFile), 0700); err != nil {
		t.Fatal(err)
	}
	s.Attach(&dsh.Client{}, nil)
	s.SetDraft("general", "retain me")
	s.SendDraft(context.Background(), "general", "retain me")
	snap := s.Snapshot()
	if snap.Draft != "retain me" || snap.Err == "" || len(snap.Messages) != 1 || snap.Messages[0].State != MessageStateFailed {
		t.Fatalf("unsent message was not retired: %+v", snap)
	}
	persisted, err := readJSON[map[string][]Message](filepath.Join(s.root, historyFile))
	if err != nil || persisted == nil || len((*persisted)["general"]) != 1 || (*persisted)["general"][0].State != MessageStateFailed {
		t.Fatalf("disk still claims a pending send: history=%+v err=%v", persisted, err)
	}
}

func TestCordisInventoryRefreshDoesNotReplaceInspectManifest(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		calls = append(calls, r.URL.Path)
		mu.Unlock()
		var value any
		switch r.URL.Path {
		case "/api/dynamicCordisRunner/inventory":
			value = []any{map[string]any{"pluginId": "fixture", "activeRun": map[string]any{"pluginRunId": "run"}}}
		case "/api/dynamicCordisRunner/getClientCode":
			value = map[string]any{"code": "", "pluginId": "fixture", "pluginRunId": "run"}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	done := make(chan struct{}, 1)
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() { done <- struct{}{} })
	s.RefreshCordisInventory(context.Background())
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("inventory refresh did not finish")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 2 || calls[0] != "/api/dynamicCordisRunner/inventory" || calls[1] != "/api/dynamicCordisRunner/getClientCode" {
		t.Fatalf("inventory refresh dispatched extra operations: %v", calls)
	}
}

func TestParityLocalSaveFailureRetainsDraft(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	s.root = file
	s.selectedID = "local"
	s.SetDraft("local", "retain me")
	s.SendDraft(context.Background(), "local", "retain me")
	snap := s.Snapshot()
	if snap.Draft != "retain me" || snap.Err == "" || len(snap.Messages) != 0 {
		t.Fatal("failed local persistence cleared or claimed the draft", snap)
	}
}
