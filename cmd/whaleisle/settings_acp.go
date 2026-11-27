package main

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"
	"whaleisle/internal/desktop/acp"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/settings"
)

type acpPanelState struct {
	owner                   *dsh.Client
	catalog                 acp.Catalog
	registry                []acp.RegistryAgent
	selected, error, status string
	loaded, busy            bool
	unavailable             bool
	cancel                  context.CancelFunc
	generation              uint64
}

func isPluginNotMountedError(err error) bool {
	var httpErr *pluginrpc.HTTPError
	return errors.As(err, &httpErr) && (httpErr.StatusCode == 404 || httpErr.StatusCode == 405)
}

func (s *mainState) acpUnavailableLocked(client *dsh.Client) bool {
	if s.acpPanel.unavailable {
		return true
	}
	if client == nil {
		return false
	}
	if s.bootStatus.State == harness.Ready && !s.bootStatus.ACPMounted {
		return true
	}
	if s.cfgStore != nil {
		disabled := s.effectiveConfigFor(s.cfgStore).DisabledPlugins
		if slices.Contains(disabled, "dsh-acp") || slices.Contains(disabled, "dsh-acp-agents") {
			return true
		}
	}
	return false
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
	if s.acpUnavailableLocked(client) {
		s.mu.Unlock()
		return nil
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

// acpActionLead names the action a person just took, so a page failure reads as
// "refreshing the agent registry did not finish" instead of "an operation".
func acpActionLead(method string) string {
	switch method {
	case "catalog":
		return resumeText("刷新 Agent 目录", "Refreshing the agent registry")
	case "agent/registry-search":
		return resumeText("搜索 Agent 目录", "Searching the agent registry")
	case "agent/save":
		return resumeText("保存 Agent 设置", "Saving the agent setting")
	case "agent/toggle":
		return resumeText("切换 Agent 开关", "Changing the agent switch")
	case "agent/delete":
		return resumeText("删除 Agent", "Deleting the agent")
	case "agent/test":
		return resumeText("测试 Agent", "Testing the agent")
	case "agent/create-account":
		return resumeText("创建 Agent 账号", "Creating the agent account")
	case "sessions/stop":
		return resumeText("停止 Agent 会话", "Stopping the agent session")
	default:
		return resumeText("更新 Agent 设置", "Updating the agent setting")
	}
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
	if client == nil || s.acpPanel.owner != client || s.acpPanel.busy || s.acpUnavailableLocked(client) {
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
			if isPluginNotMountedError(err) {
				s.acpPanel.unavailable = true
				s.acpPanel.error = ""
				s.acpPanel.status = ""
			} else {
				s.acpPanel.error = errcopy.Under(acpActionLead(method), err)
			}
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
				s.acpPanel.status = resumeText("操作已完成", "Action completed")
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
