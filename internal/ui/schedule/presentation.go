package schedule

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func scheduleText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleZH {
		return zh
	}
	return en
}

// The catalog and detail use the original TaskManagerPage.module.css geometry.
func scheduleGutter(width float32) float32 { return min(float32(48), max(float32(24), width*.04)) }
func detailGutter(width float32) float32 {
	if width <= 400 {
		return 16
	}
	if width <= 1100 {
		return 20
	}
	return 24
}

func (p *Panel) catalogView(c *ui.Context, m Model, snap schedstore.ViewState, colors theme.Colors, width float32) {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	ui.Scroll(c.Key("schedule-catalog-scroll")).Label("Task catalog scroll").Grow(1).MinWidth(0).MinHeight(0).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Justify(ui.Center).Children(func() {
			ui.Column(c.Key("schedule-catalog-content")).Label("Task catalog content").Grow(1).MaxWidth(960).MinWidth(0).Padding(0, scheduleGutter(width), 48).Children(func() {
				ui.Row(c.Key("schedule-heading")).FillWidth().MinWidth(0).Wrap().Gap(16).Padding(28, 0, 24).AlignItems(ui.Center).Children(func() {
					ui.Text(c, scheduleText("自动化任务", "Automation tasks")).Bind(&p.headingFocus).Role(ui.RoleHeading).Focusable().FocusRing(false).FontSize(theme.InterfaceSize(20)).FixedLineHeight(theme.InterfaceSize(28)).FontWeight(500).TextColor(fg).Grow(1).MinWidth(0)
					if m.OnNewTask != nil {
						newTask := widgets.Button(c, colors, scheduleText("新建", "New"), widgets.ButtonSpec{
							Variant:  widgets.ButtonPrimary,
							Size:     widgets.ButtonSmall,
							Disabled: !snap.ClientReady,
							Leading: func(c *ui.Context) {
								ui.Icon(c, scheduleIcons["plus"]).Size(13, 13).TextColor(chrome.Color(colors.ButtonFG))
							},
						})
						if newTask.Clicked() {
							m.OnNewTask()
						}
					}
				})
				ui.Row(c.Key("schedule-filters")).Label("Task status").FillWidth().Wrap().Gap(8).Padding(0, 0, 14).Children(func() {
					for _, option := range []struct{ id, zh, en string }{{"all", "全部", "All"}, {"active", "已启用", "Enabled"}, {"inactive", "已结束", "Inactive"}} {
						b := ui.Button(c.Key("schedule-filter-"+option.id), scheduleText(option.zh, option.en)).Height(28).Padding(0, 10).Radius(14).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(14))
						if p.statusFilter == option.id || (option.id == "all" && p.statusFilter == "") {
							b.Background(chrome.Color(colors.Hover)).TextColor(fg).FontWeight(500)
						}
						if b.Clicked() {
							p.statusFilter = option.id
						}
					}
				})
				ui.Row(c.Key("schedule-search-field")).FillWidth().MinWidth(0).Height(36).Padding(0, 10).Gap(6).Border(.5, chrome.Color(colors.Border)).Radius(12).Background(chrome.Color(colors.Field)).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, scheduleIcons["search"]).Size(14, 14).TextColor(muted).Shrink(0)
					ui.TextInputBase(c.Key("schedule-search"), &p.query).Label("Search tasks").Placeholder(scheduleText("搜索自动化任务", "Search automation tasks")).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).TextColor(fg)
					if p.query != "" && scheduleIconButton(c, "schedule-search-clear", "Clear search", "close", muted).Clicked() {
						p.query = ""
					}
				})
				ui.Box(c).Height(16)
				rows := schedstore.FilterEntries(snap.Rows, p.query, p.statusFilter)
				feedback := false
				if p.selected == "" {
					feedback = p.catalogFeedback(c, m, snap, colors, len(rows) > 0)
				}
				if !feedback && len(rows) == 0 {
					label := scheduleText("没有匹配的任务", "No matching tasks")
					if len(snap.Rows) == 0 {
						label = scheduleText("暂无任务。在会话中创建的任务会显示在这里。", "No tasks yet. Tasks created in your sessions appear here.")
					}
					if p.statusFilter == "inactive" && strings.TrimSpace(p.query) == "" {
						label = scheduleText("暂无已结束的自动化任务", "No inactive automation tasks")
					}
					scheduleEmpty(c, label, colors)
					if m.OnNewTask != nil {
						ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
							if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), scheduleText("新建自动化任务", "New automation task"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: !snap.ClientReady}).Clicked() {
								m.OnNewTask()
							}
						})
					}
				}
				ui.Column(c.Key("schedule-catalog-rows")).FillWidth().MinWidth(0).Gap(2).Children(func() {
					for _, entry := range rows {
						row := entry
						if p.rowFocus == nil {
							p.rowFocus = map[string]*ui.Handle{}
						}
						if p.rowFocus[taskKey(row)] == nil {
							p.rowFocus[taskKey(row)] = &ui.Handle{}
						}
						b := ui.Button(c.Key("schedule-row-"+taskKey(row)), "").Bind(p.rowFocus[taskKey(row)]).Label("Open task: " + row.Title).FillWidth().MinWidth(0).Padding(8).Radius(12).BorderWidth(0)
						rowBg := ui.Transparent
						if p.selected == taskKey(row) || b.Hovered() || b.FocusWithin() {
							rowBg = chrome.Color(colors.Hover)
						}
						b.Background(rowBg)
						b.Children(func() {
							ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Start).Children(func() {
								ui.Icon(c, scheduleIcons["clock"]).Size(16, 20).TextColor(muted).Shrink(0)
								ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
									ink := fg
									if row.Status == "inactive" && p.selected != taskKey(row) && !b.Hovered() {
										ink = muted
									}
									ui.Text(c, row.Title).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(23)).FontWeight(500).NoWrap().Ellipsis("…").FillWidth().MinWidth(0).TextColor(ink)
									ui.Text(c, scheduleRowSummary(row)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(21)).FillWidth().MinWidth(0).TextColor(muted)
								})
							})
						})
						if b.Clicked() {
							p.selectCatalogTask(row)
							p.returnRow = taskKey(row)
						}
					}
				})
			})
		})
	})
}

func scheduleEmpty(c *ui.Context, text string, colors theme.Colors) {
	ui.Column(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Padding(48, 20).Gap(12).Children(func() {
		ui.Icon(c, scheduleIcons["clock"]).Size(24, 24).TextColor(chrome.Color(colors.Caption))
		ui.Text(c, text).FontSize(theme.InterfaceSize(14)).TextColor(chrome.Color(colors.Muted)).FillWidth().MinWidth(0).TextAlign(ui.Center)
	})
}

// Initial loading and failures are explicit; refreshing retained rows never blanks them.
func (p *Panel) catalogFeedback(c *ui.Context, m Model, snap schedstore.ViewState, colors theme.Colors, populated bool) bool {
	if !snap.ClientReady {
		scheduleEmpty(c, scheduleText("连接 Host 后可管理任务。", "Connect to the Host to manage tasks."), colors)
		return true
	}
	if snap.Loading && !populated {
		ui.Column(c).Label("Loading tasks").FillWidth().AlignItems(ui.Center).Padding(48, 20).Children(func() { ui.Spinner(c).Size(20, 20).TextColor(chrome.Color(colors.Muted)) })
		return true
	}
	if snap.Err != "" {
		if p.catalogError != snap.Err || p.catalogErrorRequest != snap.CatalogRequest {
			p.catalogError, p.catalogErrorRequest, p.catalogDismissed = snap.Err, snap.CatalogRequest, false
		}
		retry := func() { m.Store.Refresh(context.Background(), m.SessionID) }
		if p.catalogDismissed {
			ui.Column(c.Key("schedule-catalog-retry")).FillWidth().MinWidth(0).Gap(8).Padding(12, 16).Role(ui.RoleStatus).Children(func() {
				ui.Text(c, scheduleText("任务列表待重新读取", "The task list needs a fresh read.")).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted))
				if widgets.Button(c, colors, scheduleText("重试", "Retry"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: snap.Loading}).Clicked() {
					retry()
				}
			})
		} else {
			scheduleNotice(c, "schedule-catalog-failure", scheduleText("无法加载任务。", "Could not load tasks."), colors, 24, []scheduleNoticeAction{{Label: scheduleText("重试", "Retry"), Disabled: snap.Loading, Run: retry}}, func() { p.catalogDismissed = true })
		}
		return true
	}
	p.catalogError, p.catalogDismissed = "", false
	return false
}

func (p *Panel) detailView(c *ui.Context, m Model, snap schedstore.ViewState, d *taskDraft, colors theme.Colors, width float32) {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	available := false
	for _, row := range snap.Rows {
		if taskKey(row) == taskKey(d.row) {
			available = true
			break
		}
	}
	p.mu.Lock()
	pending := p.actionPending
	p.mu.Unlock()
	editable := available && d.row.Status != "inactive" && snap.ClientReady && !snap.Busy && !snap.Loading && !pending
	detail := ui.Column(c.Key("schedule-detail")).Bind(&p.detailFocus).Label("Task details").Focusable().FocusRing(false).MinWidth(0).MinHeight(0)
	if p.focusDetail {
		detail.Focus()
		p.focusDetail = false
	}
	if m.DetailID != "" || width <= 760 {
		detail.Grow(1).FillWidth()
	} else {
		detail.WidthPercent(47).Border(.5, chrome.Color(colors.Border)).BorderWidth(0, 0, 0, .5)
	}
	detail.Children(func() {
		p.detailStrip(c, m, snap, d, colors, width)
		if snap.Err != "" {
			p.catalogFeedback(c, m, snap, colors, true)
		}
		if p.detailTab == "records" {
			p.mu.Lock()
			load := p.historyKey != taskKey(d.row) || p.historyLatest != latestDelivery(d.row) || (!p.historyLoading && !p.historyState.Loaded && p.historyState.Error == "")
			p.mu.Unlock()
			if load {
				p.loadDeliveryHistory(m, d.row, "", true)
			}
			p.deliveryHistoryView(c, m, fg, muted)
		} else {
			ui.Scroll(c.Key("schedule-rule-scroll-" + taskKey(d.row))).Label("Task rules scroll").Grow(1).MinHeight(0).FillWidth().MinWidth(0).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Padding(24, detailGutter(width), 80).Gap(4).Children(func() {
					if !available {
						ui.Text(c, scheduleText("任务已不可用。草稿已保留，无法保存。", "This task is unavailable. Your draft is retained and cannot be saved.")).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
					}
					if d.row.Status == "inactive" {
						ui.Text(c, d.Title).FontSize(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg).FillWidth().MinWidth(0)
					} else {
						ui.TextInputBase(c.Key("schedule-task-name"), &d.Title).Label("Automation task name").FillWidth().MinWidth(0).Height(32).FontSize(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg).Disabled(!editable)
					}
					ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Children(func() {
						ui.Text(c, scheduleNextRun(d.row)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
					})
					ui.Box(c).Height(16)
					if d.row.Status == "inactive" {
						promptCard := ui.Column(c).FillWidth().MinWidth(0).Padding(12).Radius(16).Background(chrome.Color(colors.Field))
						// The prompt is the installed quiet field, not an elevated card.
						promptCard.Children(func() {
							ui.Text(c, d.Prompt).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(24)).TextColor(fg).FillWidth().MinWidth(0).Selectable()
						})
					} else {
						ui.TextArea(c.Key("schedule-task-instruction"), &d.Prompt).Label("Automation task instruction").FillWidth().MinWidth(0).Height(112).MaxHeight(160).Padding(12).BorderWidth(0).Radius(16).Background(chrome.Color(colors.Field)).FontSize(theme.InterfaceSize(14)).TextColor(fg).Disabled(!editable)
					}
					ui.Box(c).Height(20)
					ui.Text(c, scheduleText("运行时间", "Run time")).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(500).TextColor(muted)
					runCard := ui.Column(c).FillWidth().MinWidth(0).Padding(0, 12).Gap(0).Border(.5, chrome.Color(colors.Border)).Radius(16)
					runCard.Background(ui.Transparent)
					runCard.Children(func() {
						if d.row.Status == "inactive" {
							ui.Text(c, scheduleFrequency(d.row)).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg).FillWidth().MinWidth(0)
							if d.Zone != "" {
								ui.Text(c, d.Zone).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
							}
							ui.Text(c, scheduleText("已结束的任务为只读，无法修改其运行时间。", "Inactive tasks are read-only. Their timing cannot be edited.")).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
						} else {
							p.timingEditor(c, d, editable)
						}
					})
				})
			})
		}
		if d.dirty() && d.row.Status != "inactive" {
			saveBar := ui.Row(c.Key("schedule-save-bar")).Label("Unsaved task changes").Absolute().Left(12).Right(12).Bottom(12).Padding(8, 12).Gap(8).AlignItems(ui.Center).BorderWidth(0).Radius(12)
			chrome.GlassSurface(saveBar, chrome.Color(colors.Field), chrome.MaterialPanel, 12)
			saveBar.Children(func() {
				if width > 760 {
					ui.Box(c).Grow(1)
				} else {
					ui.Box(c).Grow(1)
				}
				palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
				cancel := widgets.Button(c, palette, scheduleText("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: snap.Busy || pending})
				cancel.Label("Cancel task changes")
				if cancel.Clicked() {
					*d = *newTaskDraft(d.row)
					p.message("")
				}
				if widgets.Button(c, palette, scheduleText("保存更改", "Save changes"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: !editable}).Clicked() {
					p.saveDraft(m, d)
				}
			})
		}
		if p.detailTab != "records" {
			if notice := scheduleSessionNotice(m, d.row.SessionID); notice != "" {
				ui.Text(c, notice).Role(ui.RoleStatus).FillWidth().MinWidth(0).Padding(8, detailGutter(width), 12).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
		}
		p.detailFeedback(c, m, d, colors, !editable)
	})
}

func (p *Panel) detailStrip(c *ui.Context, m Model, snap schedstore.ViewState, d *taskDraft, colors theme.Colors, width float32) {
	muted := chrome.Color(colors.Muted)
	height := float32(44)
	if m.DetailID != "" {
		height = 37
	}
	ui.Row(c.Key("schedule-detail-strip")).Label("Task detail views").FillWidth().MinWidth(0).Height(height).Shrink(0).Padding(0, detailGutter(width)).Gap(12).Border(.5, chrome.Color(colors.Border)).BorderWidth(0, 0, .5, 0).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(36).Children(func() {
			var tabs []ui.Element
			for _, option := range []struct{ id, zh, en string }{{"rule", "规则", "Rules"}, {"records", "记录", "Records"}} {
				selected := p.detailTab == option.id || p.detailTab == "" && option.id == "rule"
				button := ui.Button(c.Key("schedule-tab-"+option.id), scheduleText(option.zh, option.en)).Height(height).Padding(0).Radius(0).Background(ui.Transparent).BorderWidth(0).TextColor(muted).FontSize(theme.InterfaceSize(13))
				if selected {
					button.TextColor(chrome.Color(colors.Accent)).FontWeight(500).Border(2, chrome.Color(colors.Accent)).BorderWidth(0, 0, 2, 0)
				}
				if button.Clicked() {
					p.selectDetailTab(option.id)
				}
				tabs = append(tabs, button)
			}
			for index, button := range tabs {
				next := -1
				if button.Shortcut(0, ui.KeyLeft) || button.Shortcut(0, ui.KeyRight) {
					next = 1 - index
				} else if button.Shortcut(0, ui.KeyHome) {
					next = 0
				} else if button.Shortcut(0, ui.KeyEnd) {
					next = 1
				}
				if next >= 0 {
					p.selectDetailTab([]string{"rule", "records"}[next])
					tabs[next].Focus()
				}
			}
		})
		ui.Row(c).Shrink(0).Gap(0).Children(func() {
			if d != nil {
				if m.ViewerSessionID != d.row.SessionID && m.OnOpenSession != nil && p.detailTab != "records" {
					label, allowed := m.SessionLinks[d.row.SessionID]
					if m.SessionLinkState != nil {
						allowed = m.SessionLinkState(d.row.SessionID) == "available"
					}
					if label == "" {
						label = d.row.SessionID
					}
					if notice := scheduleSessionNotice(m, d.row.SessionID); notice != "" {
						label += "\n" + notice
					}
					if scheduleIconButton(c, "schedule-linked-session", scheduleText("打开原会话", "Open original session"), "session", muted).Tooltip(label).Disabled(!allowed || d.row.Origin != snap.Origin).Clicked() {
						m.OnOpenSession(d.row)
					}
				}
				more := scheduleIconButton(c, "schedule-task-actions", "Automation task actions", "more", muted).Bind(&p.deleteTrigger)
				more.Menu(func(menu *ui.Menu) {
					if menu.Item(scheduleText("删除任务", "Delete task")).Disabled(snap.Busy || snap.Loading || !snap.ClientReady || !scheduleEntryAvailable(snap, d.row)).Chosen() {
						row := d.row
						p.deleting = &row
						p.focusDelete = true
					}
				})
			}
			closeLabel := "Close details"
			if m.DetailID != "" {
				closeLabel = "Close task"
			}
			if scheduleIconButton(c, "schedule-close-detail", closeLabel, "close", muted).Clicked() {
				p.dismissDetails(c, m, snap)
			}
		})
	})
}

func (p *Panel) deleteDialog(c *ui.Context, m Model, snap schedstore.ViewState, colors theme.Colors) {
	if p.deleting == nil {
		return
	}
	row := *p.deleting
	open := true
	_, height := c.Size()
	ui.DialogBase(c.Key("schedule-delete-dialog"), &open, func(backdrop, panel ui.Element) {
		panel.Label("Delete this task?").Width(480).MaxWidthPercent(92).MaxHeight(max(float32(0), height-48)).Background(chrome.Color(colors.BG)).Radius(28).Padding(20).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).MinHeight(0).Gap(16).Children(func() {
				ui.Text(c, scheduleText("删除此任务？", "Delete this task?")).FontSize(theme.InterfaceSize(18)).FontWeight(500).TextColor(chrome.Color(colors.FG))
				ui.Scroll(c.Key("schedule-delete-content")).MinHeight(0).MaxHeight(max(float32(40), height-180)).FillWidth().Children(func() {
					ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
						ui.Text(c, scheduleText("这将停止任务触发，并删除任务及其已保存的投递记录。原会话和消息将保留；已排队的消息不会撤回。", "This stops the task from triggering and deletes it together with its saved delivery records. The original session and its messages remain; queued messages are not withdrawn.")).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.Muted)).FillWidth().MinWidth(0)
						ui.Text(c, row.Title).FontSize(theme.InterfaceSize(14)).TextColor(chrome.Color(colors.FG)).FillWidth().MinWidth(0)
					})
				})
				ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
					palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
					cancel := widgets.Button(c, palette, scheduleText("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonMedium, Disabled: snap.Busy}).Bind(&p.deleteCancel)
					cancel.Label("Cancel task deletion")
					if p.focusDelete {
						cancel.Focus()
						p.focusDelete = false
					}
					if cancel.Clicked() {
						p.deleting = nil
						p.deleteTrigger.RequestFocus(c)
					}
					if widgets.Button(c, palette, scheduleText("确认删除", "Confirm deletion"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonMedium, Disabled: snap.Busy || !snap.ClientReady}).Clicked() {
						p.deleting = nil
						p.deleteTrigger.RequestFocus(c)
						p.deleteTask(m, row)
					}
				})
			})
		})
		chrome.DialogDecor(c, backdrop, panel, colors, 28)
	})
	if !open {
		p.deleting = nil
		p.deleteTrigger.RequestFocus(c)
	}
}

func scheduleRowSummary(row schedstore.Entry) string {
	parts := []string{}
	if row.Status == "inactive" {
		parts = append(parts, scheduleText("已结束", "Inactive"))
	}
	parts = append(parts, scheduleFrequency(row))
	if row.Status == "active" {
		parts = append(parts, scheduleNextRun(row))
	}
	return strings.Join(parts, " · ")
}

func scheduleNextRun(row schedstore.Entry) string {
	if row.Status == "inactive" {
		return scheduleText("已结束", "Inactive")
	}
	at, err := time.Parse(time.RFC3339Nano, row.ScheduledAt)
	if err != nil {
		return scheduleText("下次运行时间不可用", "Next run unavailable")
	}
	// The manager's next-run pair uses the reader's clock, not the rule zone.
	// Saved delivery records use the rule zone separately in deliveryTime.
	now := time.Now()
	return scheduleText("下次运行：", "Next run: ") + scheduleAbsolute(at.In(time.Local)) + " (" + scheduleRelative(at, now) + ")"
}

func scheduleFrequency(row schedstore.Entry) string {
	if text := formatManagerFrequency(row.Raw, time.Now(), i18n.Locale(), systemScheduleZone()); text != "" {
		return text
	}
	return scheduleText("运行规则不可用", "Run rule unavailable")
}

func scheduleIconButton(c *ui.Context, key, label, icon string, ink ui.Color) ui.Element {
	b := ui.Button(c.Key(key), "").Label(label).Tooltip(label).Size(28, 28).Padding(6).BorderWidth(0).Background(ui.Transparent).Radius(6)
	b.Children(func() { ui.Icon(c, scheduleIcons[icon]).Size(16, 16).TextColor(ink) })
	return b
}

var scheduleIcons = map[string]*ui.SVG{
	"clock":   ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M8 14a6 6 0 1 0 0-12 6 6 0 0 0 0 12Z" stroke="currentColor"/><path d="M8 4.31v4.15L11 10.08" stroke="currentColor"/></svg>`)),
	"search":  ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><circle cx="7" cy="7" r="4.5" stroke="currentColor"/><path d="m10.5 10.5 3 3" stroke="currentColor"/></svg>`)),
	"close":   ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="m4 4 8 8M12 4l-8 8" stroke="currentColor"/></svg>`)),
	"more":    ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="currentColor"><circle cx="3" cy="8" r="1"/><circle cx="8" cy="8" r="1"/><circle cx="13" cy="8" r="1"/></svg>`)),
	"session": ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M13.5 10a2 2 0 0 1-2 2H6l-3.5 2V4a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v6Z" stroke="currentColor"/></svg>`)),
	"plus":    ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M8 3v10M3 8h10" stroke="currentColor"/></svg>`)),
}
