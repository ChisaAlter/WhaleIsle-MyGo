package assistantctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/store/session"
)

func TestConversationOpenAndDraftConflictPreservation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		switch r.URL.Path {
		case "/dsh-whale/assistant/ensure":
			writeRPCResponse(w, req.RPCID, map[string]any{
				"ok":        true,
				"sessionId": "session-whale-42",
				"created":   false,
			})
		case "/dshbot/describe":
			writeRPCResponse(w, req.RPCID, map[string]any{
				"view": map[string]any{
					"ns":       "dsh-bot",
					"revision": 2,
					"value": map[string]any{
						"items": []map[string]any{
							{"id": "bot-1", "kind": "bot", "name": "Helper", "sessionId": "sess-bot-1"},
						},
					},
				},
			})
		case "/dshbot/bot/open":
			writeRPCResponse(w, req.RPCID, map[string]any{
				"sessionId": "sess-bot-1",
			})
		default:
			writeRPCResponse(w, req.RPCID, map[string]any{})
		}
	}))
	defer srv.Close()

	cfgStore, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = cfgStore.Update(func(c *config.Config) {
		c.WhaleAssistantEnabled = true
		c.DshbotEnabled = true
	})

	dshHome := t.TempDir()
	whaleDataDir := filepath.Join(dshHome, "data", "whale")
	if err := os.MkdirAll(whaleDataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dshHome, "sessions", "session-whale-42"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(whaleDataDir, "settings.json"), []byte(`{"name":"小鲸鱼","sessionId":"session-whale-42"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	ep := harness.Endpoint{Origin: srv.URL, Cookie: "c=1"}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL, Cookie: "c=1"}}
	sessions := session.New()
	sessions.Attach(client, nil)
	sessions.Select("other-session")

	var mu sync.Mutex
	var submitted []string
	var staged []string
	var revoked atomic.Int32
	var conflictFlag atomic.Bool

	mh := &mockHost{
		ep:        ep,
		bootState: harness.Ready,
		gen:       1,
		cfg:       cfgStore,
		client:    client,
		sessions:  sessions,
		win:       "win",
		winState:  WindowState{Exists: true, Visible: true},
		dshHome:   dshHome,
		vendorDir: t.TempDir(),
	}
	deps := mh.deps()
	deps.RevokePhoneGit = func() { revoked.Add(1) }
	deps.HasSessionDraftConflict = func(string) bool { return conflictFlag.Load() }
	deps.StageSessionDraft = func(sid, text string) bool {
		mu.Lock()
		staged = append(staged, sid+":"+text)
		mu.Unlock()
		return true
	}
	deps.SubmitComposer = func(_ any, _ *session.Store, sid, _ string, _ uint64) {
		mu.Lock()
		submitted = append(submitted, sid)
		mu.Unlock()
	}

	svc := NewService(deps)
	t.Cleanup(svc.SuspendAll)
	asst := assiststore.New()
	svc.AttachStores(asst, nil, nil)

	// Verify WhalePersistedInfo reads disk session & custom name.
	sid, name, _ := svc.WhalePersistedInfo()
	if sid != "session-whale-42" || name != "小鲸鱼" {
		t.Fatalf("WhalePersistedInfo got (%q, %q)", sid, name)
	}

	// Conflict must preserve existing draft and refuse submission.
	conflictFlag.Store(true)
	svc.SetOpen(true)
	svc.SetDraft("hello whale")
	svc.HandleNativePrompt(ep.Origin, ep, "hello whale")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !asst.Snapshot().Value.Busy && asst.Snapshot().Value.LastErr != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	if len(submitted) != 0 || len(staged) != 0 {
		mu.Unlock()
		t.Fatalf("draft conflict still submitted prompt: staged=%v submitted=%v", staged, submitted)
	}
	mu.Unlock()

	// Clear conflict and send via HandleNativePrompt.
	conflictFlag.Store(false)
	svc.SetOpen(true)
	svc.SetDraft("hello whale")
	svc.HandleNativePrompt(ep.Origin, ep, "hello whale")
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := len(submitted) == 1
		mu.Unlock()
		if done && !svc.Open() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	if len(submitted) != 1 || submitted[0] != "session-whale-42" || len(staged) != 1 || staged[0] != "session-whale-42:hello whale" {
		mu.Unlock()
		t.Fatalf("expected submit to session-whale-42, got staged=%v submitted=%v", staged, submitted)
	}
	mu.Unlock()
	if svc.Draft() != "" {
		t.Fatalf("expected matching assistant draft cleared, got %q", svc.Draft())
	}

	// OpenWhaleFromSidebar must select persisted session immediately and revoke phone git.
	sessions.Select("other-session")
	beforeRevoke := revoked.Load()
	svc.OpenWhaleFromSidebar()
	if sessions.SelectedSessionID() != "session-whale-42" || revoked.Load() <= beforeRevoke {
		t.Fatalf("OpenWhaleFromSidebar did not select session or revoke phone git: selected=%q revoked=%d", sessions.SelectedSessionID(), revoked.Load())
	}
}

func TestBotRoomRoutineTaskControllersAndWindowLifecycle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		switch r.URL.Path {
		case "/api/settings/describe":
			writeRPCResponse(w, req.RPCID, map[string]any{"writable": true})
		case "/dshbot/describe":
			writeRPCResponse(w, req.RPCID, map[string]any{
				"view": map[string]any{
					"ns":       "dsh-bot",
					"revision": 5,
					"value": map[string]any{
						"items": []map[string]any{
							{"id": "bot-1", "kind": "bot", "name": "Alpha", "sessionId": "sess-1", "workspaceId": "ws-1"},
							{"id": "room-1", "kind": "room", "name": "Room1", "sessionId": "sess-r1", "memberBotIds": []string{"bot-1"}},
						},
						"sections": []map[string]any{{"id": "sec-1", "name": "Default"}},
					},
				},
			})
		case "/dshbot/tasks/list":
			writeRPCResponse(w, req.RPCID, map[string]any{"available": true, "tasks": []any{}})
		case "/dshbot/routines/list":
			writeRPCResponse(w, req.RPCID, map[string]any{"botId": "bot-1", "routines": []any{}})
		case "/dshbot/routines/missed":
			writeRPCResponse(w, req.RPCID, map[string]any{"ids": []string{}})
		case "/dsh-whale/tasks/list":
			writeRPCResponse(w, req.RPCID, map[string]any{"available": true, "tasks": []any{}})
		default:
			writeRPCResponse(w, req.RPCID, map[string]any{})
		}
	}))
	defer srv.Close()

	cfgStore, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = cfgStore.Update(func(c *config.Config) {
		c.WhaleAssistantEnabled = true
		c.DshbotEnabled = true
	})
	roster, err := dshbotstore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{
		View: pluginrpc.DshbotSettingsView{
			NS:       "dsh-bot",
			Revision: 5,
			Value: pluginrpc.DshbotCatalogValue{
				Items: []pluginrpc.DshbotCatalogItem{
					{ID: "bot-1", Kind: "bot", Name: "Alpha", SessionID: "sess-1"},
					{ID: "room-1", Kind: "room", Name: "Room1", SessionID: "sess-r1"},
				},
			},
		},
	}, false, srv.URL)

	ep := harness.Endpoint{Origin: srv.URL, Cookie: "c=1"}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL, Cookie: "c=1"}}
	mh := &mockHost{
		ep:        ep,
		bootState: harness.Ready,
		gen:       1,
		cfg:       cfgStore,
		client:    client,
		sessions:  session.New(),
		win:       "win",
		winState:  WindowState{Exists: true, Visible: true},
		dshHome:   t.TempDir(),
		vendorDir: t.TempDir(),
	}
	var copied []string
	deps := mh.deps()
	deps.WriteClipboard = func(text string) { copied = append(copied, text) }
	deps.OpenFileDialog = func(_ any, opts FileDialogOptions) ([]string, error) {
		return []string{"/tmp/a.txt", "/tmp/b.md"}, nil
	}

	svc := NewService(deps)
	t.Cleanup(svc.SuspendAll)
	asst := assiststore.New()
	asst.SetMode(assiststore.ModeDshbot)
	asst.SelectDshbot("bot-1", srv.URL)
	svc.AttachStores(asst, nil, roster)
	svc.SetOpen(true)

	// Exercise ContactsModel refresh & roster sync.
	cm := svc.ContactsModel(ep)
	if cm.OnRefresh == nil {
		t.Fatal("expected ContactsModel.OnRefresh")
	}
	// Exercise TasksModel, RoutinesModel, RoutineCreateModel, BotCreateModel, BotDeleteModel, BotDuplicateModel, BotVisibilityModel.
	tm := svc.TasksModel(ep, "bot-1")
	rm := svc.RoutinesModel(ep, "bot-1")
	rcm := svc.RoutineCreateModel(ep, "bot-1", rm.State, rm.OnRefresh)
	bcm := svc.BotCreateModel(ep)
	bdm := svc.BotDeleteModel(ep, "bot-1")
	bdupm := svc.BotDuplicateModel(ep, "bot-1")
	bvm := svc.BotVisibilityModel(ep)

	if tm.OnRefresh == nil || rm.OnRefresh == nil || rcm.OnStart == nil || bcm.OnStart == nil || bdm.OnReview == nil || bdupm.OnStart == nil || bvm.OnRefresh == nil {
		t.Fatal("expected non-nil sub-controller callbacks")
	}

	// RevealBotVisibility works for active Dshbot host.
	if !svc.RevealBotVisibility(ep, client, "bot-1", "sess-1") {
		t.Fatal("expected RevealBotVisibility to succeed")
	}

	// Room models when selected ID is room-1.
	asst.SelectDshbot("room-1", srv.URL)
	roomCreate := svc.RoomCreateModel(ep)
	roomEdit := svc.RoomEditModel(ep, "room-1")
	roomRuntime := svc.RoomRuntimeModel(ep, "room-1")
	if roomCreate.OnStart == nil || roomEdit.OnStart == nil || roomRuntime.OnRefresh == nil {
		t.Fatal("expected non-nil room callbacks")
	}

	// WhaleTasksModel file picker in Whale mode.
	asst.SetMode(assiststore.ModeWhale)
	wt := svc.WhaleTasksModel(ep)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		wt = svc.WhaleTasksModel(ep)
		if !wt.State.Busy && wt.State.Loaded {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	wt.OnPickFiles()
	if got := svc.WhaleTasksModel(ep).State.Files; got != "/tmp/a.txt\n/tmp/b.md" {
		t.Fatalf("WhaleTasksModel OnPickFiles got %q", got)
	}

	// ResolveDshbot fences mismatched client.
	if _, _, err := svc.ResolveDshbot(context.Background(), harness.Endpoint{Origin: "http://127.0.0.1:9999"}, mh.sessions, "bot-1"); err == nil {
		t.Fatal("expected ResolveDshbot to reject mismatched endpoint")
	}

	// Window lifecycle & StopDSH hooks must run cleanly without panic.
	svc.OnWindowFocus()
	svc.OnWindowHide()
	svc.OnWindowMinimize()
	svc.StopDSH()
	svc.OnWindowClosed()
	_ = copied
}
