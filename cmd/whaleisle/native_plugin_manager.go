package main

import (
	"context"
	"encoding/json"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/dsh"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
	managerui "whaleisle/internal/ui/pluginmanager"
	"whaleisle/internal/ui/settings"
)

type nativePluginManagerState struct {
	controller *manager.Controller
	client     *dsh.Client
	owner      uint64
}

func (s *mainState) nativePluginController() *manager.Controller {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.nativePlugins.controller == nil {
		s.nativePlugins.controller = manager.New()
	}
	return s.nativePlugins.controller
}

func (s *mainState) bindNativePluginManager(client *dsh.Client) {
	c := s.nativePluginController()
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	notify := func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	// An interface carrying (*dsh.Client)(nil) is non-nil. Detach explicitly.
	if client == nil {
		c.Detach()
		c.SetNotify(notify)
	} else {
		c.Attach(client, notify)
	}
	s.mu.Lock()
	s.nativePlugins.client = client
	s.nativePlugins.owner = c.Snapshot().Owner
	s.mu.Unlock()
}

// The original ui-plugin-manager consumes these one-argument $events emits.
// Capture the transport owner once; a late packet cannot target a new Host.
func (s *mainState) nativePluginEventHandler(client *dsh.Client) func(dsh.EventsEmit) {
	s.mu.Lock()
	binding := s.nativePlugins
	s.mu.Unlock()
	if client == nil || binding.client != client || binding.controller == nil {
		return func(dsh.EventsEmit) {}
	}
	c, owner := binding.controller, binding.owner
	return func(emit dsh.EventsEmit) {
		if c.Snapshot().Owner != owner {
			return
		}
		switch emit.Event {
		case "plugin-manager/changed":
			if c.Snapshot().Status != "idle" {
				c.Refresh(context.Background())
			}
		case "plugin-manager/install-log":
			if len(emit.Args) != 1 {
				return
			}
			var chunk manager.InstallLog
			if json.Unmarshal(emit.Args[0], &chunk) == nil && chunk.RequestID != "" {
				c.AppendLog(owner, chunk)
			}
		case "plugin-manager/install-state":
			if len(emit.Args) != 1 {
				return
			}
			var progress manager.InstallProgress
			if json.Unmarshal(emit.Args[0], &progress) == nil && progress.RequestID != "" {
				c.Progress(owner, progress)
			}
		}
	}
}

func (s *mainState) pluginManagerView(c *ui.Context, themeID string) {
	s.mu.Lock()
	store := s.settings
	s.mu.Unlock()
	contributions := []managerui.Contribution{}
	if store != nil {
		served := map[string]bool{}
		for _, ns := range store.Snapshot().Namespaces {
			served[ns.NS] = true
		}
		add := func(key, zh, en string, order int, namespaces ...string) {
			active := []string{}
			for _, ns := range namespaces {
				if served[ns] {
					active = append(active, ns)
				}
			}
			if len(active) == 0 {
				return
			}
			title := en
			if strings.HasPrefix(i18n.Locale(), "zh") {
				title = zh
			}
			contributions = append(contributions, managerui.Contribution{Kind: "item", Key: key, Title: title, Order: order, Render: func(c *ui.Context, _ manager.Subject) {
				for _, ns := range active {
					settings.NamespaceBody(c, settings.Model{Store: store, ThemeID: themeID, SystemDark: mygo.Theme.IsDark()}, ns)
				}
			}})
		}
		add("subagent", "子代理", "Subagents", 10, "subagent", "subagent-model-selection-settings")
		if served["web-search-deepseek"] {
			add("web-search", "网页搜索", "Web search", 20, "web-search-deepseek", "web", "web-deepseek-api")
		}
		add("agent-loop", "代理循环", "Agent loop", 30, "agent-loop")
		if served["pwsh-sandbox"] {
			add("shell", "终端", "Shell", 40, "pwsh-sandbox")
		} else {
			add("shell", "终端", "Shell", 40, "bash-sandbox")
		}
	}
	managerui.View(c, managerui.Model{
		Controller: s.nativePluginController(), ThemeID: themeID, SystemDark: mygo.Theme.IsDark(),
		Contributions: contributions,
		OnClose:       func() { s.selectMainPanel("") },
	})
}
