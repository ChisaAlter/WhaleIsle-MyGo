package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/ui/chrome"
)

func pluginModuleFailures(c *ui.Context, owner *pluginui.Owner) {
	if owner == nil {
		return
	}
	state := owner.ClientModules()
	if len(state.Failures) == 0 && state.Error == "" {
		return
	}
	ui.Column(c.Key("plugin-client-module-failures")).FillWidth().Gap(6).Padding(10).Border(1, c.Theme().Border).Radius(6).Children(func() {
		ui.Text(c, pluginCopy("插件客户端未能加载", "Plugin client failed to load")).Bold().TextColor(c.Theme().Warning)
		ui.Text(c, pluginCopy("此页面保持受限环境；依赖 Electron 桌面接口的插件需要专门原生适配。", "This page is isolated. Plugins requiring Electron desktop APIs need a dedicated native adapter.")).FillWidth()
		for _, failure := range state.Failures {
			ui.Text(c, failure.ID+": "+failure.Message).FillWidth().Selectable()
		}
		if state.Error != "" {
			ui.Text(c, state.Error).FillWidth().TextColor(c.Theme().Warning)
		}
		if ui.Button(c, pluginCopy("重试此页面的插件", "Retry this page's plugins")).Disabled(!state.Connected || state.Syncing || state.Retrying).Clicked() {
			_ = owner.RetryClientModules(state.Page)
		}
	})
}
func (s *mainState) renderInlinePluginDiagnostics(c *ui.Context, owner *pluginui.Owner) {
	w, h := c.Size()
	open := true
	ui.DialogBase(c.Key("native-session-plugin-diagnostics"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), max(260, min(float32(700), w-48)), 16).MaxHeight(max(0, h-48)).Gap(12).Children(func() {
			ui.Row(c).FillWidth().Gap(12).Children(func() {
				ui.Text(c, pluginCopy("会话插件", "Session plugins")).Grow(1).Bold()
				if ui.Button(c, pluginCopy("关闭", "Close")).Clicked() {
					open = false
				}
			})
			pluginModuleFailures(c, owner)
			snapshot := owner.Snapshot()
			if snapshot.Error != "" {
				ui.Text(c, snapshot.Error).FillWidth().TextColor(c.Theme().Warning)
			}
			if err := owner.RichError(); err != "" {
				ui.Text(c, err).FillWidth().TextColor(c.Theme().Warning)
			}
			ui.Text(c, pluginCopy("已加载的贡献在当前会话原位显示。此检查和重试仅针对当前会话的原页面。", "Loaded contributions appear in place in the current conversation. Diagnostics and retry address only its retained source page.")).FillWidth()
		})
	})
	if !open {
		s.mu.Lock()
		if p := s.nativePlugins.session; p != nil && p.owner == owner {
			p.open = false
		}
		s.mu.Unlock()
	}
}
