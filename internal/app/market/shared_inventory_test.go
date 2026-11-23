package marketctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"whaleisle/internal/dsh"
)

func TestSharedInventoryAttachReadyDemandAndRealChange(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	enabled := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope map[string]any
		_ = json.NewDecoder(r.Body).Decode(&envelope)
		mu.Lock()
		calls[r.URL.Path]++
		current := enabled
		mu.Unlock()
		var value any = map[string]any{}
		switch r.URL.Path {
		case "/api/pluginInventory/list":
			value = map[string]any{"managementAvailable": true, "entries": []any{map[string]any{"entryId": "overlay", "moduleName": "qa-panel", "enabled": current, "fiberPhase": "active"}}}
		case "/api/pluginManager/listBundles":
			value = []any{}
		case "/api/pluginManager/listPlugins":
			value = []any{map[string]any{"entryId": "overlay", "moduleName": "qa-panel", "enabled": current, "fiberPhase": "active"}}
		case "/api/pluginRegistryProbe/fastest":
			value = ""
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": envelope["rpcId"], "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	coord := NewCoordinator(&stubHostDeps{})
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	coord.AttachHost(client, t.TempDir(), nil)
	defer coord.DetachHost()
	coord.HostEventsReady(client)
	waitMarketFixture(t, func() bool { coord.mu.Lock(); defer coord.mu.Unlock(); return !coord.inventoryRefreshRunning })
	mu.Lock()
	initial := map[string]int{}
	for method, count := range calls {
		initial[method] = count
	}
	mu.Unlock()
	if len(initial) != 1 || initial["/api/pluginInventory/list"] != 1 {
		t.Fatalf("startup loaded duplicate or full market inventory: %v", initial)
	}
	if coord.NativeController().Snapshot().Value.Status != "idle" {
		t.Fatal("runtime-only read activated full native manager")
	}
	coord.PrepareInstalledPlugin()
	coord.Store().EnsureInventory(context.Background())
	waitMarketFixture(t, func() bool {
		return coord.InstalledPluginState("qa-panel").Known && !coord.Store().Snapshot().Value.Loading
	})
	if !coord.InstalledPluginState("qa-panel").Enabled {
		t.Fatal("patch-only overlay lost")
	}
	mu.Lock()
	runtimeReads, bundleReads, pluginReads := calls["/api/pluginInventory/list"], calls["/api/pluginManager/listBundles"], calls["/api/pluginManager/listPlugins"]
	enabled = false
	mu.Unlock()
	if runtimeReads != 1 || bundleReads != 1 || pluginReads != 1 {
		t.Fatalf("demand consumers duplicated reads: %d %d %d", runtimeReads, bundleReads, pluginReads)
	}
	before := coord.InstalledPluginState("qa-panel")
	coord.NativeEventHandler(client)(dsh.EventsEmit{Event: "plugin-manager/changed"})
	waitMarketFixture(t, func() bool {
		state := coord.InstalledPluginState("qa-panel")
		return state.Known && state.Revision > before.Revision && !state.Enabled && !coord.Store().Snapshot().Value.Loading
	})
	if state := coord.Inventory().Snapshot().Value; len(state.Global) != 1 || state.Global[0].Enabled != "disabled" {
		t.Fatalf("settings retained enabled overlay: %+v", state)
	}
	mu.Lock()
	if calls["/api/pluginInventory/list"] != 2 || calls["/api/pluginManager/listBundles"] != 2 || calls["/api/pluginManager/listPlugins"] != 2 {
		t.Fatalf("changed generation reread or missed a consumer: %v", calls)
	}
	enabled = true
	mu.Unlock()
	// A legacy manual refresh invalidates the same owner and must restore the
	// settings/A2 projections even without a subsequent event notification.
	coord.Store().Refresh(context.Background())
	waitMarketFixture(t, func() bool {
		return coord.InstalledPluginState("qa-panel").Known && coord.InstalledPluginState("qa-panel").Enabled && !coord.Store().Snapshot().Value.Loading
	})
	mu.Lock()
	defer mu.Unlock()
	if calls["/api/pluginInventory/list"] != 3 || calls["/api/pluginManager/listBundles"] != 3 || calls["/api/pluginManager/listPlugins"] != 3 {
		t.Fatalf("manual refresh missed shared consumers: %v", calls)
	}
}

func TestWorkspaceInventoryDeferralKeepsExplicitConsumerAndCurrentOwner(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "workspace-first", true: "explicit-before-workspace"}[explicit], func(t *testing.T) {
			var mu sync.Mutex
			calls := map[string]int{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var envelope map[string]any
				_ = json.NewDecoder(r.Body).Decode(&envelope)
				mu.Lock()
				calls[r.URL.Path]++
				mu.Unlock()
				var value any = map[string]any{}
				switch r.URL.Path {
				case "/api/pluginInventory/list":
					value = map[string]any{"managementAvailable": true, "entries": []any{map[string]any{"entryId": "overlay", "moduleName": "qa-panel", "enabled": false, "fiberPhase": "active"}}}
				case "/api/pluginManager/listBundles":
					value = []any{}
				case "/api/pluginManager/listPlugins":
					value = []any{map[string]any{"entryId": "overlay", "moduleName": "qa-panel", "enabled": false, "fiberPhase": "active"}}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": envelope["rpcId"], "result": map[string]any{"ok": true, "value": value}})
			}))
			defer srv.Close()
			coord := NewCoordinator(&stubHostDeps{})
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			if coord.WorkspaceReady(client) {
				t.Fatal("pre-attachment refresh consumed the release")
			}
			coord.AttachHostForWorkspace(client, t.TempDir(), nil)
			defer coord.DetachHost()
			coord.HostEventsReady(client)
			coord.HostEventsReady(client) // reconnect invalidates, but has no cold consumer
			if coord.WorkspaceReady(&dsh.Client{Endpoint: client.Endpoint}) {
				t.Fatal("another attachment released the current inventory")
			}
			mu.Lock()
			if len(calls) != 0 {
				t.Fatalf("deferred startup sent inventory RPCs: %v", calls)
			}
			mu.Unlock()
			if explicit {
				coord.PrepareInstalledPlugin()
				waitMarketFixture(t, func() bool { return coord.InstalledPluginState("qa-panel").Known })
				if coord.InstalledPluginState("qa-panel").Enabled {
					t.Fatal("explicit consumer treated a disabled overlay as enabled")
				}
			}
			if !coord.WorkspaceReady(client) || !coord.WorkspaceReady(client) {
				t.Fatal("current attachment did not accept its release")
			}
			waitMarketFixture(t, func() bool { return coord.Inventory().Snapshot().Value.Ready })
			mu.Lock()
			if calls["/api/pluginInventory/list"] != 1 {
				t.Fatalf("workspace and explicit consumer duplicated inventory: %v", calls)
			}
			if !explicit && len(calls) != 1 {
				t.Fatalf("workspace release fetched a full catalog: %v", calls)
			}
			mu.Unlock()
			coord.DetachHost()
			if coord.WorkspaceReady(client) {
				t.Fatal("detached attachment was accepted")
			}
		})
	}
}
