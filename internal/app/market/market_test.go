package marketctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/dsh"
	marketstore "whaleisle/internal/store/market"
)

type catalogFixture struct{}

func (catalogFixture) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{
		Items: []desktopmarket.CatalogItem{{
			ID:          "fixture/plugin",
			Owner:       "fixture",
			Repo:        "plugin",
			PackageName: "fixture-plugin",
			InstallSpec: "fixture-plugin",
			Homepage:    "https://github.com/fixture/plugin",
			Description: "Fixture",
		}},
		Source: "fixture",
	}, nil
}

func (catalogFixture) Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error) {
	return desktopmarket.Details{Readme: "Fixture README"}, nil
}

func waitMarketFixture(t *testing.T, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("market fixture timed out")
}

func TestMarketCatalogInstallResultAndInventory(t *testing.T) {
	for _, application := range []string{"applied", "failed"} {
		t.Run(application, func(t *testing.T) {
			var installed atomic.Bool
			var waited atomic.Bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var env map[string]any
				_ = json.NewDecoder(r.Body).Decode(&env)
				var value any = map[string]any{}
				switch r.URL.Path {
				case "/api/pluginManager/installBundle":
					value = map[string]any{"application": "pending", "stage": "queued"}
				case "/api/pluginManager/waitForInstall":
					waited.Store(true)
					installed.Store(application == "applied")
					value = map[string]any{"application": application, "stage": "install", "changed": application == "applied"}
				case "/api/pluginManager/listBundles":
					value = []any{}
					if installed.Load() {
						value = []any{map[string]any{"name": "fixture-plugin", "installed": true, "enabled": true, "removable": true, "version": "1.0.0"}}
					}
				case "/api/pluginRegistryProbe/fastest":
					value = ""
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": env["rpcId"], "result": map[string]any{"ok": true, "value": value}})
			}))
			defer srv.Close()

			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			historyPath := filepath.Join(t.TempDir(), "personal.json")
			store := marketstore.NewWithPersonalPath(historyPath)
			store.SetCatalogService(catalogFixture{})
			store.Attach(client, nil)
			store.RefreshCatalog(true, "en")
			waitMarketFixture(t, func() bool { return !store.Discovery().Loading })
			store.SetQuery("fixture")
			store.SelectDetails("fixture/plugin", false)
			waitMarketFixture(t, func() bool { return !store.Discovery().DetailsLoading })
			target, err := store.CaptureTarget("fixture/plugin")
			if err != nil {
				t.Fatal(err)
			}

			coord := NewCoordinator(nil)
			coord.SetStore(store)
			coord.SetClient(client)
			coord.RunCatalogInstall(target)

			waitMarketFixture(t, func() bool { return !coord.ViewSnapshot().Value.WorkerBusy })
			waitMarketFixture(t, func() bool { return !store.Snapshot().Value.Loading })
			message := coord.ViewSnapshot().Value.InstallMsg

			history := marketstore.NewWithPersonalPath(historyPath).Personal().History
			if len(history) != 1 || history[0].Result != application || history[0].Plugin != "fixture/plugin" {
				t.Fatalf("incorrect persisted history: %+v", history)
			}
			if !waited.Load() {
				t.Fatal("queued installation was not awaited")
			}
			if application == "applied" {
				if marketstore.InstalledName(store.Discovery().Rows[0], store.Snapshot().Value.Rows) != "fixture-plugin" {
					t.Fatal("applied install was not reconciled to inventory")
				}
			} else {
				if len(store.Snapshot().Value.Rows) != 0 || strings.Contains(message, "Applied") || strings.Contains(message, "已应用") {
					t.Fatalf("failure misreported: %q", message)
				}
			}
		})
	}
}

func TestMarketRenderedOwnerCannotActAfterReconnect(t *testing.T) {
	for _, reconnect := range []string{"same-client", "other-client", "other-store"} {
		t.Run(reconnect, func(t *testing.T) {
			client := &dsh.Client{}
			store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "personal.json"))
			store.Attach(client, nil)
			coord := NewCoordinator(nil)
			coord.SetStore(store)
			coord.SetClient(client)
			owner := coord.CaptureOwner(store)

			switch reconnect {
			case "same-client":
				store.Detach()
				store.Attach(client, nil)
			case "other-client":
				coord.SetClient(&dsh.Client{})
			case "other-store":
				coord.SetStore(marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "other.json")))
			}
			for _, install := range []bool{false, true} {
				op, _ := coord.BeginAttemptForOwner("fixture-plugin", nil, install, &owner)
				if op != nil || coord.IsBusy() {
					t.Fatal("stale rendered action acquired a new Host worker")
				}
			}
		})
	}
}

func TestMarketRenderedCurrentOwnerCanBegin(t *testing.T) {
	client := &dsh.Client{}
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "personal.json"))
	store.Attach(client, nil)
	coord := NewCoordinator(nil)
	coord.SetStore(store)
	coord.SetClient(client)
	owner := coord.CaptureOwner(store)
	op, _ := coord.BeginAttemptForOwner("", nil, false, &owner)
	if op == nil || op.Client != client || op.Store != store {
		t.Fatal("current rendered action did not retain its Host")
	}
	op.Cancel()
}

func TestNativePluginEventsFollowCapturedInstallAndHost(t *testing.T) {
	type nativeRPC struct{}
	var _ = nativeRPC{}
	release := make(chan struct{})
	defer close(release)
	started := make(chan struct{}, 1)
	fixture := nativeCallerFunc(func(ctx context.Context, method string, args, out any) error {
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

	coord := NewCoordinator(nil)
	ctrl := coord.NativeController()
	ctrl.Attach(fixture, nil)
	defer ctrl.Detach()
	if err := ctrl.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	owner := ctrl.Snapshot().Value.Owner
	transport := &dsh.Client{}
	coord.mu.Lock()
	coord.nativePlugins.client = transport
	coord.nativePlugins.owner = owner
	coord.mu.Unlock()

	emit := coord.NativeEventHandler(transport)
	a := ctrl.Actions(owner)
	a.OpenInstall()
	a.EditInstall("fixture", "")
	a.Inspect()
	waitMarketFixture(t, func() bool { return ctrl.Snapshot().Value.Install.Inspection != nil })
	a.RequestInstall()
	a.Confirm()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("fake install did not start")
	}
	id := ctrl.Snapshot().Value.Install.RequestID
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
	if got := ctrl.Snapshot().Value.Install; got.Phase != "applying" || got.Log != "fixture output" {
		t.Fatalf("native event wire lost owner/phase/output: %+v", got)
	}
	ctrl.Attach(fixture, nil)
	send("plugin-manager/install-log", map[string]any{"requestId": id, "jobId": "job", "text": "stale host"})
	send("plugin-manager/install-state", map[string]any{"requestId": id, "phase": "applying"})
	if got := ctrl.Snapshot().Value.Install; got.Log != "" || got.Phase != "" {
		t.Fatalf("old transport changed replacement Host: %+v", got)
	}
}

func TestCoordinatorConcurrentRepairSlotAndSnapshots(t *testing.T) {
	coord := NewCoordinator(nil)
	release, ok := coord.TryAcquireRepairSlot()
	if !ok {
		t.Fatal("expected initial repair slot")
	}
	if _, second := coord.TryAcquireRepairSlot(); second {
		t.Fatal("repair slot acquired twice")
	}
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = coord.ViewSnapshot()
			_ = coord.IsBusy()
		}()
	}
	wg.Wait()
	release()
	if coord.IsBusy() {
		t.Fatal("repair slot did not release")
	}
}

type nativeCallerFunc func(context.Context, string, any, any) error

func (f nativeCallerFunc) Call(ctx context.Context, method string, args, out any) error {
	return f(ctx, method, args, out)
}
