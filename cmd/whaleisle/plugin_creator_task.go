package main

import (
	"context"
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func pluginCreatorText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

// Background owner checks use state only; native visibility queries belong to
// UI-thread admission and post-frame observation.
func (s *mainState) pluginCreatorNavigationCurrent(source *session.Store) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions == source && s.win != nil && !s.win.IsDestroyed() && s.marketOpen && s.marketManager && !s.projectWindowHidden && !s.quitting && !s.settingsOpen && !s.launcherOpen && !s.usageOpen && !s.imOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen
}

func (s *mainState) pluginCreatorVisible(source *session.Store) bool {
	if !s.pluginCreatorNavigationCurrent(source) {
		return false
	}
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	return win != nil && win.IsVisible() && !win.IsMinimized()
}

func (s *mainState) suspendPluginCreator() {
	s.mu.Lock()
	creator := s.pluginCreatorTask
	s.mu.Unlock()
	if creator != nil {
		creator.ObserveVisible(false)
	}
}

func (s *mainState) observePluginCreator() {
	s.mu.Lock()
	creator, source := s.pluginCreatorTask, s.sessions
	s.mu.Unlock()
	if creator != nil {
		creator.ObserveVisible(s.pluginCreatorVisible(source))
	}
}

func (s *mainState) refreshPluginCreatorRoster() {
	s.mu.Lock()
	source := s.sessions
	s.mu.Unlock()
	if source != nil && s.pluginCreatorVisible(source) {
		source.RefreshAgentPresets(context.Background())
	}
}

func (s *mainState) pluginCreatorAccess() (ready bool, description string, create func()) {
	s.mu.Lock()
	source, win := s.sessions, s.win
	if source != nil && s.pluginCreatorTask == nil {
		s.pluginCreatorTask = session.NewCreatorTask(source, func() bool { return s.pluginCreatorNavigationCurrent(source) })
	}
	creator := s.pluginCreatorTask
	s.mu.Unlock()
	if creator == nil || source == nil {
		return false, i18n.T(i18n.KeyCmdCreatorNeedsHost), nil
	}
	creator.ObserveVisible(s.pluginCreatorVisible(source))
	loaded, loading, available, failure := source.CreatorPresetCatalog()
	if !loaded && !loading && failure == "" && source.ClientReady() {
		source.RefreshAgentPresets(context.Background())
		loading = true
	}
	state := creator.Snapshot()
	description = pluginCreatorText("打开一个独立的 Creator 任务来创建插件", "Open a separate Creator task to create a plugin")
	switch {
	case !source.ClientReady():
		description = i18n.T(i18n.KeyCmdCreatorUnavailable)
	case loading || !loaded && failure == "":
		description = i18n.T(i18n.KeyCmdCreatorChecking)
	case failure != "":
		description = i18n.T(i18n.KeyCmdCreatorPresetsUnreadable)
	case !available:
		description = pluginCreatorText("当前 Host 没有可用的 cordis 预设", "This Host has no available cordis preset")
	case state.Busy:
		description = pluginCreatorText("正在创建 Creator 任务…", "Creating a Creator task…")
	case state.Attempted && !state.Opened:
		description = pluginCreatorText("请先查看下方保留的创建结果", "Inspect the retained creation result below")
	default:
		ready = true
	}
	create = func() {
		if !ready || !s.pluginCreatorVisible(source) {
			return
		}
		creator.ObserveVisible(true)
		creator.Prepare()
		prepared := creator.Snapshot()
		if !prepared.Review || prepared.Busy {
			if win != nil {
				win.Update(func() {})
			}
			return
		}
		// The user's menu pick is the admission. No second review dialog,
		// draft seed, model prompt or persisted default mutation is involved.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			opened := creator.Create(ctx, prepared.Ticket)
			if win != nil {
				win.Update(func() {
					if opened {
						s.openPluginCreator(creator, source, prepared.Ticket, true)
					}
				})
			}
		}()
	}
	return
}

func (s *mainState) openPluginCreator(creator *session.CreatorTask, source *session.Store, ticket uint64, automatic bool) {
	if !s.pluginCreatorVisible(source) || !creator.Open(ticket, automatic) {
		return
	}
	s.selectMainPanel("")
	creator.ObserveVisible(false)
}

// Remains in the manager notice area after failures, lost receipts or a user
// departure. Recovery reads/opens this exact identity and never creates again.
func (s *mainState) renderPluginCreatorNotice(c *ui.Context) {
	s.mu.Lock()
	creator, source, win := s.pluginCreatorTask, s.sessions, s.win
	s.mu.Unlock()
	if creator == nil || source == nil {
		return
	}
	state := creator.Snapshot()
	if state.SessionID == "" || state.Opened {
		return
	}
	ui.Column(c.Key("plugin-creator-result")).FillWidth().Gap(6).Padding(12).Border(1, c.Theme().Border).Radius(8).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyCmdCreatorResult)).Bold()
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyCmdCreatorResultMetadata), state.Host, state.Cwd, state.SessionID, state.Phase)).FontSize(12).Selectable()
		if state.Error != "" {
			ui.Text(c, state.Error).FontSize(12).Selectable()
		}
		ui.Row(c).Gap(8).Children(func() {
			if state.Attempted && !state.Created && ui.Button(c, pluginCreatorText("定位已提交目标", "Locate attempted task")).Disabled(state.Busy).Clicked() {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
					defer cancel()
					creator.Locate(ctx, state.Ticket)
					if win != nil {
						win.Update(func() {})
					}
				}()
			}
			if state.Created && ui.Button(c, i18n.T(i18n.KeyCmdCreatorOpenExisting)).Disabled(state.Busy).Clicked() {
				s.openPluginCreator(creator, source, state.Ticket, false)
			}
		})
	})
}
