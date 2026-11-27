package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func TestACPActionSnapshotsDraftAndDiscardsOldHostResponse(t *testing.T) {
	entered, release := make(chan map[string]any, 1), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		entered <- req
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{"agents": []any{map[string]any{"id": "old-host"}}}}})
	}))
	defer server.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}, HTTP: server.Client()}
	s := &mainState{dshClient: client, acpPanel: acpPanelState{owner: client}}
	env := map[string]string{"KEY": "owned-original"}
	s.runACP("agent/save", map[string]any{"agent": map[string]any{"env": env}})
	env["KEY"] = "edited-after-dispatch"
	select {
	case req := <-entered:
		got := req["payload"].(map[string]any)["agent"].(map[string]any)["env"].(map[string]any)["KEY"]
		if got != "owned-original" {
			close(release)
			t.Fatal("in-flight request read mutable draft", got)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("no ACP request")
	}
	s.dshMu.Lock()
	s.dshClient = nil
	s.dshMu.Unlock()
	s.acpSettingsAccess() // invalidates/cancels old ownership before response arrives
	close(release)
	// Wait for the request's cancellation; it must not repopulate the new owner.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		p := s.acpPanel
		s.mu.Unlock()
		if p.owner != nil || p.busy || len(p.catalog.Agents) != 0 || p.status != "" {
			t.Fatal("old Host polluted ACP panel", p)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestACPSettingsAccessHiddenWithoutAutoCatalogWhenUnmounted(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}, HTTP: server.Client()}
	s := &mainState{
		dshClient:  client,
		bootStatus: harness.Status{State: harness.Ready, ACPMounted: false},
	}
	if access := s.acpSettingsAccess(); access != nil {
		t.Fatalf("expected nil ACPAccess when Host has ACPMounted=false, got %+v", access)
	}
	time.Sleep(30 * time.Millisecond)
	if n := hits.Load(); n != 0 {
		t.Fatalf("unmounted ACP fired %d catalog request(s), want 0", n)
	}
}

func TestACPDegradesCleanlyOnLocalHTTP405WithoutErrorBanner(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}, HTTP: server.Client()}
	s := &mainState{
		dshClient:  client,
		bootStatus: harness.Status{State: harness.Ready, ACPMounted: true},
	}
	if access := s.acpSettingsAccess(); access == nil {
		t.Fatal("expected initial ACPAccess when ACPMounted=true")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		done := !s.acpPanel.busy && s.acpPanel.loaded
		p := s.acpPanel
		s.mu.Unlock()
		if done {
			if !p.unavailable {
				t.Fatalf("expected ACP panel to mark unavailable on HTTP 405, got %+v", p)
			}
			if p.error != "" {
				t.Fatalf("expected empty error banner on local HTTP 405 degrade, got %q", p.error)
			}
			break
		}
		time.Sleep(time.Millisecond)
	}
	if access := s.acpSettingsAccess(); access != nil {
		t.Fatalf("expected subsequent acpSettingsAccess() to return nil after HTTP 405, got %+v", access)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected exactly 1 catalog attempt before disabling, got %d", hits.Load())
	}
}

func TestACPActionStatusLocalized(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  req["rpcId"],
			"result": map[string]any{"ok": true, "value": map[string]any{"agents": []any{}}},
		})
	}))
	defer server.Close()

	for _, tc := range []struct {
		locale string
		want   string
	}{
		{i18n.LocaleEN, "Action completed"},
		{i18n.LocaleZH, "操作已完成"},
	} {
		i18n.SetLocale(tc.locale)
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}, HTTP: server.Client()}
		s := &mainState{dshClient: client, acpPanel: acpPanelState{owner: client}}
		s.runACP("agent/toggle", map[string]any{"id": "demo", "enabled": true})
		deadline := time.Now().Add(2 * time.Second)
		got := ""
		for time.Now().Before(deadline) {
			s.mu.Lock()
			done := !s.acpPanel.busy && s.acpPanel.loaded
			got = s.acpPanel.status
			s.mu.Unlock()
			if done {
				break
			}
			time.Sleep(time.Millisecond)
		}
		if got != tc.want {
			t.Fatalf("locale %s: acp status = %q, want %q", tc.locale, got, tc.want)
		}
	}
}
