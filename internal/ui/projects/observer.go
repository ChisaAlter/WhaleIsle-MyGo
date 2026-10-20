package projects

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
)

// ObserverState is retained by the coordinator's resource pane, including while
// its floating presentation is a strip. Data and commands belong to this observer’s independent project controller.
// Host is intentionally runtime-only: restoring layout cannot revive authority.
type ObserverState struct {
	ProjectID            string                 `json:"projectId"`
	CoordinatorSessionID string                 `json:"coordinatorSessionId"`
	Host                 string                 `json:"-"`
	Section              string                 `json:"section"`
	PendingSection       string                 `json:"-"`
	Initialized          bool                   `json:"-"`
	RequestModes         map[string]RequestMode `json:"requestModes,omitempty"`
	requests             map[string]projectflow.RequestCard
	requested            map[string]string
	reading              string
}

type RequestMode struct {
	Mode      string `json:"mode"`
	Completed bool   `json:"completed"`
	Tab       string `json:"tab,omitempty"`
}

func (o *ObserverState) Project(m Model) (projectflow.Project, bool) {
	if o == nil || o.Host == "" || o.Host != m.ObserverHost || !m.Listing.Available {
		return projectflow.Project{}, false
	}
	for _, p := range m.Listing.Projects {
		if p.ID == o.ProjectID && p.CoordinatorSessionID == o.CoordinatorSessionID && p.CoordinatorSessionID == m.SelectedSession {
			return p, true
		}
	}
	return projectflow.Project{}, false
}

func (o *ObserverState) navigate(m Model, section string) bool {
	if m.Busy {
		return false
	}
	if o.Section == "materials" && section != "materials" && m.Material != nil && m.Material.ProjectID == o.ProjectID && m.Material.Dirty() {
		o.PendingSection = section
		return false
	}
	o.Section, o.PendingSection = section, ""
	if m.OnObserverSection != nil {
		m.OnObserverSection(o.ProjectID, section)
	}
	switch section {
	case "tasks":
		if m.RequestProjectID != o.ProjectID && m.OnRequests != nil {
			m.OnRequests(o.ProjectID)
		}
	case "team":
		if m.TeamID != o.ProjectID && m.OnTeam != nil {
			m.OnTeam(o.ProjectID)
		}
	case "materials":
		if (m.MaterialCollection == nil || m.MaterialCollection.ProjectID != o.ProjectID) && m.OnMaterialCollection != nil {
			m.OnMaterialCollection(o.ProjectID)
		}
	}
	return true
}

// ObserverView renders task strips/cards in the retained resource pane. Host
// data and commands still belong to its independent project controller.
func ObserverView(c *ui.Context, m Model, o *ObserverState) {
	p, ok := o.Project(m)
	if !ok {
		ui.Text(c, tr("此项目概览当前不可用。请从原项目会话重新打开。", "Project overview is unavailable. Reopen it from its Project conversation."))
		return
	}
	if o.Section == "" {
		o.Section = "tasks"
	}
	if !o.Initialized && !m.Busy {
		o.Initialized = o.navigate(m, o.Section)
	}
	// A report can open its original member process; retain the same Project route.
	originalReportWork := m.OnReportWork
	if originalReportWork != nil {
		m.OnReportWork = func() {
			o.Section = "team"
			if m.OnObserverSection != nil {
				m.OnObserverSection(o.ProjectID, "team")
			}
			originalReportWork()
		}
	}
	root := ui.Column(c).Fill().MinWidth(0).MinHeight(0).Gap(5)
	m.presentation = ui.Local(root, "observer-presentation", newProjectPresentation)
	overlay := m
	hasChild := m.Historical != nil || m.AssetDiff != nil || m.Detail != nil || m.Documents != nil || m.Deliveries != nil || m.Reports != nil || m.Material != nil || m.Activity != nil || m.RequestCard != nil
	inline := m
	inline.Detail = nil
	inline.AssetDiff = nil
	inline.RequestCard = nil
	if m.RequestCard != nil && m.RequestCard.ProjectID == p.ID {
		if o.requests == nil {
			o.requests = map[string]projectflow.RequestCard{}
		}
		o.requests[m.RequestCard.ID] = *m.RequestCard
		if !m.Busy && o.reading == m.RequestCard.ID {
			o.reading = ""
		}
	}
	if !m.Busy && m.Error != "" {
		o.reading = ""
	}
	// Full task cards stay in the observer. Their process/report/file subviews
	// continue to use the existing independent modal routes.
	if m.RequestCard != nil {
		if _, exists := o.RequestModes[m.RequestCard.ID]; exists {
			overlay.RequestCard = nil
			hasChild = m.Historical != nil || m.AssetDiff != nil || m.Detail != nil || m.Documents != nil || m.Deliveries != nil || m.Reports != nil || m.Material != nil || m.Activity != nil
		}
	}
	root.Children(func() {
		colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
		ui.Column(c).FillWidth().MinWidth(0).Shrink(0).Gap(2).Padding(0, 4).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
				projectIcon(c, "folder", 14, chrome.Color(colors.Muted), false)
				ui.Text(c, p.Title).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).FontWeight(600).SingleLine()
				if status := projectStateLabel(p); status != "" {
					ui.Text(c, status).FontSize(fonttheme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted))
				}
			})
			if p.CanonicalWorkingDirectory != "" {
				ui.Text(c, p.CanonicalWorkingDirectory).FillWidth().MinWidth(0).SingleLine().Ellipsis("…").Tooltip(p.CanonicalWorkingDirectory).
					Font("monospace").FontSize(fonttheme.InterfaceSize(10)).TextColor(chrome.Color(colors.Muted))
			}
			ui.Row(c).FillWidth().Wrap().Gap(4).Children(func() {
				for _, section := range []struct{ id, zh, en string }{{"tasks", "任务与要求", "Tasks / requests"}, {"team", "成员", "Team"}, {"materials", "材料", "Materials"}} {
					label := tr(section.zh, section.en)
					active := o.Section == section.id
					button := ui.Button(c.Key("observer-section-"+section.id), label).Role(ui.RoleTab).Height(26).Radius(6).Padding(0, 8).FontSize(fonttheme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).Selected(active).Disabled(m.Busy)
					if active {
						button.Background(c.Theme().SurfaceHover)
					}
					if button.Clicked() {
						o.navigate(m, section.id)
					}
				}
			})
		})
		if m.Error != "" {
			ui.Text(c, m.Error).Selectable()
		}
		if m.Busy {
			ui.Text(c, tr("正在读取项目…", "Loading Project…"))
		}
		if o.PendingSection != "" {
			ui.Text(c, tr("请先保存或放弃材料草稿，再切换。", "Save or discard the material draft before switching."))
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
				if ui.Button(c, tr("继续编辑", "Keep editing")).Clicked() {
					o.PendingSection = ""
				}
				if ui.Button(c, tr("保存", "Save")).Disabled(m.Busy).Clicked() && m.OnSaveMaterial != nil {
					m.OnSaveMaterial()
				}
				if ui.Button(c, tr("放弃并切换", "Discard and switch")).Disabled(m.Busy).Clicked() && m.OnDiscardMaterial != nil {
					m.OnDiscardMaterial()
					o.navigate(m, o.PendingSection)
				}
				if ui.Button(c, tr("继续切换", "Continue switching")).Disabled(m.Busy || m.Material != nil && m.Material.Dirty()).Clicked() {
					o.navigate(m, o.PendingSection)
				}
			})
		}
		ui.Scroll(c).FillWidth().MinHeight(0).Grow(1).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Padding(0, 4, 4, 4).Children(func() {
				if m.RequestProjectID == p.ID {
					observerRequests(c, m, o, p)
				}
				switch o.Section {
				case "team":
					if m.TeamID == p.ID && (m.Team == nil || m.Team.Project.ID == p.ID) && (m.Detail == nil || m.Detail.Workstream.ProjectID == p.ID) {
						teamView(c, inline)
					} else {
						observerLoad(c, m, func() {
							if m.OnTeam != nil {
								m.OnTeam(p.ID)
							}
						})
					}
				case "materials":
					switch {
					case m.MaterialCollection != nil && m.MaterialCollection.ProjectID == p.ID:
						materialCollectionView(c, m)
					default:
						observerLoad(c, m, func() {
							if m.OnMaterialCollection != nil {
								m.OnMaterialCollection(p.ID)
							}
						})
					}
				default:
					if m.RequestProjectID != p.ID {
						observerLoad(c, m, func() {
							if m.OnRequests != nil {
								m.OnRequests(p.ID)
							}
						})
					}
				}
			})
		})
	})
	if !hasChild {
		overlay.TeamID = ""
		overlay.RequestProjectID = ""
		overlay.MaterialCollection = nil
	}
	projectModals(c, overlay)
}
func observerLoad(c *ui.Context, m Model, load func()) {
	if ui.Button(c, tr("读取此项目内容", "Load this Project")).Disabled(m.Busy).Clicked() {
		load()
	}
}

// projectStateLabel is the observer header's one-word status: paused wins, then
// the coordinator's activity, then the lifecycle stage.
func projectStateLabel(p projectflow.Project) string {
	switch {
	case p.Paused:
		return tr("已暂停", "Paused")
	case p.Archiving:
		return tr("归档中", "Archiving")
	case p.Activity.Running > 0:
		return tr("运行中", "Running")
	case p.Activity.Queued > 0:
		return tr("排队中", "Queued")
	case p.Lifecycle != "":
		return p.Lifecycle
	}
	return ""
}