package main

import (
	"context"
	"time"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/dsh"
	"whaleisle/internal/ui/settings"
)

type notifyingInventoryCaller struct {
	plugininventory.Caller
	notify func()
}

func (c notifyingInventoryCaller) Call(ctx context.Context, method string, args, out any) error {
	c.notify()
	return c.Caller.Call(ctx, method, args, out)
}

func (s *mainState) attachPluginInventory(client *dsh.Client) {
	var caller plugininventory.Caller
	if client != nil {
		caller = notifyingInventoryCaller{Caller: client, notify: s.updatePluginInventoryUI}
	}
	s.settingsPluginInventory.Attach(caller)
}
func (s *mainState) updatePluginInventoryUI() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}
func (s *mainState) refreshPluginInventory() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.settingsPluginInventory.Refresh(ctx)
		s.updatePluginInventoryUI()
	}()
}
func (s *mainState) pluginInventoryAccess() *settings.PluginInventoryAccess {
	snap := s.settingsPluginInventory.Snapshot()
	s.mu.Lock()
	host := s.webviewHost
	endpoint := s.bootStatus.Endpoint
	s.mu.Unlock()
	if host != nil && !host.MatchesSession(endpoint.Origin, endpoint.Cookie) {
		host = nil
	}
	clientModules := host.ClientModules()
	clientPage := ""
	if clientModules.Connected {
		clientPage = host.URL()
	}
	return &settings.PluginInventoryAccess{
		Snapshot:      snap,
		ClientModules: clientModules,
		ClientPage:    clientPage,
		OnRetryClientModules: func() {
			s.mu.Lock()
			current, endpoint := s.webviewHost, s.bootStatus.Endpoint
			s.mu.Unlock()
			if current != host || host == nil || !host.MatchesSession(endpoint.Origin, endpoint.Cookie) {
				return
			}
			go func() { _ = host.RetryClientModules(clientModules.Page); s.updatePluginInventoryUI() }()
		},
		OnRefresh:      s.refreshPluginInventory,
		OnQuery:        func(query string) { s.settingsPluginInventory.SetQuery(snap.Host, query); s.updatePluginInventoryUI() },
		OnSelectPreset: func(id string) { s.settingsPluginInventory.SelectPreset(snap.Host, id); s.updatePluginInventoryUI() },
		OnToggleRow: func(key string) {
			s.settingsPluginInventory.ToggleExpanded(snap.Host, key)
			s.updatePluginInventoryUI()
		},
		OnToggleGlobal: func() { s.settingsPluginInventory.ToggleGlobal(snap.Host); s.updatePluginInventoryUI() },
	}
}

// Keep the existing Host page alive, but hidden, only while native Settings
// inspects its client modules. Every other panel exit retains normal disposal.
func (s *mainState) parkPluginViewForSettings(settingsOpen bool) {
	s.mu.Lock()
	host, endpoint := s.webviewHost, s.bootStatus.Endpoint
	s.mu.Unlock()
	if settingsOpen && host != nil && host.HasSession() && host.Alive() && host.MatchesSession(endpoint.Origin, endpoint.Cookie) {
		host.SetVisible(false)
		return
	}
	s.closeWebViewHost()
}
