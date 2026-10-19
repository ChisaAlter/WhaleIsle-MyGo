// Package projects draws the native Project entry beside workspace sessions.
package projects

import (
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

type Model struct {
	RequestRevert                                                        func(string, int) RequestRevertState
	VisualPage                                                           func(string, projectflow.DeliveryPageReference, int) VisualPagePreview
	modalBody                                                            bool
	presentation                                                         *projectPresentation
	ThemeID                                                              string
	SystemDark                                                           bool
	OnProjectDirectoryFor                                                func(string)
	ObserverModel                                                        func(*ObserverState) Model
	ObserverHost                                                         string
	OnObserverSection                                                    func(string, string)
	OnObserver                                                           func(string)
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
	OnRequestDiff      func(string)
	OnRevertRequest    func(string, int)
	OnRevertAskDraft   func()
	RequestRevertError string
	OnProjectDirectory func()
	OnCloseDiff        func()

	RequestProjectID, RequestKind   string
	RequestPage                     projectflow.RequestPage
	RequestCard                     *projectflow.RequestCard
	RequestPageNumber               int
	RequestLoaded, RequestMutable   bool
	RequestQueue                    *projectflow.RequestQueue
	OnRequestQueue                  func(string, string)
	OnRequestWorkDetail             func(string, string)
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
