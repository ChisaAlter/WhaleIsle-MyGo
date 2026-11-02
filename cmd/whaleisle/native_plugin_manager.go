package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/dsh"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
	managerui "whaleisle/internal/ui/pluginmanager"
	"whaleisle/internal/ui/settings"
)

type nativePluginManagerState struct {
	controller                                                *manager.Controller
	client                                                    *dsh.Client
	owner                                                     uint64
	contributions                                             *pluginui.Owner
	session                                                   *nativePluginSessionState
	contributionGeneration                                    int
	contributionOrigin, contributionCookie, contributionError string
}

func (s *mainState) nativePluginController() *manager.Controller {
	s.mu.Lock()
	created := s.nativePlugins.controller == nil
	if s.nativePlugins.controller == nil {
		s.nativePlugins.controller = manager.New()
	}
	c, cfg := s.nativePlugins.controller, s.cfgStore
	s.mu.Unlock()
	if created {
		c.SetRegistryPersistence(func() (*manager.RegistryChoice, error) {
			if cfg == nil {
				return nil, nil
			}
			p := cfg.Get().GoPluginInstallRegistry
			if p == nil {
				return nil, nil
			}
			return &manager.RegistryChoice{Kind: p.Kind, Registry: p.Registry, URL: p.URL}, nil
		}, func(choice *manager.RegistryChoice) error {
			if cfg == nil {
				return fmt.Errorf("%s", i18n.T(i18n.KeyCmdPluginPreferencesUnavailable))
			}
			return cfg.Update(func(value *config.Config) {
				value.GoPluginInstallRegistry = nil
				if choice != nil {
					value.GoPluginInstallRegistry = &config.PluginInstallRegistry{Kind: choice.Kind, Registry: choice.Registry, URL: choice.URL}
				}
			})
		})
	}
	return c
}

func (s *mainState) bindNativePluginManager(client *dsh.Client) {
	s.suspendPluginContributions()
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
			c.BundlesChanged(owner)
		case "plugin-market/changed":
			c.SourcesChanged(owner)
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
	creatorReady, creatorDescription, createPlugin := s.pluginCreatorAccess()
	s.mu.Lock()
	store := s.settings
	sessions, win := s.sessions, s.win
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
				if settings.ServedPluginBody(c, settings.Model{Store: store, ThemeID: themeID, SystemDark: mygo.Theme.IsDark()}, key) {
					return
				}
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
	owner, contributionError := s.nativePluginContributions()
	var dynamic pluginui.Snapshot
	if owner != nil {
		current := s.nativePluginController().Snapshot()
		owner.Select(contributionSubject(current.View, current.Packages))
		dynamic = owner.Snapshot()
		contributions = mergePluginContributions(contributions, owner, dynamic)
	}
	pluginContributionNotice(c, owner, contributionError)
	managerui.View(c, managerui.Model{
		SVGImages:  s.svgImageOwner(win, sessions),
		Controller: s.nativePluginController(), ThemeID: themeID, SystemDark: mygo.Theme.IsDark(),
		Contributions: contributions,
		RequestListEnable: func(name string) bool {
			if owner == nil {
				return false
			}
			for _, entry := range dynamic.Entries {
				if entry.Slot == "plugins.bundle.activation" && entry.Key == name {
					current := s.nativePluginController().Snapshot()
					for _, pkg := range current.Packages {
						if pkg.Name == name && !pkg.Enabled && !pkg.SwitchLocked() && !current.ActionBusy(name) {
							if err := owner.EnableActivation(dynamic.Page, dynamic.Revision, name); err != nil {
								s.mu.Lock()
								s.nativePlugins.contributionError = err.Error()
								s.mu.Unlock()
							}
							return true
						}
					}
					return true
				}
			}
			return false
		},
		OnNativeOverlay: func(masked bool) {
			if owner != nil {
				owner.Mask(masked)
			}
		},
		DetailActions: func(c *ui.Context, subject manager.Subject) {
			if owner == nil {
				return
			}
			sub := contributionSubject(subject, s.nativePluginController().Snapshot().Packages)
			for _, action := range dynamic.Actions {
				if ui.Button(c, action.Label).Disabled(action.Disabled).Clicked() {
					_ = owner.Click(dynamic.Page, dynamic.Revision, sub, action.Handle)
				}
			}
		},
		DetailBadges: func(c *ui.Context, _ manager.Subject) {
			for _, badge := range dynamic.Badges {
				ui.Text(c, badge).FontSize(12).TextColor(c.Theme().TextMuted)
			}
		},
		DetailSections: func(c *ui.Context, _ manager.Subject) {
			if owner == nil {
				return
			}
			if dynamic.Section {
				renderPluginContributionSurface(c, owner, "plugins.detail.section")
			}
			if dynamic.Surface == "plugins.detail.actions" {
				renderPluginContributionSurface(c, owner, "plugins.detail.actions")
			}
		},
		AddActions: func(c *ui.Context) {
			if owner == nil {
				return
			}
			for _, entry := range dynamic.Entries {
				if entry.Slot == "plugins.add.actions" {
					if ui.Button(c, pluginCopy("插件提供的操作", "Plugin actions")).Clicked() {
						owner.AddActions()
					}
					break
				}
			}
		},
		Supplement: func(c *ui.Context) {
			if owner != nil {
				slot := owner.Snapshot().Surface
				if slot == "plugins.add.actions" || slot == "plugins.bundle.activation" {
					renderPluginContributionSurface(c, owner, slot)
				}
			}
		},
		OnCreatePlugin: createPlugin, CreatePluginReady: creatorReady, CreatePluginDescription: creatorDescription,
		OnOpenAddMenu: s.refreshPluginCreatorRoster,
		CreatorNotice: s.renderPluginCreatorNotice,
		RenderConfiguration: func(c *ui.Context, subject manager.Subject) bool {
			return subject.Kind == "bundle" && s.renderVoiceDeviceSettings(c, subject.Name)
		},
		OnClose:   func() { s.selectMainPanel("") },
		OnOpenURL: func(raw string) { _ = mygo.Shell.OpenExternal(raw) },
	})
}
