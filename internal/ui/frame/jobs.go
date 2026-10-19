package frame

import (
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/jobflow"
	"whaleisle/internal/ui/chrome"
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
	s := controller.Snapshot()
	if len(s.Rows) == 0 && s.Error == "" && !s.Open {
		return
	}
	live, settled := splitJobs(s.Rows)
	label := jobsCountLabel(len(live), len(s.Rows))
	if len(s.Rows) == 0 {
		label = archiveText("后台任务读取失败", "Background jobs need a fresh read")
	}
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	anchor := ui.Button(c.Key("conversation-jobs"), "").Label(label).Tooltip(label).Height(28).MinWidth(28).Shrink(0).Padding(3, 2).Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(muted)
	if compact {
		anchor.Size(28, 28).Padding(0)
	}
	anchor.Children(func() {
		if compact {
			ui.Icon(c, conversationHeaderIcons["CodeOutline"]).Size(14, 14).TextColor(muted)
			if len(live) > 0 {
				ui.Spinner(c).Size(10, 10).Absolute().Top(0).Right(0).TextColor(muted)
			}
			return
		}
		ui.Row(c).AlignItems(ui.Center).Gap(3).Children(func() {
			if len(live) > 0 {
				ui.Spinner(c).Size(10, 10).TextColor(muted)
			}
			if !compact {
				ui.Text(c, label).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted).SingleLine().MaxWidth(220).Ellipsis("…")
				icon := ui.Icon(c, jobIcons["chevron"]).Size(12, 12).TextColor(muted)
				if s.Open {
					icon.Rotate(180)
				}
			}
		})
	})
	if anchor.Clicked() {
		if s.Open {
			controller.Close(s.Generation)
		} else {
			controller.Open(s.Generation)
		}
		s = controller.Snapshot()
	}
	open := s.Open
	w, h := c.Size()
	ui.PopoverBase(c.Key("jobs-popover"), anchor, &open, func(panel ui.Element) {
		headerPopupInset(c, panel, anchor, min(float32(500), max(float32(0), w-32)), w, 12)
		panel.Label("Background jobs popover").Width(min(float32(500), max(float32(0), w-32))).MaxHeight(max(float32(0), min(float32(480), h-140))).Margin(5, 0, 0, 0).Padding(4).Background(chrome.Color(colors.BG)).Radius(12).Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, .18)).Children(func() {
			ui.Scroll(c.Key("jobs-roster")).Label("Background jobs list").FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(float32(0), min(float32(472), h-148))).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(1).Children(func() {
					if s.Loading {
						ui.Row(c).Label("Loading jobs").Gap(6).Padding(8).Children(func() {
							ui.Spinner(c).Size(14, 14)
							ui.Text(c, archiveText("正在读取后台任务…", "Loading background jobs…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						})
					}
					if s.Error != "" {
						ui.Column(c).FillWidth().MinWidth(0).Gap(6).Padding(8).Children(func() {
							ui.Text(c, s.Error).FontSize(theme.InterfaceSize(12)).TextColor(fg).FillWidth().MinWidth(0)
							if ui.Button(c, archiveText("重新读取任务", "Retry jobs")).AlignSelf(ui.Start).Disabled(s.Loading).Clicked() {
								controller.Retry(s.Generation)
							}
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
						ui.Row(c).FillWidth().Gap(4).Padding(4).Children(func() {
							text := fmt.Sprintf(archiveText("已结束 %d", "Finished %d"), len(settled))
							button := ui.Button(c, text).Label("Toggle finished jobs").MinWidth(0).Padding(2, 4).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(11))
							if button.Clicked() {
								controller.ToggleSettled(s.Generation)
							}
							ui.Box(c).Grow(1)
							if ui.Button(c, archiveText("清除", "Clear")).Label("Clear finished jobs").Padding(2, 4).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(11)).Clicked() {
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
						ui.Text(c, archiveText("暂无后台任务", "No background jobs")).Padding(12).TextColor(muted).FontSize(theme.InterfaceSize(12))
					}
				})
			})
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
	ui.Text(c, label).Padding(4, 8).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(ink)
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
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	label := job.Label
	if label == "" {
		label = job.ID
	}
	live := job.Live()
	ui.Column(c.Key("job-" + job.ID)).FillWidth().MinWidth(0).Children(func() {
		line := ui.Row(c.Key("job-line-" + job.ID)).Label("Job row: " + job.ID).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Center)
		if live {
			line.Padding(4, 8, 4, 2).Radius(8).Background(chrome.Color(colors.Field))
		}
		line.Children(func() {
			button := ui.Button(c.Key("job-toggle-"+job.ID), "").Label("Toggle job output: "+job.ID).Grow(1).Shrink(1).MinWidth(0).Padding(4, 7).Radius(6).BorderWidth(0).Background(ui.Transparent).Disabled(!job.Observable())
			button.Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
					if live {
						ui.Spinner(c).Size(10, 10).TextColor(muted)
					} else {
						ui.Box(c).Size(6, 6).Radius(3).Background(muted)
					}
					if !live {
						ui.Text(c, job.Kind).FontSize(theme.InterfaceSize(10)).Padding(0, 5).Radius(4).Background(chrome.Color(colors.Field)).TextColor(muted).Shrink(0)
						ui.Text(c, label).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(label)
						detail := job.Progress
						if detail == "" {
							detail = job.Detail
						}
						if detail == "" {
							detail = jobStatus(job.Status)
						}
						ui.Text(c, detail).FontSize(theme.InterfaceSize(10)).TextColor(muted).MaxWidthPercent(35).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(detail)
						ui.Text(c, jobDuration(job, time.Now())).FontSize(theme.InterfaceSize(11)).TextColor(muted).Shrink(0)
					} else {
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
							ink := fg
							if !live {
								ink = muted
							}
							ui.Text(c, label).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(17)).TextColor(ink).FillWidth().MinWidth(0).SingleLine().Ellipsis("…").Tooltip(label)
							ui.Row(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
								ui.Text(c, job.Kind).FontSize(theme.InterfaceSize(10)).FixedLineHeight(theme.InterfaceSize(16)).Padding(0, 5).Radius(4).Background(chrome.Color(colors.Sidebar)).TextColor(muted).Shrink(0)
								detail := job.Progress
								if detail == "" {
									detail = job.Detail
								}
								if detail == "" {
									detail = jobStatus(job.Status)
								}
								ui.Text(c, detail).FontSize(theme.InterfaceSize(10)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(muted).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(detail)
								ui.Text(c, jobDuration(job, time.Now())).FontSize(theme.InterfaceSize(11)).TextColor(muted).Shrink(0)
							})
						})
					}
					if job.Observable() {
						ui.Box(c).Size(20, 20).Shrink(0).Border(.5, chrome.Color(colors.Border)).Radius(4).Background(chrome.Color(colors.BG)).Center().Children(func() {
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
				stop := ui.Button(c.Key("job-stop-"+job.ID), "").Label("Stop task: "+job.ID).Tooltip(archiveText("停止任务", "Stop task")).Size(20, 20).Padding(4).Border(.5, chrome.Color(colors.Border)).Radius(4).Background(chrome.Color(colors.BG)).Disabled(s.Loading || s.Error != "" || s.Kill.Phase == "pending")
				stop.Children(func() {
					if s.Kill.JobID == job.ID && s.Kill.Phase == "pending" {
						ui.Spinner(c).Size(10, 10).TextColor(muted)
					} else {
						ui.Icon(c, jobIcons["stop"]).Size(10, 10).TextColor(muted)
					}
				})
				if stop.Clicked() {
					if s.Kill.JobID == job.ID && s.Kill.Phase == "armed" {
						controller.ConfirmKill(s.Generation, job.ID)
					} else {
						controller.ArmKill(s.Generation, job.ID)
					}
				}
			}
		})
		if s.Kill.JobID == job.ID && (s.Kill.Phase == "armed" || s.Kill.Error != "") {
			ui.Column(c).FillWidth().MinWidth(0).Padding(6, 8).Gap(4).Children(func() {
				message := s.Kill.Error
				if s.Kill.Phase == "armed" {
					message = archiveText("再次点击确认停止", "Click again to confirm")
				}
				ui.Text(c, message).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				ui.Row(c).Gap(6).Children(func() {
					if s.Kill.Phase == "armed" && ui.Button(c, archiveText("确认停止", "Confirm stop")).FontSize(theme.InterfaceSize(11)).Clicked() {
						controller.ConfirmKill(s.Generation, job.ID)
					}
					if ui.Button(c, archiveText("关闭提示", "Dismiss")).FontSize(theme.InterfaceSize(11)).Clicked() {
						controller.CancelKill(s.Generation)
					}
				})
			})
		}
		if s.SelectedID == job.ID {
			jobOutputView(c, controller, s, job, colors)
		}
	})
}

func jobOutputView(c *ui.Context, controller *jobflow.Controller, s jobflow.Snapshot, job jobflow.Job, colors theme.Colors) {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	o := s.Observed
	panel := ui.Column(c.Key("job-output-panel-"+job.ID)).FillWidth().MinWidth(0).Padding(0, 8, 8).Gap(6)
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
			ui.Text(c, archiveText("更早的输出已截断", "Earlier output was dropped")).FontSize(theme.InterfaceSize(11)).TextColor(muted).FillWidth().MinWidth(0)
		}
		if o.Error != "" && !notice.dismissed {
			ui.Text(c, o.Error).FontSize(theme.InterfaceSize(11)).TextColor(muted).FillWidth().MinWidth(0)
			if ui.Button(c, archiveText("重试输出", "Retry output")).Clicked() {
				controller.RetryOutput(s.Generation)
			}
			if ui.Button(c, archiveText("关闭提示", "Dismiss")).Clicked() {
				notice.dismissed = true
			}
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
			ui.Text(c, job.Label).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(1).MinWidth(0)
			if ui.Button(c, archiveText("复制", "Copy")).Label("Copy job command").FontSize(theme.InterfaceSize(11)).Disabled(job.Label == "").Clicked() {
				mygo.Clipboard.WriteText(job.Label)
			}
		})
		ui.Scroll(c.Key("job-output-" + job.ID)).Label("Job output scroll: " + job.ID).FillWidth().MinWidth(0).MaxHeight(288).Children(func() {
			body := o.Text
			if body == "" {
				body = archiveText("暂无输出", "No output yet.")
			}
			ui.Text(c, body).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(12)).TextColor(fg).FillWidth().MinWidth(0).Selectable()
		})
	})
}
func jobDuration(job jobflow.Job, now time.Time) string {
	end := job.FinishedAt
	if job.Live() {
		end = float64(now.UnixMilli())
	}
	seconds := int64(max(0, end-job.StartedAt) / 1000)
	if seconds >= 3600 {
		return fmt.Sprintf("%dh %dm", seconds/3600, seconds%3600/60)
	}
	if seconds >= 60 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	return fmt.Sprintf("%ds", seconds)
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
		return archiveText("失败", "failed")
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
