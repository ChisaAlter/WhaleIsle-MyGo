package im

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/errcopy"
	officestore "whaleisle/internal/store/office"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func officeSettingsView(c *ui.Context, m Model) {
	s := &m.Store.Office
	v := s.Snapshot().Value
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)
	danger := design.For(c).Destructive
	blocked := v.Busy || !v.Available || m.PluginDisabled || !m.HarnessReady

	refreshCaption := channelText("刷新状态", "Refresh status")
	if v.Busy {
		refreshCaption = channelText("读取中…", "Loading…")
	}
	refreshTip := channelText("当前 Host：", "Current Host: ") + v.Host + "\n" + channelText("最近读取：", "Last loaded: ") + v.UpdatedAt

	page := ui.Column(c.Key(fmt.Sprintf("office-%d", v.Scope))).FillWidth().MinWidth(0).Background(ui.Transparent)
	if !m.Embedded {
		page.Grow(1).MinHeight(0)
	}
	page.Children(func() {
		widgets.AckLine(c, "office-notice", v.Notice)
		if !m.Embedded {
			leftPad, rightPad := chrome.TitleBarSafeHorizontalPadding(c.TitleBar(), design.Sp(3))
			ui.Row(c.Key("office-header")).FillWidth().MinWidth(0).Height(44).Shrink(0).Padding(design.Sp(1.5), rightPad, design.Sp(1.5), leftPad).Gap(design.Sp(2)).AlignItems(ui.Center).DragWindow().Children(func() {
				ui.Text(c, "AI Office").FontSize(theme.InterfaceSize(15)).Bold().Grow(1).MinWidth(0).SingleLine().Ellipsis("…").TextColor(fg)
				if coss.Button(c, refreshCaption, coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonSM,
					Disabled: blocked,
					Key:      "office-refresh",
				}).Shrink(0).Tooltip(refreshTip).Clicked() {
					s.Refresh(context.Background())
				}
				coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonOutline,
					Size:    coss.ButtonIconSM,
					Icon:    coss.IconEllipsis,
					Label:   channelText("Office 页面选项", "Office page options"),
					Key:     "office-options",
				}).Shrink(0).Menu(func(menu *ui.Menu) {
					if m.OnOpenPluginWeb != nil && menu.Item(channelText("打开原版插件页面", "Open original plugin page")).Chosen() {
						s.Hide()
						m.OnOpenPluginWeb()
					}
					if menu.Item(channelText("返回 IM", "Back to IM")).Chosen() {
						s.Hide()
					}
					if m.OnClose != nil && menu.Item(channelText("关闭", "Close")).Chosen() {
						s.Hide()
						m.OnClose()
					}
				})
			})
			ui.Box(c).FillWidth().Height(0.5).Shrink(0).Background(border)
		}
		if v.Error != "" {
			padTop := design.Sp(3)
			padX := design.Sp(3)
			if m.Embedded {
				padTop, padX = 0, 0
			}
			ui.Box(c).FillWidth().Shrink(0).Padding(padTop, padX, 0, padX).Children(func() {
				widgets.Alert(c, widgets.AlertOptions{
					ID:       "office/" + fmt.Sprint(v.Scope),
					Message:  v.Error,
					Action:   channelText("重试", "Retry"),
					OnAction: func() { s.Refresh(context.Background()) },
				})
			})
		}

		windowWidth, _ := c.Size()
		narrow := windowWidth <= 680
		if m.Embedded {
			narrow = windowWidth <= 900
		}
		card := func(key string, body func()) {
			chrome.FrostSurface(ui.Column(c.Key(key)).FillWidth().MinWidth(0).Padding(design.Sp(4)).Gap(design.Sp(3)).Border(0.5, border), colors, m.SystemDark, chrome.Color(colors.Field), design.RadiusXL).Children(body)
		}
		renderBody := func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(4)).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(3)).Padding(design.Sp(4)).Radius(theme.ContextRadii(c).LG).Border(0.5, border).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(1.5)).Children(func() {
						ui.Text(c, "AI Office Connector").FontSize(theme.InterfaceSize(17)).Bold().TextColor(fg)
						ui.Text(c, channelText("让当前 Host 主动接入 Office，接收任务并回传执行结果。", "Let the current Host connect outbound to Office to receive jobs and return results.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					})
					ui.Row(c).Shrink(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
						ink := muted
						if v.Status.Connected {
							ink = chrome.Color(colors.Accent)
						}
						pill := officestore.StateLabel(v.Status)
						if v.Busy {
							pill = channelText("正在读取…", "Loading…")
						}
						ui.Text(c, pill).MaxWidth(180).FontSize(theme.InterfaceSize(12)).TextColor(ink).Padding(design.Sp(1.5), design.Sp(2.5)).Radius(design.RadiusFull).Background(chrome.FrostInset(colors, m.SystemDark, chrome.Color(colors.ToolbarFill)))
						if m.Embedded {
							if coss.Button(c, refreshCaption, coss.ButtonOptions{
								Variant:  coss.ButtonOutline,
								Size:     coss.ButtonSM,
								Disabled: blocked,
								Key:      "office-embedded-refresh",
							}).Shrink(0).Tooltip(refreshTip).Clicked() {
								s.Refresh(context.Background())
							}
						}
					})
				})
				if !v.Loaded {
					card("unavailable", func() {
						title := channelText("还没有读取到 Office 配置。", "Office configuration has not been loaded yet.")
						hint := channelText("刷新后重试；也可以打开插件页完成配置。", "Refresh to try again, or open the plugin page to configure it.")
						if v.Busy {
							title, hint = channelText("正在读取 Office 配置…", "Loading Office configuration…"), ""
						} else if !m.HarnessReady {
							title = channelText("连接 Host 后显示 Office 配置。", "Connect to a Host to view Office configuration.")
							hint = channelText("连接到 Host 后刷新。", "Refresh after connecting to the Host.")
						} else if !v.Available || m.PluginDisabled {
							title = channelText("dsh-im 插件未就绪。", "The dsh-im plugin is not ready.")
							hint = channelText("安装并启用 dsh-im 后即可读取和修改 Office 配置。", "Install and enable dsh-im to read and update Office settings.")
						}
						ui.Text(c, title).FontSize(14).Bold().TextColor(fg)
						if hint != "" {
							ui.Text(c, hint).FontSize(12).TextColor(muted)
						}
						if m.OnOpenPluginWeb != nil && coss.Button(c, channelText("打开插件页", "Open plugin page"), coss.ButtonOptions{
							Variant: coss.ButtonOutline,
							Size:    coss.ButtonSM,
							Key:     "office-open-plugin",
						}).Clicked() {
							s.Hide()
							m.OnOpenPluginWeb()
						}
					})
					return
				}
				h := v.Status.Health
				metrics := []struct{ label, value string }{
					{channelText("最近心跳", "Last heartbeat"), h.LastHeartbeatAt},
					{channelText("最近事件", "Last event"), h.LastEventType},
					{channelText("重连次数", "Reconnects"), fmt.Sprint(h.Reconnects)},
					{"Job Offer", fmt.Sprint(h.JobsOffered)},
					{channelText("运行中", "Running"), fmt.Sprint(h.Jobs.Running)},
					{channelText("已完成", "Completed"), fmt.Sprint(h.Jobs.Completed)},
				}
				columns := 3
				if narrow {
					columns = 2
				}
				ui.Column(c).FillWidth().Gap(design.Sp(2)).Children(func() {
					for offset := 0; offset < len(metrics); offset += columns {
						start := offset
						ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(func() {
							for i := start; i < min(start+columns, len(metrics)); i++ {
								metric := metrics[i]
								if metric.value == "" {
									metric.value = channelText("尚无", "None")
								}
								ui.Column(c.Key(metric.label)).Grow(1).Basis(0).MinWidth(0).Padding(design.Sp(3)).Radius(design.RadiusLG).Border(0.5, border).Gap(design.Sp(1.5)).Children(func() {
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
					ui.Column(c.Key(name)).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Children(func() {
						ui.Text(c, name).FontSize(12).TextColor(fg)
						coss.Input(c, value, coss.InputOptions{
							Size:        coss.InputMD,
							Placeholder: hint,
							Disabled:    locked,
							Password:    password,
							Label:       name,
							Key:         "input",
						})
					})
				}
				pair := func(first, second func()) {
					if narrow {
						first()
						second()
					} else {
						ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Children(func() {
							ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(first)
							ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(second)
						})
					}
				}
				card("device", func() {
					ui.Text(c, channelText("设备连接", "Device connection")).FontSize(14).Bold().TextColor(fg)
					if !v.Fresh {
						ui.Text(c, channelText("当前显示上次状态。刷新成功后可修改配置。", "Showing the last cached status. Refresh successfully before editing.")).FontSize(12).TextColor(chrome.Color(colors.Accent))
					}
					if v.Conflict {
						ui.Text(c, channelText("配置已在其他窗口更改。草稿保留；核对后放弃草稿以采用最新配置。", "Configuration changed in another window. Your draft is kept; discard it to adopt the latest settings.")).FontSize(12).TextColor(danger)
					}
					if h.Error != nil {
						ui.Text(c, errcopy.SentenceFor(h.Error.Code, h.Error.Message)).FontSize(12).TextColor(danger).Selectable()
					}
					field("Office Base URL", &draft.BaseURL, "https://office.example.com", false)
					hint := channelText("输入对应 Office 的 Device Token", "Enter the Device Token for this Office")
					if v.Status.TokenConfigured {
						hint = channelText("已配置；留空保留现有 Token", "Configured; leave empty to keep the existing token")
					}
					pair(func() { field("Device ID", &draft.DeviceID, "local-harness", false) }, func() { field("Device Token", &draft.DeviceToken, hint, true) })
					ui.Text(c, channelText("Token 只写入 Host 凭据存储，不读取或回显。更换 Office 或 Device ID 时需重新输入。", "The token is only written to Host credential storage and never read back. Re-enter it if you change Office or Device ID.")).FontSize(11).TextColor(muted)
					pair(func() {
						field(channelText("最大并发（1–4）", "Max concurrency (1–4)"), &draft.MaxConcurrency, "1", false)
					}, func() {
						field(channelText("Heartbeat 秒数（10–300）", "Heartbeat interval in seconds (10–300)"), &draft.HeartbeatSeconds, "30", false)
					})
					ui.Column(c).FillWidth().Gap(design.Sp(1.5)).Children(func() {
						ui.Text(c, channelText("Workspace 映射", "Workspace mappings")).FontSize(12).TextColor(fg)
						ui.TextArea(c.Key("office-workspaces"), &draft.Workspaces).FillWidth().MinHeight(86).Lines(3, 8).Disabled(locked)
						ui.Text(c, channelText("每行 alias=Host 上的绝对路径；Office 仅收到 alias。", "One alias=/absolute/path/on/host per line; Office only sees the alias.")).FontSize(11).TextColor(muted)
					})
					ui.Column(c).FillWidth().Gap(design.Sp(1.5)).Children(func() {
						ui.Text(c, channelText("Instruction Preset 映射", "Instruction preset mappings")).FontSize(12).TextColor(fg)
						ui.TextArea(c.Key("office-presets"), &draft.InstructionPresets).FillWidth().MinHeight(86).Lines(3, 8).Disabled(locked)
						ui.Text(c, channelText("每行 alias=指令。", "One alias=instruction per line.")).FontSize(11).TextColor(muted)
					})
					if draft != v.Draft {
						s.Edit(v.Scope, draft)
					}
					ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).Children(func() {
						for _, action := range []struct {
							name, key string
							variant   coss.ButtonVariant
							disabled  bool
						}{
							{channelText("保存并连接…", "Save & connect…"), "configure", coss.ButtonDefault, false},
							{channelText("测试连接…", "Test connection…"), "test", coss.ButtonOutline, !v.Status.Configured || v.Dirty},
							{channelText("重新连接…", "Reconnect…"), "reconnect", coss.ButtonOutline, !v.Status.Configured || v.Dirty},
							{channelText("移除连接…", "Remove connection…"), "remove", coss.ButtonDestructiveOutline, !v.Status.Configured || v.Dirty},
						} {
							if coss.Button(c, action.name, coss.ButtonOptions{
								Variant:  action.variant,
								Size:     coss.ButtonMD,
								Disabled: locked || action.disabled,
								Key:      "office-action-" + action.key,
							}).Clicked() {
								s.Review(v.Scope, action.key)
							}
						}
						if (v.Dirty || v.Conflict) && coss.Button(c, channelText("放弃草稿，采用最新配置", "Discard draft & use latest"), coss.ButtonOptions{
							Variant:  coss.ButtonGhost,
							Size:     coss.ButtonMD,
							Disabled: v.Busy,
							Key:      "office-discard",
						}).Clicked() {
							s.Discard(v.Scope)
						}
					})
					ui.Text(c, channelText("配置已保存不代表已连接。Hook 未上线时 Host 会重试；404 通常表示 Hook 未就绪或地址不正确。", "Saving configuration does not mean it is connected. Host retries until hooks are online; 404 usually means hooks are not ready or the URL is wrong.")).FontSize(11).TextColor(muted)
				})
				card("hooks", func() {
					ui.Text(c, channelText("协议 Hook 预览", "Protocol hook preview")).FontSize(14).Bold().TextColor(fg)
					ui.Text(c, officestore.ProtocolVersion+channelText(" · Host 主动建立连接，不开放入站端口。", " · Host initiates outbound connections; no inbound ports are opened.")).FontSize(12).TextColor(muted)
					hooks := officestore.HookURLs(draft.BaseURL)
					for _, name := range []string{"SSE", "Heartbeat", "Job", "Result"} {
						value := hooks[name]
						if value == "" {
							value = channelText("Base URL 无效", "Invalid Base URL")
						}
						ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Children(func() {
							ui.Text(c, name).Width(82).Shrink(0).FontSize(12).TextColor(muted)
							ui.Text(c, value).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(value).Selectable().FontSize(12).TextColor(fg)
						})
					}
				})
			})
		}

		if m.Embedded {
			renderBody()
		} else {
			ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).MinWidth(0).Padding(design.Sp(4)).Children(renderBody)
		}
	})
	if v.ReviewAction != "" {
		open := true
		ui.DialogBase(c.Key("office-review"), &open, func(backdrop, panel ui.Element) {
			chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 500, 20).MaxHeightPercent(88).Gap(design.Sp(3)).Children(func() {
				ui.Text(c, channelText("确认 Office 操作", "Confirm Office action")).FontSize(16).Bold().TextColor(fg)
				ui.Scroll(c).FillWidth().MinHeight(0).Children(func() {
					ui.Column(c).FillWidth().Gap(design.Sp(2)).Children(func() {
						ui.Text(c, officestore.ReviewSummary(v)).FontSize(12).TextColor(fg)
						ui.Text(c, channelText("提交前重读配置。请避免同时在其他窗口修改。", "Configuration is re-read before applying. Avoid editing in another window at the same time.")).FontSize(11).TextColor(muted)
					})
				})
				ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).Justify(ui.End).Children(func() {
					if coss.Button(c, channelText("取消", "Cancel"), coss.ButtonOptions{
						Variant:  coss.ButtonOutline,
						Size:     coss.ButtonSM,
						Disabled: v.Busy,
						Key:      "office-review-cancel",
					}).Clicked() {
						open = false
					}
					caption := channelText("确认执行", "Confirm")
					if v.Busy {
						caption = channelText("执行中…", "Applying…")
					}
					if coss.Button(c, caption, coss.ButtonOptions{
						Variant:  coss.ButtonDefault,
						Size:     coss.ButtonSM,
						Disabled: blocked || !v.Fresh || v.Conflict,
						Key:      "office-review-confirm",
					}).Clicked() {
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
