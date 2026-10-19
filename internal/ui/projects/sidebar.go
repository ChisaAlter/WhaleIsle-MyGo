package projects

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// View keeps navigation in the sidebar while reader/editor routes occupy their
// own modal layer. Collapsing the section never withdraws an open route.
func View(c *ui.Context, m Model) {
	if !m.Listing.Available && len(m.Listing.Projects) == 0 {
		projectModals(c, m)
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	visible := filteredProjects(m.Listing.Projects, m.SearchQuery, m.History)
	section := ui.Column(c).FillWidth().MinWidth(0).Padding(8, 0).Gap(4)
	m.presentation = ui.Local(section, "project-presentation", newProjectPresentation)
	section.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(4).AlignItems(ui.Center).Children(func() {
			category := ui.Button(c, "").Label("Project").Height(36).Grow(1).MinWidth(0).Padding(0, 4).BorderWidth(0).Background(ui.Transparent)
			category.Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(4).AlignItems(ui.Center).Children(func() {
					projectIcon(c, "chevron", 14, muted, m.Expanded)
					label := "Project"
					if m.History {
						label += " · " + tr("已归档", "Archived")
					}
					ui.Text(c, label).FontSize(theme.InterfaceSize(14)).TextColor(muted).MinWidth(0).SingleLine()
					if len(visible) > 0 {
						ui.Text(c, fmt.Sprint(len(visible))).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(0, 6).Radius(9).Background(chrome.Color(colors.Hover))
					}
				})
			})
			if category.Clicked() && m.OnToggle != nil {
				m.OnToggle()
			}
			add := projectIconButton(c, "project-add", tr("添加项目…", "Add Project…"), "add", muted).Disabled(m.Busy || !m.Listing.Available || m.Workspace == "")
			if add.Clicked() && m.OnCreate != nil {
				m.OnCreate()
			}
			projectIconButton(c, "project-options", tr("项目选项", "Project options"), "more", muted).Menu(func(menu *ui.Menu) {
				label := tr("显示已归档项目", "Show archived projects")
				if m.History {
					label = tr("显示未归档项目", "Show unarchived projects")
				}
				if menu.Item(label).Chosen() && m.OnHistory != nil {
					m.OnHistory(!m.History)
					if !m.Expanded && m.OnToggle != nil {
						m.OnToggle()
					}
				}
			})
		})
		if !m.Expanded {
			return
		}
		if m.Error != "" {
			ui.Text(c, m.Error).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		}
		if m.Busy && !m.Loaded {
			ui.Text(c, tr("正在加载项目…", "Loading Projects…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if m.RefreshNote != "" {
			ui.Text(c, m.RefreshNote).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		if m.Loaded && !m.Listing.Available {
			ui.Text(c, tr("项目当前不可用，请刷新重试。", "Projects are unavailable; refresh to retry.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if m.Loaded && m.Listing.Available && !m.Busy && len(visible) == 0 {
			if !m.History && m.SearchQuery == "" {
				add := ui.Button(c, "").Label(tr("添加 Project…", "Add Project…")).FillWidth().MinWidth(0).Padding(12).Margin(4, 0).Radius(12).Border(1, chrome.Color(colors.Border)).Background(ui.Transparent)
				add.Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Start).Children(func() {
						ui.Box(c).Size(28, 28).Radius(8).Background(chrome.Color(colors.Pressed)).AlignItems(ui.Center).Justify(ui.Center).Children(func() { projectIcon(c, "add", 16, fg, false) })
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
							ui.Text(c, tr("添加 Project…", "Add Project…")).FontSize(theme.InterfaceSize(14)).FontWeight(500).FixedLineHeight(20)
							ui.Text(c, tr("添加项目目录，开始持续对话。", "Add a project directory to start a continuing conversation.")).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(muted)
						})
					})
				})
				if add.Clicked() && m.OnCreate != nil {
					m.OnCreate()
				}
			} else {
				ui.Text(c, projectEmptyLabel(m)).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(8)
			}
		}
		for _, p := range visible {
			projectSidebarRow(c.Key(m.ObserverHost+"/"+p.ID), m, p, projectDirectoryLabel(p, visible), colors)
		}
	})
	projectModals(c, m)
}

func projectSidebarRow(c *ui.Context, m Model, p projectflow.Project, directory string, colors theme.Colors) {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	if p.Lifecycle == "archived" {
		fg = chrome.Color(colors.Caption)
		muted = fg
	}
	row := ui.Row(c).FillWidth().MinWidth(0).MinHeight(36).Padding(0, 4, 0, 8).Gap(4).Radius(12).AlignItems(ui.Center)
	selected := p.CoordinatorSessionID != "" && p.CoordinatorSessionID == m.SelectedSession
	if selected {
		row.Background(chrome.Color(colors.Pressed))
	} else if row.Hovered() || row.FocusWithin() {
		row.Background(chrome.Color(colors.Hover))
	}
	menu := func(menu *ui.Menu) { projectRowMenu(menu, m, p) }
	row.ContextMenu(menu)
	row.Children(func() {
		open := ui.Button(c, "").Label(p.Title).Tooltip(p.Title+"\n"+p.CanonicalWorkingDirectory).Grow(1).MinWidth(0).MinHeight(36).Padding(6, 0).BorderWidth(0).Background(ui.Transparent).Disabled(m.Busy || !m.Listing.Available)
		open.Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				folderInk := muted
				if selected {
					folderInk = fg
				}
				projectIcon(c, "folder", 16, folderInk, false)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					title := ui.Text(c, p.Title).FontSize(theme.InterfaceSize(14)).TextColor(fg).MinWidth(0).FillWidth().SingleLine()
					if selected {
						title.FontWeight(500)
					}
					if directory != "" {
						ui.Text(c, directory).FontSize(theme.InterfaceSize(12)).TextColor(muted).MinWidth(0).FillWidth().SingleLine()
					}
				})
			})
		})
		if open.Clicked() && m.OnOpen != nil {
			m.OnOpen(p.ID)
		}
		status := projectSidebarStatus(p)
		showMenu := row.Hovered() || row.FocusWithin()
		if status != "" && !showMenu {
			ink := muted
			if p.Lifecycle != "archived" {
				switch p.Activity.State {
				case "running", "arranging", "active", "queued":
					ink = chrome.Color(colors.Accent)
				case "blocked", "paused", "summaryFailed", "pendingSummary", "needsReview":
					ink = ui.RGB(221, 134, 41)
				}
			}
			ui.Row(c).Gap(4).MaxWidth(96).MinWidth(0).AlignItems(ui.Center).Tooltip(fmt.Sprintf("%s · %d %s · %d %s", stateLabel(p.Activity.State), p.Activity.Running, tr("进行中", "running"), p.Activity.Queued, tr("排队中", "queued"))).Children(func() {
				ui.Box(c).Size(6, 6).Shrink(0).Radius(3).Background(ink)
				ui.Text(c, status).FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(ink).MinWidth(0).SingleLine()
			})
		}
		more := projectIconButton(c, "project-row-menu", tr("更多：", "More: ")+p.Title, "more", muted).Disabled(m.Busy)
		if m.presentation != nil {
			handle := m.presentation.Rows[p.ID]
			if handle == nil {
				handle = &ui.Handle{}
				m.presentation.Rows[p.ID] = handle
			}
			more.Bind(handle)
		}
		more.Menu(menu)
		if !showMenu && !more.Focused() {
			more.Opacity(0).Width(0)
		}
	})
}

func projectRowMenu(menu *ui.Menu, m Model, p projectflow.Project) {
	markOrigin := func() {
		if m.presentation != nil {
			m.presentation.Origin = p.ID
		}
	}
	blocked := m.Busy || !m.Listing.Available
	if menu.Item(tr("重命名", "Rename")).Disabled(blocked).Chosen() && m.OnBeginRename != nil {
		markOrigin()
		m.OnBeginRename(p)
	}
	if menu.Item(tr("打开项目目录", "Open project directory")).Disabled(blocked || m.OnProjectDirectoryFor == nil).Chosen() && m.OnProjectDirectoryFor != nil {
		m.OnProjectDirectoryFor(p.ID)
	}
	if p.Lifecycle == "archived" {
		if menu.Item(tr("恢复项目", "Restore Project")).Disabled(blocked || p.Archiving).Chosen() && m.OnRestoreProject != nil {
			m.OnRestoreProject(p.ID)
		}
	} else if menu.Item(tr("归档项目", "Archive Project")).Disabled(blocked || p.Archiving || p.Lifecycle != "ready").Chosen() && m.OnArchiveProject != nil {
		markOrigin()
		m.OnArchiveProject(p.ID)
	}
}

func projectSidebarStatus(p projectflow.Project) string {
	if p.Lifecycle == "archived" {
		return tr("已归档", "Archived")
	}
	if p.Activity.State == "" || p.Activity.State == "idle" || p.Activity.State == "done" {
		return ""
	}
	return stateLabel(p.Activity.State)
}

func projectDirectoryLabel(p projectflow.Project, projects []projectflow.Project) string {
	var peers []projectflow.Project
	for _, other := range projects {
		if other.ID != p.ID && other.Title == p.Title {
			peers = append(peers, other)
		}
	}
	if len(peers) == 0 {
		return ""
	}
	parts := func(path string) []string {
		return strings.FieldsFunc(strings.ReplaceAll(path, "\\", "/"), func(r rune) bool { return r == '/' })
	}
	path := parts(p.CanonicalWorkingDirectory)
	for count := 1; count < len(path); count++ {
		label := strings.Join(path[len(path)-count:], "/")
		unique := true
		for _, other := range peers {
			peer := parts(other.CanonicalWorkingDirectory)
			start := max(0, len(peer)-count)
			if strings.Join(peer[start:], "/") == label {
				unique = false
				break
			}
		}
		if unique {
			return "…/" + label
		}
	}
	return p.CanonicalWorkingDirectory
}

func projectIconButton(c *ui.Context, key, label, icon string, ink ui.Color) ui.Element {
	button := ui.Button(c.Key(key), "").Label(label).Tooltip(label).Width(28).Height(28).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent)
	button.Children(func() { projectIcon(c, icon, 16, ink, false) })
	return button
}

func projectIcon(c *ui.Context, kind string, size float32, ink ui.Color, expanded bool) {
	ui.Box(c).Width(size).Height(size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		x, y, s := r.X, r.Y, r.W/16
		switch kind {
		case "add":
			path.MoveTo(x+8*s, y+3*s)
			path.LineTo(x+8*s, y+13*s)
			path.MoveTo(x+3*s, y+8*s)
			path.LineTo(x+13*s, y+8*s)
		case "more":
			for _, at := range []float32{3, 8, 13} {
				path.Circle(x+at*s, y+8*s, .8*s)
			}
		case "close":
			path.MoveTo(x+4*s, y+4*s)
			path.LineTo(x+12*s, y+12*s)
			path.MoveTo(x+12*s, y+4*s)
			path.LineTo(x+4*s, y+12*s)
		case "up":
			path.MoveTo(x+4*s, y+10*s)
			path.LineTo(x+8*s, y+6*s)
			path.LineTo(x+12*s, y+10*s)
		case "down":
			path.MoveTo(x+4*s, y+6*s)
			path.LineTo(x+8*s, y+10*s)
			path.LineTo(x+12*s, y+6*s)
		case "chevron":
			if expanded {
				path.MoveTo(x+4*s, y+6*s)
				path.LineTo(x+8*s, y+10*s)
				path.LineTo(x+12*s, y+6*s)
			} else {
				path.MoveTo(x+6*s, y+4*s)
				path.LineTo(x+10*s, y+8*s)
				path.LineTo(x+6*s, y+12*s)
			}
		default:
			path.MoveTo(x+2*s, y+4*s)
			path.LineTo(x+6*s, y+4*s)
			path.LineTo(x+8*s, y+6*s)
			path.LineTo(x+14*s, y+6*s)
			path.LineTo(x+14*s, y+13*s)
			path.LineTo(x+2*s, y+13*s)
			path.Close()
		}
		p.StrokePath(path, 1.2, ink)
	})
}
