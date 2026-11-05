package projects

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type projectPresentation struct {
	Rows                            map[string]*ui.Handle
	Scrolls                         map[string]*ui.ScrollState
	Origin, LastRoute               string
	PendingProject, PendingMaterial string
	PendingClose, Saving            bool
	ChildOrigin                     *projectChildOrigin
}

// Child origins are runtime handles owned by the current observer and request
// revision, like the connected activeElement captured by the source overlay.
type projectChildOrigin struct {
	route, host, session string
	request              projectflow.RequestCard
	handle               *ui.Handle
}

func (origin *projectChildOrigin) current(m Model) bool {
	return origin != nil && origin.host == m.ObserverHost && origin.session == m.SelectedSession && taskCardCurrent(m, origin.request)
}

func newProjectPresentation() projectPresentation {
	return projectPresentation{Rows: map[string]*ui.Handle{}, Scrolls: map[string]*ui.ScrollState{}}
}

type projectRoute struct {
	key, title, project string
	close               func()
	body                func(*ui.Context, Model)
}

func currentProjectRoute(m Model) (r projectRoute) {
	switch {
	case m.Historical != nil:
		h := m.Historical
		r = projectRoute{key: "history/" + h.Target.ProjectID + "/" + h.Target.WorkstreamID + "/" + h.Target.DelegationRef, title: tr("成员历史", "Member history"), project: h.Target.ProjectID, close: func() {
			if m.OnHistoricalAction != nil {
				m.OnHistoricalAction("close")
			}
		}, body: historicalProcessView}
	case m.AssetDiff != nil:
		r = projectRoute{key: "diff/" + m.TeamID + "/" + m.RequestProjectID, title: tr("工作区更改", "Workspace changes"), project: m.TeamID, close: m.OnCloseDiff, body: assetDiffView}
		if changes := m.AssetDiff.Requirement; changes != nil {
			r.key = "requirement-diff/" + changes.ProjectID + "/" + changes.RequestID
			r.title = tr("要求的改动", "Requirement changes")
			r.project = changes.ProjectID
		}
		if r.project == "" {
			r.project = m.RequestProjectID
		}
	case m.Documents != nil:
		d := m.Documents
		r = projectRoute{key: "documents/" + d.Scope.ProjectID + "/" + d.SelectedPath, title: tr("项目文档", "Project documents"), project: d.Scope.ProjectID, close: func() {
			if m.OnDocumentAction != nil {
				m.OnDocumentAction("close", "")
			}
		}, body: documentsView}
	case m.Deliveries != nil:
		d := m.Deliveries
		key := "deliveries/" + d.Scope.ProjectID
		if d.Selected != nil {
			key += "/" + d.Selected.ID
		}
		if d.ReportRef != nil {
			key += "/" + d.ReportRef.DelegationRef
		}
		r = projectRoute{key: key, title: tr("历次交付", "Delivery history"), project: d.Scope.ProjectID, close: func() {
			if m.OnDeliveryAction != nil {
				m.OnDeliveryAction("close", "")
			}
		}, body: deliveryHistoryView}
	case m.Reports != nil:
		d := m.Reports
		key := "reports/" + d.ProjectID + "/" + d.WorkID
		if d.Selected != nil {
			key += "/" + d.Selected.DelegationRef
		}
		r = projectRoute{key: key, title: tr("成员报告", "Recorded reports"), project: d.ProjectID, close: m.OnCloseReports, body: reportsView}
	case m.Material != nil:
		d := m.Material
		r = projectRoute{key: "material/" + d.ProjectID + "/" + d.Path, title: tr("项目材料", "Project materials"), project: d.ProjectID, close: m.OnCloseMaterial, body: materialsView}
	case m.Activity != nil:
		d := m.Activity
		r = projectRoute{key: "activity/" + d.Scope.ProjectID + "/" + d.Scope.RequestID, title: tr("要求活动", "Request activity"), project: d.Scope.ProjectID, close: func() {
			if m.OnActivityAction != nil {
				m.OnActivityAction("close", "")
			}
		}, body: requestActivityView}
	case m.RequestProjectID != "":
		key := "requests/" + m.RequestProjectID + "/" + m.RequestKind
		if m.RequestCard != nil {
			key += "/" + m.RequestCard.ID
		}
		r = projectRoute{key: key, title: tr("要求与交付", "Requests and deliveries"), project: m.RequestProjectID, close: m.OnCloseRequests, body: requestsView}
	case m.TeamID != "":
		if m.Detail != nil {
			title := tr("工作详情", "Work details")
			if m.Process != nil {
				title = tr("成员历史", "Member history")
			}
			r = projectRoute{key: "detail/" + m.Detail.Workstream.ProjectID + "/" + m.Detail.Workstream.ID + fmt.Sprint(m.Process != nil), title: title, project: m.Detail.Workstream.ProjectID, close: m.OnCloseDetail, body: detailView}
			break
		}
		r = projectRoute{key: "team/" + m.TeamID, title: tr("成员", "Team"), project: m.TeamID, close: m.OnCloseTeam, body: teamView}
	case m.MaterialCollection != nil:
		d := m.MaterialCollection
		r = projectRoute{key: "materials/" + d.ProjectID, title: tr("项目材料", "Project materials"), project: d.ProjectID, close: func() {
			if m.OnMaterialCollectionAction != nil {
				m.OnMaterialCollectionAction("close")
			}
		}, body: materialCollectionView}
	}
	for _, p := range m.Listing.Projects {
		if p.ID == r.project {
			r.title += " · " + p.Title
			break
		}
	}
	return
}

func projectModals(c *ui.Context, m Model) {
	if m.presentation != nil && m.presentation.ChildOrigin != nil && !m.presentation.ChildOrigin.current(m) {
		m.presentation.ChildOrigin = nil
	}
	route := currentProjectRoute(m)
	if route.body != nil {
		materialRoute := m.Material != nil && strings.HasPrefix(route.key, "material/")
		memory := m.presentation
		if memory == nil {
			state := newProjectPresentation()
			memory = &state
		}
		close := func() {
			if materialRoute && m.Material.ProjectID == route.project && m.Material.Dirty() {
				memory.PendingProject = route.project
				memory.PendingClose = true
				return
			}
			if route.close != nil {
				if materialRoute {
					c.CloseToast("project-material-error")
				}
				route.close()
			}
			if origin := memory.ChildOrigin; origin != nil && origin.route == route.key {
				memory.ChildOrigin = nil
				if origin.current(m) && origin.handle != nil {
					// Bounds queries the committed source control, including if a
					// close callback is delivered between frame builds.
					if bounds := origin.handle.Bounds(c); bounds.W > 0 && bounds.H > 0 {
						origin.handle.RequestFocus(c)
					}
				}
				return
			}
			if origin := memory.Rows[memory.Origin]; origin != nil {
				origin.Focus(c)
			}
		}
		open := true
		colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
		width, height := c.Size()
		ui.DialogBase(c, &open, func(back, panel ui.Element) {
			chrome.DialogReveal(c, back, panel)
			panel.Label(route.title).Width(min(760, max(1, width-32))).MaxHeight(min(height*.8, max(1, height-48))).MinHeight(0).Padding(0, 0, 24, 0).Radius(28).Clip().Background(chrome.Color(colors.Field)).Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .2)).Children(func() {
				projectModalHeader(c, m, route.title, close, materialRoute && m.Busy, memory.LastRoute != route.key)
				if m.Error != "" {
					ui.Text(c, m.Error).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.FG)).Padding(0, 24, 8, 24).Shrink(0)
				}
				if m.Busy {
					ui.Text(c, tr("正在加载…", "Loading…")).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Muted)).Padding(0, 24, 8, 24).Shrink(0)
				}
				scroll := memory.Scrolls[route.key]
				if scroll == nil {
					scroll = &ui.ScrollState{}
					memory.Scrolls[route.key] = scroll
				}
				ui.Scroll(c.Key(route.key)).FillWidth().MinWidth(0).MinHeight(0).TrackScroll(scroll).Padding(0, 24).Children(func() {
					ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
						body := m
						body.modalBody = true
						body.OnCloseMaterial = close
						body.OnCloseTeam = close
						body.OnCloseRequests = close
						body.OnCloseReports = close
						body.OnCloseDetail = close
						body.OnCloseDiff = close
						if materialRoute {
							original := m.OnMaterial
							body.OnMaterialCollection = func(id string) {
								if m.Material.Dirty() {
									memory.PendingProject = id
									memory.PendingMaterial = "docs"
									return
								}
								if m.OnMaterialCollection != nil {
									m.OnMaterialCollection(id)
								}
							}
							body.OnMaterial = func(id, path string) {
								if m.Material.Dirty() && path != m.Material.Path {
									memory.PendingProject = id
									memory.PendingMaterial = path
									return
								}
								if original != nil {
									original(id, path)
								}
							}
						}
						route.body(c, body)
					})
				})
				if materialRoute {
					ui.Column(c).FillWidth().Shrink(0).Padding(12, 24, 0, 24).Children(func() { materialFooter(c, m) })
				}
			})
		})
		memory.LastRoute = route.key
		if !open && !(materialRoute && m.Busy) {
			close()
		}
		materialTransitionDialog(c, m, memory)
	} else if m.presentation != nil {
		m.presentation.LastRoute = ""
	}
	if m.RenameID != "" {
		projectRenameDialog(c, m)
	}
	if m.ArchiveCandidate != nil {
		projectArchiveDialog(c, m)
	}
}

func projectModalHeader(c *ui.Context, m Model, title string, close func(), disabled, autofocus bool) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Padding(22, 14, 12, 24).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(chrome.Color(colors.FG)).Grow(1).MinWidth(0).SingleLine()
		button := projectIconButton(c, "project-modal-close", tr("关闭", "Close"), "close", chrome.Color(colors.Muted)).Disabled(disabled)
		if autofocus {
			button.Focus()
		}
		if button.Clicked() && close != nil {
			close()
		}
	})
}

func materialTransitionDialog(c *ui.Context, m Model, state *projectPresentation) {
	if m.Material == nil || state.PendingProject != m.Material.ProjectID {
		state.PendingClose = false
		state.PendingMaterial = ""
		state.Saving = false
		return
	}
	if !state.PendingClose && state.PendingMaterial == "" {
		return
	}
	finish := func() {
		c.CloseToast("project-material-error")
		path := state.PendingMaterial
		state.PendingMaterial = ""
		state.PendingClose = false
		state.Saving = false
		if path == "docs" {
			if m.OnMaterialCollection != nil {
				m.OnMaterialCollection(m.Material.ProjectID)
			}
		} else if path != "" {
			if m.OnMaterial != nil {
				m.OnMaterial(m.Material.ProjectID, path)
			}
		} else if m.OnCloseMaterial != nil {
			m.OnCloseMaterial()
		}
	}
	if state.Saving && !m.Busy && !m.Material.Dirty() {
		finish()
		return
	}
	open := true
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.DialogBase(c, &open, func(back, panel ui.Element) {
		chrome.DialogReveal(c, back, panel)
		panel.Width(380).MaxWidthPercent(94).Padding(0, 0, 24, 0).Radius(28).
			Background(chrome.Color(colors.Field)).
			Draw(chrome.Glass(chrome.Color(colors.Field), chrome.MaterialPanel, 28)).Children(func() {
			cancel := func() { state.PendingClose = false; state.PendingMaterial = ""; state.Saving = false }
			projectModalHeader(c, m, tr("未保存的更改", "Unsaved changes"), cancel, m.Busy, false)
			ui.Text(c, tr("保存或放弃当前材料的更改后继续。", "Save or discard this material's changes before continuing.")).FontSize(theme.InterfaceSize(14)).Padding(0, 24, 16, 24)
			if m.Material.Error != "" {
				ui.Text(c, m.Material.Error).FontSize(theme.InterfaceSize(12)).Padding(0, 24, 12, 24)
			}
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Padding(0, 24).Gap(8).Children(func() {
				if ui.Button(c, tr("继续编辑", "Keep editing")).Disabled(m.Busy).Clicked() {
					cancel()
				}
				if ui.Button(c, tr("放弃", "Discard")).Disabled(m.Busy).Clicked() && m.OnDiscardMaterial != nil {
					m.OnDiscardMaterial()
					finish()
				}
				if ui.PrimaryButton(c, tr("保存并继续", "Save and continue")).Disabled(m.Busy || !m.MaterialMutable || len(m.Material.Draft) > materialLimit(m.Material.Path)).Clicked() && m.OnSaveMaterial != nil {
					state.Saving = true
					m.OnSaveMaterial()
				}
			})
		})
	})
	if !open && !m.Busy {
		state.PendingClose = false
		state.PendingMaterial = ""
		state.Saving = false
	}
}
