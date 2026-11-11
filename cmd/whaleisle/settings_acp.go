package main

import (
	"context"
	"encoding/json"
	"time"
	"whaleisle/internal/desktop/acp"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/ui/settings"
)

type acpPanelState struct {
	owner                   *dsh.Client
	catalog                 acp.Catalog
	registry                []acp.RegistryAgent
	selected, error, status string
	loaded, busy            bool
	cancel                  context.CancelFunc
	generation              uint64
}

func (s *mainState) acpCurrentClient() *dsh.Client {
	s.dshMu.Lock()
	defer s.dshMu.Unlock()
	return s.dshClient
}
func (s *mainState) acpSettingsAccess() *settings.ACPAccess {
	client := s.acpCurrentClient()
	s.mu.Lock()
	if s.acpPanel.owner != client {
		if s.acpPanel.cancel != nil {
			s.acpPanel.cancel()
		}
		s.acpPanel = acpPanelState{owner: client, generation: s.acpPanel.generation + 1}
	}
	p := s.acpPanel
	s.mu.Unlock()
	if client != nil && !p.loaded && !p.busy {
		s.runACP("catalog", map[string]any{})
	}
	return &settings.ACPAccess{Catalog: p.catalog, Registry: p.registry, Selected: p.selected, Error: p.error, Status: p.status, Busy: p.busy, Available: client != nil, Loaded: p.loaded,
		OnSelect: func(id string) { s.mu.Lock(); s.acpPanel.selected = id; s.mu.Unlock(); s.refreshWhaleSettingsUI() },
		OnAction: s.runACP}
}
func (s *mainState) runACP(method string, payload map[string]any) {
	// UI drafts stay editable between frames; an in-flight request owns its
	// arguments, including nested credential/environment maps.
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	var request map[string]any
	if json.Unmarshal(raw, &request) != nil {
		return
	}
	client := s.acpCurrentClient()
	s.mu.Lock()
	if client == nil || s.acpPanel.owner != client || s.acpPanel.busy {
		s.mu.Unlock()
		return
	}
	s.acpPanel.busy = true
	s.acpPanel.error = ""
	s.acpPanel.status = ""
	s.acpPanel.generation++
	generation := s.acpPanel.generation
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	s.acpPanel.cancel = cancel
	s.mu.Unlock()
	s.refreshWhaleSettingsUI()
	go func() {
		defer cancel()
		rpc := acp.Client{RPC: &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}}
		var catalog acp.Catalog
		var registry []acp.RegistryAgent
		var selected string
		var err error
		switch method {
		case "catalog":
			catalog, err = rpc.Catalog(ctx)
		case "agent/registry-search":
			query, _ := request["query"].(string)
			registry, err = rpc.Search(ctx, query)
		default:
			catalog, selected, err = rpc.Action(ctx, method, request)
		}
		current := s.acpCurrentClient() == client
		s.mu.Lock()
		if !current || s.acpPanel.owner != client || s.acpPanel.generation != generation {
			s.mu.Unlock()
			return
		}
		s.acpPanel.busy = false
		s.acpPanel.loaded = true
		s.acpPanel.cancel = nil
		if err != nil {
			s.acpPanel.error = err.Error()
		} else {
			if method == "agent/registry-search" {
				s.acpPanel.registry = registry
			} else if catalog.Agents != nil {
				s.acpPanel.catalog = catalog
			}
			if selected != "" {
				s.acpPanel.selected = selected
			}
			if method != "catalog" {
				s.acpPanel.status = "操作已完成"
			}
		}
		poll := false
		for _, a := range s.acpPanel.catalog.Agents {
			if a.AuthFlow.Active() {
				poll = true
			}
		}
		sessions := s.sessions
		s.mu.Unlock()
		s.refreshWhaleSettingsUI()
		if err == nil && method != "catalog" && method != "agent/registry-search" && sessions != nil {
			sessions.RefreshModelCatalog(context.Background())
		}
		if poll {
			time.AfterFunc(2*time.Second, func() {
				s.mu.Lock()
				active := s.settingsOpen && s.settingsAgentPresetsActive && s.acpPanel.owner == client && s.acpPanel.generation == generation
				s.mu.Unlock()
				if active {
					s.runACP("catalog", map[string]any{})
				}
			})
		}
	}()
}
