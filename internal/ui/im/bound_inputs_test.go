package im

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pluginrpc"
	imstore "whaleisle/internal/store/im"
	"whaleisle/internal/ui/i18n"
)

// cordisFixture serves one Cordis connection.status roster so the store
// settles into bridgeMode "cordis" with a selected bot. It records every RPC
// payload keyed by path so tests can observe saves.
func cordisFixture(t *testing.T) (*imstore.Store, func() map[string]map[string]any) {
	t.Helper()
	var mu sync.Mutex
	calls := map[string]map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string         `json:"rpcId"`
			Payload map[string]any `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		calls[r.Method+" "+r.URL.Path] = req.Payload
		mu.Unlock()
		value, _ := json.Marshal(map[string]any{
			"snapshot": map[string]any{
				"bots": []map[string]any{{
					"botId": "bot-1", "connected": true, "workspace": "/orig/ws",
					"bot": map[string]any{"name": "Bot One"},
				}},
			},
		})
		result, _ := json.Marshal(map[string]any{"ok": true, "value": json.RawMessage(value)})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "server-response", "rpcId": req.RPCID, "result": json.RawMessage(result),
		})
	}))
	t.Cleanup(srv.Close)
	s := imstore.New()
	s.AttachPlugin(&pluginrpc.Client{BaseURL: srv.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Refresh(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for s.Snapshot().Value.BridgeMode != "cordis" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if s.Snapshot().Value.BridgeMode != "cordis" {
		t.Fatalf("cordis probe never completed: %+v", s.Snapshot().Value)
	}
	return s, func() map[string]map[string]any {
		mu.Lock()
		defer mu.Unlock()
		out := map[string]map[string]any{}
		for k, v := range calls {
			out[k] = v
		}
		return out
	}
}

// Typing a workspace path must update the store's workspace draft. The bound
// local used to be compared before applyInputs flushed the edit, so
// SetWorkspaceDraft never saw the typed path. SetWorkspaceDraft does not
// republish, so the test forces a republish with SetDraft before asserting.
func TestIMWorkspaceInputReachesStore(t *testing.T) {
	s, calls := cordisFixture(t)
	_ = calls
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: s, ThemeID: "deepseek", HarnessReady: true})
	}, 900, 700)
	r, ok := tt.Find("工作区绝对路径")
	if !ok {
		t.Fatal("workspace settings pane missing")
	}
	// The FillWidth input sits directly under its heading.
	tt.ClickAt(r.X+80, r.Y+r.H+16)
	tt.Command("selectAll")
	tt.Type("/new/ws")
	s.SetDraft(s.Snapshot().Value.SelectedID, "probe")
	if got := s.Snapshot().Value.WorkspaceDraft; got != "/new/ws" {
		t.Fatalf("workspace edit lost: draft=%q", got)
	}
}

// The composer textarea must persist the text actually typed this frame; the
// store draft used to be written from a pre-flush read, one frame stale.
// Post-flush assertions alone cannot see the lag — the tester settles extra
// frames — so the test records the store draft at the end of every frame and
// requires it never trails the bound value.
func TestIMDraftTextareaPersistsFreshText(t *testing.T) {
	s := imstore.New()
	s.Refresh(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for s.Snapshot().Value.SelectedID == "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if s.Snapshot().Value.SelectedID == "" {
		t.Fatal("local refresh selected no channel")
	}
	draft := ""
	var drafts, snaps []string
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: s, ThemeID: "deepseek", Draft: &draft})
		drafts = append(drafts, draft)
		snaps = append(snaps, s.Snapshot().Value.Draft)
	}, 900, 700)
	r, ok := tt.Find(i18n.T(i18n.KeyIMCompose))
	if !ok {
		t.Fatal("composer heading missing")
	}
	tt.ClickAt(r.X+80, r.Y+r.H+24)
	tt.Type("hello")
	for i := range drafts {
		if drafts[i] != snaps[i] {
			t.Fatalf("frame %d: bound draft %q but store kept stale %q (all: %v vs %v)", i, drafts[i], snaps[i], drafts, snaps)
		}
	}
	if s.Snapshot().Value.Draft != "hello" {
		t.Fatalf("composer draft not persisted: %q", s.Snapshot().Value.Draft)
	}
}
