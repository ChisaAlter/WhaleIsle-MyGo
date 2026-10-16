package projects

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	fonttheme "whaleisle/internal/ui/theme"
)

// ObserverState is retained by the coordinator's resource pane, including while
// its floating presentation is a strip. Data and commands belong to this observer’s independent project controller.
// Host is intentionally runtime-only: restoring layout cannot revive authority.
type ObserverState struct {
	ProjectID            string `json:"projectId"`
	CoordinatorSessionID string `json:"coordinatorSessionId"`
	Host                 string `json:"-"`
	Section              string `json:"section"`
	PendingSection       string `json:"-"`
	Initialized          bool   `json:"-"`
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

// ObserverView deliberately routes to the existing live native views. It never
// manufactures a coordinator or clones request/material data into a pane snapshot.
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
	ui.Column(c).Fill().Gap(5).Children(func() {
		ui.Text(c, p.Title+" · "+p.Lifecycle+" · "+p.Activity.State).FontSize(fonttheme.InterfaceSize(12))
		ui.Row(c).FillWidth().Gap(4).Children(func() {
			for _, section := range []struct{ id, zh, en string }{{"tasks", "任务与要求", "Tasks / requests"}, {"team", "成员", "Team"}, {"materials", "材料", "Materials"}} {
				label := tr(section.zh, section.en)
				if o.Section == section.id {
					label = "● " + label
				}
				if ui.Button(c, label).Disabled(m.Busy).Clicked() {
					o.navigate(m, section.id)
				}
			}
		})
		if m.Error != "" {
			ui.Text(c, m.Error).Selectable()
		}
		if m.Busy {
			ui.Text(c, tr("正在读取项目…", "Loading Project…"))
		}
		if o.PendingSection != "" {
			ui.Text(c, tr("请先保存或放弃材料草稿，再切换。", "Save or discard the material draft before switching."))
			ui.Row(c).Gap(4).Children(func() {
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
		ui.Scroll(c).Fill().Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				if m.Historical != nil && m.Historical.Target.ProjectID == p.ID {
					historicalProcessView(c, m)
					return
				}
				switch o.Section {
				case "team":
					if m.TeamID == p.ID && (m.Team == nil || m.Team.Project.ID == p.ID) && (m.Detail == nil || m.Detail.Workstream.ProjectID == p.ID) {
						teamView(c, m)
					} else {
						observerLoad(c, m, func() {
							if m.OnTeam != nil {
								m.OnTeam(p.ID)
							}
						})
					}
				case "materials":
					switch {
					case m.Documents != nil && m.Documents.Scope.ProjectID == p.ID:
						documentsView(c, m)
					case m.Deliveries != nil && m.Deliveries.Scope.ProjectID == p.ID:
						deliveryHistoryView(c, m)
					case m.Reports != nil && m.Reports.ProjectID == p.ID:
						reportsView(c, m)
					case m.Material != nil && m.Material.ProjectID == p.ID:
						materialsView(c, m)
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
					if m.Activity != nil && m.Activity.Scope.ProjectID == p.ID {
						requestActivityView(c, m)
					} else if m.RequestProjectID == p.ID && (m.RequestCard == nil || m.RequestCard.ProjectID == p.ID) {
						requestsView(c, m)
					} else {
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
}
func observerLoad(c *ui.Context, m Model, load func()) {
	if ui.Button(c, tr("读取此项目内容", "Load this Project")).Disabled(m.Busy).Clicked() {
		load()
	}
}
