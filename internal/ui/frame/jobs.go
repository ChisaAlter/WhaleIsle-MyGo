package frame

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	"time"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func jobsHeader(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	jobs := m.Store.NativeJobs()
	s := jobs.Snapshot()
	if ui.Button(c, archiveText("后台任务", "Jobs")).FontSize(12).BorderWidth(0).Background(ui.Transparent).Disabled(!s.Ready).Clicked() {
		jobs.Open(s.Generation)
	}
}
func jobsDialog(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	jobs := m.Store.NativeJobs()
	s := jobs.Snapshot()
	if !s.Open {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	open := true
	_, height := c.Size()
	panelHeight := min(float32(650), max(float32(280), height-64))
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(880).MaxWidthPercent(94).Height(panelHeight).Padding(18).Background(chrome.Color(colors.BG)).Radius(14).Children(func() {
			ui.Column(c).Fill().Gap(10).Children(func() {
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, archiveText("后台任务", "Background jobs")).FontSize(16).Bold().TextColor(fg).Grow(1)
					if ui.Button(c, archiveText("清除已结束", "Clear finished")).FontSize(11).Clicked() {
						jobs.ClearSettled(s.Generation)
					}
					if ui.Button(c, archiveText("关闭", "Close")).Clicked() {
						jobs.Close(s.Generation)
					}
				})
				ui.Text(c, archiveText("会话：", "Session: ")+s.SessionID).FontSize(11).TextColor(muted)
				if s.Loading {
					ui.Text(c, archiveText("正在连接任务列表…", "Connecting to jobs…")).FontSize(12).TextColor(muted)
				}
				if s.Error != "" {
					ui.Text(c, s.Error).FontSize(12).TextColor(fg)
					if ui.Button(c, archiveText("重连任务列表", "Reconnect jobs")).Clicked() {
						jobs.Retry(s.Generation)
					}
				}
				if !s.Loading && s.Error == "" && len(s.Rows) == 0 {
					ui.Text(c, archiveText("当前没有可显示的后台任务。", "No background jobs to show.")).FontSize(13).TextColor(muted)
				}
				ui.Scroll(c.Key("jobs-roster")).Grow(1).MinHeight(0).FillWidth().Children(func() {
					ui.Column(c).FillWidth().Gap(8).Children(func() {
						for _, row := range s.Rows {
							job := row
							ui.Column(c.Key("job-"+job.ID)).FillWidth().Gap(6).Padding(10).Border(1, chrome.Color(colors.Border)).Radius(8).Children(func() {
								ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
									label := job.Label
									if label == "" {
										label = job.ID
									}
									if s.SelectedID == job.ID {
										label = "▾ " + label
									} else if job.Observable() {
										label = "▸ " + label
									}
									if ui.Button(c, label).Grow(1).FontSize(13).Disabled(!job.Observable()).Clicked() {
										jobs.Toggle(s.Generation, job.ID)
									}
									ui.Text(c, jobStatus(job.Status)).FontSize(12).TextColor(fg)
									if job.Status == "running" {
										pending := s.Kill.Phase == "pending"
										caption := archiveText("终止", "Stop")
										if s.Kill.JobID == job.ID && pending {
											caption = archiveText("正在停止…", "Stopping…")
										}
										if ui.Button(c, caption).FontSize(11).Disabled(pending).Clicked() {
											jobs.ArmKill(s.Generation, job.ID)
										}
									}
								})
								detail := job.Progress
								if detail == "" {
									detail = job.Detail
								}
								line := job.Kind + " · " + job.ID
								if detail != "" {
									line += " · " + detail
								}
								ui.Text(c, line).FontSize(11).TextColor(muted).Selectable()
								if job.StartedAt > 0 {
									at := time.UnixMilli(int64(job.StartedAt)).Format("01-02 15:04:05")
									if job.FinishedAt > 0 {
										at += fmt.Sprintf(" · %.1fs", max(0, job.FinishedAt-job.StartedAt)/1000)
									}
									ui.Text(c, at).FontSize(11).TextColor(muted)
								}
								if s.Kill.JobID == job.ID {
									if s.Kill.Phase == "armed" {
										ui.Text(c, archiveText("确定终止这个后台任务？此操作不会自动重试任务。", "Stop this background job? It will not be restarted automatically.")).FontSize(12).TextColor(fg)
										ui.Row(c).Gap(8).Children(func() {
											if ui.Button(c, archiveText("确认终止", "Confirm stop")).Clicked() {
												jobs.ConfirmKill(s.Generation, job.ID)
											}
											if ui.Button(c, archiveText("取消", "Cancel")).Clicked() {
												jobs.CancelKill(s.Generation)
											}
										})
									}
									if s.Kill.Error != "" {
										ui.Text(c, s.Kill.Error).FontSize(12).TextColor(fg)
									}
								}
								if s.SelectedID == job.ID {
									o := s.Observed
									if o.GapBefore {
										ui.Text(c, archiveText("较早输出已被保留策略或显示上限截断。", "Earlier output was omitted by retention or the display limit.")).FontSize(11).TextColor(muted)
									}
									if o.Error != "" {
										ui.Text(c, o.Error).FontSize(12).TextColor(fg)
										if ui.Button(c, archiveText("重新连接输出", "Reconnect output")).Clicked() {
											jobs.RetryOutput(s.Generation)
										}
									}
									ui.Row(c).Gap(8).Children(func() {
										state := archiveText("输出已结束", "Output ended")
										if o.Streaming {
											state = archiveText("持续接收输出…", "Following output…")
										}
										ui.Text(c, state).FontSize(11).TextColor(muted).Grow(1)
										if ui.Button(c, archiveText("复制输出", "Copy output")).FontSize(11).Disabled(o.Text == "").Clicked() {
											mygo.Clipboard.WriteText(o.Text)
										}
									})
									ui.Scroll(c.Key("job-output-" + job.ID)).FillWidth().Height(220).Children(func() {
										body := o.Text
										if body == "" {
											body = archiveText("尚无输出。", "No output yet.")
										}
										ui.Text(c, body).FontSize(12).TextColor(fg).Selectable()
									})
									if len(job.Output.SpillPaths) > 0 {
										ui.Text(c, archiveText("完整输出保留文件：", "Retained output files: ")+strings.Join(job.Output.SpillPaths, ", ")).FontSize(11).TextColor(muted).Selectable()
									}
								}
							})
						}
					})
				})
				ui.Text(c, archiveText("清除只隐藏当前面板中的已结束任务；不会删除 Host 记录。", "Clear hides finished jobs in this panel; Host records are retained.")).FontSize(11).TextColor(muted)
			})
		})
	})
	if !open {
		jobs.Close(s.Generation)
	}
}
func jobStatus(status string) string {
	switch status {
	case "running":
		return archiveText("运行中", "Running")
	case "stopping":
		return archiveText("正在停止", "Stopping")
	case "completed":
		return archiveText("已完成", "Completed")
	case "killed":
		return archiveText("已终止", "Stopped")
	case "failed":
		return archiveText("失败", "Failed")
	default:
		return status
	}
}
