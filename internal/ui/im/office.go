package im

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	officestore "whaleisle/internal/store/office"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func officeSettingsView(c *ui.Context, m Model) {
	s := &m.Store.Office
	v := s.Snapshot()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	accent := chrome.Color(colors.Accent)
	blocked := v.Busy || !v.Available || m.PluginDisabled || !m.HarnessReady
	ui.Column(c.Key(fmt.Sprintf("office-%d", v.Scope))).Grow(1).MinHeight(0).Background(chrome.Color(colors.BG)).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "AI Office Connector").FontSize(theme.InterfaceSize(15)).Bold().Grow(1).TextColor(fg)
			if ui.Button(c, "刷新状态").Disabled(blocked).Clicked() {
				s.Refresh(context.Background())
			}
			if m.OnOpenPluginWeb != nil && ui.Button(c, "打开原版插件页面").Clicked() {
				s.Hide()
				m.OnOpenPluginWeb()
			}
			if ui.Button(c, "返回 IM").Clicked() {
				s.Hide()
			}
			if m.OnClose != nil && ui.Button(c, "关闭").Clicked() {
				s.Hide()
				m.OnClose()
			}
		})
		ui.Box(c).FillWidth().Height(1).Background(border)
		ui.Scroll(c).Grow(1).MinHeight(0).Padding(12, 16).Children(func() {
			ui.Column(c).Gap(9).Children(func() {
				ui.Text(c, "本机 Host 主动连接 Office；不开放入站端口。协议："+officestore.ProtocolVersion).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				ui.Text(c, "当前 Host："+v.Host).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				if v.Busy {
					ui.Text(c, "正在等待 Host…").TextColor(muted)
				}
				if v.Error != "" {
					ui.Text(c, v.Error).FontSize(theme.InterfaceSize(12)).TextColor(accent)
				}
				if v.Notice != "" {
					ui.Text(c, v.Notice).FontSize(theme.InterfaceSize(12)).TextColor(fg)
				}
				if !v.Loaded {
					if v.Busy {
						ui.Text(c, "正在读取 Office 配置…").FontSize(theme.InterfaceSize(12)).TextColor(muted)
						return
					}
					// No state has been read yet: say so plainly and offer the
					// two real ways forward instead of a plugin fallback note.
					title, hint := "还没有读取到 Office 配置。", "刷新后重试；也可以打开插件页完成配置。"
					if !v.Available || !m.HarnessReady {
						title, hint = "连接 Host 后显示 Office 配置。", "连接到 Host 后刷新；也可以打开插件页完成配置。"
					}
					ui.Column(c).FillWidth().Background(chrome.Color(colors.Field)).Border(1, border).Radius(12).Padding(20).Gap(8).Children(func() {
						ui.Text(c, title).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
						ui.Text(c, hint).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						if m.OnOpenPluginWeb != nil && ui.Button(c, "打开插件页").Clicked() {
							s.Hide()
							m.OnOpenPluginWeb()
						}
					})
					return
				}
				ui.Text(c, officestore.StateLabel(v.Status)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
				ui.Text(c, "最近读取："+v.UpdatedAt+"；点击刷新更新连接状态。").FontSize(theme.InterfaceSize(11)).TextColor(muted)
				if !v.Fresh {
					ui.Text(c, "当前显示上次状态。刷新成功前不可修改配置。").FontSize(theme.InterfaceSize(12)).TextColor(accent)
				}
				h := v.Status.Health
				if v.Status.Configured {
					ui.Text(c, "当前已保存连接："+v.Status.Config.BaseURL+" / "+v.Status.Config.DeviceID).FontSize(theme.InterfaceSize(12)).TextColor(fg)
					heartbeat, event := h.LastHeartbeatAt, h.LastEventType
					if heartbeat == "" {
						heartbeat = "尚无"
					}
					if event == "" {
						event = "尚无"
					}
					ui.Text(c, fmt.Sprintf("最近心跳：%s\n最近事件：%s · 重连 %d 次 · Job Offer %d · 运行 %d · 完成 %d", heartbeat, event, h.Reconnects, h.JobsOffered, h.Jobs.Running, h.Jobs.Completed)).FontSize(theme.InterfaceSize(12)).TextColor(fg)
					if h.Error != nil {
						ui.Text(c, h.Error.Message).FontSize(theme.InterfaceSize(12)).TextColor(accent)
					}
				}
				if v.Conflict {
					ui.Text(c, "配置已在其他窗口更改。草稿保留，未提交覆盖；请核对后放弃草稿并采用最新配置。").TextColor(accent)
				}
				locked := blocked || v.ReviewAction != "" || !v.Fresh || v.Conflict
				draft := v.Draft
				input := func(label string, value *string, hint string) {
					ui.Text(c, label).FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg)
					ui.TextInput(c.Key(label), value).FillWidth().Placeholder(hint).Disabled(locked)
				}
				input("Office Base URL", &draft.BaseURL, "https://office.example.com")
				input("Device ID", &draft.DeviceID, "local-harness")
				ui.Text(c, "Device Token（只写入 Host 凭据存储）").FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg)
				hint := "输入对应 Office 的 Device Token"
				if v.Status.TokenConfigured {
					hint = "已配置；同一 Office 和 Device ID 留空保留"
				}
				ui.TextInput(c.Key("office-token"), &draft.DeviceToken).FillWidth().Password().Placeholder(hint).Disabled(locked)
				ui.Text(c, "不会读取或回显已保存 Token。更换 Office/Device ID 需重新输入；关闭页面或提交后清空输入。").FontSize(theme.InterfaceSize(11)).TextColor(muted)
				input("最大并发（1–4）", &draft.MaxConcurrency, "1")
				input("Heartbeat 秒数（10–300）", &draft.HeartbeatSeconds, "30")
				ui.Text(c, "Workspace 映射：每行 alias=Host 上的绝对路径；Office 仅收到 alias。").FontSize(theme.InterfaceSize(12)).TextColor(fg)
				ui.TextArea(c.Key("office-workspaces"), &draft.Workspaces).FillWidth().Lines(3, 5).Disabled(locked)
				ui.Text(c, "Instruction Preset 映射：每行 alias=指令。").FontSize(theme.InterfaceSize(12)).TextColor(fg)
				ui.TextArea(c.Key("office-presets"), &draft.InstructionPresets).FillWidth().Lines(3, 5).Disabled(locked)
				if draft != v.Draft {
					s.Edit(v.Scope, draft)
				}
				if v.Dirty || v.Conflict {
					if ui.Button(c, "放弃草稿，采用最新配置").Disabled(v.Busy).Clicked() {
						s.Discard(v.Scope)
					}
				}
				if v.ReviewAction != "" {
					ui.Column(c).Background(chrome.Color(colors.Field)).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
						ui.Text(c, "确认 Office 操作").Bold().TextColor(fg)
						ui.Text(c, officestore.ReviewSummary(v)).FontSize(theme.InterfaceSize(12)).TextColor(fg)
						ui.Text(c, "提交前会重读配置；Host 不提供原子版本校验，请避免同时在其他窗口修改。").FontSize(theme.InterfaceSize(11)).TextColor(muted)
						ui.Row(c).Gap(8).Children(func() {
							if ui.Button(c, "确认执行").Disabled(blocked || !v.Fresh || v.Conflict).Clicked() {
								s.Confirm(context.Background(), v.Scope)
							}
							if ui.Button(c, "取消").Disabled(v.Busy).Clicked() {
								s.Dismiss(v.Scope)
							}
						})
					})
				} else {
					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, "保存并连接…").Disabled(locked).Clicked() {
							s.Review(v.Scope, "configure")
						}
						if ui.Button(c, "测试连接…").Disabled(locked || !v.Status.Configured || v.Dirty).Clicked() {
							s.Review(v.Scope, "test")
						}
						if ui.Button(c, "重新连接…").Disabled(locked || !v.Status.Configured || v.Dirty).Clicked() {
							s.Review(v.Scope, "reconnect")
						}
						if ui.Button(c, "移除连接…").Disabled(locked || !v.Status.Configured || v.Dirty).Clicked() {
							s.Review(v.Scope, "remove")
						}
					})
				}
				ui.Text(c, "配置保存成功不代表已连接。Office Hook 尚未上线时，Host 会自动重试；404 通常表示 Hook 未就绪或地址不正确。").FontSize(theme.InterfaceSize(11)).TextColor(muted)
				ui.Text(c, "协议 Hook 预览（由 Office Base URL 派生）").Bold().TextColor(fg)
				hooks := officestore.HookURLs(draft.BaseURL)
				for _, name := range []string{"SSE", "Heartbeat", "Job", "Result"} {
					value := hooks[name]
					if value == "" {
						value = "Base URL 无效"
					}
					ui.Text(c, name+": "+value).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
			})
		})
	})
}
