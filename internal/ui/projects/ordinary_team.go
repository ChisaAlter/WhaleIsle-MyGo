package projects

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type OrdinaryTeamModel struct {
	SessionID, ThemeID, Error string
	SystemDark, Loading       bool
	NextRefresh               time.Time
	MaxHeight                 float32
	Team                      *projectflow.TeamProjection
	OnRefresh                 func()
	OnOpenMember              func(string)
	State                     *OrdinaryTeamViewState
	Columns                   int
}

// View state follows the Team owner between the header and its Sidebar.
// Native scroll handles remain local to each mounted presentation.
type OrdinaryTeamViewState struct {
	FloatingOpen, Pinned, HoverDismissed bool
	HoverAt, LeaveAt                     time.Time
	NavigationError                      string
	Scroll                               ui.ScrollState
	Expanded                             map[string]bool
}

// OrdinaryTeamView is TeamAction's ordinary-session body. Project filtering,
// paging and current-work scopes remain with the Project Team owner.
func OrdinaryTeamView(c *ui.Context, m OrdinaryTeamModel) {
	if !m.Loading && !m.NextRefresh.IsZero() && time.Now().After(m.NextRefresh) && m.OnRefresh != nil {
		m.OnRefresh()
	}
	if !m.NextRefresh.IsZero() {
		c.After(time.Second)
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	body := ui.Scroll(c.Key("ordinary-team:" + m.SessionID)).FillWidth().MinWidth(0).MinHeight(0)
	if m.State != nil {
		track := ui.Local(body, "scroll", func() teamSurfaceScroll { return teamSurfaceScroll{} })
		if track.track.Y != track.applied {
			m.State.Scroll.Y = track.track.Y
		}
		track.track.Y, track.applied = m.State.Scroll.Y, m.State.Scroll.Y
		body.TrackScroll(&track.track)
	}
	if m.MaxHeight > 0 {
		body.MaxHeight(m.MaxHeight)
	} else {
		body.FillHeight()
	}
	body.Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Padding(10, 12, 12, 12).Gap(12).Children(func() {
			if m.Error != "" || m.Team == nil && !m.Loading {
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					message := m.Error
					if message == "" {
						message = tr("Team 暂不可用", "Team is unavailable")
					}
					ui.Text(c, message).TextColor(fg).Selectable()
					if m.OnRefresh != nil && ui.Button(c, tr("重试", "Retry")).Disabled(m.Loading).Clicked() {
						m.OnRefresh()
					}
				})
			}
			if m.Team == nil {
				return
			}
			if m.Team.Failure != "" {
				ui.Text(c, tr("团队持久记录无效：", "Invalid persisted Team record: ")+m.Team.Failure).TextColor(chrome.Color(colors.Danger)).Selectable()
			}
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				ui.Text(c, tr("成员", "Members")).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				columns := max(1, m.Columns)
				for start := 0; start < len(m.Team.Members); start += columns {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
						for index := start; index < min(start+columns, len(m.Team.Members)); index++ {
							member := m.Team.Members[index]
							ui.Column(c).Grow(1).MinWidth(0).Children(func() {
								current := member.ID == m.SessionID
								button := ui.Button(c.Key("team-member:"+member.ID), "").Label(member.Name).Tooltip(tr("打开成员会话", "Open member conversation")).FillWidth().MinWidth(0).Padding(8).Radius(6).BorderWidth(0).Background(ui.Transparent).Disabled(current || member.Phase == "failed" || member.Phase == "provisioning" || m.OnOpenMember == nil)
								button.Children(func() {
									ui.Column(c).FillWidth().MinWidth(0).Gap(3).Children(func() {
										ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
											ui.Text(c, member.Name).FontSize(theme.InterfaceSize(13)).TextColor(fg).MinWidth(0).Shrink(1).SingleLine().Ellipsis("…")
											if current {
												ui.Text(c, tr("当前会话", "Current chat")).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Accent))
											}
										})
										status := stateLabel(member.Phase)
										if member.Phase == "active" {
											status = tr("未运行", "Inactive")
											if member.Running {
												status = tr("运行中", "Running")
											}
										}
										if member.ModelLabel != "" {
											status += " · " + tr("模型：", "Model: ") + member.ModelLabel
										}
										ui.Text(c, status).FontSize(theme.InterfaceSize(12)).TextColor(muted)
										if member.Error != "" {
											ui.Text(c, member.Error).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Danger))
										}
										if member.ModelError != "" {
											ui.Text(c, member.ModelError).FontSize(theme.InterfaceSize(12)).TextColor(muted)
										}
									})
								})
								if button.Clicked() {
									m.OnOpenMember(member.ID)
								}
							})
						}
						if columns > 1 && start+1 == len(m.Team.Members) {
							ui.Box(c).Grow(1)
						}
					})
				}
			})
			count := 0
			for _, task := range m.Team.Tasks {
				if task.Status != "deleted" {
					count++
				}
			}
			if count == 0 {
				ui.Text(c, tr("暂无共享任务，可以通过对话创建", "No shared tasks yet. Create them through the conversation.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				return
			}
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, tr("共享任务", "Shared tasks")).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				for _, task := range m.Team.Tasks {
					if task.Status == "deleted" {
						continue
					}
					task := task
					row := ui.Column(c.Key("team-task:"+task.ID)).FillWidth().MinWidth(0).Gap(6).Padding(11, 13).Radius(9).Border(.5, chrome.Color(colors.Border))
					expanded := ui.Local(row, "expanded", func() bool { return false })
					if m.State != nil {
						*expanded = m.State.Expanded[task.ID]
					}
					row.Children(func() {
						ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
							ui.Text(c, task.Subject).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg).Grow(1).MinWidth(0)
							status := stateLabel(task.Status)
							if task.Status == "completed" {
								status = tr("已完成", "Completed")
							}
							ui.Text(c, status).FontSize(theme.InterfaceSize(12)).TextColor(muted).Shrink(0)
						})
						if task.Description != "" {
							text := ui.Text(c, task.Description).FontSize(theme.InterfaceSize(12)).TextColor(fg).FillWidth()
							clamped := ui.Local(row, "clamped", func() bool { return false })
							if !*expanded {
								text.MaxLines(2).Ellipsis("…")
							}
							font := theme.RenderPreferences().Interface.Family
							text.DrawOver(func(p *ui.Painter, rect ui.Rect) {
								_, full := p.MeasureText(rect.W, ui.Span{Text: task.Description, Font: font, Size: theme.InterfaceSize(12)})
								_, two := p.MeasureText(0, ui.Span{Text: "Ag\nAg", Font: font, Size: theme.InterfaceSize(12)})
								if next := full > two+.5; next != *clamped {
									*clamped = next
									c.Services().Invalidate()
								}
							})
							label := tr("展开", "Expand")
							if *expanded {
								label = tr("收起", "Collapse")
							}
							if (*expanded || *clamped) && ui.Button(c, label).FontSize(theme.InterfaceSize(12)).Padding(0).BorderWidth(0).Background(ui.Transparent).AlignSelf(ui.Start).Clicked() {
								*expanded = !*expanded
								if m.State != nil {
									if m.State.Expanded == nil {
										m.State.Expanded = map[string]bool{}
									}
									m.State.Expanded[task.ID] = *expanded
								}
							}
						}
						owner := task.OwnerName
						if owner == "" {
							owner = tr("未分配", "Unassigned")
						}
						meta := task.ID + " · " + tr("Owner：", "Owner: ") + owner
						if task.Status == "pending" {
							if task.Ready {
								meta += " · " + tr("可开始", "Ready")
							} else {
								meta += " · " + tr("被依赖阻塞", "Blocked")
							}
						}
						ui.Text(c, meta).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						if len(task.BlockedBy) > 0 {
							ui.Text(c, tr("依赖：", "Dependencies: ")+strings.Join(task.BlockedBy, ", ")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						}
						if len(task.WriteScopes) > 0 {
							ui.Text(c, tr("写入范围：", "Write scopes: ")+strings.Join(task.WriteScopes, ", ")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						}
						for _, warning := range task.WriteScopeWarnings {
							ui.Text(c, warning).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Danger))
						}
					})
				}
			})
		})
	})
}
