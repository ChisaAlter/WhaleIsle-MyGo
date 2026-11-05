package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/paths"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
)

type viewTabsTransport func(*http.Request) (*http.Response, error)

func (f viewTabsTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type viewTabsHost struct {
	mu                     sync.Mutex
	field, writable, value bool
	namespace              string
	revision               int
	writes                 []bool
	describeGate           <-chan struct{}
	readErr, writeErr      error
}

func (h *viewTabsHost) client(origin string) *dsh.Client {
	return &dsh.Client{Endpoint: dsh.Endpoint{Origin: origin}, HTTP: &http.Client{Transport: viewTabsTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return nil, err
		}
		h.mu.Lock()
		gate := h.describeGate
		h.mu.Unlock()
		if request["method"] == "settings/describe" && gate != nil {
			<-gate
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.namespace == "" {
			h.namespace = "ui-conversation"
		}
		out := func() map[string]any {
			fields, values := map[string]any{}, map[string]any{}
			if h.field {
				fields["viewTabs"], values["viewTabs"] = 1, h.value
			}
			return map[string]any{"ns": h.namespace, "revision": h.revision, "value": values,
				"schema": map[string]any{"uid": 2, "refs": map[string]any{"1": map[string]any{"type": "boolean"}, "2": map[string]any{"type": "object", "dict": fields}}}}
		}
		var value any
		switch request["method"] {
		case "settings/describe":
			if h.readErr != nil {
				return nil, h.readErr
			}
			value = map[string]any{"writable": h.writable, "namespaces": []any{out()}}
		case "settings/mutate":
			if h.writeErr != nil {
				return nil, h.writeErr
			}
			args := request["payload"].(map[string]any)["args"].(map[string]any)
			if args["ns"] != h.namespace {
				return nil, errors.New("wrong view-tabs namespace")
			}
			for _, raw := range args["ops"].([]any) {
				op := raw.(map[string]any)
				if path := op["path"].([]any); len(path) != 1 || path[0] != "viewTabs" {
					return nil, errors.New("unexpected settings field")
				}
				h.value = op["value"].(bool)
				h.writes = append(h.writes, h.value)
			}
			h.revision++
			value = out()
		case "settings/update":
			args := request["payload"].(map[string]any)["args"].(map[string]any)
			if args["ns"] != h.namespace {
				return nil, errors.New("wrong update namespace")
			}
			h.value = args["patch"].(map[string]any)["viewTabs"].(bool)
			h.writes = append(h.writes, h.value)
			h.revision++
			value = out()
		default:
			return nil, errors.New("unexpected mock settings request")
		}
		recorder := httptest.NewRecorder()
		_ = json.NewEncoder(recorder).Encode(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": value}})
		return recorder.Result(), nil
	})}}
}

func waitViewTabsSettings(t *testing.T, st *settingsstore.Store, changed <-chan struct{}) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for {
		snap := st.Snapshot()
		// Immediate Host preferences have their own namespace queue; they do
		// not block unrelated form editors through the global Saving flag.
		if !snap.Loading && !snap.Saving && len(snap.PreferencePending) == 0 {
			return
		}
		select {
		case <-changed:
		case <-deadline.C:
			t.Fatalf("mock settings did not settle: %+v", st.Snapshot())
		}
	}
}

func viewTabsSettingsState(t *testing.T, h *viewTabsHost) (*mainState, chan struct{}) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("WHALEISLE_ISOLATED", "1")
	t.Setenv(paths.EnvUserData, dir)
	cfg, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Update(func(c *config.Config) { c.Locale = "en"; c.WallpaperPath = "" }); err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg, settings: settingsstore.New()}
	s.seedDesktopDraft()
	// A current Host-owned appearance avoids unrelated main-thread OS theme
	// effects. Its accepted empty wallpaper makes materialization a true no-op.
	themeStore := settingsstore.New()
	themeChanged := make(chan struct{}, 8)
	themeStore.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://owned-appearance.invalid"}, HTTP: &http.Client{Transport: viewTabsTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return nil, err
		}
		if request["method"] != "settings/describe" {
			return nil, errors.New("view tabs test attempted appearance mutation")
		}
		recorder := httptest.NewRecorder()
		value := map[string]any{"writable": true, "namespaces": []any{map[string]any{"ns": "ui-theme", "revision": 1, "value": map[string]any{"wallpaperImage": ""}, "schema": json.RawMessage(`{"uid":2,"refs":{"1":{"type":"string"},"2":{"type":"object","dict":{"wallpaperImage":1}}}}`)}}}
		_ = json.NewEncoder(recorder).Encode(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": value}})
		return recorder.Result(), nil
	})}}, func() {
		select {
		case themeChanged <- struct{}{}:
		default:
		}
	})
	themeStore.RefreshDescribe(context.Background())
	waitViewTabsSettings(t, themeStore, themeChanged)
	s.hostAppearance = settings.BindHostTheme(themeStore)
	s.hostAppearanceAccepted = true
	if s.hostAppearance == nil {
		t.Fatal("isolated appearance binding unavailable")
	}
	changed := make(chan struct{}, 32)
	if h != nil {
		s.settings.Attach(h.client("http://settings-owner-a.invalid"), func() {
			select {
			case changed <- struct{}{}:
			default:
			}
		})
		s.settings.RefreshDescribe(context.Background())
		waitViewTabsSettings(t, s.settings, changed)
		if snap := s.settings.Snapshot(); snap.Err != "" {
			t.Fatalf("mock description=%+v", snap)
		}
	}
	return s, changed
}

func viewTabsSettingsModel(s *mainState) settings.Model {
	m := settings.Model{Store: s.settings, InterfaceActive: true, ThemeID: "deepseek.light", Interface: &settings.InterfaceAccess{
		ShowViewTabs: &s.desktopDraft.showViewTabs, ShowArchivedList: &s.desktopDraft.showArchivedList, OnSave: s.saveInterfacePreferences,
	}}
	s.wireSettingsBodies(&m)
	return m
}

func TestViewTabsSettingsFallbackSavesAndAffectsConversation(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, connected := range []bool{false, true} {
		t.Run(map[bool]string{false: "disconnected", true: "described-without-field"}[connected], func(t *testing.T) {
			var h *viewTabsHost
			if connected {
				h = &viewTabsHost{writable: true}
			}
			s, changed := viewTabsSettingsState(t, h)
			m := viewTabsSettingsModel(s)
			if !settings.InterfaceViewTabsFallbackAllowed(m.Interface.Host) {
				t.Fatal("compatibility fallback missing")
			}
			s.desktopDraft.showViewTabs = false
			m.Interface.OnSave()
			waitViewTabsSettings(t, s.settings, changed)
			if s.desktopSaveErr != "" || !s.cfgStore.Get().HideChatViewTabs || !s.conversationViewTabsHidden(s.settings.Snapshot()) {
				t.Fatalf("fallback failed: %s", s.desktopSaveErr)
			}
			if h != nil && len(h.writes) != 0 {
				t.Fatalf("fallback wrote Host: %v", h.writes)
			}
			reopened, err := config.Open(filepath.Dir(s.cfgStore.Path()))
			if err != nil || !reopened.Get().HideChatViewTabs {
				t.Fatal("fallback did not persist", err)
			}
		})
	}
}

func TestViewTabsSettingsOldCallbacksCannotWriteReplacementHost(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	a := &viewTabsHost{field: true, writable: true, value: true, revision: 1}
	s, changed := viewTabsSettingsState(t, a)
	old := viewTabsSettingsModel(s)
	b := &viewTabsHost{field: true, writable: true, value: false, revision: 2}
	s.settings.Attach(b.client("http://settings-owner-b.invalid"), func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	before := viewTabsSettingsModel(s)
	if before.Interface.Host.Snapshot.Described || settings.InterfaceViewTabsFallbackAllowed(before.Interface.Host) {
		t.Fatal("replacement inherited description or local authority")
	}
	old.Interface.Host.OnChange("ui-conversation", "viewTabs", "true")
	s.settings.RefreshDescribe(context.Background())
	waitViewTabsSettings(t, s.settings, changed)
	old.Interface.Host.OnChange("ui-conversation", "viewTabs", "true")
	s.desktopDraft.showViewTabs = true
	old.Interface.OnSave()
	if len(a.writes) != 0 || len(b.writes) != 0 || !s.cfgStore.Get().HideChatViewTabs {
		t.Fatal("old callback retargeted replacement Host or stale local mirror", a.writes, b.writes)
	}
	if !s.conversationViewTabsHidden(s.settings.Snapshot()) {
		t.Fatal("accepted replacement false did not control conversation")
	}
}

func TestViewTabsSettingsFailedWriteKeepsAcceptedValue(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	h := &viewTabsHost{field: true, writable: true, value: false, revision: 1, writeErr: errors.New("fixture rejected mutation")}
	s, changed := viewTabsSettingsState(t, h)
	m := viewTabsSettingsModel(s)
	m.Interface.Host.OnChange("ui-conversation", "viewTabs", "true")
	waitViewTabsSettings(t, s.settings, changed)
	// A successful recovery describe may clear the general request error, but
	// the rejected preference remains owned by this field until a real retry.
	if got := s.settings.Snapshot().PreferenceErrors["ui-conversation.viewTabs"]; !strings.Contains(got, h.writeErr.Error()) {
		t.Fatalf("fixture refusal was not retained for viewTabs: %q", got)
	}
	s.desktopDraft.showViewTabs = true
	m.Interface.OnSave()
	if !s.cfgStore.Get().HideChatViewTabs || len(h.writes) != 0 || !s.conversationViewTabsHidden(s.settings.Snapshot()) {
		t.Fatal("failed write or stale draft replaced accepted false")
	}
}

func TestViewTabsSettingsUnresolvedAndUnsupportedCannotCommitHiddenDraft(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, mode := range []string{"pending", "loading", "read-error", "unsupported", "read-only"} {
		t.Run(mode, func(t *testing.T) {
			s, changed := viewTabsSettingsState(t, nil)
			h := &viewTabsHost{field: true, writable: true, value: false, revision: 1}
			client := h.client("http://incomplete-description.invalid")
			if mode == "unsupported" {
				client.HTTP = &http.Client{Transport: viewTabsTransport(func(r *http.Request) (*http.Response, error) {
					var request map[string]any
					_ = json.NewDecoder(r.Body).Decode(&request)
					recorder := httptest.NewRecorder()
					value := map[string]any{"writable": true, "namespaces": []any{map[string]any{"ns": "ui-conversation", "value": map[string]any{"viewTabs": "false"}, "schema": json.RawMessage(`{"uid":2,"refs":{"1":{"type":"string"},"2":{"type":"object","dict":{"viewTabs":1}}}}`)}}}
					_ = json.NewEncoder(recorder).Encode(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": value}})
					return recorder.Result(), nil
				})}
			}
			var gate chan struct{}
			if mode == "loading" {
				gate = make(chan struct{})
				h.describeGate = gate
			}
			if mode == "read-error" {
				h.readErr = errors.New("fixture denied describe")
			}
			if mode == "read-only" {
				h.writable = false
			}
			s.settings.Attach(client, func() {
				select {
				case changed <- struct{}{}:
				default:
				}
			})
			if mode != "pending" {
				s.settings.RefreshDescribe(context.Background())
				if gate == nil {
					waitViewTabsSettings(t, s.settings, changed)
				}
			}
			m := viewTabsSettingsModel(s)
			if settings.InterfaceViewTabsFallbackAllowed(m.Interface.Host) {
				t.Fatal("unresolved or described field exposed local fallback")
			}
			s.desktopDraft.showViewTabs = false
			if mode == "read-only" {
				s.desktopDraft.showViewTabs = true
			}
			m.Interface.OnSave()
			wantHidden := mode == "read-only"
			if s.cfgStore.Get().HideChatViewTabs != wantHidden || s.conversationViewTabsHidden(s.settings.Snapshot()) != wantHidden {
				t.Fatalf("hidden draft gained authority: mode=%s", mode)
			}
			if gate != nil {
				close(gate)
				waitViewTabsSettings(t, s.settings, changed)
			}
			if len(h.writes) != 0 {
				t.Fatal("non-writable owner caused a Host write", h.writes)
			}
		})
	}
}

func TestViewTabsSettingsSaveDoesNotRevertAcceptedHostValue(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	h := &viewTabsHost{field: true, writable: true, value: true, revision: 3}
	s, changed := viewTabsSettingsState(t, h)
	m := viewTabsSettingsModel(s)
	m.Interface.Host.OnChange("ui-conversation", "viewTabs", "false")
	waitViewTabsSettings(t, s.settings, changed)
	if on, ok := s.settings.ViewTabsEnabled(); !ok || on {
		t.Fatalf("Host switch not accepted: %v/%v", on, ok)
	}
	if !s.desktopDraft.showViewTabs {
		t.Fatal("fixture did not retain the stale local mirror")
	}
	s.desktopDraft.showArchivedList = !s.desktopDraft.showArchivedList
	synctest.Test(t, func(t *testing.T) {
		m.Interface.OnSave()
		synctest.Wait()
		if on, ok := s.settings.ViewTabsEnabled(); !ok || on {
			t.Fatalf("unrelated local Save reverted Host viewTabs: on=%v known=%v writes=%v", on, ok, h.writes)
		}
		if len(h.writes) != 1 || h.writes[0] {
			t.Fatalf("local Save wrote Host preference: %v", h.writes)
		}
	})
	if s.cfgStore.Get().HideArchivedList != !s.desktopDraft.showArchivedList {
		t.Fatal("unrelated local save did not commit")
	}
}
