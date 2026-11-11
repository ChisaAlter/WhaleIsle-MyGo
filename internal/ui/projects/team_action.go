package projects

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

const (
	ProjectTeamMemberPageSize = 6
	ProjectTeamTaskPageSize   = 5
)

// TeamViewState belongs to one Host/coordinator. Both the anchored action and
// the Team Sidebar use this state, independently of the Project overview.
type TeamViewState struct {
	FloatingOpen, FloatingVisited bool
	History                       bool
	Query                         string
	MemberPage, TaskPage          int
	ExpandedTaskIDs               map[string]bool
	Scroll                        ui.ScrollState
	NavigationError, FailedMember string
}

func (s *TeamViewState) SetFloating(open bool) {
	s.FloatingOpen = open
	if open {
		s.FloatingVisited = true
	}
}

func (s *TeamViewState) SetHistory(history bool) {
	s.History, s.Query, s.MemberPage, s.TaskPage = history, "", 0, 0
}

func (s *TeamViewState) SetQuery(query string) {
	s.Query, s.MemberPage, s.TaskPage = query, 0, 0
}

type ProjectTeamModel struct {
	Project                            projectflow.Project
	SessionID, ThemeID, Error          string
	SystemDark                         bool
	State                              *TeamViewState
	Overview                           *projectflow.Overview
	Loading, ActivityStale, CountKnown bool
	RunningCount                       int
	NextRefresh                        time.Time
	MaxHeight                          float32
	Columns                            int
	OnRefresh                          func()
	// The header opens in the main conversation; Team Sidebar opens in its own
	// pane. The frame supplies the corresponding verified Session route.
	OnOpenMember func(string) error
}

// Scroll handles belong to a presentation; only their offsets are shared.
// A pinned Sidebar and an open popover may coexist without binding one native
// handle to two different scroll elements.
type teamSurfaceScroll struct {
	track   ui.ScrollState
	applied float32
}

// ProjectTeamBody is the shared, read-only TeamAction/TeamSidebar body. It does
// not render Project controls, workstream details, or the ordinary Team browser.
func ProjectTeamBody(c *ui.Context, m ProjectTeamModel) {
	if m.State == nil || m.Project.ID == "" {
		ui.Text(c, tr("Team 暂不可用", "Team is unavailable"))
		return
	}
	if !m.Loading && !m.NextRefresh.IsZero() && !c.Now().Before(m.NextRefresh) && m.OnRefresh != nil {
		m.OnRefresh()
	}
	if !m.NextRefresh.IsZero() {
		c.After(time.Second)
	}
	s := m.State
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	root := ui.Column(c.Key("project-team-body:" + m.Project.CoordinatorSessionID)).FillWidth().MinWidth(0).MinHeight(0).Gap(10)
	scroll := ui.Local(root, "team-scroll", func() teamSurfaceScroll { return teamSurfaceScroll{} })
	if scroll.track.Y != scroll.applied {
		s.Scroll.Y = scroll.track.Y
	}
	scroll.track.Y, scroll.applied = s.Scroll.Y, s.Scroll.Y
	if m.MaxHeight > 0 {
		root.MaxHeight(m.MaxHeight)
	} else {
		root.FillHeight()
	}
	root.Children(func() {
		compact := m.Overview != nil && m.Overview.Team != nil && len(m.Overview.Team.Members) == 1 && len(m.Overview.Team.Tasks) == 0
		if m.Overview != nil && m.Overview.Team != nil && !compact {
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Label(tr("团队记录范围", "Team record scope")).Children(func() {
					for _, option := range []struct {
						history bool
						label   string
					}{{false, tr("当前与近期", "Current and recent")}, {true, tr("全部记录", "All records")}} {
						if ui.Button(c, option.label).Role(ui.RoleTab).Selected(s.History == option.history).Cursor(ui.CursorPointer).Clicked() {
							s.SetHistory(option.history)
						}
					}
				})
				if s.History {
					query := s.Query
					if ui.TextInput(c, &query).FillWidth().Label(tr("查找成员或工作", "Find a member or work")).Placeholder(tr("输入名称或工作内容", "Search names or work descriptions")).Changed() {
						s.SetQuery(query)
					}
				}
			})
		}
		body := ui.Scroll(c.Key("project-team-records")).FillWidth().MinWidth(0).MinHeight(0).Grow(1).TrackScroll(&scroll.track)
		body.Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(16).Padding(8, 12, 16, 14).Children(func() {
				if m.Error != "" || s.NavigationError != "" {
					message := m.Error
					if s.NavigationError != "" {
						message = s.NavigationError
					}
					ui.Column(c).FillWidth().Gap(5).Children(func() {
						ui.Text(c, message).TextColor(chrome.Color(colors.Danger)).FontSize(theme.InterfaceSize(12)).Selectable()
						if m.Error != "" && m.OnRefresh != nil && ui.Button(c, tr("重试", "Retry")).Cursor(ui.CursorPointer).Disabled(m.Loading).Clicked() {
							m.OnRefresh()
						}
						if s.NavigationError != "" && ui.Button(c, tr("关闭提示", "Dismiss notice")).Cursor(ui.CursorPointer).Clicked() {
							s.NavigationError = ""
						}
					})
				}
				if m.Overview == nil || m.Overview.Team == nil {
					if !m.Loading && m.Error == "" {
						ui.Text(c, tr("Team 暂不可用", "Team is unavailable")).TextColor(muted)
					}
					return
				}
				if m.ActivityStale {
					ui.Text(c, tr("团队活动暂未确认；后台命令记录为上次确认状态。", "Team activity is unconfirmed; background command records show their last confirmed state.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
				team := m.Overview.Team
				if team.Failure != "" {
					ui.Text(c, tr("团队持久记录无效：", "Invalid team persistence record: ")+team.Failure).Tooltip("Invalid persisted Team record: " + team.Failure).TextColor(chrome.Color(colors.Danger)).Selectable()
				}
				members, tasks, labels := projectTeamRecords(m)
				s.MemberPage = teamBoundedPage(s.MemberPage, len(members), ProjectTeamMemberPageSize)
				s.TaskPage = teamBoundedPage(s.TaskPage, len(tasks), ProjectTeamTaskPageSize)
				ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Padding(0, 0, 0, 4).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tr("成员", "Members")).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg)
						if len(members) > 1 {
							ui.Text(c, fmt.Sprint(len(members))).FontSize(theme.InterfaceSize(12)).FontWeight(400).TextColor(chrome.Color(colors.Caption))
						}
					})
					page := teamPage(members, s.MemberPage, ProjectTeamMemberPageSize)
					columns := max(1, m.Columns)
					for start := 0; start < len(page); start += columns {
						ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
							for index := start; index < min(start+columns, len(page)); index++ {
								member := page[index]
								ui.Column(c).Grow(1).MinWidth(0).Children(func() { projectTeamMember(c.Key("member:"+member.ID), m, member, labels[member.Name]) })
							}
							if columns > 1 && start+1 == len(page) {
								ui.Box(c).Grow(1)
							}
						})
					}
					if len(members) == 0 {
						ui.Text(c, tr("没有匹配成员，可尝试其他关键词。", "No matching members. Try another keyword.")).TextColor(muted)
					}
					projectTeamPagination(c, &s.MemberPage, len(members), ProjectTeamMemberPageSize, tr("成员分页", "Member pages"))
				})
				ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
					if len(tasks) == 0 {
						message := tr("暂无相关工作。以往工作可在全部记录中查看。", "No related work. Earlier work is available in All records.")
						if s.History && strings.TrimSpace(s.Query) != "" {
							message = tr("没有匹配工作，可尝试其他关键词。", "No matching work. Try another keyword.")
						} else if len(team.Tasks) == 0 {
							message = tr("尚未安排后台工作，在主对话中提出需求即可。", "No background work yet. Describe what you need in the main conversation.")
						}
						ui.Row(c).FillWidth().Padding(0, 0, 0, 4).Children(func() {
							ui.Text(c, message).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
						})
						return
					}
					ui.Row(c).FillWidth().MinWidth(0).Padding(0, 0, 0, 4).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, tr("工作", "Work")).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg)
						ui.Text(c, fmt.Sprint(len(tasks))).FontSize(theme.InterfaceSize(12)).FontWeight(400).TextColor(chrome.Color(colors.Caption))
					})
					if !s.History && m.Project.CurrentTaskIDs == nil && teamHasRecentCompletion(team.Tasks) {
						ui.Text(c, tr("包含近期登记的历史工作。", "Includes recently recorded historical work.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
					for _, task := range teamPage(tasks, s.TaskPage, ProjectTeamTaskPageSize) {
						projectTeamTask(c.Key("task:"+task.ID), m, task, labels)
					}
					projectTeamPagination(c, &s.TaskPage, len(tasks), ProjectTeamTaskPageSize, tr("工作分页", "Work pages"))
				})
			})
		})
	})
}

func teamBoundedPage(page, total, size int) int { return max(0, min(page, max(0, (total-1)/size))) }

func teamPage[T any](items []T, page, size int) []T {
	start := min(len(items), max(0, page)*size)
	return items[start:min(len(items), start+size)]
}

func projectTeamPagination(c *ui.Context, page *int, total, size int, label string) {
	if total <= size {
		return
	}
	ui.Row(c.Key(label)).FillWidth().MinWidth(0).Wrap().Gap(6).AlignItems(ui.Center).Label(label).Children(func() {
		ui.Text(c, fmt.Sprintf("%d–%d / %d", *page*size+1, min((*page+1)*size, total), total)).Grow(1).FontSize(theme.InterfaceSize(12))
		if ui.Button(c, tr("上一页", "Previous")).Cursor(ui.CursorPointer).Disabled(*page == 0).Clicked() {
			*page--
		}
		if ui.Button(c, tr("下一页", "Next")).Cursor(ui.CursorPointer).Disabled((*page+1)*size >= total).Clicked() {
			*page++
		}
	})
}

// Current-task association is optional: absent means a legacy Project, while
// an explicit empty slice means there is no current Request work.
func projectTeamRecords(m ProjectTeamModel) ([]projectflow.TeamMember, []projectflow.TeamTask, map[string]string) {
	team := m.Overview.Team
	tasks := make([]projectflow.TeamTask, 0, len(team.Tasks))
	byID := map[string]projectflow.TeamTask{}
	for _, task := range team.Tasks {
		if task.Status != "deleted" {
			tasks = append(tasks, task)
			byID[task.ID] = task
		}
	}
	labels := map[string]string{}
	for _, id := range m.Project.CurrentTaskIDs {
		if task, ok := byID[id]; ok && task.OwnerName != "" && labels[task.OwnerName] == "" {
			labels[task.OwnerName] = task.Subject
		}
	}
	for _, task := range tasks {
		if task.OwnerName != "" && labels[task.OwnerName] == "" {
			labels[task.OwnerName] = task.Subject
		}
	}
	for _, member := range team.Members {
		if member.Role == "lead" {
			labels[member.Name] = tr("协调者", "Coordinator")
		} else if labels[member.Name] == "" {
			labels[member.Name] = member.Description
			if labels[member.Name] == "" {
				labels[member.Name] = member.Name
			}
		}
	}
	if m.State.History {
		search := strings.ToLower(strings.TrimSpace(m.State.Query))
		matchedOwners := map[string]bool{}
		matchingTasks := map[string]bool{}
		for _, task := range tasks {
			if strings.Contains(strings.ToLower(task.Subject+"\n"+task.Description), search) {
				matchingTasks[task.ID], matchedOwners[task.OwnerName] = true, true
			}
		}
		members := make([]projectflow.TeamMember, 0, len(team.Members))
		for i := len(team.Members) - 1; i >= 0; i-- {
			member := team.Members[i]
			if strings.Contains(strings.ToLower(labels[member.Name]), search) || strings.Contains(strings.ToLower(member.Description), search) || matchedOwners[member.Name] {
				members = append(members, member)
			}
		}
		filtered := make([]projectflow.TeamTask, 0, len(tasks))
		for i := len(tasks) - 1; i >= 0; i-- {
			task := tasks[i]
			if matchingTasks[task.ID] || task.OwnerName != "" && strings.Contains(strings.ToLower(labels[task.OwnerName]), search) {
				filtered = append(filtered, task)
			}
		}
		return members, filtered, labels
	}
	live := map[string]bool{}
	jobs := map[string]bool{}
	for _, id := range m.Project.Activity.RunningJobSessionIDs {
		jobs[id] = true
		if !m.ActivityStale {
			live[id] = true
		}
	}
	activeOwners := map[string]bool{}
	for _, member := range team.Members {
		if member.Running || live[member.ID] || member.Phase == "provisioning" {
			activeOwners[member.Name] = true
		}
	}
	current := make([]projectflow.TeamTask, 0, len(tasks))
	order := map[string]int{}
	for i, id := range m.Project.CurrentTaskIDs {
		order[id] = i
	}
	for _, task := range tasks {
		if m.Project.CurrentTaskIDs == nil && (task.Status == "pending" || task.Status == "in_progress") || m.Project.CurrentTaskIDs != nil && task.OwnerName != "" && activeOwners[task.OwnerName] {
			current = append(current, task)
		}
	}
	if m.Project.CurrentTaskIDs != nil {
		sort.SliceStable(current, func(i, j int) bool {
			left, ok := order[current[i].ID]
			if !ok {
				left = len(order)
			}
			right, ok := order[current[j].ID]
			if !ok {
				right = len(order)
			}
			return left < right
		})
	}
	seen := map[string]bool{}
	for _, task := range current {
		seen[task.ID] = true
	}
	appendTask := func(task projectflow.TeamTask) {
		if !seen[task.ID] {
			current = append(current, task)
			seen[task.ID] = true
		}
	}
	if m.Project.CurrentTaskIDs != nil {
		for _, id := range m.Project.CurrentTaskIDs {
			if task, ok := byID[id]; ok {
				appendTask(task)
			}
		}
	} else {
		recent := 0
		for i := len(tasks) - 1; i >= 0 && recent < 3; i-- {
			if tasks[i].Status == "completed" {
				appendTask(tasks[i])
				recent++
			}
		}
	}
	owners := map[string]bool{}
	for _, task := range current {
		owners[task.OwnerName] = true
	}
	members := make([]projectflow.TeamMember, 0, len(team.Members))
	for _, member := range team.Members {
		if member.Role == "lead" || owners[member.Name] || member.Running || live[member.ID] || jobs[member.ID] || member.Phase == "provisioning" {
			members = append(members, member)
		}
	}
	return members, current, labels
}

func teamHasRecentCompletion(tasks []projectflow.TeamTask) bool {
	for _, task := range tasks {
		if task.Status == "completed" {
			return true
		}
	}
	return false
}

func projectTeamMember(c *ui.Context, m ProjectTeamModel, member projectflow.TeamMember, label string) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	muted := chrome.Color(colors.Muted)
	runningJob := false
	for _, id := range m.Project.Activity.RunningJobSessionIDs {
		if id == member.ID {
			runningJob = true
			break
		}
	}
	status := member.Phase
	if status == "active" {
		status = "inactive"
		if member.Running || runningJob && !m.ActivityStale {
			status = "running"
		}
	}
	current := member.ID == m.SessionID
	highlightCurrent := current && len(m.Overview.Team.Members) > 1
	inert := current || status == "failed" || status == "provisioning" || m.OnOpenMember == nil
	borderColor := chrome.Color(colors.Border)
	borderWidth := float32(.5)
	if member.Error != "" || (m.State.FailedMember == member.ID && m.State.NavigationError != "") {
		borderColor = chrome.Color(colors.Danger)
	} else if highlightCurrent {
		borderColor = chrome.MaterialFill(chrome.Color(colors.Accent), .45)
		borderWidth = 1
	}
	card := ui.Column(c.Key("project-member-card:" + member.ID)).FillWidth().MinWidth(0)
	cardHovered := card.Hovered()
	card.Children(func() {
		button := ui.Button(c, "").Label(label).FillWidth().MinWidth(0).Padding(10, 12).Radius(8).Border(borderWidth, borderColor).Background(chrome.Color(colors.Field))
		if !inert {
			button.Tooltip(tr("查看工作过程", "View work history")).Cursor(ui.CursorPointer)
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
						ui.Text(c, label).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(label).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(chrome.Color(colors.FG))
						if current {
							ui.Row(c).Shrink(0).Padding(1, 5).Radius(4).Background(chrome.MaterialFill(chrome.Color(colors.Accent), .15)).AlignItems(ui.Center).Children(func() {
								ui.Text(c, tr("当前会话", "Current chat")).FontSize(theme.InterfaceSize(10)).FixedLineHeight(theme.InterfaceSize(14)).TextColor(chrome.Color(colors.Accent)).SingleLine()
							})
						}
					})
					text := stateLabel(status)
					if status == "inactive" {
						text = tr("空闲", "Idle")
					}
					if member.Phase != "active" && member.Running {
						text += " · " + tr("回合运行中", "Turn running")
					}
					if runningJob {
						job := tr("后台命令运行中", "Background command running")
						if m.ActivityStale {
							job = tr("后台命令上次仍在运行", "Background command last seen running")
						}
						text += " · " + job
					}
					if showModel && member.ModelLabel != "" {
						text += " · " + tr("模型: ", "Model: ") + member.ModelLabel
					}
					ui.Text(c, text).FillWidth().FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(muted).SingleLine().Ellipsis("…")
					if member.Error != "" {
						ui.Text(c, member.Error).FillWidth().FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Danger)).Selectable()
					}
					if m.State.FailedMember == member.ID && m.State.NavigationError != "" {
						ui.Text(c, m.State.NavigationError).FillWidth().FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Danger))
					}
				})
			})
		})
		if !inert && button.Clicked() {
			if err := m.OnOpenMember(member.ID); err != nil {
				m.State.NavigationError, m.State.FailedMember = err.Error(), member.ID
			} else {
				m.State.NavigationError, m.State.FailedMember = "", ""
			}
		}
	})
}

func projectTeamTask(c *ui.Context, m ProjectTeamModel, task projectflow.TeamTask, labels map[string]string) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	ui.Column(c).FillWidth().MinWidth(0).Gap(6).Padding(11, 13).Radius(9).Border(.5, chrome.Color(colors.Border)).Background(chrome.Color(colors.Field)).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Start).Children(func() {
			ui.Text(c, task.Subject).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg)
			ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
				state := widgets.StateIdle
				if task.Status == "in_progress" {
					state = widgets.StateRunning
				}
				if task.Status == "completed" {
					state = widgets.StateDone
				}
				if task.Status == "pending" && !task.Ready {
					state = widgets.StateWarning
				}
				widgets.StateDot(c, state)
				label := stateLabel(task.Status)
				if task.Status == "completed" {
					label = tr("已完成", "Completed")
				}
				ui.Text(c, label).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			})
		})
		expanded := m.State.ExpandedTaskIDs[task.ID]
		if expanded && task.Description != "" {
			ui.Text(c, task.Description).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(chrome.Color(colors.Secondary)).Selectable()
		}
		owner := labels[task.OwnerName]
		if owner == "" {
			owner = task.OwnerName
		}
		if owner == "" {
			owner = tr("未分配", "Unowned")
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(10).AlignItems(ui.Start).Children(func() {
			ui.Row(c).Grow(1).MinWidth(0).Wrap().Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, tr("负责人: ", "Assignee: ")+owner).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
				if task.Status == "pending" {
					label := tr("被依赖阻塞", "Blocked by dependencies")
					if task.Ready {
						label = tr("可开始", "Ready")
					}
					ui.Text(c, label).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
				}
				if len(task.BlockedBy) > 0 {
					dependencies := make([]string, 0, len(task.BlockedBy))
					for _, id := range task.BlockedBy {
						label := tr("前置工作不可用", "Prerequisite unavailable")
						for _, candidate := range m.Overview.Team.Tasks {
							if candidate.ID == id {
								label = candidate.Subject
								break
							}
						}
						dependencies = append(dependencies, label)
					}
					ui.Text(c, tr("依赖: ", "Blocked by: ")+strings.Join(dependencies, ", ")).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
				}
				if len(task.WriteScopes) > 0 {
					ui.Text(c, tr("写入范围: ", "Write scopes: ")+strings.Join(task.WriteScopes, ", ")).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted)
				}
				for _, warning := range task.WriteScopeWarnings {
					ui.Text(c, warning).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(chrome.Color(colors.Danger))
				}
			})
			if task.Description != "" {
				label := tr("展开", "Show more")
				icon := "chevron-down"
				if expanded {
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
					if m.State.ExpandedTaskIDs == nil {
						m.State.ExpandedTaskIDs = map[string]bool{}
					}
					m.State.ExpandedTaskIDs[task.ID] = !expanded
				}
			}
		})
	})
}
