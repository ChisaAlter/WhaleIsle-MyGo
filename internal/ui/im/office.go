package im

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
	officestore "whaleisle/internal/store/office"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func officeSettingsView(c *ui.Context, m Model) {
	s := &m.Store.Office
	v := s.Snapshot()
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)
	blocked := v.Busy || !v.Available || m.PluginDisabled || !m.HarnessReady
	page := ui.Column(c.Key(fmt.Sprintf("office-%d", v.Scope))).Fill().MinWidth(0).MinHeight(0).Background(chrome.Color(colors.BG))
	feedback := ui.Local(page, "feedback", func() string { return "" })
	message, kind := v.Notice, "success"
	if v.Error != "" {
		message, kind = v.Error, "error"
	}
	if message != "" && message != *feedback {
		c.AddToast(ui.Toast{ID: fmt.Sprintf("office-%d", v.Scope), Title: message, Type: kind, Timeout: 7 * time.Second})
	}
	*feedback = message
	page.Children(func() {
		ui.Row(c).Height(44).Shrink(0).Padding(6, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "AI Office").FontSize(theme.InterfaceSize(15)).Bold().Grow(1).MinWidth(0).TextColor(fg)
			caption := "刷新状态"
			if v.Busy {
				caption = "读取中…"
			}
			if ui.Button(c, caption).Width(82).Height(28).Disabled(blocked).Tooltip("当前 Host：" + v.Host + "\n最近读取：" + v.UpdatedAt).Clicked() {
				s.Refresh(context.Background())
			}
			ui.Button(c, "•••").Size(28, 28).Label("Office 页面选项").Menu(func(menu *ui.Menu) {
				if m.OnOpenPluginWeb != nil && menu.Item("打开原版插件页面").Chosen() {
					s.Hide()
					m.OnOpenPluginWeb()
				}
				if menu.Item("返回 IM").Chosen() {
					s.Hide()
				}
				if m.OnClose != nil && menu.Item("关闭").Chosen() {
					s.Hide()
					m.OnClose()
				}
			})
		})
		ui.Box(c).FillWidth().Height(0.5).Background(border)
		scroll := ui.Scroll(c).Grow(1).MinHeight(0).MinWidth(0).Padding(16)
		width := scroll.Bounds().W
		if width <= 0 {
			width, _ = c.Size()
		}
		narrow := width <= 680
		card := func(key string, body func()) {
			ui.Column(c.Key(key)).FillWidth().MinWidth(0).Padding(16).Gap(12).Radius(14).Border(0.5, border).Background(chrome.Color(colors.Field)).Children(body)
		}
		scroll.Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(16).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(16).Padding(18).Radius(16).Border(0.5, border).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Gap(6).Children(func() {
						ui.Text(c, "AI Office Connector").FontSize(theme.InterfaceSize(17)).Bold().TextColor(fg)
						ui.Text(c, "让当前 Host 主动接入 Office，接收任务并回传执行结果。").FontSize(theme.InterfaceSize(12)).TextColor(muted)
					})
					ink := muted
					if v.Status.Connected {
						ink = chrome.Color(colors.Accent)
					}
					pill := officestore.StateLabel(v.Status)
					if v.Busy {
						pill = "正在读取…"
					}
					ui.Text(c, pill).MaxWidth(180).FontSize(theme.InterfaceSize(12)).TextColor(ink).Padding(6, 10).Radius(20).Background(chrome.Color(colors.ToolbarFill))
				})
				if !v.Loaded {
					card("unavailable", func() {
						title, hint := "还没有读取到 Office 配置。", "刷新后重试；也可以打开插件页完成配置。"
						if v.Busy {
							title, hint = "正在读取 Office 配置…", ""
						} else if !m.HarnessReady {
							title, hint = "连接 Host 后显示 Office 配置。", "连接到 Host 后刷新。"
						} else if !v.Available || m.PluginDisabled {
							title, hint = "dsh-im 插件未就绪。", "安装并启用 dsh-im 后即可读取和修改 Office 配置。"
						}
						ui.Text(c, title).FontSize(14).Bold().TextColor(fg)
						if hint != "" {
							ui.Text(c, hint).FontSize(12).TextColor(muted)
						}
						if m.OnOpenPluginWeb != nil && ui.Button(c, "打开插件页").Clicked() {
							s.Hide()
							m.OnOpenPluginWeb()
						}
					})
					return
				}
				h := v.Status.Health
				metrics := []struct{ label, value string }{{"最近心跳", h.LastHeartbeatAt}, {"最近事件", h.LastEventType}, {"重连次数", fmt.Sprint(h.Reconnects)}, {"Job Offer", fmt.Sprint(h.JobsOffered)}, {"运行中", fmt.Sprint(h.Jobs.Running)}, {"已完成", fmt.Sprint(h.Jobs.Completed)}}
				columns := 3
				if narrow {
					columns = 2
				}
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					for offset := 0; offset < len(metrics); offset += columns {
						start := offset
						ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
							for i := start; i < min(start+columns, len(metrics)); i++ {
								metric := metrics[i]
								if metric.value == "" {
									metric.value = "尚无"
								}
								ui.Column(c.Key(metric.label)).Grow(1).Basis(0).MinWidth(0).Padding(12).Radius(10).Border(0.5, border).Gap(6).Children(func() {
									ui.Text(c, metric.label).FontSize(11).TextColor(muted)
									ui.Text(c, metric.value).FontSize(13).Bold().SingleLine().Ellipsis("…").Tooltip(metric.value).TextColor(fg)
								})
							}
						})
					}
				})
				draft := v.Draft
				locked := blocked || v.ReviewAction != "" || !v.Fresh || v.Conflict
				field := func(name string, value *string, hint string, password bool) {
					ui.Column(c.Key(name)).FillWidth().MinWidth(0).Gap(6).Children(func() {
						ui.Text(c, name).FontSize(12).TextColor(fg)
						input := ui.TextInput(c.Key("input"), value).FillWidth().Height(38).Radius(9).Placeholder(hint).Disabled(locked)
						if password {
							input.Password()
						}
					})
				}
				pair := func(first, second func()) {
					if narrow {
						first()
						second()
					} else {
						ui.Row(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
							ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(first)
							ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(second)
						})
					}
				}
				card("device", func() {
					ui.Text(c, "设备连接").FontSize(14).Bold().TextColor(fg)
					if !v.Fresh {
						ui.Text(c, "当前显示上次状态。刷新成功后可修改配置。").FontSize(12).TextColor(chrome.Color(colors.Accent))
					}
					if v.Conflict {
						ui.Text(c, "配置已在其他窗口更改。草稿保留；核对后放弃草稿以采用最新配置。").FontSize(12).TextColor(chrome.Color(colors.Danger))
					}
					if h.Error != nil {
						ui.Text(c, h.Error.Code+": "+h.Error.Message).FontSize(12).TextColor(chrome.Color(colors.Danger)).Selectable()
					}
					field("Office Base URL", &draft.BaseURL, "https://office.example.com", false)
					hint := "输入对应 Office 的 Device Token"
					if v.Status.TokenConfigured {
						hint = "已配置；留空保留现有 Token"
					}
					pair(func() { field("Device ID", &draft.DeviceID, "local-harness", false) }, func() { field("Device Token", &draft.DeviceToken, hint, true) })
					ui.Text(c, "Token 只写入 Host 凭据存储，不读取或回显。更换 Office 或 Device ID 时需重新输入。").FontSize(11).TextColor(muted)
					pair(func() { field("最大并发（1–4）", &draft.MaxConcurrency, "1", false) }, func() { field("Heartbeat 秒数（10–300）", &draft.HeartbeatSeconds, "30", false) })
					ui.Column(c).FillWidth().Gap(6).Children(func() {
						ui.Text(c, "Workspace 映射").FontSize(12).TextColor(fg)
						ui.TextArea(c.Key("office-workspaces"), &draft.Workspaces).FillWidth().MinHeight(86).Lines(3, 8).Disabled(locked)
						ui.Text(c, "每行 alias=Host 上的绝对路径；Office 仅收到 alias。").FontSize(11).TextColor(muted)
					})
					ui.Column(c).FillWidth().Gap(6).Children(func() {
						ui.Text(c, "Instruction Preset 映射").FontSize(12).TextColor(fg)
						ui.TextArea(c.Key("office-presets"), &draft.InstructionPresets).FillWidth().MinHeight(86).Lines(3, 8).Disabled(locked)
						ui.Text(c, "每行 alias=指令。").FontSize(11).TextColor(muted)
					})
					if draft != v.Draft {
						s.Edit(v.Scope, draft)
					}
					ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
						for _, action := range []struct {
							name, key string
							disabled  bool
						}{{"保存并连接…", "configure", false}, {"测试连接…", "test", !v.Status.Configured || v.Dirty}, {"重新连接…", "reconnect", !v.Status.Configured || v.Dirty}, {"移除连接…", "remove", !v.Status.Configured || v.Dirty}} {
							if ui.Button(c, action.name).Height(32).Disabled(locked || action.disabled).Clicked() {
								s.Review(v.Scope, action.key)
							}
						}
						if (v.Dirty || v.Conflict) && ui.Button(c, "放弃草稿，采用最新配置").Height(32).Disabled(v.Busy).Clicked() {
							s.Discard(v.Scope)
						}
					})
					ui.Text(c, "配置已保存不代表已连接。Hook 未上线时 Host 会重试；404 通常表示 Hook 未就绪或地址不正确。").FontSize(11).TextColor(muted)
				})
				card("hooks", func() {
					ui.Text(c, "协议 Hook 预览").FontSize(14).Bold().TextColor(fg)
					ui.Text(c, officestore.ProtocolVersion+" · Host 主动建立连接，不开放入站端口。").FontSize(12).TextColor(muted)
					hooks := officestore.HookURLs(draft.BaseURL)
					for _, name := range []string{"SSE", "Heartbeat", "Job", "Result"} {
						value := hooks[name]
						if value == "" {
							value = "Base URL 无效"
						}
						ui.Row(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
							ui.Text(c, name).Width(82).Shrink(0).FontSize(12).TextColor(muted)
							ui.Text(c, value).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(value).Selectable().FontSize(12).TextColor(fg)
						})
					}
				})
			})
		})
	})
	if v.ReviewAction != "" {
		open := true
		ui.DialogBase(c.Key("office-review"), &open, func(backdrop, panel ui.Element) {
			chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 500, 20).MaxHeightPercent(88).Gap(12).Children(func() {
				ui.Text(c, "确认 Office 操作").FontSize(16).Bold().TextColor(fg)
				ui.Scroll(c).FillWidth().MinHeight(0).Children(func() {
					ui.Column(c).FillWidth().Gap(8).Children(func() {
						ui.Text(c, officestore.ReviewSummary(v)).FontSize(12).TextColor(fg)
						ui.Text(c, "提交前重读配置。请避免同时在其他窗口修改。").FontSize(11).TextColor(muted)
					})
				})
				ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, "取消").Disabled(v.Busy).Clicked() {
						open = false
					}
					caption := "确认执行"
					if v.Busy {
						caption = "执行中…"
					}
					if ui.PrimaryButton(c, caption).Disabled(blocked || !v.Fresh || v.Conflict).Clicked() {
						s.Confirm(context.Background(), v.Scope)
					}
				})
			})
		})
		if !open {
			s.Dismiss(v.Scope)
		}
	}
}
