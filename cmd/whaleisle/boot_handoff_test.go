package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/store/session"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/boot"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func bootHandoffWallpaperDataURL(t *testing.T, c color.RGBA) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

type bootHandoffFakeHost struct {
	workspace     string
	wallpaperURL  string
	releaseDesc   chan struct{}
	releaseFollow chan struct{}
	listFail      bool

	mu          sync.Mutex
	descStarted chan struct{}
}

func newBootHandoffFakeHost(workspace, wallpaperURL string, gateDescribe, gateFollow bool) *bootHandoffFakeHost {
	h := &bootHandoffFakeHost{
		workspace:     workspace,
		wallpaperURL:  wallpaperURL,
		releaseDesc:   make(chan struct{}),
		releaseFollow: make(chan struct{}),
		descStarted:   make(chan struct{}, 4),
	}
	if !gateDescribe {
		close(h.releaseDesc)
	}
	if !gateFollow {
		close(h.releaseFollow)
	}
	return h
}

func (h *bootHandoffFakeHost) serveHTTP(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			c, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer c.CloseNow()
			ctx := r.Context()
			var wsMu sync.Mutex
			writeFrame := func(frame dsh.ServerFrame) {
				wsMu.Lock()
				defer wsMu.Unlock()
				_ = wsjson.Write(ctx, c, frame)
			}
			for {
				var f dsh.ClientFrame
				if wsjson.Read(ctx, c, &f) != nil {
					return
				}
				if f.Type != dsh.FrameOpen {
					continue
				}
				switch f.Endpoint {
				case "workspace/follow":
					val, _ := json.Marshal(map[string]any{
						"type": "baseline",
						"value": map[string]any{
							"scratchCwd": h.workspace,
							"items": []map[string]any{{
								"workspaceId": "ws-default",
								"path":        h.workspace,
								"sessionIds":  []string{"auto-first", "saved-target"},
							}},
							"pinnedSessionIds":   []string{},
							"archivedSessionIds": []string{},
						},
					})
					writeFrame(dsh.ServerFrame{Type: dsh.FrameItem, StreamID: f.StreamID, Value: val})
				case "session/follow":
					go func(streamID string, payload json.RawMessage) {
						select {
						case <-h.releaseFollow:
						case <-ctx.Done():
							return
						}
						var req struct {
							SessionID string `json:"sessionId"`
						}
						_ = json.Unmarshal(payload, &req)
						val, _ := json.Marshal(map[string]any{
							"type": "snapshot",
							"records": []map[string]any{{
								"type": "event",
								"event": map[string]any{
									"seq":  1,
									"type": "message/user",
									"data": map[string]any{"content": "restored history for " + req.SessionID},
								},
							}},
						})
						writeFrame(dsh.ServerFrame{Type: dsh.FrameItem, StreamID: streamID, Value: val})
					}(f.StreamID, f.Payload)
				}
			}
		}

		var req struct {
			Method string `json:"method"`
			RPCID  string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		method := req.Method
		if method == "" && strings.HasPrefix(r.URL.Path, "/api/") {
			method = strings.TrimPrefix(r.URL.Path, "/api/")
		}
		var value any = map[string]any{}
		switch method {
		case "workspace/initializeDefault":
			value = map[string]any{"workspaceId": "ws-default", "path": h.workspace}
		case "session/list":
			if h.listFail {
				http.Error(w, "list unavailable", http.StatusBadRequest)
				return
			}
			value = map[string]any{
				"items": []map[string]any{
					{"sessionId": "auto-first", "cwd": h.workspace, "updatedAt": 100},
					{"sessionId": "saved-target", "cwd": h.workspace, "updatedAt": 90},
				},
			}
		case "settings/describe":
			select {
			case h.descStarted <- struct{}{}:
			default:
			}
			select {
			case <-h.releaseDesc:
			case <-r.Context().Done():
				return
			}
			value = map[string]any{
				"hasDocument": true,
				"writable":    true,
				"namespaces": []map[string]any{{
					"ns":       "ui-theme",
					"revision": 1,
					"schema":   map[string]any{"type": "object"},
					"value": map[string]any{
						"preference":       "dark",
						"backgroundEffect": "none",
						"wallpaperBlur":    0,
						"wallpaperPixel":   0,
						"transparent":      true,
						"wallpaperImage":   h.wallpaperURL,
					},
				}},
			}
		case "session/initializeDefaultModel":
			value = map[string]any{}
		case "session/modelCatalog":
			value = map[string]any{
				"default": map[string]any{"provider": "deepseek-official", "model": "deepseek-chat"},
				"groups": []map[string]any{{
					"id":   "deepseek-official",
					"name": "DeepSeek",
					"models": []map[string]any{{
						"id":   "deepseek-chat",
						"name": "DeepSeek Chat",
					}},
				}},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  req.RPCID,
			"result": map[string]any{"ok": true, "value": value},
		})
	}))
}

func waitBootCondition(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func setupBootHandoffState(t *testing.T, host *bootHandoffFakeHost) (*mainState, *session.Store, *settingsstore.Store, *dsh.Client, *media.Service) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("WHALEISLE_ISOLATED", "1")
	t.Setenv(paths.EnvUserData, dir)

	prevTheme := theme.CurrentPreferences()
	prevMinimal := renderpolicy.Minimal()
	prevGlass := renderpolicy.Glass()
	renderpolicy.SetMinimal(false)
	renderpolicy.SetGlass(true)
	svc := media.Init(media.Config{Workers: -1})
	t.Cleanup(func() {
		media.Init(media.Config{})
		theme.SetPreferences(prevTheme)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		renderpolicy.SetMinimal(prevMinimal)
		renderpolicy.SetGlass(prevGlass)
	})

	cfg, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Update(func(c *config.Config) {
		c.Workspace = host.workspace
		c.Theme = "dark"
		c.WallpaperPath = ""
	}); err != nil {
		t.Fatal(err)
	}

	srv := host.serveHTTP(t)
	t.Cleanup(srv.Close)

	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
	t.Cleanup(client.Close)
	mux := &dsh.Mux{Endpoint: client.Endpoint, HTTP: client.HTTP}
	t.Cleanup(func() { _ = mux.Close() })

	sessions := session.New()
	setStore := settingsstore.New()
	s := &mainState{
		cfgStore:             cfg,
		sessions:             sessions,
		settings:             setStore,
		dshMuxReady:          true,
		bootStatus:           harness.Status{State: harness.Ready},
		coldLoading:          true,
		themeID:              startupTheme(cfg.Get(), "dark", true),
		resumePendingSession: "saved-target",
		resumeWorkspace:      host.workspace,
	}
	s.seedDesktopDraft()

	sessions.Attach(client, func() { s.recordStartupWorkspaceReady(sessions) })
	t.Cleanup(sessions.Detach)
	sessions.AttachMux(mux)
	sessions.SetWorkspaceCwd(host.workspace)
	setStore.Attach(client, nil)
	t.Cleanup(setStore.Detach)
	return s, sessions, setStore, client, svc
}

func TestColdLoadingStaysUpUntilWorkbenchBaseline(t *testing.T) {
	ws := t.TempDir()
	host := newBootHandoffFakeHost(ws, "", false, false)
	s, sessions, _, client, _ := setupBootHandoffState(t, host)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner := sessions.BeginEvents(ctx, client)
	sessions.FollowWorkspace(ctx)
	waitBootCondition(t, "workspace ready before baseline", func() bool {
		snap := sessions.Snapshot()
		return snap.ClientReady && snap.WorkspaceReady && !snap.ListLoading
	})
	if sessions.WorkbenchBaseline() {
		t.Fatal("expected WorkbenchBaseline() == false before InitializeDefaultWorkspace and OnEventsReady")
	}

	s.recordStartupWorkspaceReady(sessions)
	if !s.coldLoading {
		t.Fatal("recordStartupWorkspaceReady lowered coldLoading before WorkbenchBaseline()")
	}
	if !s.bootPageCovers() {
		t.Fatal("bootPageCovers() turned false before WorkbenchBaseline()")
	}

	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		if s.paintColdLoading(c) {
			return
		}
		ui.Box(c).Fill().Background(ui.RGB(255, 0, 0))
	}, 240, 140)
	tt.SetTime(start)
	if !s.coldLoading {
		t.Fatal("paintColdLoading lowered coldLoading while WorkspaceReady && !ListLoading lacked WorkbenchBaseline()")
	}

	// Now complete the authoritative events generation baseline.
	sessions.InitializeDefaultWorkspace(ctx)
	sessions.OnEventsReady(owner)
	waitBootCondition(t, "WorkbenchBaseline", sessions.WorkbenchBaseline)
	s.recordStartupWorkspaceReady(sessions)
	if s.bootPageCovers() {
		t.Fatal("bootPageCovers() stayed true after WorkbenchBaseline armed cold handoff")
	}
}

func TestColdHandoffWaitsForHostSettingsWallpaperAndHistoryThenRevealsOnFirstFrame(t *testing.T) {
	ws := t.TempDir()
	wallURL := bootHandoffWallpaperDataURL(t, color.RGBA{R: 30, G: 110, B: 235, A: 255})
	host := newBootHandoffFakeHost(ws, wallURL, true, true)
	s, sessions, setStore, client, svc := setupBootHandoffState(t, host)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner := sessions.BeginEvents(ctx, client)
	sessions.FollowWorkspace(ctx)
	setStore.EnsureLoaded(ctx)
	sessions.InitializeDefaultWorkspace(ctx)
	sessions.OnEventsReady(owner)
	waitBootCondition(t, "WorkbenchBaseline", sessions.WorkbenchBaseline)

	var firstWorkbenchFrame struct {
		seen                 bool
		hostAccepted         bool
		hostWallpaper        string
		selectedID           string
		historyReady         bool
		wallpaperReadyOnDraw bool
		revealActive         bool
	}

	start := time.Unix(2000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		if s.paintColdLoading(c) {
			return
		}
		defer s.paintBootReveal(c)
		s.syncHostAppearance()
		s.restoreResumeSelection()
		draw := s.wallpaperPainter(s.currentWallpaperConfigPath(), true, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(draw)
		if !firstWorkbenchFrame.seen {
			firstWorkbenchFrame.seen = true
			snap := sessions.Snapshot()
			firstWorkbenchFrame.hostAccepted = s.hostAppearanceAccepted
			firstWorkbenchFrame.hostWallpaper = s.hostWallpaperImage
			firstWorkbenchFrame.selectedID = snap.SelectedID
			firstWorkbenchFrame.historyReady = snap.HistoryReady && !snap.HistoryLoading
			prefs := wallpapers.Defaults()
			if g := s.wallpaperEditor(); g != nil {
				prefs = g.EffectivePreferences()
			}
			firstWorkbenchFrame.wallpaperReadyOnDraw = chrome.PrewarmWallpaper(s.resolvedWallpaperPathFor(s.currentWallpaperConfigPath(), prefs), prefs, 240, 140)
		}
	}, 240, 140)
	tt.SetTime(start)

	// Gate 1: WorkbenchBaseline reached, but Host settings not described and
	// resumed session history still loading -> boot page must stay up.
	if !s.coldLoading || firstWorkbenchFrame.seen {
		t.Fatalf("handoff fired before Host settings/history settled: coldLoading=%v seen=%v", s.coldLoading, firstWorkbenchFrame.seen)
	}
	if s.bootReveal.Active() {
		t.Fatal("bootReveal started while coldLoading was still true")
	}
	if got := sessions.SelectedSessionID(); got != "saved-target" {
		t.Fatalf("restoreResumeSelection did not select saved-target while boot page was up: got %q", got)
	}

	// Release Host settings/describe and session/follow history, while keeping
	// wallpaper decode queued (media.Pending).
	close(host.releaseDesc)
	close(host.releaseFollow)
	waitBootCondition(t, "settings described", func() bool { return setStore.Snapshot().Value.Described })
	waitBootCondition(t, "resumed history ready", func() bool {
		snap := sessions.Snapshot()
		return snap.SelectedID == "saved-target" && snap.HistoryReady && !snap.HistoryLoading
	})

	// Gate 2: settings described + history ready, wallpaper decode still Pending.
	tt.SetTime(start.Add(100 * time.Millisecond))
	if !s.coldLoading || firstWorkbenchFrame.seen {
		t.Fatalf("handoff fired while wallpaper decode was still Pending: coldLoading=%v seen=%v", s.coldLoading, firstWorkbenchFrame.seen)
	}
	if !s.hostAppearanceAccepted || s.hostWallpaperImage != wallURL {
		t.Fatalf("Host appearance not applied during handoff wait: accepted=%v wall=%q", s.hostAppearanceAccepted, s.hostWallpaperImage)
	}
	if svc.Queued() == 0 {
		t.Fatal("PrewarmWallpaper did not enqueue the Host wallpaper decode")
	}

	// Decode wallpaper bitmap + colorMap synchronously -> next frame must hand
	// off seamlessly on the SAME frame and trigger bootReveal.
	svc.RunPending()
	tt.SetTime(start.Add(200 * time.Millisecond))
	firstWorkbenchFrame.revealActive = s.bootReveal.Active()

	if s.coldLoading {
		t.Fatal("coldLoading stayed true after settings, wallpaper and history all settled")
	}
	if !firstWorkbenchFrame.seen {
		t.Fatal("first workbench frame did not render on the handoff frame")
	}
	if !firstWorkbenchFrame.hostAccepted || firstWorkbenchFrame.hostWallpaper != wallURL {
		t.Fatalf("first workbench frame missing Host appearance: %+v", firstWorkbenchFrame)
	}
	if !firstWorkbenchFrame.wallpaperReadyOnDraw {
		t.Fatal("first workbench frame did not have prewarmed wallpaper ready")
	}
	if firstWorkbenchFrame.selectedID != "saved-target" || !firstWorkbenchFrame.historyReady {
		t.Fatalf("first workbench frame missing resumed session history: %+v", firstWorkbenchFrame)
	}
	if !firstWorkbenchFrame.revealActive {
		t.Fatal("bootReveal did not trigger on the seamless handoff frame")
	}

	// Once bootReveal finishes, the underlying pixels must show the blue Host
	// wallpaper (not pure black).
	tt.SetTime(start.Add(200*time.Millisecond + boot.RevealDuration))
	if s.bootReveal.Active() {
		t.Fatal("bootReveal stayed active past RevealDuration")
	}
	px := tt.Image().RGBAAt(120, 70)
	if px.B < 150 || px.R > 80 {
		t.Fatalf("workbench backdrop after reveal is not the decoded Host wallpaper: %+v", px)
	}
}

func TestColdHandoffTimeoutReleasesUnderControlledClock(t *testing.T) {
	ws := t.TempDir()
	wallURL := bootHandoffWallpaperDataURL(t, color.RGBA{R: 30, G: 110, B: 235, A: 255})
	// Keep settings/describe and session/follow permanently blocked to verify
	// the hard timeout fallback via c.After / controlled clock.
	host := newBootHandoffFakeHost(ws, wallURL, true, true)
	t.Cleanup(func() {
		close(host.releaseDesc)
		close(host.releaseFollow)
	})
	s, sessions, setStore, client, _ := setupBootHandoffState(t, host)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner := sessions.BeginEvents(ctx, client)
	sessions.FollowWorkspace(ctx)
	setStore.EnsureLoaded(ctx)
	sessions.InitializeDefaultWorkspace(ctx)
	sessions.OnEventsReady(owner)
	waitBootCondition(t, "WorkbenchBaseline", sessions.WorkbenchBaseline)

	workbenchPainted := false
	start := time.Unix(3000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		if s.paintColdLoading(c) {
			return
		}
		defer s.paintBootReveal(c)
		workbenchPainted = true
		ui.Box(c).Fill().Background(ui.RGB(20, 20, 20))
	}, 240, 140)

	tt.SetTime(start)
	if !s.coldLoading || workbenchPainted {
		t.Fatal("handoff fired immediately despite blocked settings/history")
	}

	tt.SetTime(start.Add(coldHandoffTimeout - time.Millisecond))
	if !s.coldLoading || workbenchPainted {
		t.Fatal("handoff fired before coldHandoffTimeout elapsed")
	}

	tt.SetTime(start.Add(coldHandoffTimeout))
	if s.coldLoading || !workbenchPainted {
		t.Fatalf("handoff did not release at coldHandoffTimeout: coldLoading=%v painted=%v", s.coldLoading, workbenchPainted)
	}
	if !s.bootReveal.Active() {
		t.Fatal("bootReveal did not trigger on timeout handoff")
	}
}

func TestColdHandoffErrorStateReleasesImmediately(t *testing.T) {
	ws := t.TempDir()
	host := newBootHandoffFakeHost(ws, "", true, true)
	host.listFail = true
	t.Cleanup(func() {
		close(host.releaseDesc)
		close(host.releaseFollow)
	})
	s, sessions, _, client, _ := setupBootHandoffState(t, host)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner := sessions.BeginEvents(ctx, client)
	sessions.OnEventsReady(owner)
	waitBootCondition(t, "list error", func() bool { return sessions.Snapshot().ListErr != nil })

	workbenchPainted := false
	tt := ui.NewTester(func(c *ui.Context) {
		if s.paintColdLoading(c) {
			return
		}
		workbenchPainted = true
		ui.Box(c).Fill().Background(ui.RGB(20, 20, 20))
	}, 240, 140)
	tt.SetTime(time.Unix(4000, 0))

	if s.coldLoading || !workbenchPainted {
		t.Fatalf("error state did not immediately release coldLoading: coldLoading=%v painted=%v", s.coldLoading, workbenchPainted)
	}
}
