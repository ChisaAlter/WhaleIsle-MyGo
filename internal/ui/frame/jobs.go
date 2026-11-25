package frame

import (
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/jobflow"
	"whaleisle/internal/ui/agents"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

func jobsHeader(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	if m.jobsHeaderMounted != nil {
		*m.jobsHeaderMounted = true
	}
	controller := m.Store.NativeJobs()
	controller.Mount()
	compact := false
	if m.Sizes != nil {
		compact = m.Sizes.Center-48 <= 560
	}
	jobsHeaderView(c, controller, compact, theme.Resolve(m.ThemeID).Active(m.SystemDark))
}

// The header remains mounted while the popup is closed: only the popup's
// expanded output observation follows visibility. PopoverBase lifts its panel
// into the window overlay, outside the clipped conversation header.
func jobsHeaderView(c *ui.Context, controller *jobflow.Controller, compact bool, colors theme.Colors) {
	_ = colors
	s := controller.Snapshot()
	if len(s.Rows) == 0 && s.Error == "" && !s.Open {
		return
	}
	live, settled := splitJobs(s.Rows)
	label := jobsCountLabel(len(live), len(s.Rows))
	if len(s.Rows) == 0 {
		label = archiveText("后台任务读取失败", "Background jobs need a fresh read")
	}
	t := tk(c)
	muted := t.MutedForeground
	headerInk := muted
	anchor := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonSM,
		Label:   label,
		Key:     "conversation-jobs",
		Content: func() {
			if compact {
				ui.Icon(c, conversationHeaderIcons["CodeOutline"]).Size(14, 14).TextColor(muted)
				if len(live) > 0 {
					coss.Spinner(c, coss.SpinnerOptions{Size: 10}).Absolute().Top(0).Right(0).TextColor(muted)
				}
				return
			}
			ui.Row(c).AlignItems(ui.Center).Gap(design.Sp(1.25)).Children(func() {
				if len(live) > 0 {
					coss.Spinner(c, coss.SpinnerOptions{Size: 10}).TextColor(muted)
				}
				if !compact {
					ui.Text(c, label).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(headerInk).SingleLine().MaxWidth(220).Ellipsis("…")
					icon := ui.Icon(c, jobIcons["chevron"]).Size(12, 12).TextColor(headerInk)
					if s.Open {
						icon.Rotate(180)
					}
				}
			})
		},
	}).Tooltip(label).MinWidth(28).Shrink(0).Padding(0, design.Sp(2)).Radius(design.RadiusSM).TextColor(headerInk)
	if compact {
		anchor.Size(28, 28).Padding(0)
	}
	if anchor.Clicked() {
		if s.Open {
			controller.Close(s.Generation)
		} else {
			controller.Open(s.Generation)
		}
		s = controller.Snapshot()
	}
	open := s.Open
	w, h := viewportDIP(c)
	ui.PopoverBase(c.Key("jobs-popover"), anchor, &open, func(panel ui.Element) {
		headerPopupInset(c, panel, anchor, min(float32(500), max(float32(0), w-32)), w, 12)
		panel = framePopoverPanel(panel.Label("Background jobs popover").Width(min(float32(500), max(float32(0), w-32))).MaxHeight(max(float32(0), min(float32(480), h-140))).Margin(design.Sp(1.25), 0, 0, 0).Padding(design.Sp(1)), t, design.RadiusXL-design.Sp(0.5))
		panel.Children(func() {
			coss.ScrollArea(c.Key("jobs-roster"), coss.ScrollAreaOptions{}, func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(design.BorderWidth).Children(func() {
					if s.Loading {
						ui.Row(c).Label("Loading jobs").Gap(design.Sp(1.5)).Padding(design.Sp(2)).Children(func() {
							coss.Spinner(c, coss.SpinnerOptions{Size: 14})
							frameText(c, archiveText("正在读取后台任务…", "Loading background jobs…"), design.TextXS, design.WeightNormal, muted)
						})
					}
					if s.Error != "" {
						ui.Box(c).FillWidth().MinWidth(0).Padding(design.Sp(2)).Children(func() {
							frameErrorAlert(c, frameAlertOptions{
								ID:       "jobs-roster",
								Message:  archiveText("没能读取后台任务，请重试。", "Background jobs could not be read. Try again."),
								Detail:   s.Error,
								Action:   archiveText("重新读取任务", "Retry jobs"),
								OnAction: func() { controller.Retry(s.Generation) },
							})
						})
					}
					if len(live) > 0 {
						jobsSectionLabel(c, archiveText("运行中", "Running"), muted)
						for _, row := range live {
							jobRowView(c, controller, s, row, colors)
						}
						c.After(time.Second)
					}
					if len(settled) > 0 {
						settledOpen := len(live) == 0
						if s.SettledOpen != nil {
							settledOpen = *s.SettledOpen
						}
						if len(live) > 0 {
							ui.Box(c).FillWidth().Height(design.BorderWidth/2).Margin(design.Sp(1), design.Sp(1), 0, design.Sp(1)).Background(t.Border)
						}
						ui.Row(c).FillWidth().Gap(design.Sp(1)).Padding(design.Sp(1)).AlignItems(ui.Center).Children(func() {
							text := fmt.Sprintf(archiveText("已结束 %d", "Finished %d"), len(settled))
							button := coss.Button(c, "", coss.ButtonOptions{
								Variant: coss.ButtonGhost,
								Size:    coss.ButtonXS,
								Label:   "Toggle finished jobs",
								Content: func() {
									ui.Row(c).Gap(design.Sp(0.75)).AlignItems(ui.Center).Children(func() {
										icon := ui.Icon(c, jobIcons["chevron"]).Size(12, 12).TextColor(muted)
										if !settledOpen {
											icon.Rotate(-90)
										}
										ui.Text(c, text).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(muted)
									})
								},
							}).MinWidth(0).Padding(design.Sp(0.5), design.Sp(1)).TextColor(muted)
							if button.Clicked() {
								controller.ToggleSettled(s.Generation)
							}
							ui.Box(c).Grow(1)
							if coss.Button(c, archiveText("清空", "Clear"), coss.ButtonOptions{
								Variant: coss.ButtonGhost,
								Size:    coss.ButtonXS,
							}).Label("Clear finished jobs").Padding(design.Sp(0.5), design.Sp(1)).Clicked() {
								controller.ClearSettled(s.Generation)
							}
						})
						if settledOpen {
							for _, row := range settled {
								jobRowView(c, controller, s, row, colors)
							}
						}
					}
					if len(s.Rows) == 0 && !s.Loading && s.Error == "" {
						ui.Text(c, archiveText("暂无后台任务", "No background jobs")).Padding(design.Sp(3)).TextColor(muted).FontSize(theme.InterfaceSize(12))
					}
				})
			}).Label("Background jobs list").FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(float32(0), min(float32(472), h-148)))
		})
		if panel.OverlayShortcut(0, ui.KeyEscape) {
			open = false
			anchor.Focus()
		}
	})
	if s.Open && !open {
		controller.Close(s.Generation)
	}
}

func jobsSectionLabel(c *ui.Context, label string, ink ui.Color) {
	ui.Text(c, label).Padding(design.Sp(1), design.Sp(2)).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(ink)
}
func splitJobs(rows []jobflow.Job) (live, settled []jobflow.Job) {
	for _, row := range rows {
		if row.Live() {
			live = append(live, row)
		} else {
			settled = append(settled, row)
		}
	}
	return
}
func jobsCountLabel(live, total int) string {
	if live > 0 {
		if live == 1 {
			return archiveText("1 个后台任务运行中", "1 background job running")
		}
		return fmt.Sprintf(archiveText("%d 个后台任务运行中", "%d background jobs running"), live)
	}
	if total == 1 {
		return archiveText("1 个后台任务", "1 background job")
	}
	return fmt.Sprintf(archiveText("%d 个后台任务", "%d background jobs"), total)
}

func jobRowView(c *ui.Context, controller *jobflow.Controller, s jobflow.Snapshot, job jobflow.Job, colors theme.Colors) {
	_ = colors
	t := tk(c)
	fg, muted := t.Foreground, t.MutedForeground
	label := job.Label
	if label == "" {
		label = job.ID
	}
	live := job.Live()
	durText := jobDuration(job, time.Now())
	durTip := agents.FormatJobsDurationTitle(live, durText)
	ui.Column(c.Key("job-" + job.ID)).FillWidth().MinWidth(0).Children(func() {
		line := ui.Row(c.Key("job-line-" + job.ID)).Label("Job row: " + job.ID).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).AlignItems(ui.Center)
		if live {
			line.Padding(design.Sp(1), design.Sp(2), design.Sp(1), design.Sp(0.5)).Radius(design.RadiusMD).Background(frameInsetBg(t))
		}
		line.Children(func() {
			button := ui.ButtonBase(c.Key("job-toggle-"+job.ID)).Label("Toggle job output: "+job.ID).Grow(1).Shrink(1).MinWidth(0).Padding(design.Sp(1), design.Sp(1.75)).Radius(design.RadiusSM).BorderWidth(0).Background(ui.Transparent).Cursor(ui.CursorPointer).Disabled(!job.Observable())
			button.Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Children(func() {
					jobStateIndicator(c, job.Status)
					if !live {
						ui.Text(c, job.Kind).FontSize(theme.InterfaceSize(10)).Padding(0, design.Sp(1.25)).Radius(design.RadiusXS * 2).Background(frameInsetBg(t)).TextColor(muted).Shrink(0)
						ui.Text(c, label).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(label)
						detail := job.Progress
						if detail == "" {
							detail = job.Detail
						}
						if detail == "" {
							detail = jobStatus(job.Status)
						}
						ui.Text(c, detail).FontSize(theme.InterfaceSize(10)).TextColor(muted).MaxWidthPercent(35).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(detail)
						if job.ExitCode != nil {
							ui.Text(c, agents.FormatJobsTerminalExitCode(*job.ExitCode)).FontSize(theme.InterfaceSize(10)).TextColor(muted).Shrink(0)
						}
						ui.Text(c, durText).Tooltip(durTip).FontSize(theme.InterfaceSize(11)).TextColor(muted).Shrink(0)
					} else {
						ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
							ink := fg
							if !live {
								ink = muted
							}
							ui.Text(c, label).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(17)).TextColor(ink).FillWidth().MinWidth(0).SingleLine().Ellipsis("…").Tooltip(label)
							ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Children(func() {
								ui.Text(c, job.Kind).FontSize(theme.InterfaceSize(10)).FixedLineHeight(theme.InterfaceSize(16)).Padding(0, design.Sp(1.25)).Radius(design.RadiusXS * 2).Background(t.Sidebar).TextColor(muted).Shrink(0)
								detail := job.Progress
								if detail == "" {
									detail = job.Detail
								}
								if detail == "" {
									detail = jobStatus(job.Status)
								}
								ui.Text(c, detail).FontSize(theme.InterfaceSize(10)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(muted).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(detail)
								ui.Text(c, durText).Tooltip(durTip).FontSize(theme.InterfaceSize(11)).TextColor(muted).Shrink(0)
							})
						})
					}
					if job.Observable() {
						ui.Box(c).Size(20, 20).Shrink(0).Border(design.BorderWidth/2, t.Border).Radius(design.RadiusXS * 2).Background(t.Background).Center().Children(func() {
							icon := ui.Icon(c, jobIcons["chevron"]).Size(12, 12).TextColor(muted)
							if s.SelectedID == job.ID {
								icon.Rotate(180)
							}
						})
					}
				})
			})
			if button.Clicked() {
				controller.Toggle(s.Generation, job.ID)
			}
			if job.Status == "running" {
				stop := coss.Button(c, "", coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonIconXS,
					Disabled: s.Loading || s.Error != "" || s.Kill.Phase == "pending",
					Label:    "Stop task " + job.Label,
					Key:      "job-stop-" + job.ID,
					Content: func() {
						if s.Kill.JobID == job.ID && s.Kill.Phase == "pending" {
							coss.Spinner(c, coss.SpinnerOptions{Size: 10}).TextColor(muted)
						} else {
							ui.Icon(c, jobIcons["stop"]).Size(10, 10).TextColor(muted)
						}
					},
				}).Tooltip(agents.FormatJobsKillStop(job.Label)).Size(20, 20).Padding(design.Sp(1)).Border(design.BorderWidth/2, t.Border).Radius(design.RadiusXS * 2).Background(t.Background)
				if stop.Clicked() {
					if s.Kill.JobID == job.ID && s.Kill.Phase == "armed" {
						controller.ConfirmKill(s.Generation, job.ID)
					} else {
						controller.ArmKill(s.Generation, job.ID)
					}
				}
			}
		})
		if s.Kill.JobID == job.ID && s.Kill.Phase == "armed" {
			ui.Row(c.Key("job-kill-confirm-"+job.ID)).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Padding(0, design.Sp(2), design.Sp(1), design.Sp(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, archiveText("再次点击确认停止", "Click again to confirm stop")).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(1).MinWidth(0)
				if coss.Button(c, archiveText("确认停止", "Confirm stop"), coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonXS,
				}).Label("Confirm stop job "+job.ID).Padding(0, design.Sp(2)).Clicked() {
					controller.ConfirmKill(s.Generation, job.ID)
				}
				if coss.Button(c, archiveText("取消", "Cancel"), coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonXS,
				}).Label("Cancel stop job "+job.ID).Padding(0, design.Sp(2)).Clicked() {
					controller.CancelKill(s.Generation)
				}
			})
		}
		if s.Kill.JobID == job.ID && s.Kill.Phase != "armed" && s.Kill.Error != "" {
			ui.Box(c).FillWidth().MinWidth(0).Padding(0, design.Sp(2), design.Sp(1), design.Sp(2)).Children(func() {
				frameErrorAlert(c, frameAlertOptions{
					ID:        "job-kill:" + job.ID,
					Message:   archiveText("没能停止这条任务，请重试。", "This task could not be stopped. Try again."),
					Detail:    s.Kill.Error,
					Action:    archiveText("重试停止", "Retry stop"),
					OnAction:  func() { controller.ArmKill(s.Generation, job.ID) },
					Dismiss:   archiveText("关闭", "Dismiss"),
					OnDismiss: func() { controller.CancelKill(s.Generation) },
				})
			})
		}
		if s.SelectedID == job.ID {
			jobOutputView(c, controller, s, job, colors)
		}
	})
}

func jobStateIndicator(c *ui.Context, status string) {
	t := tk(c)
	if status == "running" {
		coss.Spinner(c, coss.SpinnerOptions{Size: 10}).Label("Job state: running").TextColor(t.MutedForeground)
		return
	}
	ink := t.Success
	switch status {
	case "stopping", "killed":
		ink = t.Warning
	case "failed":
		ink = t.Destructive
	}
	ui.Box(c).Label("Job state: "+status).Size(10, 10).Shrink(0).Center().Children(func() {
		ui.Box(c).Size(6, 6).Radius(design.RadiusFull).Background(ink)
	})
}

func jobOutputView(c *ui.Context, controller *jobflow.Controller, s jobflow.Snapshot, job jobflow.Job, colors theme.Colors) {
	_ = colors
	t := tk(c)
	fg, muted := t.Foreground, t.MutedForeground
	o := s.Observed
	panel := ui.Column(c.Key("job-output-panel-"+job.ID)).FillWidth().MinWidth(0).Padding(0, design.Sp(2), design.Sp(2)).Gap(design.Sp(1.5))
	notice := ui.Local(panel, "output-error-notice", func() struct {
		value     string
		dismissed bool
	} {
		return struct {
			value     string
			dismissed bool
		}{}
	})
	if notice.value != o.Error {
		notice.value, notice.dismissed = o.Error, false
	}
	panel.Children(func() {
		if o.GapBefore {
			ui.Text(c, archiveText("……较早的输出已丢弃……", "… earlier output dropped …")).FontSize(theme.InterfaceSize(11)).TextColor(muted).FillWidth().MinWidth(0)
		}
		if o.Error != "" && !notice.dismissed {
			frameErrorAlert(c, frameAlertOptions{
				ID:        "job-output:" + job.ID,
				Message:   archiveText("这条任务的输出没能读到，可以再试一次。", "This task's output could not be read. Try again."),
				Detail:    o.Error,
				Action:    archiveText("重试输出", "Retry output"),
				OnAction:  func() { controller.RetryOutput(s.Generation) },
				Dismiss:   archiveText("关闭", "Dismiss"),
				OnDismiss: func() { notice.dismissed = true },
			})
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(func() {
			ui.Text(c, job.Label).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(1).MinWidth(0)
			if coss.Button(c, archiveText("复制", "Copy"), coss.ButtonOptions{
				Variant:  coss.ButtonGhost,
				Size:     coss.ButtonXS,
				Label:    "Copy job command",
				Disabled: job.Label == "",
			}).Clicked() {
				mygo.Clipboard.WriteText(job.Label)
			}
		})
		coss.ScrollArea(c.Key("job-output-"+job.ID), coss.ScrollAreaOptions{}, func() {
			body := o.Text
			if body == "" {
				body = archiveText("（无输出）", "(no output)")
			}
			ui.Text(c, body).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(12)).TextColor(fg).FillWidth().MinWidth(0).Selectable()
		}).Label("Job output scroll: " + job.ID).FillWidth().MinWidth(0).MaxHeight(288)
	})
}
func jobDuration(job jobflow.Job, now time.Time) string {
	end := job.FinishedAt
	if job.Live() {
		end = float64(now.UnixMilli())
	}
	seconds := int64(max(0, end-job.StartedAt) / 1000)
	if seconds >= 3600 {
		return fmt.Sprintf(archiveText("%d小时%d分", "%dh %dm"), seconds/3600, seconds%3600/60)
	}
	if seconds >= 60 {
		return fmt.Sprintf(archiveText("%d分%d秒", "%dm %ds"), seconds/60, seconds%60)
	}
	return fmt.Sprintf(archiveText("%d秒", "%ds"), seconds)
}
func jobStatus(status string) string {
	switch status {
	case "running":
		return archiveText("运行中", "running")
	case "stopping":
		return archiveText("正在停止", "stopping")
	case "completed":
		return archiveText("已完成", "completed")
	case "killed":
		return archiveText("已取消", "cancelled")
	case "failed":
		return archiveText("已失败", "failed")
	}
	return status
}

var jobIcons = map[string]*ui.SVG{
	"chevron": ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="m4 6 4 4 4-4" stroke="currentColor"/></svg>`)),
	"stop":    ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><rect x="3" y="3" width="10" height="10" rx="1" fill="currentColor"/></svg>`)),
	"trash":   ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M2 4h12M6 4V2h4v2M4 4l1 10h6l1-10M6 6v6M10 6v6" stroke="currentColor"/></svg>`)),
}

// PopoverBase supplies clipping-free placement and focus order. Its window
// margin is 4px; these two original header menus use 12px and 16px instead.
func headerPopupInset(c *ui.Context, panel, anchor ui.Element, width, viewport, margin float32) {
	box := anchor.Bounds()
	base := box.X
	over := func(x float32) float32 { return max(float32(0), 4-x) + max(float32(0), x+width-(viewport-4)) }
	if over(base) > 0 {
		flipped := box.X + box.W - width
		if over(flipped) < over(base) {
			base = flipped
		}
		if over(base) > 0 {
			base = max(float32(4), min(base, viewport-4-width))
		}
	}
	target := max(margin, min(box.X, viewport-margin-width))
	panel.Left(target - base)
	panel.DrawOver(func(_ *ui.Painter, _ ui.Rect) {
		if anchor.Bounds() != box {
			c.Services().Invalidate()
		}
	})
}
