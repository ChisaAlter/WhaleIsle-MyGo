package im

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestDir(t *testing.T) {
	got := Dir(`C:\data`)
	want := filepath.Join(`C:\data`, "im")
	if got != want {
		t.Fatalf("Dir = %q, want %q", got, want)
	}
}

func TestOpenSeedsDefaultChannels(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	root := Dir(dir)
	if s.Root() != root {
		t.Fatalf("Root = %q, want %q", s.Root(), root)
	}
	snap := s.Snapshot()
	if len(snap.Rows) < 2 {
		t.Fatalf("seeded rows = %d, want >= 2", len(snap.Rows))
	}
	if snap.SelectedID == "" {
		t.Fatal("expected selected channel after Open")
	}
	if snap.LocalMode != true {
		// hasIMRPC false until Refresh; LocalMode = !hasIMRPC
		t.Fatalf("LocalMode = %v, want true before RPC refresh", snap.LocalMode)
	}

	raw, err := os.ReadFile(filepath.Join(root, channelsFile))
	if err != nil {
		t.Fatal(err)
	}
	var channels []Entry
	if err := json.Unmarshal(raw, &channels); err != nil {
		t.Fatal(err)
	}
	if len(channels) != len(snap.Rows) {
		t.Fatalf("channels.json len = %d, snap = %d", len(channels), len(snap.Rows))
	}
}

func TestPersistDraftAndHistoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	id := s.Snapshot().SelectedID
	if id == "" {
		t.Fatal("no selected id")
	}

	s.SetDraft(id, "hello draft")
	if err := s.Persist(); err != nil {
		t.Fatal(err)
	}
	s.SendDraft(context.Background(), id, "first message")
	snap := s.Snapshot()
	if len(snap.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(snap.Messages))
	}
	if snap.Messages[0].Text != "first message" {
		t.Fatalf("msg text = %q", snap.Messages[0].Text)
	}
	if snap.Draft != "" {
		t.Fatalf("draft after send = %q, want empty", snap.Draft)
	}
	if snap.LastSend == "" {
		t.Fatal("expected LastSend note in stub mode")
	}

	// Re-open from disk.
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s2.Select(id)
	snap2 := s2.Snapshot()
	if len(snap2.Messages) != 1 || snap2.Messages[0].Text != "first message" {
		t.Fatalf("reloaded history = %+v", snap2.Messages)
	}

	// Draft was cleared on send; set and persist a new one, reopen.
	s2.SetDraft(id, "kept draft")
	if err := s2.Persist(); err != nil {
		t.Fatal(err)
	}
	s3, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s3.Select(id)
	if got := s3.Snapshot().Draft; got != "kept draft" {
		t.Fatalf("reloaded draft = %q, want kept draft", got)
	}
}

func TestSelectSwitchesDraftAndHistory(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	rows := s.Snapshot().Rows
	if len(rows) < 2 {
		t.Fatal("need two seeded channels")
	}
	a := channelKey(rows[0])
	b := channelKey(rows[1])

	s.Select(a)
	s.SetDraft(a, "draft-a")
	s.SendDraft(context.Background(), a, "msg-a")
	s.SetDraft(a, "draft-a-again")

	s.Select(b)
	s.SetDraft(b, "draft-b")
	if snap := s.Snapshot(); snap.Draft != "draft-b" || len(snap.Messages) != 0 {
		t.Fatalf("channel b snap = draft %q msgs %d", snap.Draft, len(snap.Messages))
	}

	s.Select(a)
	snap := s.Snapshot()
	if snap.Draft != "draft-a-again" {
		t.Fatalf("channel a draft = %q, want draft-a-again", snap.Draft)
	}
	if len(snap.Messages) != 1 || snap.Messages[0].Text != "msg-a" {
		t.Fatalf("channel a history = %+v", snap.Messages)
	}
}

func TestDetachKeepsLocalData(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	id := s.Snapshot().SelectedID
	s.SendDraft(context.Background(), id, "stay")
	s.Detach()
	snap := s.Snapshot()
	if !snap.LocalMode {
		t.Fatal("LocalMode after Detach")
	}
	if len(snap.Rows) == 0 {
		t.Fatal("rows cleared on Detach")
	}
	if len(snap.Messages) != 1 {
		t.Fatalf("history cleared on Detach: %d", len(snap.Messages))
	}
	if snap.ClientReady {
		t.Fatal("ClientReady should be false after Detach")
	}
}

func TestRefreshWithoutClientKeepsLocalChannels(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot().Rows
	s.Refresh(context.Background())
	after := s.Snapshot()
	if after.Loading {
		t.Fatal("still loading")
	}
	if after.HasIMRPC {
		t.Fatal("expected no IM RPC without client")
	}
	if after.BridgeMode != "local" {
		t.Fatalf("BridgeMode = %q, want local", after.BridgeMode)
	}
	if len(after.Rows) != len(before) {
		t.Fatalf("rows changed: %d -> %d", len(before), len(after.Rows))
	}
	if after.PluginDiag == "" {
		t.Fatal("expected plugin diag note without client")
	}
}

func TestRefreshListAndHistoryMergeHTTPtest(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.URL.Path]++
		mu.Unlock()

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcID, _ := body["rpcId"].(string)

		switch r.URL.Path {
		case "/api/im/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": rpcID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"channels": []map[string]any{
							{"id": "c1", "title": "Alpha", "preview": "hi"},
							{"id": "c2", "name": "Beta"},
						},
					},
				},
			})
		case "/api/im/history":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": rpcID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"messages": []map[string]any{
							{"id": "r1", "text": "from server", "at": "2026-01-01T00:00:00Z", "outbound": false},
							{"id": "local-keep", "text": "should merge", "at": "2026-01-01T00:01:00Z", "fromMe": true},
						},
					},
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": rpcID,
				"result": map[string]any{
					"ok": false,
					"error": map[string]any{
						"code":    "NOT_FOUND",
						"message": "no " + r.URL.Path,
					},
				},
			})
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Seed a local message that should survive merge (different id).
	s.Select("general")
	s.SendDraft(context.Background(), "general", "local only")

	done := make(chan struct{})
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		snap := s.Snapshot()
		if !snap.Loading && snap.ListMethod == "im/list" {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	s.Refresh(context.Background())

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for refresh")
	}
	// Allow history merge finish.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snap := s.Snapshot()
		if snap.SyncedRemote && snap.HistoryMethod == "im/history" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	snap := s.Snapshot()
	if !snap.HasIMRPC || snap.ListMethod != "im/list" {
		t.Fatalf("list snap = %+v", snap)
	}
	if snap.BridgeMode != "rpc" {
		t.Fatalf("BridgeMode = %q", snap.BridgeMode)
	}
	if len(snap.Rows) != 2 || snap.Rows[0].ID != "c1" {
		t.Fatalf("rows = %+v", snap.Rows)
	}
	if snap.HistoryMethod != "im/history" {
		t.Fatalf("HistoryMethod = %q", snap.HistoryMethod)
	}
	if !snap.SyncedRemote {
		t.Fatal("expected SyncedRemote")
	}
	s.Select("c1")
	msgs := s.Snapshot().Messages
	if len(msgs) < 2 {
		t.Fatalf("merged messages = %+v", msgs)
	}
	texts := map[string]bool{}
	for _, m := range msgs {
		texts[m.Text] = true
	}
	if !texts["from server"] {
		t.Fatalf("missing remote message: %+v", msgs)
	}

	mu.Lock()
	defer mu.Unlock()
	if seen["/api/im/list"] < 1 || seen["/api/im/history"] < 1 {
		t.Fatalf("seen paths = %+v", seen)
	}
}

func TestRefreshInventoryBridgeHTTPtest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcID, _ := body["rpcId"].(string)
		if r.URL.Path == "/api/pluginInventory/list" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": rpcID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"managementAvailable": true,
						"entries": []map[string]any{
							{
								"entryId":    "xmanrui-dsh-im",
								"moduleName": "@xmanrui/dsh-im",
								"enabled":    true,
								"fiberPhase": "active",
							},
						},
					},
				},
			})
			return
		}
		// All im/* candidates fail.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": rpcID,
			"result": map[string]any{
				"ok":    false,
				"error": map[string]any{"code": "NOT_FOUND", "message": "missing"},
			},
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	before := len(s.Snapshot().Rows)

	done := make(chan struct{})
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		snap := s.Snapshot()
		if !snap.Loading && snap.BridgeMode == "inventory" && strings.Contains(snap.PluginDiag, "@xmanrui/dsh-im") {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	s.Refresh(context.Background())

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout; snap=%+v", s.Snapshot())
	}
	snap := s.Snapshot()
	if snap.HasIMRPC {
		t.Fatal("HasIMRPC should be false on inventory bridge")
	}
	if snap.LocalMode != true {
		t.Fatal("LocalMode expected")
	}
	if len(snap.Rows) != before {
		t.Fatalf("local rows should remain: %d -> %d", before, len(snap.Rows))
	}
	if !strings.Contains(snap.PluginDiag, "fiber=active") {
		t.Fatalf("PluginDiag = %q", snap.PluginDiag)
	}
}

func TestSendDraftCallsIMSendHTTPtest(t *testing.T) {
	got := make(chan map[string]any, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcID, _ := body["rpcId"].(string)
		if r.URL.Path == "/api/im/send" {
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			got <- args
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": rpcID,
				"result": map[string]any{
					"ok":    true,
					"value": map[string]any{"ok": true},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  rpcID,
			"result": map[string]any{"ok": false, "error": map[string]any{"code": "NOT_FOUND", "message": "x"}},
		})
	}))
	defer srv.Close()

	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.SendDraft(context.Background(), "general", "hello rpc")

	select {
	case args := <-got:
		if args["channelId"] != "general" || args["text"] != "hello rpc" {
			t.Fatalf("args = %#v", args)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("im/send not called")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(s.Snapshot().LastSend, "im/send") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("LastSend = %q", s.Snapshot().LastSend)
}

func TestMergeMessagesPrefersRemoteAndSorts(t *testing.T) {
	local := []Message{
		{ID: "a", Text: "local-a", At: "2026-01-01T00:02:00Z", Outbound: true},
		{ID: "b", Text: "keep-local", At: "2026-01-01T00:00:00Z"},
	}
	remote := []Message{
		{ID: "a", Text: "remote-a", At: "2026-01-01T00:02:00Z", Outbound: false, Remote: true},
		{ID: "c", Text: "remote-c", At: "2026-01-01T00:01:00Z", Remote: true},
	}
	out := mergeMessages(local, remote)
	if len(out) != 3 {
		t.Fatalf("len = %d %+v", len(out), out)
	}
	if out[0].ID != "b" || out[1].ID != "c" || out[2].ID != "a" {
		t.Fatalf("order = %+v", out)
	}
	if out[2].Text != "remote-a" || !out[2].Remote {
		t.Fatalf("merged a = %+v", out[2])
	}
}

func TestRefreshCordisConnectionStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		rpcID, _ := req["rpcId"].(string)
		if r.URL.Path == "/feishu/connection.status" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": rpcID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"snapshot": map[string]any{
							"bots": []map[string]any{
								{
									"botId":     "bot-1",
									"connected": true,
									"state":     "connected",
									"bot":       map[string]any{"name": "飞书测试"},
									"health":    map[string]any{"summary": "运行正常"},
								},
							},
						},
					},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": rpcID,
			"result": map[string]any{
				"ok":    false,
				"error": map[string]any{"code": "NOT_FOUND", "message": r.URL.Path},
			},
		})
	}))
	defer srv.Close()

	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		snap := s.Snapshot()
		if !snap.Loading && snap.BridgeMode == "cordis" {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	s.Refresh(context.Background())
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout; snap=%+v", s.Snapshot())
	}
	snap := s.Snapshot()
	if !snap.HasIMRPC || snap.ListMethod != "connection.status" {
		t.Fatalf("snap = %+v", snap)
	}
	if len(snap.Rows) != 1 || snap.Rows[0].ID != "feishu:bot-1" {
		t.Fatalf("rows = %+v", snap.Rows)
	}
	if !strings.Contains(snap.Rows[0].Detail, "运行正常") {
		t.Fatalf("Detail = %q", snap.Rows[0].Detail)
	}
	if !strings.Contains(snap.PluginDiag, "feishu") {
		t.Fatalf("PluginDiag = %q", snap.PluginDiag)
	}
}
