package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/dsh"
	"whaleisle/internal/paths"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

// TestWallpaperSourcePath pins which value the window backdrop paints. The
// seeded fixture field (goWallpaperPath / configPath) must survive an empty Host
// appearance: the Host theme arrives with wallpaperImage "" on every profile
// that never set one, and preferring that empty value dropped the seeded image
// as soon as the harness connected.
func TestWallpaperSourcePath(t *testing.T) {
	const cfgPath = `C:\tmp\wall.png`
	const hostData = "data:image/png;base64,AAAA"
	for _, tc := range []struct {
		name       string
		hostAccept bool
		hostImage  string
		configPath string
		want       string
	}{
		{"host appearance not accepted yet", false, "", cfgPath, cfgPath},
		{"host accepted but empty: keep the seeded config", true, "", cfgPath, cfgPath},
		{"host accepted with whitespace: keep the seeded config", true, "   ", cfgPath, cfgPath},
		{"host wallpaper wins", true, hostData, cfgPath, hostData},
		{"host only", true, hostData, "", hostData},
		{"nothing set", true, "", "", ""},
		{"trimmed config path", false, "", "  " + cfgPath + "  ", cfgPath},
	} {
		if got := wallpaperSourcePath(tc.hostAccept, tc.hostImage, tc.configPath); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestWallpaperPainterUsesAcceptedHostBackground(t *testing.T) {
	useGlassTheme(t)
	t.Cleanup(func() { chrome.ConfigureBackdrop("", wallpapers.Defaults()) })
	for _, localNone := range []bool{false, true} {
		name := "missing local preference"
		if localNone {
			name = "saved local none before Host adoption"
		}
		t.Run(name, func(t *testing.T) {
			cfg, err := config.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if localNone {
				if err := cfg.SaveWallpaperBackground(wallpapers.Defaults()); err != nil {
					t.Fatal(err)
				}
			}
			prefs := wallpapers.Defaults()
			prefs.Effect, prefs.EffectExplicit = "gradient", true
			s := &mainState{cfgStore: cfg, hostAppearanceAccepted: true, wallpaperControls: &settings.WallpaperEditor{Prefs: prefs, Saved: prefs}}
			for _, effect := range []string{"gradient", "none", "gradient"} {
				s.wallpaperControls.Saved.Effect = effect
				s.wallpaperPainter("", true, true)
				if got, want := chrome.BackdropActive(), effect == "gradient"; got != want {
					t.Fatalf("accepted Host %s painted backdrop active=%v", effect, got)
				}
			}
			// A settings draft is previewed only in its sandbox until Save.
			s.wallpaperControls.Prefs.Effect = "none"
			s.wallpaperPainter("", true, true)
			if !chrome.BackdropActive() {
				t.Fatal("unfinished effect draft changed the global renderer")
			}
			if cfg.HasWallpaperBackground() != localNone {
				t.Fatal("rendering changed local appearance storage")
			}
		})
	}
}

func TestWallpaperPainterKeepsOfflineFlatFallback(t *testing.T) {
	t.Cleanup(func() { chrome.ConfigureBackdrop("", wallpapers.Defaults()) })
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg}
	s.wallpaperPainter("", true, true)
	if chrome.BackdropActive() || cfg.HasWallpaperBackground() {
		t.Fatal("offline first run seeded a backdrop")
	}
	prefs := wallpapers.Defaults()
	prefs.EffectExplicit = true
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}
	s.wallpaperControls = nil
	s.wallpaperPainter("", true, true)
	if chrome.BackdropActive() || s.wallpaperControls.Prefs.Effect != "none" || !s.wallpaperControls.Prefs.EffectExplicit {
		t.Fatal("offline saved none was not preserved")
	}
}

type wallpaperMockTransport func(*http.Request) (*http.Response, error)

func (f wallpaperMockTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type wallpaperThemeHost struct {
	mu             sync.Mutex
	writable       bool
	revision       int
	wallpaperImage string
	writes         []string
}

func (h *wallpaperThemeHost) setWallpaper(image string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.wallpaperImage = image
	h.revision++
}

func (h *wallpaperThemeHost) client() *dsh.Client {
	return &dsh.Client{
		Endpoint: dsh.Endpoint{Origin: "http://wallpaper-theme-host.invalid"},
		HTTP: &http.Client{Transport: wallpaperMockTransport(func(r *http.Request) (*http.Response, error) {
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, err
			}
			h.mu.Lock()
			defer h.mu.Unlock()
			out := func() map[string]any {
				return map[string]any{
					"ns":       "ui-theme",
					"revision": h.revision,
					"value":    map[string]any{"wallpaperImage": h.wallpaperImage},
					"schema":   json.RawMessage(`{"uid":2,"refs":{"1":{"type":"string"},"2":{"type":"object","dict":{"wallpaperImage":1}}}}`),
				}
			}
			var value any
			switch request["method"] {
			case "settings/describe":
				value = map[string]any{"writable": h.writable, "namespaces": []any{out()}}
			case "settings/mutate":
				if !h.writable {
					return nil, errors.New("read-only host appearance")
				}
				args := request["payload"].(map[string]any)["args"].(map[string]any)
				for _, raw := range args["ops"].([]any) {
					op := raw.(map[string]any)
					if path := op["path"].([]any); len(path) == 1 && path[0] == "wallpaperImage" {
						v, _ := op["value"].(string)
						h.wallpaperImage = v
						h.writes = append(h.writes, v)
					}
				}
				h.revision++
				value = out()
			case "settings/update":
				if !h.writable {
					return nil, errors.New("read-only host appearance")
				}
				args := request["payload"].(map[string]any)["args"].(map[string]any)
				if v, ok := args["patch"].(map[string]any)["wallpaperImage"].(string); ok {
					h.wallpaperImage = v
					h.writes = append(h.writes, v)
				}
				h.revision++
				value = out()
			default:
				return nil, errors.New("unexpected mock settings request")
			}
			recorder := httptest.NewRecorder()
			_ = json.NewEncoder(recorder).Encode(map[string]any{
				"type":   "server-response",
				"rpcId":  request["rpcId"],
				"result": map[string]any{"ok": true, "value": value},
			})
			return recorder.Result(), nil
		})},
	}
}

func waitWallpaperHostStore(t *testing.T, st *settingsstore.Store, changed <-chan struct{}) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for {
		snap := st.Snapshot().Value
		if !snap.Loading && !snap.Saving && len(snap.PreferencePending) == 0 {
			return
		}
		select {
		case <-changed:
		case <-deadline.C:
			t.Fatalf("mock settings did not settle: %+v", st.Snapshot().Value)
		}
	}
}

func newWallpaperHostState(t *testing.T, host *wallpaperThemeHost, localWallpaper string) (*mainState, *config.Store, chan struct{}) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("WHALEISLE_ISOLATED", "1")
	t.Setenv(paths.EnvUserData, dir)
	prevTheme := theme.CurrentPreferences()
	t.Cleanup(func() {
		theme.SetPreferences(prevTheme)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
	})
	cfg, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Update(func(c *config.Config) { c.WallpaperPath = localWallpaper }); err != nil {
		t.Fatal(err)
	}
	store := settingsstore.New()
	changed := make(chan struct{}, 16)
	store.Attach(host.client(), func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	store.RefreshDescribe(context.Background())
	waitWallpaperHostStore(t, store, changed)
	s := &mainState{cfgStore: cfg, settings: store}
	s.seedDesktopDraft()
	// Match the initial scheme so syncHostAppearance does not call mygo.onMain
	// when running without a live UI event loop.
	s.themeID = startupTheme(cfg.Get(), "", true)
	return s, cfg, changed
}

func TestSyncHostAppearancePreservesLocalWallpaperOnEmptyHostDefault(t *testing.T) {
	const localWall = `C:\tmp\seeded-local.png`
	const hostWall = "data:image/png;base64,BBBB"

	host := &wallpaperThemeHost{writable: true, revision: 1, wallpaperImage: ""}
	s, _, changed := newWallpaperHostState(t, host, localWall)

	if s.desktopDraft.wallpaperPath != localWall {
		t.Fatalf("seedDesktopDraft() wallpaperPath = %q, want %q", s.desktopDraft.wallpaperPath, localWall)
	}

	// Host startup sends WallpaperImageExplicit=true with empty default "".
	s.syncHostAppearance()
	if !s.hostAppearanceAccepted {
		t.Fatal("expected hostAppearanceAccepted = true")
	}
	if s.hostWallpaperImage != "" {
		t.Fatalf("hostWallpaperImage = %q, want empty", s.hostWallpaperImage)
	}
	if s.desktopDraft.wallpaperPath != localWall {
		t.Fatalf("syncHostAppearance wiped local wallpaperPath: got %q, want %q", s.desktopDraft.wallpaperPath, localWall)
	}
	if got := s.resolvedWallpaperPath(s.desktopDraft.wallpaperPath); got != localWall {
		t.Fatalf("resolvedWallpaperPath = %q, want %q", got, localWall)
	}
	if got := s.resolvedWallpaperPathFor(s.desktopDraft.wallpaperPath, wallpapers.Defaults()); got != localWall {
		t.Fatalf("resolvedWallpaperPathFor = %q, want %q", got, localWall)
	}

	// When Host publishes a non-empty wallpaperImage, desktopDraft adopts it.
	host.setWallpaper(hostWall)
	s.settings.RefreshDescribe(context.Background())
	waitWallpaperHostStore(t, s.settings, changed)
	s.syncHostAppearance()
	if s.hostWallpaperImage != hostWall || s.desktopDraft.wallpaperPath != hostWall {
		t.Fatalf("non-empty Host wallpaper not adopted: host=%q draft=%q", s.hostWallpaperImage, s.desktopDraft.wallpaperPath)
	}
	if got := s.resolvedWallpaperPathFor(s.desktopDraft.wallpaperPath, wallpapers.Defaults()); got != hostWall {
		t.Fatalf("resolvedWallpaperPathFor with host wallpaper = %q, want %q", got, hostWall)
	}

	// When Host explicitly clears a previously non-empty wallpaperImage back to "",
	// desktopDraft clears as well.
	host.setWallpaper("")
	s.settings.RefreshDescribe(context.Background())
	waitWallpaperHostStore(t, s.settings, changed)
	s.syncHostAppearance()
	if s.hostWallpaperImage != "" || s.desktopDraft.wallpaperPath != "" {
		t.Fatalf("Host clear from non-empty did not clear draft: host=%q draft=%q", s.hostWallpaperImage, s.desktopDraft.wallpaperPath)
	}
}

func TestWallpaperEditorApplyClearClearsHostAndLocalWallpaper(t *testing.T) {
	const localWall = `C:\tmp\seeded-local.png`
	const hostWall = "data:image/png;base64,CCCC"

	t.Run("empty-host-default-with-local-config", func(t *testing.T) {
		host := &wallpaperThemeHost{writable: true, revision: 1, wallpaperImage: ""}
		s, cfg, _ := newWallpaperHostState(t, host, localWall)
		s.syncHostAppearance()

		g := s.wallpaperEditor()
		if g == nil {
			t.Fatal("wallpaperEditor() = nil")
		}
		if err := g.Apply(""); err != nil {
			t.Fatalf("Apply(\"\") error = %v", err)
		}
		if got := cfg.Get().WallpaperPath; got != "" {
			t.Fatalf("cfg.WallpaperPath = %q, want empty", got)
		}
		if s.desktopDraft.wallpaperPath != "" || s.hostWallpaperImage != "" {
			t.Fatalf("draft/host not cleared: draft=%q host=%q", s.desktopDraft.wallpaperPath, s.hostWallpaperImage)
		}
		if got := s.resolvedWallpaperPathFor(s.desktopDraft.wallpaperPath, wallpapers.Defaults()); got != "" {
			t.Fatalf("resolvedWallpaperPathFor after clear = %q, want empty", got)
		}
	})

	t.Run("non-empty-host-with-local-config", func(t *testing.T) {
		host := &wallpaperThemeHost{writable: true, revision: 1, wallpaperImage: hostWall}
		s, cfg, _ := newWallpaperHostState(t, host, localWall)
		s.syncHostAppearance()

		g := s.wallpaperEditor()
		if g == nil {
			t.Fatal("wallpaperEditor() = nil")
		}
		if err := g.Apply(""); err != nil {
			t.Fatalf("Apply(\"\") error = %v", err)
		}
		if got := cfg.Get().WallpaperPath; got != "" {
			t.Fatalf("cfg.WallpaperPath = %q, want empty", got)
		}
		if s.desktopDraft.wallpaperPath != "" || s.hostWallpaperImage != "" {
			t.Fatalf("draft/host not cleared: draft=%q host=%q", s.desktopDraft.wallpaperPath, s.hostWallpaperImage)
		}
	})

	t.Run("read-only-host-keeps-local-config-on-error", func(t *testing.T) {
		host := &wallpaperThemeHost{writable: false, revision: 1, wallpaperImage: hostWall}
		s, cfg, _ := newWallpaperHostState(t, host, localWall)
		s.syncHostAppearance()

		g := s.wallpaperEditor()
		if err := g.Apply(""); err == nil {
			t.Fatal("expected error when Host appearance is read-only")
		}
		if got := cfg.Get().WallpaperPath; got != localWall {
			t.Fatalf("cfg.WallpaperPath = %q, want %q on failed clear", got, localWall)
		}
	})
}
