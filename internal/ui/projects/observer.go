package projects

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/coss"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// Requirements retain one reading state while moving between the header and Sidebar.
type ObserverState struct {
	ProjectID            string                             `json:"projectId"`
	CoordinatorSessionID string                             `json:"coordinatorSessionId"`
	Host                 string                             `json:"-"`
	Section              string                             `json:"section"`
	PendingSection       string                             `json:"-"`
	Initialized          bool                               `json:"-"`
	RequestModes         map[string]RequestMode             `json:"requestModes,omitempty"`
	FloatingOpen, Pinned bool                               `json:"-"`
	HistoryKind          string                             `json:"historyKind,omitempty"`
	Scroll               ui.ScrollState                     `json:"-"`
	Attention            projectflow.RequestPage            `json:"-"`
	Pages                map[string]projectflow.RequestPage `json:"-"`
	Cursors              map[string][]string                `json:"-"`
	CatalogLoaded        map[string]bool                    `json:"-"`
	CatalogErrors        map[string]string                  `json:"-"`
	ReadingOrder         []string                           `json:"readingOrder,omitempty"`
	OnLoadCatalog        func(string, int)                  `json:"-"`
	OnPin, OnReturn      func()                             `json:"-"`
	requests             map[string]projectflow.RequestCard
	requested            map[string]string
	reading              string
}
type RequestMode struct {
	Mode      string `json:"mode"`
	Completed bool   `json:"completed"`
	Tab       string `json:"tab,omitempty"`
}

// ObserverReading retains navigation and editor positions without callbacks,
// controllers or live resources. Host projections are refreshed after attach.
type ObserverReading struct {
	Section, HistoryKind string
	RequestModes         map[string]RequestMode
	ReadingOrder         []string
	ScrollX, ScrollY     float32
}

func (o *ObserverState) Reading() ObserverReading {
	modes := make(map[string]RequestMode, len(o.RequestModes))
	for key, mode := range o.RequestModes {
		modes[key] = mode
	}
	return ObserverReading{Section: o.Section, HistoryKind: o.HistoryKind, RequestModes: modes, ReadingOrder: append([]string(nil), o.ReadingOrder...), ScrollX: o.Scroll.X, ScrollY: o.Scroll.Y}
}

func (o *ObserverState) RestoreReading(r ObserverReading) {
	o.Section, o.HistoryKind = r.Section, r.HistoryKind
	o.RequestModes = make(map[string]RequestMode, len(r.RequestModes))
	for key, mode := range r.RequestModes {
		o.RequestModes[key] = mode
	}
	o.ReadingOrder = append([]string(nil), r.ReadingOrder...)
	o.Scroll.X, o.Scroll.Y = r.ScrollX, r.ScrollY
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
	o.Section = section
	if m.OnObserverSection != nil {
		m.OnObserverSection(o.ProjectID, section)
	}
	if m.RequestProjectID != o.ProjectID && m.OnRequests != nil {
		m.OnRequests(o.ProjectID)
	}
	return true
}
func (o *ObserverState) RefreshCatalog() {
	o.CatalogLoaded = map[string]bool{}
	o.CatalogErrors = map[string]string{}
	o.Attention = projectflow.RequestPage{}
}

// DetachReads keeps the reading position and edited content while releasing
// callbacks and making retained Host projections refresh on the next attach.
func (o *ObserverState) DetachReads() {
	o.OnLoadCatalog, o.OnPin, o.OnReturn = nil, nil, nil
	o.Initialized = false
	o.requested = nil
	o.reading = ""
	o.CatalogLoaded = nil
	o.CatalogErrors = nil
}

func ObserverView(c *ui.Context, m Model, o *ObserverState) {
	p, ok := o.Project(m)
	if !ok {
		ui.Text(c, tr("此需求观察器当前不可用，请从原项目会话重新打开。", "Requirements are unavailable. Reopen from the source Project conversation."))
		return
	}
	if o.Section != "members" {
		o.Section = "tasks"
	}
	if o.Pages == nil {
		o.Pages = map[string]projectflow.RequestPage{}
	}
	if o.Cursors == nil {
		o.Cursors = map[string][]string{}
	}
	if o.CatalogLoaded == nil {
		o.CatalogLoaded = map[string]bool{}
	}
	if o.CatalogErrors == nil {
		o.CatalogErrors = map[string]string{}
	}
	if m.RequestProjectID == p.ID && m.RequestLoaded && m.RequestKind != "" {
		o.Pages[m.RequestKind] = m.RequestPage
	}
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
	root := ui.Column(c).Fill().MinWidth(0).MinHeight(0)
	m.presentation = ui.Local(root, "requirements-presentation", newProjectPresentation)
	overlay := m
	if m.RequestCard != nil {
		if _, exists := o.RequestModes[m.RequestCard.ID]; exists {
			overlay.RequestCard = nil
		}
	}
	root.Children(func() {
		colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
		ui.Row(c).FillWidth().MinWidth(0).Padding(10, 12).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, p.Title).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(p.CanonicalWorkingDirectory).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600)
			if pButton(c, tr("任务材料", "Materials"), m.Busy).Clicked() && m.OnMaterialCollection != nil {
				m.OnMaterialCollection(p.ID)
			}
			label := tr("固定观察", "Keep observing")
			action := o.OnPin
			if o.Pinned {
				label = tr("返回浮层", "Return to popover")
				action = o.OnReturn
			}
			if pButton(c, label, action == nil || m.Busy).Clicked() {
				action()
			}
			refreshLabel := tr("刷新需求", "Refresh requirements")
			if coss.Button(c, "↻", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Disabled: m.Busy}).Label(refreshLabel).Tooltip(refreshLabel).Clicked() {
				o.RefreshCatalog()
			}
			moreLabel := tr("更多需求", "More requirements")
			coss.Button(c, "…", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM}).Label(moreLabel).Tooltip(moreLabel).Menu(func(menu *ui.Menu) {
				for _, kind := range []string{"cancelled", "archived"} {
					kind := kind
					name := tr("已取消", "Cancelled")
					if kind == "archived" {
						name = tr("已归档", "Archived")
					}
					if menu.Item(fmt.Sprintf("%s · %d", name, o.Pages[kind].Total)).Chosen() {
						if o.HistoryKind == kind {
							o.HistoryKind = ""
						} else {
							o.HistoryKind = kind
						}
					}
				}
			})
		})
		track := ui.Local(root, "requirements-scroll", func() teamSurfaceScroll { return teamSurfaceScroll{} })
		if track.track.Y != track.applied {
			o.Scroll.Y = track.track.Y
		}
		track.track.Y, track.applied = o.Scroll.Y, o.Scroll.Y
		ui.Scroll(c.Key("requirements-reading")).FillWidth().MinWidth(0).Grow(1).MinHeight(0).TrackScroll(&track.track).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Padding(0, 12, 12, 12).Children(func() { observerRequests(c, m, o, p) })
		})
		if m.Error != "" {
			widgets.Alert(c, widgets.AlertOptions{ID: "project-requirements/" + o.ProjectID, Message: m.Error,
				Action: tr("重试", "Retry"), OnAction: func() { o.RefreshCatalog() }})
		}
		_ = colors
	})
	hasChild := overlay.Historical != nil || overlay.AssetDiff != nil || overlay.Detail != nil || overlay.Documents != nil || overlay.Deliveries != nil || overlay.Reports != nil || overlay.Material != nil || overlay.Activity != nil || overlay.RequestCard != nil
	if !hasChild {
		overlay.TeamID = ""
		overlay.RequestProjectID = ""
		overlay.MaterialCollection = nil
	}
	projectModals(c, overlay)
	if !m.Busy && o.OnLoadCatalog != nil && o.reading == "" {
		for _, kind := range []string{"current", "completed", "cancelled", "archived"} {
			if !o.CatalogLoaded[kind] && o.CatalogErrors[kind] == "" {
				o.OnLoadCatalog(kind, 0)
				break
			}
		}
	}
}
func observerLoad(c *ui.Context, m Model, load func()) {
	if pPrimary(c, tr("读取此项目内容", "Load this Project"), m.Busy).Clicked() {
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
