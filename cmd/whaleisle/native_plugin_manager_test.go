package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
	manager "whaleisle/internal/pluginmanager"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

type nativeManagerFixture func(context.Context, string, any, any) error

func (f nativeManagerFixture) Call(ctx context.Context, method string, args, out any) error {
	return f(ctx, method, args, out)
}

func TestNativePluginEventsFollowCapturedInstallAndHost(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	started := make(chan struct{}, 1)
	fixture := nativeManagerFixture(func(ctx context.Context, method string, args, out any) error {
		var value any
		switch method {
		case "pluginInventory/list":
			value = map[string]any{"managementAvailable": true}
		case "pluginManager/listBundles", "pluginManager/listPlugins":
			value = []any{}
		case "pluginManager/registries":
			value = map[string]any{}
		case "pluginManager/inspect":
			value = map[string]any{"status": "accepted", "name": "fixture"}
		case "pluginManager/installBundle":
			started <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
			value = map[string]any{"application": "applied", "changed": true}
		default:
			t.Errorf("unexpected fixture method %s", method)
		}
		if out != nil {
			raw, _ := json.Marshal(value)
			return json.Unmarshal(raw, out)
		}
		return nil
	})
	var s mainState
	c := s.nativePluginController()
	transport := &dsh.Client{}
	s.marketCoordinator().BindNativePluginManagerCaller(transport, fixture)
	defer c.Detach()
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	owner := c.Snapshot().Value.Owner
	emit := s.nativePluginEventHandler(transport)
	a := c.Actions(owner)
	a.OpenInstall()
	a.EditInstall("fixture", "")
	a.Inspect()
	waitNativeManager(t, c, func(v manager.State) bool { return v.Install.Inspection != nil })
	a.RequestInstall()
	a.Confirm()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("fake install did not start")
	}
	id := c.Snapshot().Value.Install.RequestID
	if id == "" {
		t.Fatal("install request has no identity")
	}
	send := func(event string, value any) {
		raw, _ := json.Marshal(value)
		emit(dsh.EventsEmit{Event: event, Args: []json.RawMessage{raw}})
	}
	send("plugin-manager/install-log", map[string]any{"requestId": "other", "jobId": "job", "text": "wrong request"})
	send("plugin-manager/install-state", map[string]any{"requestId": id, "phase": "applying"})
	send("plugin-manager/install-state", map[string]any{"requestId": id, "phase": "installing"})
	send("plugin-manager/install-log", map[string]any{"requestId": id, "jobId": "job", "argv": []string{"pnpm", "install", "fixture"}, "cwd": "C:/fixture", "stream": "stdout", "text": "fixture output"})
	if got := c.Snapshot().Value.Install; got.Phase != "applying" || got.Log != "fixture output" {
		t.Fatalf("native event wire lost owner/phase/output: %+v", got)
	}
	// Replacing the controller Host invalidates the entire old event callback.
	c.Attach(fixture, nil)
	send("plugin-manager/install-log", map[string]any{"requestId": id, "jobId": "job", "text": "stale host"})
	send("plugin-manager/install-state", map[string]any{"requestId": id, "phase": "applying"})
	if got := c.Snapshot().Value.Install; got.Log != "" || got.Phase != "" {
		t.Fatalf("old transport changed replacement Host: %+v", got)
	}
}

func waitNativeManager(t *testing.T, c *manager.Controller, ready func(manager.State) bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ready(c.Snapshot().Value) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("isolated plugin state did not settle")
}

func TestCachedPluginContributionsInvalidatesOnChanges(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	var includeSearch atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasSuffix(r.URL.Path, "/settings/describe") {
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{}}})
			return
		}
		ns := []map[string]any{{
			"ns": "agent-loop", "schema": json.RawMessage(`{"type":"object","properties":{}}`),
			"value": map[string]any{}, "autoGenerate": true, "applies": "live", "secrets": []any{}, "revision": 1.0,
		}}
		if includeSearch.Load() {
			ns = append(ns, map[string]any{
				"ns": "web-search-deepseek", "schema": json.RawMessage(`{"type":"object","properties":{}}`),
				"value": map[string]any{}, "autoGenerate": true, "applies": "live", "secrets": []any{}, "revision": 1.0,
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "server-response", "rpcId": body["rpcId"],
			"result": map[string]any{"ok": true, "value": map[string]any{"writable": true, "hasDocument": true, "namespaces": ns}},
		})
	}))
	defer srv.Close()

	st := settingsstore.New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && len(st.Snapshot().Value.Namespaces) < 1 {
		time.Sleep(time.Millisecond)
	}
	if len(st.Snapshot().Value.Namespaces) != 1 {
		t.Fatalf("settings namespaces did not load: %+v", st.Snapshot().Value.Namespaces)
	}

	var s mainState
	first := s.cachedPluginContributions(st, "default", false)
	second := s.cachedPluginContributions(st, "default", false)
	if len(first) != 1 || &first[0] != &second[0] {
		t.Fatalf("repeated identical call did not reuse cached contributions: len=%d", len(first))
	}
	if first[0].Title != "Agent loop" {
		t.Fatalf("got EN title %q, want Agent loop", first[0].Title)
	}

	i18n.SetLocale(i18n.LocaleZH)
	zh := s.cachedPluginContributions(st, "default", false)
	if len(zh) != 1 || zh[0].Title != "Agent 循环" {
		t.Fatalf("locale change did not invalidate contributions cache: %+v", zh)
	}

	dark := s.cachedPluginContributions(st, "default", true)
	if &dark[0] == &zh[0] {
		t.Fatal("systemDark change did not rebuild contributions cache")
	}

	includeSearch.Store(true)
	st.RefreshDescribe(context.Background())
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && len(st.Snapshot().Value.Namespaces) < 2 {
		time.Sleep(time.Millisecond)
	}
	withSearch := s.cachedPluginContributions(st, "default", true)
	if len(withSearch) != 2 {
		t.Fatalf("namespace mask change did not rebuild contributions cache: len=%d", len(withSearch))
	}
}
