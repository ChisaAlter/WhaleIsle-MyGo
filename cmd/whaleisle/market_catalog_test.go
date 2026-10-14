package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/dsh"
	marketstore "whaleisle/internal/store/market"
)

type catalogFixture struct{}

func (catalogFixture) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{Items: []desktopmarket.CatalogItem{{ID: "fixture/plugin", Owner: "fixture", Repo: "plugin", PackageName: "fixture-plugin", InstallSpec: "fixture-plugin", Homepage: "https://github.com/fixture/plugin", Description: "Fixture"}}, Source: "fixture"}, nil
}
func (catalogFixture) Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error) {
	return desktopmarket.Details{Readme: "Fixture README"}, nil
}
func waitCatalogFixture(t *testing.T, predicate func() bool) {
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
					args, _ := env["args"].(map[string]any)
					_ = args
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
			waitCatalogFixture(t, func() bool { return !store.Discovery().Loading })
			store.SetQuery("fixture")
			store.SelectDetails("fixture/plugin", false)
			waitCatalogFixture(t, func() bool { return !store.Discovery().DetailsLoading })
			target, e := store.CaptureTarget("fixture/plugin")
			if e != nil {
				t.Fatal(e)
			}
			s := &mainState{dshClient: client, market: store}
			s.runMarketCatalogInstall(target)
			waitCatalogFixture(t, func() bool { s.mu.Lock(); defer s.mu.Unlock(); return s.marketWorker == nil })
			waitCatalogFixture(t, func() bool { return !store.Snapshot().Loading })
			s.mu.Lock()
			message := s.marketInstallMsg
			s.mu.Unlock()
			history := marketstore.NewWithPersonalPath(historyPath).Personal().History
			if len(history) != 1 || history[0].Result != application || history[0].Plugin != "fixture/plugin" {
				t.Fatalf("incorrect persisted history: %+v", history)
			}
			if !waited.Load() {
				t.Fatal("queued installation was not awaited")
			}
			if application == "applied" {
				if marketstore.InstalledName(store.Discovery().Rows[0], store.Snapshot().Rows) != "fixture-plugin" {
					t.Fatal("applied install was not reconciled to inventory")
				}
			} else {
				if len(store.Snapshot().Rows) != 0 || strings.Contains(message, "Applied") || strings.Contains(message, "已应用") {
					t.Fatalf("failure misreported: %q", message)
				}
			}
		})
	}
}
