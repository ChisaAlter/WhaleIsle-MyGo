package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/paths"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

type projectFlowState struct {
	observers                     map[*projects.ObserverState]*projectController
	observerSection               string
	observerOpen                  bool
	materialCollection            *projectMaterialCollectionState
	teamQuery                     string
	searchQuery                   string
	history                       bool
	historical                    *projectHistoricalState
	historicalGeneration          uint64
	activity                      *projectActivityState
	deliveries                    *projectDeliveryState
	documents                     *projectDocumentState
	refreshLoop                   *projectRefreshLoop
	refreshNote                   string
	processCancel                 context.CancelFunc
	processMux                    *dsh.Mux
	processWatchGeneration        uint64
	reports                       *projectflow.ReportBrowser
	materials                     map[string]*projectflow.MaterialDraft
	materialKey                   string
	archiveCandidate              *projectflow.Project
	lifecycleNote                 string
	assetDiff                     *projectflow.WorkDiff
	assetNote                     string
	requestRevertError            string
	visualResults                 *projectVisualResultsState
	requestProjectID, requestKind string
	requestPage                   projectflow.RequestPage
	requestCard                   *projectflow.RequestCard
	requestCards                  map[string]projectflow.RequestCard
	requestReverts                *projectRequestRevertState
	requestCursors                []string
	requestLoaded                 bool
	requestGeneration             uint64

	detail                     *projectflow.WorkDetail
	process                    *projectflow.ProcessSnapshot
	detailGeneration           uint64
	teamID, teamNote           string
	team                       *projectflow.Overview
	navigation                 uint64
	client                     *dsh.Client
	listing                    projectflow.Listing
	expanded, busy, loaded     bool
	err, renameID, renameDraft string
	generation                 uint64
}

func (f *projectFlowState) visible() bool { return f.expanded || f.observerOpen }

func (s *projectController) projectModel(win *mygo.Window) projects.Model {
	s.mu.Lock()
	f := s.projectFlow
	if f.client != s.dshClient {
		for _, observer := range f.observers {
			observer.disposeLocked()
		}
		s.closeProjectMaterialCollectionLocked()
		s.closeProjectHistoricalLocked()
		s.closeProjectActivityLocked()
		s.closeProjectDeliveriesLocked()
		s.closeProjectDocumentsLocked()
		s.closeProjectVisualResultsLocked()
		s.stopProjectRefreshLocked()
		s.stopProjectProcessLocked()
		expanded := f.expanded
		generation := f.generation + 1
		*f = projectFlowState{client: s.dshClient, expanded: expanded, generation: generation}
	}
	workspace := ""
	if s.cfgStore != nil {
		workspace = s.cfgStore.Get().Workspace
	}
	if workspace == "" {
		workspace = paths.DefaultWorkspace()
	}
	s.syncProjectProcessLocked()
	s.ensureProjectRefreshLocked(win)
	listing := f.listing
	listing.Projects = append([]projectflow.Project(nil), listing.Projects...)
	m := projects.Model{ArchiveCandidate: f.archiveCandidate, LifecycleNote: f.lifecycleNote, AssetDiff: f.assetDiff, AssetNote: f.assetNote, RequestProjectID: f.requestProjectID, RequestKind: f.requestKind, RequestPage: f.requestPage, RequestCard: f.requestCard, RequestPageNumber: len(f.requestCursors), RequestLoaded: f.requestLoaded, Detail: f.detail, Process: f.process, TeamID: f.teamID, Team: f.team, TeamNote: f.teamNote, Expanded: f.expanded, Busy: f.busy, Loaded: f.loaded, Listing: listing, Error: f.err, Workspace: workspace, RenameID: f.renameID, RenameDraft: &f.renameDraft}
	// Ephemeral authenticated Host identity; no credentials enter pane persistence.
	m.ObserverHost = fmt.Sprintf("%p", f.client)
	m.RequestRevertError = f.requestRevertError
	m.TeamQuery = f.teamQuery
	m.SearchQuery = f.searchQuery
	m.History = f.history
	filterClient := f.client
	queueOwner := projectQueueSubmission{Client: f.client, Generation: f.generation, Navigation: f.navigation, RequestGeneration: f.requestGeneration}
	for _, p := range listing.Projects {
		if p.ID == f.requestProjectID {
			m.RequestMutable = p.Lifecycle == "ready" && !p.Archiving
			m.RequestQueue = p.Queue
		}
	}
	observerTeam := f.observerOpen && f.observerSection == "team"
	m.ProcessWatching = f.processCancel != nil
	m.RefreshNote = projectRefreshStatus(f.refreshLoop != nil, f.refreshNote)
	sessions := s.sessions
	s.mu.Unlock()
	if sessions != nil {
		m.SelectedSession = sessions.Snapshot().SelectedID
	}
	m.OnToggle = func() {
		s.mu.Lock()
		s.projectFlow.expanded = !s.projectFlow.expanded
		if !s.projectFlow.visible() {
			s.closeProjectMaterialCollectionLocked()
			s.closeProjectHistoricalLocked()
			s.closeProjectActivityLocked()
			s.closeProjectDeliveriesLocked()
			s.closeProjectDocumentsLocked()
			s.closeProjectVisualResultsLocked()
			s.stopProjectRefreshLocked()
			s.stopProjectProcessLocked()
		}
		if !s.projectFlow.observerOpen {
			s.projectFlow.navigation++
		}
		load := s.projectFlow.expanded && !s.projectFlow.loaded
		s.mu.Unlock()
		if load {
			s.runProjectAction(win, "list", "", "")
		}
	}
	m.OnSearchQuery = func(query string) {
		s.dshMu.Lock()
		client := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		if client == filterClient && s.projectFlow.client == filterClient {
			s.projectFlow.searchQuery = query
		}
		s.mu.Unlock()
	}
	m.OnHistory = func(history bool) {
		s.dshMu.Lock()
		client := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		if client == filterClient && s.projectFlow.client == filterClient {
			s.projectFlow.history = history
		}
		s.mu.Unlock()
	}
	m.OnObserverSection = func(id, section string) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.dshClient == filterClient && s.projectFlow.client == filterClient && s.projectFlow.observerOpen {
			for _, p := range s.projectFlow.listing.Projects {
				if p.ID == id && p.CoordinatorSessionID == m.SelectedSession {
					s.projectFlow.observerSection = section
					return
				}
			}
		}
	}
	m.OnObserver = func(id string) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.dshClient != filterClient || s.projectFlow.client != filterClient {
			return
		}
		for _, p := range s.projectFlow.listing.Projects {
			if p.ID == id && p.CoordinatorSessionID == m.SelectedSession {
				s.projectFlow.observerOpen = true
				return
			}
		}
	}
	m.OnRefresh = func() { s.runProjectAction(win, "list", "", "") }
	m.OnCreate = func() { s.runProjectAction(win, "create", workspace, "") }
	m.OnOpen = func(id string) { s.runProjectAction(win, "open", id, "") }
	m.OnBeginRename = func(p projectflow.Project) {
		s.mu.Lock()
		s.projectFlow.renameID = p.ID
		s.projectFlow.renameDraft = p.Title
		s.mu.Unlock()
	}
	m.OnCancelRename = func() { s.mu.Lock(); s.projectFlow.renameID = ""; s.projectFlow.renameDraft = ""; s.mu.Unlock() }
	m.OnRename = func(id, title string) { s.runProjectAction(win, "rename", id, title) }
	m.OnTeamQuery = func(query string) {
		s.mu.Lock()
		if s.projectFlow.client == filterClient && s.dshClient == filterClient && s.projectFlow.teamID == m.TeamID {
			s.projectFlow.teamQuery = query
		}
		s.mu.Unlock()
	}
	m.OnTeam = func(id string) { s.runProjectTeam(win, id, "inspect", "") }
	m.OnCloseTeam = func() {
		s.mu.Lock()
		s.stopProjectProcessLocked()
		s.projectFlow.teamID = ""
		s.projectFlow.team = nil
		s.projectFlow.detail = nil
		s.projectFlow.assetDiff = nil
		s.projectFlow.process = nil
		s.projectFlow.detailGeneration++
		s.mu.Unlock()
	}
	m.OnControl = func(action, workID string) { s.runProjectTeam(win, m.TeamID, action, workID) }
	m.OnNextWork = func(cursor string) { s.runProjectTeam(win, m.TeamID, "next", cursor) }
	m.OnPreviousWork = func() { s.runProjectTeam(win, m.TeamID, "previous", "") }
	m.OnInspect = func(workID, memberID string) { s.runProjectDetail(win, m.TeamID, "detail", workID, memberID) }
	m.OnCloseDetail = func() {
		s.mu.Lock()
		s.stopProjectProcessLocked()
		s.projectFlow.detail = nil
		s.projectFlow.assetDiff = nil
		s.projectFlow.process = nil
		s.projectFlow.detailGeneration++
		s.mu.Unlock()
	}
	m.OnStopProcess = func() { s.mu.Lock(); s.stopProjectProcessLocked(); s.mu.Unlock() }
	m.OnProcess = func(earlier bool) {
		if earlier {
			s.runProjectDetail(win, m.TeamID, "earlier", "", "")
			return
		}
		s.watchProjectProcess(win, m.TeamID)
	}
	m.OnRetryDraft = func() { s.runProjectDetail(win, m.TeamID, "retry-draft", "", "") }
	m.OnSummaryRetryDraft = func() { s.runProjectRecoveryDraft(win, m.TeamID, "summary") }
	m.OnHandoverRetryDraft = func() { s.runProjectRecoveryDraft(win, m.RequestProjectID, "handover") }
	m.OnRequests = func(id string) { s.runProjectRequests(win, id, "list", "current") }
	m.OnRequestList = func(kind string) { s.runProjectRequests(win, m.RequestProjectID, "list", kind) }
	m.OnRequestGet = func(id string) { s.runProjectRequests(win, m.RequestProjectID, "get", id) }
	m.OnRequestQueue = func(id, action string) {
		change, err := projectQueueChange(m.RequestPage, m.RequestCard, m.RequestQueue, id, action)
		if err != nil {
			s.mu.Lock()
			if queueOwner.matches(s.projectFlow, s.dshClient) {
				s.projectFlow.err = err.Error()
			}
			s.mu.Unlock()
			return
		}
		submission := queueOwner
		submission.Change = change
		s.runProjectRequests(win, m.RequestProjectID, "queue", id, submission)
	}
	m.OnRequestPage = func(direction int) {
		action := "next"
		if direction < 0 {
			action = "previous"
		}
		s.runProjectRequests(win, m.RequestProjectID, action, "")
	}
	m.OnRequestArchive = func(id string, archive bool) {
		action := "restore"
		if archive {
			action = "archive"
		}
		s.runProjectRequests(win, m.RequestProjectID, action, id)
	}
	m.OnBackRequests = func() {
		s.mu.Lock()
		s.projectFlow.requestCard = nil
		s.projectFlow.assetDiff = nil
		s.projectFlow.assetNote = ""
		s.projectFlow.requestGeneration++
		s.mu.Unlock()
	}
	m.OnCloseRequests = func() {
		s.mu.Lock()
		s.closeProjectVisualResultsLocked()
		s.projectFlow.requestProjectID = ""
		s.projectFlow.requestCard = nil
		s.projectFlow.assetDiff = nil
		s.projectFlow.assetNote = ""
		s.projectFlow.requestGeneration++
		s.mu.Unlock()
	}
	m.OnArtifact = func(file projectflow.Artifact, open func(string)) {
		s.runProjectAsset(win, m.RequestProjectID, "file", file, "", open)
	}
	m.OnWorkDiff = func(workID string) {
		if (m.RequestProjectID == "" || observerTeam) && m.TeamID != "" && m.Detail != nil {
			s.runProjectDetail(win, m.TeamID, "diff", workID, "")
			return
		}
		s.runProjectAsset(win, m.RequestProjectID, "diff", projectflow.Artifact{}, workID, nil)
	}
	m.OnRequestDiff = func(requestID string) {
		s.runProjectRequestChanges(win, m.RequestProjectID, requestID, queueOwner)
	}
	m.OnRevertRequest = func(requestID string, revision int) {
		s.revertProjectRequest(win, m.RequestProjectID, requestID, revision, queueOwner)
	}
	m.OnRevertAskDraft = func() { s.runProjectRecoveryDraft(win, m.RequestProjectID, "revert") }
	m.OnProjectDirectory = func() { s.runProjectAsset(win, m.RequestProjectID, "directory", projectflow.Artifact{}, "", nil) }
	m.OnProjectDirectoryFor = func(id string) { s.runProjectAsset(win, id, "directory", projectflow.Artifact{}, "", nil, queueOwner) }
	m.OnRequestWorkDetail = func(requestID, workID string) {
		s.runProjectRequestWorkDetail(win, m.RequestProjectID, requestID, workID, queueOwner)
	}
	m.OnCloseDiff = func() { s.mu.Lock(); s.projectFlow.assetDiff = nil; s.mu.Unlock() }
	m.OnArchiveProject = func(id string) { s.runProjectLifecycle(win, id, "prepare") }
	m.OnRestoreProject = func(id string) { s.runProjectLifecycle(win, id, "restore") }
	m.OnConfirmArchive = func() {
		if m.ArchiveCandidate != nil {
			s.runProjectLifecycle(win, m.ArchiveCandidate.ID, "archive")
		}
	}
	m.OnCancelArchive = func() { s.mu.Lock(); s.projectFlow.archiveCandidate = nil; s.mu.Unlock() }
	s.bindProjectHistory(win, &m)
	s.bindProjectMaterials(win, &m)
	s.bindProjectActivity(win, &m)
	s.bindProjectDeliveries(win, &m)
	s.bindProjectDocuments(win, &m)
	s.bindProjectHistorical(win, &m)
	s.bindProjectMaterialCollection(win, &m)
	s.bindProjectRequestReverts(win, &m)
	s.bindProjectRequestCardActions(win, &m)
	return m
}
func (s *projectController) runProjectAction(win *mygo.Window, action, arg, title string) {
	if win == nil {
		return
	}
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	if f.busy {
		s.mu.Unlock()
		return
	}
	if client == nil || f.client != client {
		f.err = resumeText("Host 尚未连接，请刷新。", "Host is not connected; refresh when ready.")
		s.mu.Unlock()
		return
	}
	if action != "list" && !f.listing.Available {
		f.err = resumeText("Project Host 不可用。", "Project Host is unavailable.")
		s.mu.Unlock()
		return
	}
	workspace := ""
	if s.cfgStore != nil {
		workspace = s.cfgStore.Get().Workspace
	}
	if workspace == "" {
		workspace = paths.DefaultWorkspace()
	}
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	navigation := f.navigation
	sessions := s.sessions
	s.mu.Unlock()
	selected := ""
	if sessions != nil {
		selected = sessions.Snapshot().SelectedID
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var listing projectflow.Listing
		var err error
		var lead string
		var changed bool
		switch action {
		case "list":
			listing, err = rpc.List(ctx)
		case "create":
			var token [16]byte
			_, err = rand.Read(token[:])
			if err == nil {
				var p projectflow.Project
				p, err = rpc.Create(ctx, arg, []string{workspace}, hex.EncodeToString(token[:]))
				if err == nil {
					changed = true
					lead, err = rpc.Open(ctx, p.ID)
				}
			}
		case "rename":
			_, err = rpc.Rename(ctx, arg, title)
			changed = err == nil
		case "open":
			lead, err = rpc.Open(ctx, arg)
		default:
			err = fmt.Errorf("unsupported Project action")
		}
		// A successful mutation is followed by authoritative list state. A failed read
		// does not replay the mutation or pretend it did not happen.
		if changed {
			var refreshErr error
			listing, refreshErr = rpc.List(ctx)
			if err == nil {
				err = refreshErr
			}
		}
		win.Update(func() {
			s.mu.Lock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				s.mu.Unlock()
				return
			}
			f.busy = false
			if action == "list" || changed {
				f.loaded = true
				f.listing = listing
			}
			if err != nil {
				f.err = err.Error()
				if action == "list" {
					f.listing.Available = false
					f.listing.Projects = nil
				}
			} else {
				f.err = listing.Error
				if action == "rename" {
					f.renameID = ""
					f.renameDraft = ""
				}
			}
			stillOpen := f.visible() && f.navigation == navigation
			s.mu.Unlock()
			if lead != "" && stillOpen && sessions != nil && sessions.Snapshot().SelectedID == selected {
				sessions.RefreshList(context.Background())
				sessions.Select(lead)
			}
		})
	}()
}
