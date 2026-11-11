package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/paths"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

type projectFlowState struct {
	projectTeams                  map[string]*projectTeamState
	observers                     map[*projects.ObserverState]*projectController
	observerSection               string
	observerOpen                  bool
	materialCollection            *projectMaterialCollectionState
	teamQuery                     string
	teamHistory                   bool
	loadAttempted                 bool
	pendingCreate                 string
	createAdoption                *workspaceAdoption
	directoryIntent               *projectDirectoryIntent
	directoryChecks               map[string]*projectDirectoryCheck
	createError                   string
	ordinaryTeam                  *ordinaryTeamState
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
		var ordinaryTeam *ordinaryTeamState
		if f.ordinaryTeam != nil && f.ordinaryTeam.client == s.dshClient {
			ordinaryTeam = f.ordinaryTeam
		}
		*f = projectFlowState{client: s.dshClient, expanded: expanded, generation: generation, ordinaryTeam: ordinaryTeam}
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
	initialLoad := f.client != nil && !f.loadAttempted && !f.busy
	listing := f.listing
	listing.Projects = append([]projectflow.Project(nil), listing.Projects...)
	m := projects.Model{ArchiveCandidate: f.archiveCandidate, LifecycleNote: f.lifecycleNote, AssetDiff: f.assetDiff, AssetNote: f.assetNote, RequestProjectID: f.requestProjectID, RequestKind: f.requestKind, RequestPage: f.requestPage, RequestCard: f.requestCard, RequestPageNumber: len(f.requestCursors), RequestLoaded: f.requestLoaded, Detail: f.detail, Process: f.process, TeamID: f.teamID, Team: f.team, TeamNote: f.teamNote, Expanded: f.expanded, Busy: f.busy, Loaded: f.loaded, Listing: listing, Error: f.err, Workspace: workspace, RenameID: f.renameID, RenameDraft: &f.renameDraft}
	// Ephemeral authenticated Host identity; no credentials enter pane persistence.
	m.ObserverHost = fmt.Sprintf("%p", f.client)
	m.RequestRevertError = f.requestRevertError
	m.TeamQuery = f.teamQuery
	m.TeamHistory = f.teamHistory
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
	if initialLoad {
		s.runProjectAction(win, "list", "", "")
	}
	if sessions != nil {
		m.SelectedSession = sessions.SelectedSessionID()
		snapshot := sessions.RenderSnapshot()
		m.PlanSessionID, m.PlanBoard = snapshot.SelectedID, snapshot.PlanBoard
		if s.projectFlow == &s.mainState.projectFlow {
			s.bindProjectHero(win, &m, sessions, filterClient, snapshot)
		}
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
	m.OnCreate = func() { s.mainState.createProjectIntent(win, workspace, false, false) }
	m.OnOpen = func(id string) {
		s.mu.Lock()
		s.projectFlow.expanded = true
		s.mu.Unlock()
		s.runProjectAction(win, "open", id, "")
	}
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
	m.OnTeamHistory = func(history bool) {
		s.mu.Lock()
		if s.projectFlow.client == filterClient && s.dshClient == filterClient {
			s.projectFlow.teamHistory = history
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
	m.OnStopProject = func(id string) { s.runProjectStop(win, id) }
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
		card := m.RequestCard
		s.mu.Lock()
		if saved, ok := s.projectFlow.requestCards[projectRequestCardKey(m.RequestProjectID, id)]; ok {
			copy := saved
			card = &copy
		}
		s.mu.Unlock()
		change, err := projectQueueChange(m.RequestPage, card, m.RequestQueue, id, action)
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
	m.OnArtifactTarget = func(file projectflow.Artifact, open func(projectflow.FileTarget)) {
		s.runProjectAssetForCard(win, m.RequestProjectID, "file", file, "", nil, nil, nil, open)
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
		if action == "create" {
			f.directoryIntent, f.createAdoption = nil, nil
		}
		f.err = i18n.T(i18n.KeyCmdHostIsNotConnectedRefreshWhenReady)
		s.mu.Unlock()
		return
	}
	if action != "list" && !f.listing.Available {
		if action == "create" {
			f.directoryIntent, f.createAdoption = nil, nil
		}
		f.err = i18n.T(i18n.KeyCmdProjectHostIsUnavailable)
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
	if action == "list" {
		f.loadAttempted = true
	}
	f.err = ""
	f.generation++
	generation := f.generation
	navigation := f.navigation
	sessions := s.sessions
	adoption := f.createAdoption
	if action == "create" {
		f.createAdoption = nil
		f.createError = ""
	}
	s.mu.Unlock()
	selected := ""
	authorized := []string{workspace}
	if sessions != nil {
		selected = sessions.SelectedSessionID()
		if choices, ready := sessions.CachedWorkspaceChoices(client); ready {
			for _, choice := range choices {
				if choice.Path != "" {
					authorized = append(authorized, choice.Path)
				}
			}
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var listing projectflow.Listing
		var err error
		var lead string
		var changed bool
		var acceptedList bool
		var receipt projectflow.Project
		switch action {
		case "list":
			listing, err = rpc.List(ctx)
			acceptedList = err == nil
		case "create":
			var token [16]byte
			_, err = rand.Read(token[:])
			if err == nil {
				var p projectflow.Project
				p, err = rpc.Create(ctx, arg, authorized, hex.EncodeToString(token[:]))
				if err == nil {
					changed = true
					receipt = p
					lead, err = rpc.Open(ctx, p.ID)
				}
			}
		case "rename":
			receipt, err = rpc.Rename(ctx, arg, title)
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
			acceptedList = refreshErr == nil
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
			if action == "create" {
				f.directoryIntent = nil
			}
			if acceptedList {
				f.loaded = true
				f.listing = listing
			} else if changed && receipt.ID != "" {
				// Keep the accepted mutation even if its subsequent read fails.
				found := false
				for i := range f.listing.Projects {
					if f.listing.Projects[i].ID == receipt.ID {
						f.listing.Projects[i] = receipt
						found = true
						break
					}
				}
				if !found {
					f.listing.Projects = append(f.listing.Projects, receipt)
				}
			}
			if err != nil {
				f.err = err.Error()
			} else {
				f.err = listing.Error
			}
			if action == "rename" && changed {
				f.renameID, f.renameDraft = "", ""
			}
			stillOpen := f.visible() && f.navigation == navigation
			if action == "create" && lead != "" && stillOpen && adoption != nil && adoption.carry && sessions != nil && sessions.SelectedSessionID() == selected {
				if s.composerDrafts[adoption.sourceID] == adoption.draft && workspaceSessionDraftRevision(adoption.draft) == adoption.revision && bytes.Equal(adoption.draft.SaveDraft(), adoption.snapshot) {
					if lead == adoption.sourceID || !workspaceSessionDraftOccupied(s.composerDrafts[lead]) {
						s.composerDrafts[lead] = adoption.draft
						adoption.draft.TargetID = lead
						if lead != adoption.sourceID {
							delete(s.composerDrafts, adoption.sourceID)
						}
					} else {
						f.err = "Lead 已有未发送草稿；原草稿已保留，未覆盖。"
					}
				}
			}
			if action == "create" {
				f.createError = f.err
			}
			pendingCreate := ""
			if action == "list" && f.pendingCreate != "" {
				if acceptedList && listing.Available {
					pendingCreate = f.pendingCreate
				} else {
					f.directoryIntent, f.createAdoption = nil, nil
				}
				f.pendingCreate = ""
			}
			s.mu.Unlock()
			if pendingCreate != "" {
				s.runProjectAction(win, "create", pendingCreate, "")
			}
			if lead != "" && stillOpen && sessions != nil && sessions.SelectedSessionID() == selected {
				sessions.RefreshList(context.Background())
				sessions.Select(lead)
			}
		})
	}()
}

// createProjectFromDirectory is consumed by both the registered workspace row
// menu and the new-conversation directory actions. RPC admission rechecks Host
// membership. A click on a registered directory is already an explicit native
// choice; global Add Project still presents the native directory dialog.
func (s *mainState) createProjectFromDirectory(win *mygo.Window, path string) {
	s.createProjectIntent(win, path, false, true)
}
func (s *mainState) createProjectWithDraft(win *mygo.Window, path string) {
	s.createProjectIntent(win, path, true, true)
}
func (s *mainState) createProjectIntent(win *mygo.Window, path string, carry, registered bool) {
	controller := s.primaryProjectController()
	controller.projectModel(win)
	s.mu.Lock()
	sessions := s.sessions
	s.mu.Unlock()
	var adoption *workspaceAdoption
	if carry && sessions != nil {
		value := s.captureWorkspaceAdoption(sessions)
		adoption = &value
	}
	s.mu.Lock()
	f := controller.projectFlow
	if f.busy && f.loaded {
		s.mu.Unlock()
		return
	}
	f.expanded = true
	f.createAdoption = adoption
	f.directoryIntent = nil
	if registered {
		f.directoryIntent = &projectDirectoryIntent{client: f.client, path: path}
	}
	if !f.loaded {
		f.pendingCreate = path
		busy := f.busy
		s.mu.Unlock()
		if !busy {
			controller.runProjectAction(win, "list", "", "")
		}
		return
	}
	s.mu.Unlock()
	controller.runProjectAction(win, "create", path, "")
}
