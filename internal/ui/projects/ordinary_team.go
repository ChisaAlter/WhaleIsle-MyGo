package projects

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
		ui.Column(c).FillWidth().MinWidth(0).Padding(8, 12, 16, 14).Gap(16).Children(func() {
			if m.Loading && m.Team == nil {
				ui.Row(c).FillWidth().Justify(ui.Center).Padding(4, 0).Children(func() {
					widgets.StateDot(c, widgets.StateRunning)
				})
			}
			if m.Error != "" || m.Team == nil && !m.Loading {
				ui.Column(c).FillWidth().Gap(6).Padding(4).Children(func() {
					message := m.Error
					if message == "" {
						message = tr("Team 暂不可用", "Team is unavailable")
					}
					ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
						widgets.StateDot(c, widgets.StateWarning)
						ui.Text(c, message).FontSize(theme.InterfaceSize(12)).TextColor(fg).Selectable()
					})
					if m.OnRefresh != nil && ui.Button(c, tr("重试", "Retry")).Cursor(ui.CursorPointer).Disabled(m.Loading).Clicked() {
						m.OnRefresh()
					}
				})
			}
			if m.Team == nil {
				return
			}
			if m.Team.Failure != "" {
				ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Padding(4).Children(func() {
					widgets.StateDot(c, widgets.StateError)
					ui.Text(c, tr("团队持久记录无效：", "Invalid team persistence record: ")+m.Team.Failure).Tooltip("Invalid persisted Team record: " + m.Team.Failure).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Danger)).Selectable()
				})
			}
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Padding(0, 0, 0, 4).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, tr("成员", "Members")).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg)
					if len(m.Team.Members) > 1 {
						ui.Text(c, fmt.Sprint(len(m.Team.Members))).FontSize(theme.InterfaceSize(12)).FontWeight(400).TextColor(chrome.Color(colors.Caption))
					}
				})
				columns := max(1, m.Columns)
				for start := 0; start < len(m.Team.Members); start += columns {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
						for index := start; index < min(start+columns, len(m.Team.Members)); index++ {
							member := m.Team.Members[index]
							card := ui.Column(c.Key("team-card:" + member.ID)).Grow(1).MinWidth(0)
							cardHovered := card.Hovered()
							card.Children(func() {
								status := member.Phase
								if status == "active" {
									if member.Running {
										status = "running"
									} else {
										status = "inactive"
									}
								}
								current := member.ID == m.SessionID
								highlightCurrent := current && len(m.Team.Members) > 1
								inert := current || status == "failed" || status == "provisioning" || m.OnOpenMember == nil
								borderColor := chrome.Color(colors.Border)
								borderWidth := float32(.5)
								if member.Error != "" {
									borderColor = chrome.Color(colors.Danger)
								} else if highlightCurrent {
									borderColor = chrome.MaterialFill(chrome.Color(colors.Accent), .45)
									borderWidth = 1
								}
								button := ui.Button(c.Key("team-member:"+member.ID), "").Label(member.Name).FillWidth().MinWidth(0).Padding(10, 12).Radius(8).Border(borderWidth, borderColor).Background(chrome.Color(colors.Field))
								if !inert {
									button.Tooltip(tr("打开成员会话", "Open member conversation")).Cursor(ui.CursorPointer)
								} else {
									button.Cursor(ui.CursorDefault)
								}
								showModel := cardHovered || button.Hovered() || button.Focused()
								button.Children(func() {
									ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
										ui.Row(c).Width(14).Height(theme.InterfaceSize(18)).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
											switch status {
											case "running", "provisioning":
												widgets.StateDot(c, widgets.StateRunning)
											case "failed":
												widgets.StateDot(c, widgets.StateError)
											default:
												projectIcon(c, "user", 14, muted, false)
											}
										})
										ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
											ui.Row(c).FillWidth().MinWidth(0).Gap(5).AlignItems(ui.Center).Children(func() {
												ui.Text(c, member.Name).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(fg).MinWidth(0).Shrink(1).SingleLine().Ellipsis("…")
												if current {
													ui.Row(c).Shrink(0).Padding(1, 5).Radius(4).Background(chrome.MaterialFill(chrome.Color(colors.Accent), .15)).AlignItems(ui.Center).Children(func() {
														ui.Text(c, tr("当前会话", "Current chat")).FontSize(theme.InterfaceSize(10)).FixedLineHeight(theme.InterfaceSize(14)).TextColor(chrome.Color(colors.Accent)).SingleLine()
													})
												}
											})
											statusText := stateLabel(status)
											if member.Phase != "active" && member.Running {
												statusText += " · " + tr("回合运行中", "Turn running")
											}
											if showModel && member.ModelLabel != "" {
												statusText += " · " + tr("模型: ", "Model: ") + member.ModelLabel
											}
											ui.Text(c, statusText).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(muted).SingleLine().Ellipsis("…")
											if member.Error != "" {
												ui.Text(c, member.Error).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Danger))
											}
										})
									})
								})
								if !inert && button.Clicked() {
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
				ui.Row(c).FillWidth().Padding(0, 0, 0, 4).Children(func() {
					ui.Text(c, tr("暂无共享任务，可以通过对话创建", "No shared tasks yet. Create them through the conversation.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				})
				return
			}
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Padding(0, 0, 0, 4).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, tr("共享任务", "Shared tasks")).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg)
					ui.Text(c, fmt.Sprint(count)).FontSize(theme.InterfaceSize(12)).FontWeight(400).TextColor(chrome.Color(colors.Caption))
				})
				for _, task := range m.Team.Tasks {
					if task.Status == "deleted" {
						continue
					}
					task := task
					row := ui.Column(c.Key("team-task:"+task.ID)).FillWidth().MinWidth(0).Gap(6).Padding(11, 13).Radius(9).Border(.5, chrome.Color(colors.Border)).Background(chrome.Color(colors.Field))
					expanded := ui.Local(row, "expanded", func() bool { return false })
					if m.State != nil {
						*expanded = m.State.Expanded[task.ID]
					}
					row.Children(func() {
						ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Text(c, task.Subject).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg).Grow(1).MinWidth(0)
							ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
								dotState := widgets.StateIdle
								switch task.Status {
								case "in_progress":
									dotState = widgets.StateRunning
								case "completed":
									dotState = widgets.StateDone
								case "pending":
									if !task.Ready {
										dotState = widgets.StateWarning
									}
								}
								widgets.StateDot(c, dotState)
								status := stateLabel(task.Status)
								if task.Status == "completed" {
									status = tr("已完成", "Completed")
								}
								ui.Text(c, status).FontSize(theme.InterfaceSize(11)).TextColor(muted)
							})
						})
						clamped := ui.Local(row, "clamped", func() bool { return false })
						if task.Description != "" {
							text := ui.Text(c, task.Description).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(chrome.Color(colors.Secondary)).FillWidth()
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
						}
						owner := task.OwnerName
						if owner == "" {
							owner = tr("未分配", "Unowned")
						}
						ui.Row(c).FillWidth().MinWidth(0).Gap(10).AlignItems(ui.Start).Children(func() {
							ui.Row(c).Grow(1).MinWidth(0).Wrap().Gap(10).AlignItems(ui.Center).Children(func() {
								ui.Text(c, task.ID).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
								ui.Text(c, tr("Owner: ", "Owner: ")+owner).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
								if task.Status == "pending" {
									readyLabel := tr("被依赖阻塞", "Blocked by dependencies")
									if task.Ready {
										readyLabel = tr("可开始", "Ready")
									}
									ui.Text(c, readyLabel).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
								}
								if len(task.BlockedBy) > 0 {
									ui.Text(c, tr("依赖: ", "Blocked by: ")+strings.Join(task.BlockedBy, ", ")).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
								}
								if len(task.WriteScopes) > 0 {
									ui.Text(c, tr("写入范围: ", "Write scopes: ")+strings.Join(task.WriteScopes, ", ")).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
								}
								for _, warning := range task.WriteScopeWarnings {
									ui.Text(c, warning).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(chrome.Color(colors.Danger))
								}
							})
							if task.Description != "" && (*expanded || *clamped) {
								label := tr("展开", "Show more")
								icon := "chevron-down"
								if *expanded {
									label = tr("收起", "Show less")
									icon = "chevron-up"
								}
								toggle := ui.Button(c, "").Label(label).Padding(0).BorderWidth(0).Background(ui.Transparent).Shrink(0).Cursor(ui.CursorPointer)
								toggleColor := muted
								if toggle.Hovered() || toggle.Focused() {
									toggleColor = fg
								}
								toggle.Children(func() {
									ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
										ui.Text(c, label).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(toggleColor)
										projectIcon(c, icon, 12, toggleColor, false)
									})
								})
								if toggle.Clicked() {
									*expanded = !*expanded
									if m.State != nil {
										if m.State.Expanded == nil {
											m.State.Expanded = map[string]bool{}
										}
										m.State.Expanded[task.ID] = *expanded
									}
								}
							}
						})
					})
				}
			})
		})
	})
}
