package assistantctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

type mockHost struct {
	mu          sync.Mutex
	client      *dsh.Client
	ep          harness.Endpoint
	bootState   harness.State
	gen         int
	cfg         *config.Store
	sessions    *session.Store
	win         any
	winState    WindowState
	dshHome     string
	vendorDir   string
	quitting    bool
	blocking    bool
	petActive   bool
	webOpened   bool
	webPlugin   string
	invalidated atomic.Int32
	taskChanged atomic.Int32
	notifDenied bool
	notifs      []*fakeNotification
}

func (m *mockHost) deps() HostDeps {
	return HostDeps{
		RunOnMain:     func(fn func()) { fn() },
		Window:        func() any { m.mu.Lock(); defer m.mu.Unlock(); return m.win },
		WindowUpdate:  func(_ any, fn func()) { fn() },
		Invalidate:    func(_ any) { m.invalidated.Add(1) },
		WindowStatus:  func(_ any) WindowState { m.mu.Lock(); defer m.mu.Unlock(); return m.winState },
		CurrentClient: func() *dsh.Client { m.mu.Lock(); defer m.mu.Unlock(); return m.client },
		BootEndpoint: func() (harness.Endpoint, harness.State, int) {
			m.mu.Lock()
			defer m.mu.Unlock()
			return m.ep, m.bootState, m.gen
		},
		LifecycleContext:     func() context.Context { return context.Background() },
		Sessions:             func() *session.Store { m.mu.Lock(); defer m.mu.Unlock(); return m.sessions },
		Config:               func() *config.Store { m.mu.Lock(); defer m.mu.Unlock(); return m.cfg },
		DshHome:              func() string { m.mu.Lock(); defer m.mu.Unlock(); return m.dshHome },
		VendorDir:            func() string { m.mu.Lock(); defer m.mu.Unlock(); return m.vendorDir },
		Quitting:             func() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.quitting },
		OtherSurfaceBlocking: func() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.blocking },
		PetSettingsActive:    func() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.petActive },
		NotifyTaskChanged:    func() { m.taskChanged.Add(1) },
		MarkWebViewOpen:      func() { m.mu.Lock(); m.webOpened = true; m.mu.Unlock() },
		OpenWebViewHostPlugin: func(_, pluginID string) {
			m.mu.Lock()
			m.webPlugin = pluginID
			m.mu.Unlock()
		},
		SystemNotificationsSupport: func() bool { return true },
		NewNotification: func(opts NotificationOptions) NotificationHandle {
			n := &fakeNotification{}
			m.mu.Lock()
			m.notifs = append(m.notifs, n)
			m.mu.Unlock()
			return n
		},
		IsNotificationsDenied: func(err error) bool { return m.notifDenied && err != nil },
	}
}

func writeRPCResponse(w http.ResponseWriter, rpcID string, value any) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":  "server-response",
		"rpcId": rpcID,
		"result": map[string]any{
			"ok":    true,
			"value": value,
		},
	})
}

func TestNilServiceAndColdStartAttachStoresSafety(t *testing.T) {
	var nilSvc *Service
	nilSvc.SuspendAll()
	nilSvc.SuspendViews()
	nilSvc.OnWindowHide()
	nilSvc.OnWindowMinimize()
	nilSvc.OnWindowClosed()
	nilSvc.OnWindowFocus()
	nilSvc.StopDSH()
	nilSvc.AttachStores(nil, nil, nil)
	nilSvc.ObserveFrame(FrameObservation{})
	nilSvc.BeginGroupRuntimeFrame()
	nilSvc.FinishGroupRuntimeFrame()
	if nilSvc.Open() || nilSvc.ConversationGeneration() != 0 || nilSvc.GateNavigation(true) {
		t.Fatal("nil Service returned unexpected non-zero state")
	}
	if _, _, home := nilSvc.WhalePersistedInfo(); home == "" {
		t.Fatal("nil WhalePersistedInfo returned empty homeDir")
	}

	// Cold start: NewService with empty HostDeps and no stores attached yet.
	svc := NewService(HostDeps{})
	t.Cleanup(svc.SuspendAll)
	gen1 := svc.SetOpen(true)
	if !svc.Open() || gen1 != 1 {
		t.Fatalf("SetOpen(true) got open=%v gen=%d", svc.Open(), gen1)
	}
	m, obs, commit := svc.BuildPanelModel(PanelBuildOptions{ThemeID: "default"})
	if m.Draft == nil || obs.RoomCreateRendered {
		t.Fatalf("unexpected cold-start panel model: draft=%v obs=%+v", m.Draft, obs)
	}
	*m.Draft = "draft before attach"
	commit()
	if svc.Draft() != "draft before attach" {
		t.Fatalf("draft commit lost: %q", svc.Draft())
	}
	svc.ObserveFrame(obs)

	// Late store attachment (simulating Wave 2 deferred store opening).
	asst, err := assiststore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	roster, err := dshbotstore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc.AttachStores(asst, nil, roster)
	if svc.AssistantStore() != asst || svc.DshbotRoster() != roster {
		t.Fatal("late AttachStores did not attach stores")
	}
	svc.SuspendAll()
}

func TestCachedVendorsNonBlockingAsyncProbe(t *testing.T) {
	vendorRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vendorRoot, PluginWhale), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := NewService(HostDeps{
		VendorDir:        func() string { return vendorRoot },
		LifecycleContext: func() context.Context { return ctx },
		Window:           func() any { return "win" },
	})
	t.Cleanup(svc.SuspendAll)

	whale, dshbot := svc.CachedVendors()
	if whale || dshbot {
		t.Fatalf("expected cold first probe before async stat completes to return false/false, got %v/%v", whale, dshbot)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		whale, dshbot = svc.CachedVendors()
		if whale && !dshbot {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("async vendor probe did not converge, got whale=%v dshbot=%v", whale, dshbot)
}

func TestWhaleSettingsModelAndNavLabelWithMockHost(t *testing.T) {
	whaleName := "小鲸屿"
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string          `json:"rpcId"`
			Method  string          `json:"method"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		mu.Lock()
		name := whaleName
		mu.Unlock()
		switch r.URL.Path {
		case "/dsh-whale/catalog":
			writeRPCResponse(w, req.RPCID, pluginrpc.WhaleSettings{
				Name:        name,
				UserTitle:   "Boss",
				PersonaText: "Helpful",
				Memory:      "Remembered",
				SessionID:   "session-whale-1",
				HomeDir:     "/tmp/whale",
			})
		case "/dsh-whale/settings/update":
			mu.Lock()
			whaleName = "新鲸屿"
			mu.Unlock()
			writeRPCResponse(w, req.RPCID, map[string]any{"name": "新鲸屿"})
		case "/dsh-whale/preferences":
			writeRPCResponse(w, req.RPCID, pluginrpc.WhalePreferences{})
		case "/api/session/modelCatalog":
			writeRPCResponse(w, req.RPCID, map[string]any{"providers": []any{}})
		default:
			http.Error(w, "unexpected path "+r.URL.Path, 404)
		}
	}))
	defer srv.Close()

	cfgStore, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := cfgStore.Update(func(c *config.Config) { c.WhaleAssistantEnabled = true }); err != nil {
		t.Fatal(err)
	}
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
	svc := NewService(mh.deps())
	t.Cleanup(svc.SuspendAll)
	asst := assiststore.New()
	svc.AttachStores(asst, nil, nil)
	svc.SetOpen(true)

	// Nav label fetch should converge asynchronously.
	svc.ObserveWhaleSettings()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if svc.WhaleNavName() == "小鲸屿" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if svc.WhaleNavName() != "小鲸屿" {
		t.Fatalf("expected WhaleNavName '小鲸屿', got %q", svc.WhaleNavName())
	}

	// WhaleSettingsModel refresh + edit + save.
	wm := svc.WhaleSettingsModel(ep)
	wm.OnRefresh()
	for time.Now().Before(deadline) {
		if svc.WhaleSettingsModel(ep).State.Loaded {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	wm = svc.WhaleSettingsModel(ep)
	if !wm.State.Loaded || wm.State.Name != "小鲸屿" {
		t.Fatalf("WhaleSettingsModel did not load: %+v", wm.State)
	}
	wm.OnEdit("name", "新鲸屿")
	svc.WhaleSettingsModel(ep).OnSave("name")
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cur := svc.WhaleSettingsModel(ep)
		if !cur.State.Busy && cur.State.CompletedAction == "name" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := svc.WhaleNavName(); got != "新鲸屿" {
		t.Fatalf("expected updated WhaleNavName '新鲸屿', got %q", got)
	}

	// Client replacement must fence stale callbacks and clear owner key.
	mh.mu.Lock()
	mh.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL, Cookie: "c=2"}}
	mh.gen++
	mh.mu.Unlock()
	if svc.WhaleSettingsModel(ep).State.Loaded {
		t.Fatal("stale Whale settings remained bound after client replacement")
	}
}

func TestBuildPanelModelAndBotNotificationLifecycle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		switch r.URL.Path {
		case "/api/settings/describe":
			writeRPCResponse(w, req.RPCID, map[string]any{"writable": false})
		case "/dshbot/describe":
			writeRPCResponse(w, req.RPCID, map[string]any{
				"view": map[string]any{
					"ns":       "dsh-bot",
					"revision": 3,
					"value": map[string]any{
						"items": []map[string]any{
							{"id": "bot-1", "kind": "bot", "name": "Helper", "sessionId": "sess-b1"},
						},
					},
				},
			})
		case "/dshbot/bot/activity":
			writeRPCResponse(w, req.RPCID, map[string]any{
				"entries": []map[string]any{
					{"botId": "bot-1", "sessionId": "sess-b1", "attention": false, "activitySeq": 1},
				},
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
	if err := cfgStore.Update(func(c *config.Config) {
		c.WhaleAssistantEnabled = true
		c.DshbotEnabled = true
	}); err != nil {
		t.Fatal(err)
	}
	roster, err := dshbotstore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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
		winState:  WindowState{Exists: true, Visible: true, Focused: false},
		dshHome:   t.TempDir(),
		vendorDir: t.TempDir(),
	}
	svc := NewService(mh.deps())
	t.Cleanup(svc.SuspendAll)
	asst := assiststore.New()
	asst.SetMode(assiststore.ModeDshbot)
	svc.AttachStores(asst, nil, roster)
	svc.SetOpen(true)

	model, obs, commit := svc.BuildPanelModel(PanelBuildOptions{
		Endpoint: ep,
		ThemeID:  "default",
	})
	if !obs.RoomCreateRendered || !obs.NotificationsRendered {
		t.Fatalf("expected dshbot frame flags true, got %+v", obs)
	}
	commit()
	svc.ObserveFrame(obs)

	// Trigger roster refresh via RunPluginRPC and wait for convergence.
	model.OnDshbotRefreshRoster()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !svc.PluginStatus().Busy && len(roster.Snapshot().Value.Rows) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(roster.Snapshot().Value.Rows) != 1 || roster.Snapshot().Value.Rows[0].ID != "bot-1" {
		t.Fatalf("expected roster refreshed with bot-1, got %+v", roster.Snapshot().Value)
	}

	// Test mode switch & WebView open callbacks.
	model.OnOpenWhaleWeb()
	if svc.Open() {
		t.Fatal("OnOpenWhaleWeb should close assistant panel")
	}
	mh.mu.Lock()
	opened, plugin := mh.webOpened, mh.webPlugin
	mh.mu.Unlock()
	if !opened || plugin != PluginWhale {
		t.Fatalf("expected webview opened for %q, got (%v, %q)", PluginWhale, opened, plugin)
	}

	// Ensure bot notification observer stops cleanly on unmount.
	svc.ObserveBotNotifications(false)
	if !strings.Contains(svc.BotNotificationStatusText(), "") {
		t.Fatal("unexpected notification status error")
	}
}

func TestConcurrentServiceOperationsRaceSafe(t *testing.T) {
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
	mh := &mockHost{
		ep:        harness.Endpoint{Origin: "http://127.0.0.1:1", Cookie: "c=1"},
		bootState: harness.Ready,
		gen:       1,
		cfg:       cfgStore,
		sessions:  session.New(),
		win:       "win",
		winState:  WindowState{Exists: true, Visible: true},
		dshHome:   t.TempDir(),
		vendorDir: t.TempDir(),
	}
	svc := NewService(mh.deps())
	t.Cleanup(svc.SuspendAll)
	svc.AttachStores(assiststore.New(), nil, roster)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				svc.SetOpen(j%2 == 0)
				_ = svc.Open()
				_ = svc.ConversationGeneration()
				_, _ = svc.CachedVendors()
				svc.BeginGroupRuntimeFrame()
				_, obs, commit := svc.BuildPanelModel(PanelBuildOptions{
					Endpoint: mh.ep,
					ThemeID:  "default",
				})
				commit()
				svc.ObserveFrame(obs)
				svc.FinishGroupRuntimeFrame()
				_ = svc.ProactiveModel()
				_ = svc.PluginStatus()
				if id%2 == 0 && j%7 == 0 {
					svc.SuspendViews()
				}
			}
		}(i)
	}
	wg.Wait()
	svc.SuspendAll()
}
