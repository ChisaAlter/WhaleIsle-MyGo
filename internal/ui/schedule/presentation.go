package schedule

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
)

func scheduleText(zh, en string) string {
	return i18n.Pick(zh, en)
}

func detailGutterSp(width float32) float32 {
	if width <= 400 {
		return design.Sp(4)
	}
	if width <= 1100 {
		return design.Sp(5)
	}
	return design.Sp(6)
}

func (p *Panel) catalogView(c *ui.Context, m Model, snap schedstore.ViewState, epoch uint64, width float32) {
	t := design.For(c)
	pagekit.Page(c, pagekit.PageOptions{MaxWidth: 1056, Scroll: &p.catalogScroll}, func() {
		pagekit.Header(c, scheduleText("自动化任务", "Automation tasks"), "", func() {
			if m.OnNewTask != nil {
				if coss.Button(c, scheduleText("新建", "New"), coss.ButtonOptions{
					Key:      "schedule-new",
					Icon:     coss.IconPlus,
					Disabled: !snap.ClientReady,
				}).Clicked() {
					m.OnNewTask()
				}
			}
		}).Bind(&p.headingFocus).Focusable().FocusRing(false)

		ui.Row(c.Key("schedule-filters")).Label("Task status").FillWidth().Wrap().Gap(design.Sp(2)).Children(func() {
			for _, option := range []struct{ id, zh, en string }{{"all", "全部", "All"}, {"active", "已开启", "Enabled"}, {"inactive", "已结束", "Inactive"}} {
				on := p.statusFilter == option.id || (option.id == "all" && p.statusFilter == "")
				if coss.Toggle(c, &on, scheduleText(option.zh, option.en), coss.ToggleOptions{
					Key:     "schedule-filter-" + option.id,
					Variant: coss.ToggleOutline,
					Size:    coss.ToggleSM,
				}).Radius(design.RadiusFull).Changed() {
					p.statusFilter = option.id
				}
			}
		})

		query := p.query
		var clearEnd func()
		if p.query != "" {
			clearEnd = func() {
				if coss.Button(c, "", coss.ButtonOptions{
					Key:     "schedule-search-clear",
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconXS,
					Icon:    coss.IconX,
					Label:   "Clear search",
				}).Clicked() {
					p.query = ""
				}
			}
		}
		if coss.InputGroup(c, &query, coss.InputGroupOptions{
			InputOptions: coss.InputOptions{
				Key:         "schedule-search",
				Label:       "Search tasks",
				Placeholder: scheduleText("搜索自动化任务", "Search automation tasks"),
			},
			Start: func() { coss.AddonIcon(c, coss.IconSearch) },
			End:   clearEnd,
		}).Changed() {
			p.query = query
		}

		rows := schedstore.FilterEntries(snap.Rows, p.query, p.statusFilter)
		feedback := false
		if p.selected == "" {
			feedback = p.catalogFeedback(c, m, snap, len(rows) > 0)
		}
		if !feedback && len(rows) == 0 {
			label := scheduleText("没有匹配的自动化任务", "No matching tasks")
			if len(snap.Rows) == 0 {
				label = scheduleText("还没有自动化任务，在会话中创建的任务会显示在这里", "No tasks yet. Tasks created in your sessions appear here.")
			}
			if p.statusFilter == "inactive" && strings.TrimSpace(p.query) == "" {
				label = scheduleText("没有已结束的自动化任务", "No inactive automation tasks")
			}
			var action func()
			if m.OnNewTask != nil {
				action = func() {
					if coss.Button(c, scheduleText("新建自动化任务", "New automation task"), coss.ButtonOptions{
						Variant:  coss.ButtonOutline,
						Disabled: !snap.ClientReady,
					}).Clicked() {
						m.OnNewTask()
					}
				}
			}
			coss.Empty(c, coss.EmptyOptions{
				Icon:        coss.IconClock,
				Description: label,
			}, action)
		}

		if len(rows) > 0 {
			baseVer := scheduleMemoBase(epoch, snap).
				String(p.query).
				String(p.statusFilter).
				Uint(uint64(len(rows)))
			ui.Column(c.Key("schedule-catalog-rows")).FillWidth().MinWidth(0).
				Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).Clip().
				Children(func() {
					for i, entry := range rows {
						row := entry
						if i > 0 {
							coss.Separator(c, coss.Horizontal)
						}
						if p.rowFocus == nil {
							p.rowFocus = map[string]*ui.Handle{}
						}
						key := taskKey(row)
						if p.rowFocus[key] == nil {
							p.rowFocus[key] = &ui.Handle{}
						}
						selected := p.selected == key
						b := ui.ButtonBase(c.Key("schedule-row-"+key)).
							Bind(p.rowFocus[key]).
							Label("Open task: "+row.Title).
							FillWidth().MinWidth(0).Justify(ui.Start).Cursor(ui.CursorPointer).
							Padding(design.Sp(3), design.Sp(4)).Gap(design.Sp(3)).AlignItems(ui.Start).FocusRing(false)
						hovered := b.Hovered()
						focused := b.FocusWithin()
						if selected || hovered || focused {
							b.Background(t.Accent)
						}
						rowVer := baseVer.
							String(key).
							String(row.Title).
							String(row.Status).
							String(row.ScheduledAt).
							String(string(row.Raw)).
							Bool(selected).
							Bool(hovered).
							Bool(focused)
						if row.Status == "active" {
							rowVer = rowVer.Uint(uint64(time.Now().Unix()))
						}
						b.Memo(rowVer.Key()).Children(func() {
							ui.Icon(c, coss.IconClock.SVG()).Size(design.IconSM, design.IconSM).Margin(design.Sp(0.5), 0, 0, 0).TextColor(t.MutedForeground).Shrink(0)
							ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
								ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
									titleEl := pagekit.Strong(c, row.Title).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
									if row.Status == "inactive" && !selected && !hovered {
										titleEl.TextColor(t.MutedForeground)
									}
									if row.Status == "inactive" {
										coss.Badge(c, scheduleText("已结束", "Inactive"), coss.BadgeOptions{Variant: coss.BadgeSecondary, Size: coss.BadgeSM})
									}
								})
								pagekit.Caption(c, scheduleRowSummary(row)).FillWidth().MinWidth(0)
							})
						})
						if b.Clicked() {
							p.selectCatalogTask(row)
							p.returnRow = key
						}
					}
				})
		}
	}).Label("Task catalog content")
}

// Initial loading and failures are explicit; refreshing retained rows never blanks them.
func (p *Panel) catalogFeedback(c *ui.Context, m Model, snap schedstore.ViewState, populated bool) bool {
	if !snap.ClientReady {
		coss.Empty(c, coss.EmptyOptions{
			Icon:        coss.IconWifiOff,
			Description: scheduleText("连接 Host 后可管理任务。", "Connect to the Host to manage tasks."),
		}, nil)
		return true
	}
	if snap.Loading && !populated {
		pagekit.SkeletonRows(c, 3, scheduleText("正在读取任务…", "Loading tasks"))
		return true
	}
	if snap.Err != nil {
		if p.catalogError != snap.Err || p.catalogErrorRequest != snap.CatalogRequest {
			p.catalogError, p.catalogErrorRequest, p.catalogDismissed = snap.Err, snap.CatalogRequest, false
		}
		retry := func() { m.Store.Refresh(context.Background(), m.SessionID) }
		pagekit.FaultSentence(c, scheduleText("无法加载任务", "Could not load tasks."), snap.Err.SentenceText(), scheduleText("重试", "Retry"), retry)
		return true
	}
	p.catalogError, p.catalogDismissed = nil, false
	return false
}

func (p *Panel) detailView(c *ui.Context, m Model, snap schedstore.ViewState, d *taskDraft, epoch uint64, width float32) {
	t := design.For(c)
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
		detail.WidthPercent(47).BorderWidth(0, 0, 0, design.BorderWidth).BorderColor(t.Border)
	}
	detail.Children(func() {
		p.detailStrip(c, m, snap, d, width)
		if snap.Err != nil {
			ui.Column(c).FillWidth().Padding(design.Sp(2), detailGutterSp(width), 0, detailGutterSp(width)).Children(func() {
				p.catalogFeedback(c, m, snap, true)
			})
		}
		p.detailFeedback(c, m, d, !editable, detailGutterSp(width))
		if p.detailTab == "records" {
			p.mu.Lock()
			load := p.historyKey != taskKey(d.row) || p.historyLatest != latestDelivery(d.row) || (!p.historyLoading && !p.historyState.Loaded && p.historyState.Error == "")
			p.mu.Unlock()
			if load {
				p.loadDeliveryHistory(m, d.row, "", true)
			}
			p.deliveryHistoryView(c, m, epoch)
		} else {
			coss.ScrollArea(c, coss.ScrollAreaOptions{Key: "schedule-rule-scroll-" + taskKey(d.row)}, func() {
				ui.Column(c).FillWidth().MinWidth(0).Padding(design.Sp(6), detailGutterSp(width), design.Sp(20), detailGutterSp(width)).Gap(design.Sp(4)).Children(func() {
					if !available {
						pagekit.Note(c, coss.AlertWarning, scheduleText("任务已不可用。草稿已保留，无法保存。", "This task is unavailable. Your draft is retained and cannot be saved."), "", nil)
					}
					ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Children(func() {
						if d.row.Status == "inactive" {
							pagekit.Title(c, d.Title).FillWidth().MinWidth(0)
						} else {
							title := d.Title
							if coss.Input(c, &title, coss.InputOptions{
								Key:      "schedule-task-name",
								Label:    "Automation task name",
								Size:     coss.InputLG,
								Disabled: !editable,
							}).Changed() {
								d.Title = title
							}
						}
						pagekit.Caption(c, scheduleDetailNextRun(d.row)).FillWidth()
					})
					if d.row.Status == "inactive" {
						coss.Card(c, coss.CardOptions{}, func() {
							pagekit.Body(c, d.Prompt).FillWidth().MinWidth(0).Selectable()
						}).FillWidth()
					} else {
						prompt := d.Prompt
						if coss.Textarea(c, &prompt, coss.TextareaOptions{
							Key:      "schedule-task-instruction",
							Label:    "Automation task instruction",
							MaxLines: 8,
							Disabled: !editable,
						}).Changed() {
							d.Prompt = prompt
						}
					}
					pagekit.Section(c, scheduleText("运行时间", "Run time"), nil, func() {
						runCard := ui.Column(c).FillWidth().MinWidth(0).Padding(0, design.Sp(3)).
							Radius(design.RadiusXL2).Border(design.BorderWidth, t.Border).Background(t.Card)
						runCard.Children(func() {
							if d.row.Status == "inactive" {
								ui.Column(c).FillWidth().MinWidth(0).PaddingY(design.Sp(3)).Gap(design.Sp(1)).Children(func() {
									pagekit.Strong(c, scheduleFrequency(d.row)).FillWidth()
									if d.Zone != "" {
										pagekit.Caption(c, d.Zone).FillWidth()
									}
									pagekit.Caption(c, scheduleText("已结束的任务为只读，无法修改时间", "Inactive tasks are read-only. Their timing cannot be edited.")).FillWidth()
								})
							} else {
								p.timingEditor(c, d, editable)
							}
						})
					})
				})
			}).Label("Task rules scroll").Grow(1).MinHeight(0).FillWidth().MinWidth(0)
		}
		if d.dirty() && d.row.Status != "inactive" {
			saveBar := ui.Row(c.Key("schedule-save-bar")).Label("Unsaved task changes").
				Absolute().Left(design.Sp(3)).Right(design.Sp(3)).Bottom(design.Sp(3)).
				Padding(design.Sp(2), design.Sp(3)).Gap(design.Sp(2)).AlignItems(ui.Center).
				Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Popover)
			for _, l := range design.ShadowLg {
				saveBar.Shadow(l.X, l.Y, l.Blur, l.Spread, l.Color.Alpha(0.05))
			}
			saveBar.Children(func() {
				ui.Spacer(c)
				cancel := coss.Button(c, scheduleText("取消", "Cancel"), coss.ButtonOptions{
					Variant:  coss.ButtonGhost,
					Size:     coss.ButtonSM,
					Disabled: snap.Busy || pending,
				})
				cancel.Label("Cancel task changes")
				if cancel.Clicked() {
					*d = *newTaskDraft(d.row)
					p.message("")
				}
				if coss.Button(c, scheduleText("保存修改", "Save changes"), coss.ButtonOptions{
					Size:     coss.ButtonSM,
					Disabled: !editable,
					Loading:  pending,
				}).Clicked() {
					p.saveDraft(m, d)
				}
			})
		}
		if p.detailTab != "records" {
			if notice := scheduleSessionNotice(m, d.row.SessionID); notice != "" {
				pagekit.Caption(c, notice).Role(ui.RoleStatus).FillWidth().MinWidth(0).Padding(design.Sp(2), detailGutterSp(width), design.Sp(3), detailGutterSp(width))
			}
		}
	})
}

func (p *Panel) detailStrip(c *ui.Context, m Model, snap schedstore.ViewState, d *taskDraft, width float32) {
	t := design.For(c)
	height := float32(44)
	if m.DetailID != "" {
		height = 37
	}
	ui.Row(c.Key("schedule-detail-strip")).Label("Task detail views").FillWidth().MinWidth(0).Height(height).Shrink(0).
		Padding(0, detailGutterSp(width)).Gap(design.Sp(3)).BorderWidth(0, 0, design.BorderWidth, 0).BorderColor(t.Border).AlignItems(ui.Center).Children(func() {
		tabIdx := 0
		if p.detailTab == "records" {
			tabIdx = 1
		}
		ui.Row(c).Grow(1).MinWidth(0).Children(func() {
			if coss.Tabs(c, &tabIdx, []string{scheduleText("规则", "Rules"), scheduleText("任务运行记录", "Records")}, coss.TabsOptions{
				Variant: coss.TabsUnderline,
				Size:    coss.SegmentedSM,
				Label:   scheduleText("任务视图标签", "Task view tabs"),
				Key:     "schedule-detail-tabs",
			}).Changed() {
				p.selectDetailTab([]string{"rule", "records"}[tabIdx])
			}
		})
		ui.Row(c).Shrink(0).Gap(design.Sp(1)).AlignItems(ui.Center).Children(func() {
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
					if coss.Button(c, "", coss.ButtonOptions{
						Key:      "schedule-linked-session",
						Variant:  coss.ButtonGhost,
						Size:     coss.ButtonIconSM,
						Icon:     coss.IconMessage,
						Label:    scheduleText("打开原会话", "Open original session"),
						Disabled: !allowed || d.row.Origin != snap.Origin,
					}).Tooltip(label).Clicked() {
						m.OnOpenSession(d.row)
					}
				}
				more := coss.Button(c, "", coss.ButtonOptions{
					Key:     "schedule-task-actions",
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconSM,
					Icon:    coss.IconEllipsis,
					Label:   "Automation task actions",
				}).Bind(&p.deleteTrigger)
				menuOpen := ui.Local(more, "menu-open", func() bool { return false })
				if id := coss.Menu(c, more, menuOpen, []coss.MenuItem{
					{
						ID:          "delete",
						Label:       scheduleText("删除任务", "Delete task"),
						Icon:        coss.IconTrash,
						Destructive: true,
						Disabled:    snap.Busy || snap.Loading || !snap.ClientReady || !scheduleEntryAvailable(snap, d.row),
					},
				}, coss.MenuOptions{
					Align: coss.AlignEnd,
					Label: "Automation task actions",
				}); id == "delete" {
					row := d.row
					p.deleting = &row
					p.focusDelete = true
				}
			}
			closeLabel := "Close details"
			if m.DetailID != "" {
				closeLabel = "Close task"
			}
			if coss.Button(c, "", coss.ButtonOptions{
				Key:     "schedule-close-detail",
				Variant: coss.ButtonGhost,
				Size:    coss.ButtonIconSM,
				Icon:    coss.IconX,
				Label:   closeLabel,
			}).Tooltip(closeLabel).Clicked() {
				p.dismissDetails(c, m, snap)
			}
		})
	})
}

func (p *Panel) deleteDialog(c *ui.Context, m Model, snap schedstore.ViewState) {
	if p.deleting == nil {
		return
	}
	row := *p.deleting
	open := true
	panel := coss.AlertDialog(c, &open, coss.DialogOptions{
		Key:         "schedule-delete-dialog",
		Title:       scheduleText("删除此任务？", "Delete this task?"),
		Description: scheduleText("任务将停止触发，并连同其已保存的任务运行记录一并删除。原会话及其消息仍然保留；已排队的消息不会被撤回。", "This stops the task from triggering and deletes it together with its saved delivery records. The original session and its messages remain; queued messages are not withdrawn."),
		MaxWidth:    380,
		Footer: func() {
			cancel := coss.Button(c, scheduleText("取消", "Cancel"), coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Disabled: snap.Busy,
			}).Bind(&p.deleteCancel)
			cancel.Label("Cancel task deletion")
			if p.focusDelete {
				cancel.Focus()
				p.focusDelete = false
			}
			if cancel.Clicked() {
				open = false
			}
			if coss.Button(c, scheduleText("确认删除", "Confirm deletion"), coss.ButtonOptions{
				Variant:  coss.ButtonDestructive,
				Disabled: snap.Busy || !snap.ClientReady,
			}).Clicked() {
				p.deleting = nil
				p.deleteTrigger.RequestFocus(c)
				p.deleteTask(m, row)
			}
		},
	}, func() {
		if strings.TrimSpace(row.Prompt) != "" {
			pagekit.Body(c, row.Prompt).FillWidth().MinWidth(0)
		}
	})
	if panel != (ui.Element{}) {
		panel.Label("Delete this task?")
		if panel.Bounds().H == 0 {
			c.Services().Invalidate()
		}
	}
	if !open && p.deleting != nil {
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
	return scheduleText("下次计划时间：", "Next run: ") + scheduleAbsolute(at.In(time.Local)) + " (" + scheduleRelative(at, now) + ")"
}

// scheduleDetailNextRun is the detail header's `detail.nextRun` line; the list
// rows use `list.nextPrefix` instead (ui-schedule task-manager-locales).
func scheduleDetailNextRun(row schedstore.Entry) string {
	if row.Status == "inactive" {
		return scheduleText("已结束", "Inactive")
	}
	at, err := time.Parse(time.RFC3339Nano, row.ScheduledAt)
	if err != nil {
		return scheduleText("下次运行时间不可用", "Next run unavailable")
	}
	return scheduleText("下次运行 ", "Next run ") + scheduleAbsolute(at.In(time.Local)) + " (" + scheduleRelative(at, time.Now()) + ")"
}

func scheduleFrequency(row schedstore.Entry) string {
	if text := formatManagerFrequency(row.Raw, time.Now(), i18n.Locale(), systemScheduleZone()); text != "" {
		return text
	}
	return scheduleText("运行规则不可用", "Run rule unavailable")
}
