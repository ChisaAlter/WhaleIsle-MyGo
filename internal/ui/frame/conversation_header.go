package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

// ConversationSessionHeader: titleCluster (crumbs + actions), utilities, corner.
// Occupants share the original 560 DIP title-row compact threshold.
func conversationHeaderTitleRow(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors) {
	t := tk(c)
	rightInset := max(float32(28), m.captionRightInset)
	compact := m.Sizes.Center-20-rightInset <= 560
	gap := design.Sp(2)
	utilitiesMargin := design.Sp(3)
	if compact {
		gap = design.Sp(1)
		utilitiesMargin = design.Sp(2)
	}
	muted := t.MutedForeground
	secondary := muted
	managed := conversationIsManaged(snap)
	// The phone band floats the drawer's menu button over the band's leading
	// edge (AppFrame .phoneMenu at 10,10 36x36), so the title clears it.
	leftInset := m.Sizes.headerLeftInset()
	// The caption owns the title and actions. View tabs are a separate,
	// non-draggable row, independent of compact title actions.
	ui.Row(c).Key("conversation-title-band").Label("Conversation title band").FillWidth().Height(workspaceCaptionContent).Padding(0, rightInset, 0, leftInset).AlignItems(ui.Center).DragWindow().Children(func() {
		if isWhaleAssistantSession(snap) {
			if img := chrome.BrandAvatar(); img != nil {
				ui.Image(c, img).Key("conversation-whale-mark").Width(16).Height(16).Shrink(0)
			}
		}
		if m.ConversationContributions != nil {
			m.ConversationContributions(c, snap, "conversation.session.header.leading")
		}
		ui.Row(c).Key("conversation-title-cluster").Grow(1).Shrink(1).MinWidth(0).AlignItems(ui.Center).Gap(gap).Children(func() {
			// The crumb seat spans the band's whole content row (y = 12..44) and
			// centres its 20px line in it, as .titleRow's min-height 32 does.
			ui.Box(c).Key("conversation-title-crumbs").Label("Conversation session title").Shrink(1).MaxWidth(280).MinWidth(0).Height(workspaceCaptionContent).Justify(ui.Center).Clip().Children(func() {
				if snap.SelectedID != "" {
					sessionAncestryHeader(c, m, snap)
				} else {
					ui.Text(c, i18n.T(i18n.KeyAppName)).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(design.WeightMedium).TextColor(t.Foreground)
				}
			})
			ui.Row(c).Key("conversation-header-actions").Shrink(0).AlignItems(ui.Center).Gap(gap).Children(func() {
				if managed {
					return
				}
				conversationTeamAction(c, m, snap, compact, secondary)
				preset := strings.TrimSpace(snap.SelectedAgentPreset)
				if preset == "" && m.AgentPresetPick != nil {
					preset = strings.TrimSpace(*m.AgentPresetPick)
				}
				if preset != "" {
					preset = composer.PresetDisplayName(preset)
					presetRow := ui.Row(c).Key("conversation-active-preset").Label("Conversation agent preset").Tooltip(preset).Height(28).MaxWidth(180).MinWidth(14).Shrink(0).Gap(design.Sp(1.25)).AlignItems(ui.Center).Padding(0, design.Sp(2))
					if compact {
						presetRow = presetRow.Height(22).Gap(design.Sp(1)).Padding(0, design.Sp(0.5), 0, 0)
					}
					presetRow.Children(func() {
						ui.Icon(c, conversationHeaderIcons["AgentPresetOutline"]).Width(14).Height(14).Shrink(0).TextColor(secondary).Opacity(0.85)
						if !compact {
							ui.Text(c, preset).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(secondary).MinWidth(0).MaxWidth(160).NoWrap().Ellipsis("…")
						}
					})
				}
				conversationMoreMenu(c, m, snap, secondary)
				if m.OnOpenWhaleSettings != nil && isWhaleAssistantSession(snap) {
					conversationHeaderButton(c, "conversation-whale-settings", archiveText("助理设置", "Assistant settings"), "SettingsOutline", true, muted, m.OnOpenWhaleSettings, 0)
				}
				ui.Box(c).Size(0, 0).PassThrough().Label(i18n.T(i18n.KeyParityConvSessionMoreViews))
				ui.Box(c).Size(0, 0).PassThrough().Label(i18n.T(i18n.KeyParityConvSessionMoreViewsAlt))
				projectObserverHeader(c, m)
				subagentsHeader(c, m, snap, compact, secondary)
				jobsHeader(c, m)
				if m.ConversationContributions != nil {
					m.ConversationContributions(c, snap, "conversation.session.header.actions")
				}
			})
		})
		ui.Row(c).Key("conversation-header-utilities").Shrink(0).AlignItems(ui.Center).Gap(gap).Padding(0, 0, 0, utilitiesMargin).Children(func() {
			if managed {
				return
			}
			conversationOpenInApp(c, m, snap, colors)
			scheduleHeader(c, m, snap, colors)
			if m.ConversationContributions != nil {
				m.ConversationContributions(c, snap, "conversation.session.header.utilities")
			}
			if m.WorkspaceTrailing == nil && !m.TitlebarPreferences.HideSurfacesToggle {
				conversationHeaderButton(c, "conversation-docs", i18n.T(i18n.KeyFrameCaptionProjectDocs), "PanelLeftOutline", true, muted, m.OnToggleTools, 180)
			}
			if m.WorkspaceTrailing == nil && !m.TitlebarPreferences.HideTerminalToggle {
				conversationHeaderButton(c, "conversation-host-terminal", termui.HostTitle(), "CodeOutline", compact, muted, m.OnOpenHostTerminal, 0)
				conversationHeaderButton(c, "conversation-terminal", i18n.T(i18n.KeyTerminalTitle), "PanelLeftOutline", true, muted, m.OnToggleTerminal, 90)
			}
		})
		ui.Row(c).Key("conversation-header-corner").Shrink(0).AlignItems(ui.Center).Children(func() {
			if false && m.OnOpenWhaleSettings != nil {
				for _, item := range snap.Items {
					if item.SessionID == snap.SelectedID && item.Presentation != nil && item.Presentation.Owner == "dsh-whale:assistant" {
						conversationHeaderButton(c, "conversation-whale-settings", archiveText("助理设置", "Assistant settings"), "SettingsOutline", true, muted, m.OnOpenWhaleSettings, 0)
						break
					}
				}
			}
			if m.ConversationContributions != nil {
				m.ConversationContributions(c, snap, "conversation.session.header.corner")
			}
			if m.WorkspaceTrailing == nil {
				conversationHeaderButton(c, "conversation-settings", i18n.T(i18n.KeySettingsOpen), "SettingsOutline", true, muted, m.OnOpenSettings, 0)
			}
		})
	})
}

func isWhaleAssistantSession(snap session.ViewState) bool {
	if snap.SelectedID == "" {
		return false
	}
	if strings.HasPrefix(snap.SelectedID, "session-whale-") {
		return true
	}
	for _, item := range snap.Items {
		if item.SessionID == snap.SelectedID {
			return item.Presentation != nil && item.Presentation.Owner == "dsh-whale:assistant"
		}
	}
	return false
}

type scheduleHeaderState struct {
	open bool
}

func scheduleHeader(c *ui.Context, m Model, sessionSnap session.ViewState, colors theme.Colors) {
	if m.ScheduleStore == nil || !sessionSnap.ClientReady || sessionSnap.SelectedID == "" {
		return
	}
	m.ScheduleStore.EnsureContext(context.Background(), sessionSnap.SelectedID, nil)
	scheduleHeaderView(c, m, m.ScheduleStore.Snapshot().Value, sessionSnap.SelectedID, colors)
}

func scheduleHeaderView(c *ui.Context, m Model, snap schedstore.ViewState, sessionID string, colors theme.Colors) {
	_ = colors
	if !snap.ClientReady {
		return
	}
	rows := make([]schedstore.Entry, 0)
	for _, row := range snap.Rows {
		if row.SessionID == sessionID && row.Status == "active" && row.Origin == snap.Origin {
			rows = append(rows, row)
		}
	}
	rows = schedstore.FilterEntries(rows, "", "active")
	t := tk(c)
	muted, fg := t.MutedForeground, t.Foreground
	root := ui.Row(c.Key("schedule-header:" + snap.Origin + ":" + sessionID)).Shrink(0)
	state := ui.Local(root, "schedule-menu", func() *scheduleHeaderState { return &scheduleHeaderState{} })
	local := *state
	if len(rows) == 0 && snap.Err == nil {
		root.Size(0, 0)
		return
	}
	root.Padding(0, 0, 0, design.Sp(1))
	root.Children(func() {
		label := archiveText("定时任务", "Scheduled tasks")
		if len(rows) == 1 {
			label = archiveText("1 个定时任务", "1 scheduled task")
		} else if len(rows) > 1 {
			label = fmt.Sprintf(archiveText("%d 个定时任务", "%d scheduled tasks"), len(rows))
		}
		tip := label
		if snap.Err != nil {
			tip = archiveText("定时任务读取失败", "Scheduled tasks could not be read")
		}
		anchor := coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonIconSM,
			Label:   label,
			Key:     "conversation-schedule",
			Content: func() {
				if snap.Loading {
					coss.Spinner(c, coss.SpinnerOptions{Size: 16}).TextColor(muted)
				} else {
					ui.Icon(c, conversationHeaderIcons["ClockOutline"]).Size(16, 16).TextColor(muted)
				}
				if snap.Err != nil && !snap.Loading {
					ui.Box(c).Label("Scheduled tasks failed").Size(6, 6).Radius(design.RadiusFull).Absolute().Top(3).Right(3).Background(t.Destructive)
				}
			},
		}).Tooltip(tip).Padding(design.Sp(1.5)).Radius(design.RadiusXL)
		if anchor.Clicked() {
			if len(rows) == 1 && !snap.Loading && snap.Err == nil && !local.open {
				if m.OnOpenScheduleTask != nil {
					m.OnOpenScheduleTask(rows[0])
				}
			} else {
				local.open = !local.open
			}
		}
		w, h := viewportDIP(c)
		ui.PopoverBase(c.Key("schedule-header-popover"), anchor, &local.open, func(panel ui.Element) {
			headerPopupInset(c, panel, anchor, min(float32(336), max(float32(0), w-32)), w, 16)
			panel = framePopoverPanel(panel.Label("Scheduled tasks popover").Width(min(float32(336), max(float32(0), w-32))).MaxHeight(max(float32(0), min(float32(420), h-140))).Margin(design.Sp(1.25), 0, 0, 0).Padding(design.Sp(0.75)), t, design.RadiusXL2)
			panel.Children(func() {
				coss.ScrollArea(c.Key("schedule-header-list"), coss.ScrollAreaOptions{}, func() {
					ui.Column(c).FillWidth().MinWidth(0).Gap(design.BorderWidth).Children(func() {
						if snap.Err != nil {
							ui.Box(c).FillWidth().MinWidth(0).Padding(design.Sp(1.5)).Children(func() {
								frameErrorAlert(c, frameAlertOptions{
									ID:      "schedule-catalog",
									Message: archiveText("没能读取这个对话的定时任务，请重试。", "This conversation's scheduled tasks could not be read. Try again."),
									Detail:  snap.Err.SentenceText(),
									Action:  archiveText("重试", "Retry schedules"),
									OnAction: func() {
										if m.ScheduleStore != nil {
											m.ScheduleStore.Refresh(context.Background(), sessionID)
										}
									},
								})
							})
						}
						if snap.Loading {
							ui.Row(c).Label("Loading scheduled tasks").Padding(design.Sp(2.5), design.Sp(3)).Gap(design.Sp(1.5)).Children(func() {
								coss.Spinner(c, coss.SpinnerOptions{Size: 14})
								frameText(c, archiveText("正在读取任务…", "Loading tasks…"), design.TextXS, design.WeightNormal, muted)
							})
						}
						for _, entry := range rows {
							row := entry
							ui.Row(c.Key("header-task-"+row.ID)).FillWidth().MinWidth(0).Gap(design.Sp(0.5)).Padding(design.Sp(2.5), design.Sp(3)).Radius(design.RadiusLG).AlignItems(ui.Start).Children(func() {
								button := ui.ButtonBase(c).Label("Open scheduled task: " + row.Title).Grow(1).MinWidth(0).Padding(0).BorderWidth(0).Background(ui.Transparent).Cursor(ui.CursorPointer).Disabled(m.OnOpenScheduleTask == nil)
								button.Children(func() {
									ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(0.75)).Children(func() {
										ui.Text(c, row.Title).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(fg).FillWidth().MinWidth(0)
										frameText(c, scheduleHeaderFrequency(row), design.TextXS, design.WeightNormal, muted).FillWidth().MinWidth(0)
										frameText(c, scheduleHeaderNext(row), design.TextXS, design.WeightNormal, muted).FillWidth().MinWidth(0)
									})
								})
								if button.Clicked() {
									local.open = false
									m.OnOpenScheduleTask(row)
								}
								remove := coss.Button(c, "", coss.ButtonOptions{
									Variant:  coss.ButtonGhost,
									Size:     coss.ButtonIconXS,
									Disabled: snap.Busy || snap.Loading || snap.Err != nil || m.ScheduleStore == nil,
									Label:    "Delete scheduled task: " + row.Title,
									Content:  func() { ui.Icon(c, jobIcons["trash"]).Size(14, 14).TextColor(muted) },
								}).Tooltip(archiveText("删除任务", "Delete task")).Padding(design.Sp(1.25))
								if remove.Clicked() {
									services := c.Services()
									go func() {
										_ = m.ScheduleStore.DeleteEntry(context.Background(), row)
										services.Invalidate()
										if m.OnScheduleInvalidate != nil {
											m.OnScheduleInvalidate()
										}
									}()
								}
							})
						}
					})
				}).Label("Scheduled tasks list").FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(float32(0), min(float32(414), h-146)))
			})
			if panel.OverlayShortcut(0, ui.KeyEscape) {
				local.open = false
				anchor.Focus()
			}
		})
		if local.open {
			c.After(time.Second)
		}
	})
}

func scheduleHeaderFrequency(row schedstore.Entry) string {
	var rule struct {
		Kind, Time, TimeZone, Expression string
		EverySeconds                     int
		Weekdays                         []int
	}
	_ = json.Unmarshal(row.Raw, &rule)
	switch rule.Kind {
	case "every":
		if rule.EverySeconds%3600 == 0 {
			return fmt.Sprintf(archiveText("每 %d 小时", "Every %d hours"), rule.EverySeconds/3600)
		}
		if rule.EverySeconds%60 == 0 {
			return fmt.Sprintf(archiveText("每 %d 分钟", "Every %d minutes"), rule.EverySeconds/60)
		}
		return fmt.Sprintf(archiveText("每 %d 秒", "Every %d seconds"), rule.EverySeconds)
	case "daily":
		return archiveText("每天 ", "Every day ") + rule.Time
	case "weekly":
		return fmt.Sprintf(archiveText("每周 %v · %s", "Weekly %v · %s"), rule.Weekdays, rule.Time)
	case "cron":
		return archiveText("自定义 ", "Custom ") + rule.Expression
	default:
		return archiveText("仅一次", "Once")
	}
}
func scheduleHeaderNext(row schedstore.Entry) string {
	stamp, err := time.Parse(time.RFC3339Nano, row.ScheduledAt)
	if err != nil {
		return archiveText("下次运行时间不可用", "Next run unavailable")
	}
	seconds := int64(time.Until(stamp).Seconds())
	relative := archiveText("即将运行", "due now")
	if seconds >= 3600 {
		relative = fmt.Sprintf(archiveText("%d 小时后", "in %dh"), seconds/3600)
	} else if seconds >= 60 {
		relative = fmt.Sprintf(archiveText("%d 分钟后", "in %dm"), seconds/60)
	} else if seconds > 0 {
		relative = fmt.Sprintf(archiveText("%d 秒后", "in %ds"), seconds)
	}
	return archiveText("下次运行：", "Next run: ") + stamp.In(time.Local).Format("2006-01-02 15:04:05") + " (" + relative + ")"
}

func conversationHeaderButton(c *ui.Context, key, label, icon string, compact bool, ink ui.Color, action func(), rotation float32) ui.Element {
	if action == nil {
		return ui.Element{}
	}
	padX := design.Sp(2)
	if compact {
		padX = design.Sp(1.5)
	}
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonSM,
		Label:   label,
		Key:     key,
		Content: func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.Center).Gap(design.Sp(1.25)).Children(func() {
				ui.Icon(c, conversationHeaderIcons[icon]).Width(16).Height(16).TextColor(ink).Rotate(rotation)
				if !compact {
					ui.Text(c, label).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).NoWrap().TextColor(ink)
				}
			})
		},
	}).Tooltip(label).MinWidth(28).Shrink(0).Padding(0, padX).Radius(design.RadiusSM).TextColor(ink)
	if button.Clicked() {
		action()
	}
	return button
}

func subagentsHeader(c *ui.Context, m Model, snap session.ViewState, compact bool, headerInk ui.Color) {
	if snap.SelectedID == "" || compact {
		return
	}
	itemRunning := make(map[string]bool, len(snap.Items))
	for _, it := range snap.Items {
		if it.Running {
			itemRunning[it.SessionID] = true
		}
	}
	subCount := len(snap.SubagentCatalog)
	runningCount := 0
	if subCount > 0 {
		for _, entry := range snap.SubagentCatalog {
			if itemRunning[entry.ID] {
				runningCount++
			}
		}
	} else {
		seen := map[string]bool{}
		for _, it := range snap.Items {
			if it.ParentID == snap.SelectedID && it.SessionID != "" && it.SessionID != snap.SelectedID {
				seen[it.SessionID] = true
				if it.Running {
					runningCount++
				}
			}
		}
		for _, s := range snap.CostSources {
			if s.Parent == snap.SelectedID && s.Origin == "subagent" && s.ID != "" && s.ID != snap.SelectedID && !seen[s.ID] {
				seen[s.ID] = true
				if itemRunning[s.ID] {
					runningCount++
				}
			}
		}
		subCount = len(seen)
	}
	if runningCount == 0 {
		if act, ok := snap.SidebarSubagentActivity[snap.SelectedID]; ok && act.Count > 0 {
			runningCount = act.Count
		}
	}
	if subCount == 0 {
		return
	}
	summary := fmt.Sprintf(archiveText("%d 个子智能体", "%d subagents"), subCount)
	if subCount == 1 {
		summary = archiveText("1 个子智能体", "1 subagent")
	}
	btn := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonSM,
		Label:   "Subagents",
		Key:     "conversation-subagents",
		Content: func() {
			ui.Row(c).AlignItems(ui.Center).Gap(design.Sp(1.25)).Children(func() {
				if runningCount > 0 {
					coss.Spinner(c, coss.SpinnerOptions{Size: 10}).TextColor(headerInk)
				}
				ui.Text(c, summary).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(headerInk).NoWrap()
				ui.Icon(c, jobIcons["chevron"]).Size(12, 12).TextColor(headerInk)
			})
		},
	}).Tooltip(summary).MinWidth(28).Shrink(0).Padding(0, design.Sp(2)).Radius(design.RadiusSM).TextColor(headerInk)
	if btn.Clicked() && m.OnOpenAgents != nil {
		m.OnOpenAgents()
	}
}
