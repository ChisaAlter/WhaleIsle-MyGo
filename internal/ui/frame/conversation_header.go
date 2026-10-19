package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

// ConversationSessionHeader: titleCluster (crumbs + actions), utilities, corner.
// Occupants share the original 560 DIP title-row compact threshold.
func conversationHeaderTitleRow(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors) {
	compact := m.Sizes.Center-48 <= 560
	gap := float32(8)
	utilitiesMargin := float32(12)
	if compact {
		gap = 4
		utilitiesMargin = 8
	}
	rightInset := float32(28)
	if m.OnOpenSettings != nil {
		rightInset = 12
	}
	muted := chrome.Color(colors.Muted)
	ui.Row(c).Key("conversation-title-band").Label("Conversation title band").FillWidth().MinHeight(40).Padding(8, rightInset, 0, 20).AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Row(c).Key("conversation-title-cluster").Grow(1).MinWidth(0).AlignItems(ui.Center).Gap(gap).Children(func() {
			ui.Box(c).Key("conversation-title-crumbs").Label("Conversation session title").Grow(1).Shrink(1).MaxWidth(280).MinWidth(0).Clip().Children(func() {
				if snap.SelectedID != "" {
					sessionAncestryHeader(c, m, snap)
				} else {
					ui.Text(c, i18n.T(i18n.KeyAppName)).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(500).TextColor(chrome.Color(colors.FG))
				}
			})
			ui.Row(c).Key("conversation-header-actions").Shrink(0).AlignItems(ui.Center).Gap(gap).Children(func() {
				preset := strings.TrimSpace(snap.SelectedAgentPreset)
				if preset == "" && m.AgentPresetPick != nil {
					preset = strings.TrimSpace(*m.AgentPresetPick)
				}
				if preset != "" {
					ui.Row(c).Key("conversation-active-preset").Label("Conversation agent preset").Tooltip(preset).Height(22).MaxWidth(180).MinWidth(14).Shrink(0).Gap(4).AlignItems(ui.Center).Padding(0, 2, 0, 0).Children(func() {
						ui.Icon(c, conversationHeaderIcons["AgentPresetOutline"]).Width(14).Height(14).Shrink(0).TextColor(muted).Opacity(0.7)
						if !compact {
							ui.Text(c, preset).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(muted).MinWidth(0).MaxWidth(160).NoWrap().Ellipsis("…")
						}
					})
				}
				jobsHeader(c, m)
				conversationHeaderButton(c, "conversation-more", i18n.T(i18n.KeyLauncherOpen), "EllipsisOutline", true, muted, m.OnOpenLauncher, 0)
			})
		})
		ui.Row(c).Key("conversation-header-utilities").Shrink(0).AlignItems(ui.Center).Gap(gap).Padding(0, 0, 0, utilitiesMargin).Children(func() {
			scheduleHeader(c, m, snap, colors)
			openAgents := m.OnOpenAgents
			if openAgents == nil {
				openAgents = m.OnOpenAssistant
			}
			conversationHeaderButton(c, "conversation-agents", i18n.T(i18n.KeyFrameCaptionAgents), "AgentPresetOutline", compact, muted, openAgents, 0)
			if !m.TitlebarPreferences.HideSurfacesToggle {
				conversationHeaderButton(c, "conversation-docs", i18n.T(i18n.KeyFrameCaptionProjectDocs), "PanelLeftOutline", true, muted, m.OnToggleTools, 180)
			}
			if !m.TitlebarPreferences.HideTerminalToggle {
				conversationHeaderButton(c, "conversation-host-terminal", termui.HostTitle(), "CodeOutline", compact, muted, m.OnOpenHostTerminal, 0)
				conversationHeaderButton(c, "conversation-terminal", i18n.T(i18n.KeyTerminalTitle), "PanelLeftOutline", true, muted, m.OnToggleTerminal, 90)
			}
		})
		ui.Row(c).Key("conversation-header-corner").Shrink(0).AlignItems(ui.Center).Children(func() {
			conversationHeaderButton(c, "conversation-settings", i18n.T(i18n.KeySettingsOpen), "SettingsOutline", true, muted, m.OnOpenSettings, 0)
		})
	})
}

type scheduleHeaderState struct {
	open   bool
	mu     sync.Mutex
	notice string
}

func scheduleHeader(c *ui.Context, m Model, sessionSnap session.ViewState, colors theme.Colors) {
	if m.ScheduleStore == nil || !sessionSnap.ClientReady || sessionSnap.SelectedID == "" {
		return
	}
	m.ScheduleStore.EnsureContext(context.Background(), sessionSnap.SelectedID, nil)
	scheduleHeaderView(c, m, m.ScheduleStore.Snapshot(), sessionSnap.SelectedID, colors)
}

func scheduleHeaderView(c *ui.Context, m Model, snap schedstore.ViewState, sessionID string, colors theme.Colors) {
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
	muted, fg := chrome.Color(colors.Muted), chrome.Color(colors.FG)
	root := ui.Row(c.Key("schedule-header:" + snap.Origin + ":" + sessionID)).Shrink(0)
	state := ui.Local(root, "schedule-menu", func() *scheduleHeaderState { return &scheduleHeaderState{} })
	local := *state
	local.mu.Lock()
	notice := local.notice
	local.notice = ""
	local.mu.Unlock()
	if notice != "" {
		c.Toast(notice)
	}
	if len(rows) == 0 && snap.Err == "" {
		root.Size(0, 0)
		return
	}
	root.Padding(0, 0, 0, 4)
	root.Children(func() {
		label := archiveText("定时任务", "Scheduled tasks")
		if len(rows) == 1 {
			label = archiveText("1 个定时任务", "1 scheduled task")
		} else if len(rows) > 1 {
			label = fmt.Sprintf(archiveText("%d 个定时任务", "%d scheduled tasks"), len(rows))
		}
		anchor := ui.Button(c.Key("conversation-schedule"), "").Label(label).Tooltip(label).Size(28, 28).Padding(6).Radius(14).BorderWidth(0).Background(ui.Transparent)
		anchor.Children(func() {
			if snap.Loading {
				ui.Spinner(c).Size(16, 16).TextColor(muted)
			} else {
				ui.Icon(c, conversationHeaderIcons["ClockOutline"]).Size(16, 16).TextColor(muted)
			}
		})
		if anchor.Clicked() {
			if len(rows) == 1 && !snap.Loading && snap.Err == "" && !local.open {
				if m.OnOpenScheduleTask != nil {
					m.OnOpenScheduleTask(rows[0])
				}
			} else {
				local.open = !local.open
			}
		}
		w, h := c.Size()
		ui.PopoverBase(c.Key("schedule-header-popover"), anchor, &local.open, func(panel ui.Element) {
			headerPopupInset(c, panel, anchor, min(float32(336), max(float32(0), w-32)), w, 16)
			panel.Label("Scheduled tasks popover").Width(min(float32(336), max(float32(0), w-32))).MaxHeight(max(float32(0), min(float32(420), h-140))).Margin(5, 0, 0, 0).Padding(3).Radius(16).Background(chrome.Color(colors.BG)).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, .18)).Children(func() {
				ui.Scroll(c.Key("schedule-header-list")).Label("Scheduled tasks list").FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(float32(0), min(float32(414), h-146))).Children(func() {
					ui.Column(c).FillWidth().MinWidth(0).Gap(1).Children(func() {
						if snap.Loading {
							ui.Row(c).Label("Loading scheduled tasks").Padding(10, 12).Gap(6).Children(func() {
								ui.Spinner(c).Size(14, 14)
								ui.Text(c, archiveText("正在读取任务…", "Loading tasks…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
							})
						}
						if snap.Err != "" {
							ui.Column(c).FillWidth().MinWidth(0).Padding(10, 12).Gap(6).Children(func() {
								ui.Text(c, snap.Err).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
								if ui.Button(c, archiveText("重试", "Retry schedules")).Disabled(snap.Loading).Clicked() && m.ScheduleStore != nil {
									m.ScheduleStore.Refresh(context.Background(), sessionID)
								}
							})
						}
						for _, entry := range rows {
							row := entry
							ui.Row(c.Key("header-task-"+row.ID)).FillWidth().MinWidth(0).Gap(2).Padding(10, 12).Radius(10).AlignItems(ui.Start).Children(func() {
								button := ui.Button(c, "").Label("Open scheduled task: " + row.Title).Grow(1).MinWidth(0).Padding(0).BorderWidth(0).Background(ui.Transparent).Disabled(m.OnOpenScheduleTask == nil)
								button.Children(func() {
									ui.Column(c).FillWidth().MinWidth(0).Gap(3).Children(func() {
										ui.Text(c, row.Title).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(fg).FillWidth().MinWidth(0)
										ui.Text(c, scheduleHeaderFrequency(row)).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
										ui.Text(c, scheduleHeaderNext(row)).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
									})
								})
								if button.Clicked() {
									local.open = false
									m.OnOpenScheduleTask(row)
								}
								remove := ui.Button(c, "").Label("Delete scheduled task: "+row.Title).Tooltip(archiveText("删除任务", "Delete task")).Size(24, 24).Padding(5).BorderWidth(0).Background(ui.Transparent).Disabled(snap.Busy || snap.Loading || snap.Err != "" || m.ScheduleStore == nil)
								remove.Children(func() { ui.Icon(c, jobIcons["trash"]).Size(14, 14).TextColor(muted) })
								if remove.Clicked() {
									services := c.Services()
									go func() {
										err := m.ScheduleStore.DeleteEntry(context.Background(), row)
										local.mu.Lock()
										if err != nil {
											local.notice = err.Error()
										} else {
											local.notice = archiveText("任务已删除", "Task deleted")
										}
										local.mu.Unlock()
										services.Invalidate()
										if m.OnScheduleInvalidate != nil {
											m.OnScheduleInvalidate()
										}
									}()
								}
							})
						}
					})
				})
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
	button := ui.Button(c.Key(key), "").Label(label).Tooltip(label).Height(28).MinWidth(28).Shrink(0).Padding(0, 6).Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(ink)
	button.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.Center).Gap(4).Children(func() {
			ui.Icon(c, conversationHeaderIcons[icon]).Width(16).Height(16).TextColor(ink).Rotate(rotation)
			if !compact {
				ui.Text(c, label).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(20)).NoWrap().TextColor(ink)
			}
		})
	})
	if button.Clicked() {
		action()
	}
	return button
}
