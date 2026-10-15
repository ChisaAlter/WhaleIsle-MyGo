// Package projects draws the native Project entry beside workspace sessions.
package projects

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

type Model struct {
	MaterialCollection                                                   *projectflow.MaterialCollection
	OnMaterialCollection                                                 func(string)
	OnMaterialCollectionAction                                           func(string)
	TeamQuery                                                            string
	OnTeamQuery                                                          func(string)
	SearchQuery                                                          string
	History                                                              bool
	OnSearchQuery                                                        func(string)
	OnHistory                                                            func(bool)
	Historical                                                           *projectflow.HistoricalSnapshot
	OnHistoricalSource                                                   func(string)
	OnHistoricalAction                                                   func(string)
	Activity                                                             *projectflow.ActivitySnapshot
	OnRequestActivity                                                    func()
	OnActivityAction                                                     func(string, string)
	OnActivityReport                                                     func(string, projectflow.DeliveryReportRef)
	Deliveries                                                           *projectflow.DeliverySnapshot
	OnDeliveries                                                         func(string)
	OnDeliveryAction                                                     func(string, string)
	OnDeliveryReport                                                     func(string, projectflow.DeliveryReportRef)
	Documents                                                            *projectflow.DocumentSnapshot
	OnDocuments                                                          func(string)
	OnDocumentAction                                                     func(string, string)
	RefreshNote                                                          string
	ProcessWatching                                                      bool
	OnStopProcess                                                        func()
	Reports                                                              *projectflow.ReportBrowser
	OnReports                                                            func(string)
	OnReportAction                                                       func(string, string)
	OnReportFile                                                         func(string, func(string))
	OnCloseReports, OnReportWork                                         func()
	Material                                                             *projectflow.MaterialDraft
	MaterialMutable                                                      bool
	OnMaterial                                                           func(string, string)
	OnSaveMaterial, OnReloadMaterial, OnDiscardMaterial, OnCloseMaterial func()
	ArchiveCandidate                                                     *projectflow.Project
	LifecycleNote                                                        string
	OnArchiveProject, OnRestoreProject                                   func(string)
	OnConfirmArchive, OnCancelArchive                                    func()

	AssetNote          string
	AssetDiff          *projectflow.WorkDiff
	OnArtifact         func(projectflow.Artifact, func(string))
	OnResolvedFile     func(string)
	OnWorkDiff         func(string)
	OnProjectDirectory func()
	OnCloseDiff        func()

	RequestProjectID, RequestKind   string
	RequestPage                     projectflow.RequestPage
	RequestCard                     *projectflow.RequestCard
	RequestPageNumber               int
	RequestLoaded, RequestMutable   bool
	OnRequests                      func(string)
	OnRequestList, OnRequestGet     func(string)
	OnRequestPage                   func(int)
	OnRequestArchive                func(string, bool)
	OnCloseRequests, OnBackRequests func()

	Detail                                        *projectflow.WorkDetail
	Process                                       *projectflow.ProcessSnapshot
	OnInspect                                     func(string, string)
	OnCloseDetail, OnRetryDraft                   func()
	OnSummaryRetryDraft, OnHandoverRetryDraft     func()
	OnProcess                                     func(bool)
	TeamID, TeamNote                              string
	Team                                          *projectflow.Overview
	OnTeam                                        func(string)
	OnCloseTeam                                   func()
	OnControl                                     func(string, string)
	OnNextWork                                    func(string)
	OnPreviousWork                                func()
	Expanded, Busy, Loaded                        bool
	Listing                                       projectflow.Listing
	Error, Workspace, SelectedSession, RenameID   string
	RenameDraft                                   *string
	OnToggle, OnRefresh, OnCreate, OnCancelRename func()
	OnOpen                                        func(string)
	OnBeginRename                                 func(projectflow.Project)
	OnRename                                      func(string, string)
}

func tr(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}
func View(c *ui.Context, m Model) {
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		ui.Row(c).FillWidth().Gap(4).Children(func() {
			arrow := "▸ "
			if m.Expanded {
				arrow = "▾ "
			}
			if ui.Button(c, arrow+"Projects").Grow(1).Clicked() && m.OnToggle != nil {
				m.OnToggle()
			}
			if m.Expanded && ui.Button(c, tr("刷新", "Refresh")).Disabled(m.Busy).Clicked() && m.OnRefresh != nil {
				m.OnRefresh()
			}
		})
		if !m.Expanded {
			return
		}
		if m.RefreshNote != "" {
			ui.Text(c, m.RefreshNote).FontSize(11)
		}
		if m.Busy {
			ui.Text(c, tr("正在处理…", "Working…")).FontSize(12)
		}
		if m.Error != "" {
			ui.Text(c, m.Error).FontSize(12)
		}
		if m.Loaded && !m.Listing.Available {
			ui.Text(c, tr("Project 不可用：需要支持 Project 的 Host 和桌面桥。", "Project unavailable: a compatible Host and desktop bridge are required.")).FontSize(12)
		}
		if m.LifecycleNote != "" {
			ui.Text(c, m.LifecycleNote).FontSize(11)
		}
		if m.ArchiveCandidate != nil {
			archiveConfirmation(c, m)
			return
		}
		if m.Historical != nil {
			historicalProcessView(c, m)
			return
		}
		if m.Documents != nil {
			documentsView(c, m)
			return
		}
		if m.Deliveries != nil {
			deliveryHistoryView(c, m)
			return
		}
		if m.Reports != nil {
			reportsView(c, m)
			return
		}
		if m.Material != nil {
			materialsView(c, m)
			return
		}
		if m.Activity != nil {
			requestActivityView(c, m)
			return
		}
		if m.RequestProjectID != "" {
			requestsView(c, m)
			return
		}
		if m.TeamID != "" {
			teamView(c, m)
			return
		}
		if m.MaterialCollection != nil {
			materialCollectionView(c, m)
			return
		}
		projectFilterControls(c, &m)
		visible := filteredProjects(m.Listing.Projects, m.SearchQuery, m.History)
		if m.Workspace != "" {
			ui.Text(c, tr("初始目录：", "Starting directory: ")+m.Workspace).FontSize(11)
		}
		if ui.Button(c, tr("选择目录并授权创建 Project", "Choose and authorize a Project directory")).FillWidth().Disabled(m.Busy || !m.Listing.Available || m.Workspace == "").Clicked() && m.OnCreate != nil {
			m.OnCreate()
		}
		if m.Loaded && m.Listing.Available && len(visible) == 0 {
			ui.Text(c, projectEmptyLabel(m)).FontSize(12)
		}
		if len(visible) == 0 {
			return
		}
		ui.Scroll(c).Height(200).Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				for _, p := range visible {
					p := p
					ui.Column(c).FillWidth().Gap(2).Children(func() {
						title := p.Title
						if p.CoordinatorSessionID == m.SelectedSession {
							title = "● " + title
						}
						ui.Row(c).FillWidth().Gap(4).Children(func() {
							if ui.Button(c, title).Grow(1).Tooltip(p.CanonicalWorkingDirectory).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnOpen != nil {
								m.OnOpen(p.ID)
							}
							if ui.Button(c, tr("要求", "Requests")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnRequests != nil {
								m.OnRequests(p.ID)
							}
							if ui.Button(c, "Team").Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnTeam != nil {
								m.OnTeam(p.ID)
							}
							if ui.Button(c, tr("改名", "Rename")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnBeginRename != nil {
								m.OnBeginRename(p)
							}
						})
						if p.Lifecycle == "archived" {
							if ui.Button(c, tr("恢复 Project", "Restore Project")).Disabled(m.Busy || !m.Listing.Available || p.Archiving).Clicked() && m.OnRestoreProject != nil {
								m.OnRestoreProject(p.ID)
							}
						} else {
							if ui.Button(c, tr("归档 Project", "Archive Project")).Disabled(m.Busy || !m.Listing.Available || p.Archiving || p.Lifecycle != "ready").Clicked() && m.OnArchiveProject != nil {
								m.OnArchiveProject(p.ID)
							}
						}
						ui.Row(c).Gap(4).Children(func() {
							for _, path := range []string{"notes.md", "preferences.md"} {
								if ui.Button(c, path).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnMaterial != nil {
									m.OnMaterial(p.ID, path)
								}
							}
						})
						if ui.Button(c, tr("材料集合", "Materials")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnMaterialCollection != nil {
							m.OnMaterialCollection(p.ID)
						}
						if ui.Button(c, tr("项目资料", "Project documents")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnDocuments != nil {
							m.OnDocuments(p.ID)
						}
						if ui.Button(c, tr("历次交付", "Delivery history")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnDeliveries != nil {
							m.OnDeliveries(p.ID)
						}
						if ui.Button(c, tr("历史报告", "Recorded reports")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnReports != nil {
							m.OnReports(p.ID)
						}
						status := p.Lifecycle
						if p.Activity.State != "" {
							status += " · " + p.Activity.State
						}
						ui.Text(c, status).FontSize(11)
						if m.RenameID == p.ID && m.RenameDraft != nil {
							ui.TextInput(c, m.RenameDraft).FillWidth()
							ui.Row(c).Gap(4).Children(func() {
								if ui.Button(c, tr("保存", "Save")).Disabled(m.Busy).Clicked() && m.OnRename != nil {
									m.OnRename(p.ID, *m.RenameDraft)
								}
								if ui.Button(c, tr("取消", "Cancel")).Disabled(m.Busy).Clicked() && m.OnCancelRename != nil {
									m.OnCancelRename()
								}
							})
						}
					})
				}
			})
		})
	})
}
