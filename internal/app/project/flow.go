package projectctl

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

// Model builds the controller's projects.Model snapshot and binds interactive callbacks.
func (c *Controller) Model(win any) projects.Model {
	snap := c.mgr.hostSnapshot()
	cfgStore := c.mgr.configStore()
	sessions := c.mgr.sessionsStore()
	lc := c.mgr.lifecycleContext()

	c.mgr.mu.Lock()
	f := &c.flow
	if f.client != snap.Client {
		if c.isPrimary {
			for _, observer := range c.mgr.observers {
				observer.disposeLocked()
			}
			c.mgr.observers = make(map[*projects.ObserverState]*Controller)
			c.mgr.observerReadings = nil
		}
		c.closeMaterialCollectionLocked()
		c.closeHistoricalLocked()
		c.closeActivityLocked()
		c.closeDeliveriesLocked()
		c.closeDocumentsLocked()
		c.closeVisualResultsLocked()
		if f.catalogRead != nil {
			c.finishReadLocked(f.catalogRead)
		}
		c.stopRefreshLocked()
		c.stopProcessLocked()
		expanded := f.expanded
		generation := f.generation + 1
		observerOpen := !c.isPrimary && f.observerOpen
		observerSection := ""
		if !c.isPrimary {
			observerSection = f.observerSection
		}
		*f = flowState{
			client:          snap.Client,
			expanded:        expanded,
			generation:      generation,
			observerOpen:    observerOpen,
			observerSection: observerSection,
		}
	}
	workspace := currentWorkspace(cfgStore)
	c.syncProcessLocked(snap.Mux)
	c.ensureRefreshLocked(win, snap, lc)
	initialLoad := f.client != nil && !f.loadAttempted && !f.busy
	listing := f.listing
	listing.Projects = append([]projectflow.Project(nil), listing.Projects...)
	m := projects.Model{
		ArchiveCandidate:  f.archiveCandidate,
		LifecycleNote:     f.lifecycleNote,
		AssetDiff:         f.assetDiff,
		AssetNote:         f.assetNote,
		RequestProjectID:  f.requestProjectID,
		RequestKind:       f.requestKind,
		RequestPage:       f.requestPage,
		RequestCard:       f.requestCard,
		RequestPageNumber: len(f.requestCursors),
		RequestLoaded:     f.requestLoaded,
		Detail:            f.detail,
		Process:           f.process,
		TeamID:            f.teamID,
		Team:              f.team,
		TeamNote:          f.teamNote,
		Expanded:          f.expanded,
		Busy:              f.busy,
		Loaded:            f.loaded,
		Listing:           listing,
		Error:             f.err,
		Workspace:         workspace,
		RenameID:          f.renameID,
		RenameDraft:       &f.renameDraft,
	}
	// Ephemeral authenticated Host identity; no credentials enter pane persistence.
	m.ObserverHost = fmt.Sprintf("%p", f.client)
	m.RequestRevertError = f.requestRevertError
	m.TeamQuery = f.teamQuery
	m.TeamHistory = f.teamHistory
	m.SearchQuery = f.searchQuery
	m.History = f.history
	filterClient := f.client
	queueOwner := QueueSubmission{
		Client:            f.client,
		Generation:        f.generation,
		Navigation:        f.navigation,
		RequestGeneration: f.requestGeneration,
	}
	for _, p := range listing.Projects {
		if p.ID == f.requestProjectID {
			m.RequestMutable = p.Lifecycle == "ready" && !p.Archiving
			m.RequestQueue = p.Queue
		}
	}
	observerTeam := f.observerOpen && f.observerSection == "team"
	m.ProcessWatching = f.processCancel != nil
	m.RefreshNote = refreshStatus(f.refreshLoop != nil, f.refreshNote)
	c.mgr.mu.Unlock()

	if initialLoad {
		c.runAction(win, "list", "", "")
	}
	if sessions != nil {
		m.SelectedSession = sessions.SelectedSessionID()
		snapshot := sessions.RenderSnapshot()
		m.PlanSessionID, m.PlanBoard = snapshot.SelectedID, snapshot.PlanBoard
		if c.isPrimary {
			c.bindHero(win, &m, sessions, filterClient, snapshot)
		}
	}
	m.OnToggle = func() {
		c.mgr.mu.Lock()
		c.flow.expanded = !c.flow.expanded
		if !c.flow.visible() {
			c.closeMaterialCollectionLocked()
			c.closeHistoricalLocked()
			c.closeActivityLocked()
			c.closeDeliveriesLocked()
			c.closeDocumentsLocked()
			c.closeVisualResultsLocked()
			c.stopRefreshLocked()
			c.stopProcessLocked()
		}
		if !c.flow.observerOpen {
			c.flow.navigation++
		}
		load := c.flow.expanded && !c.flow.loaded
		c.mgr.mu.Unlock()
		if load {
			c.runAction(win, "list", "", "")
		}
	}
	m.OnSearchQuery = func(query string) {
		current := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		if current.Client == filterClient && c.flow.client == filterClient {
			c.flow.searchQuery = query
		}
		c.mgr.mu.Unlock()
	}
	m.OnHistory = func(history bool) {
		current := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		if current.Client == filterClient && c.flow.client == filterClient {
			c.flow.history = history
		}
		c.mgr.mu.Unlock()
	}
	m.OnObserverSection = func(id, section string) {
		current := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		defer c.mgr.mu.Unlock()
		if current.Client == filterClient && c.flow.client == filterClient && c.flow.observerOpen {
			for _, p := range c.flow.listing.Projects {
				if p.ID == id && p.CoordinatorSessionID == m.SelectedSession {
					c.flow.observerSection = section
					return
				}
			}
		}
	}
	m.OnObserver = func(id string) {
		current := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		defer c.mgr.mu.Unlock()
		if current.Client != filterClient || c.flow.client != filterClient {
			return
		}
		for _, p := range c.flow.listing.Projects {
			if p.ID == id && p.CoordinatorSessionID == m.SelectedSession {
				c.flow.observerOpen = true
				return
			}
		}
	}
	m.OnRefresh = func() { c.runAction(win, "list", "", "") }
	m.OnCreate = func() { c.mgr.CreateIntent(win, workspace, false, false) }
	m.OnOpen = func(id string) {
		c.mgr.mu.Lock()
		c.flow.expanded = true
		c.mgr.mu.Unlock()
		c.runAction(win, "open", id, "")
	}
	m.OnBeginRename = func(p projectflow.Project) {
		c.mgr.mu.Lock()
		c.flow.renameID = p.ID
		c.flow.renameDraft = p.Title
		c.mgr.mu.Unlock()
	}
	m.OnCancelRename = func() {
		c.mgr.mu.Lock()
		c.flow.renameID = ""
		c.flow.renameDraft = ""
		c.mgr.mu.Unlock()
	}
	m.OnRename = func(id, title string) { c.runAction(win, "rename", id, title) }
	m.OnTeamQuery = func(query string) {
		current := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		if c.flow.client == filterClient && current.Client == filterClient && c.flow.teamID == m.TeamID {
			c.flow.teamQuery = query
		}
		c.mgr.mu.Unlock()
	}
	m.OnTeamHistory = func(history bool) {
		current := c.mgr.hostSnapshot()
		c.mgr.mu.Lock()
		if c.flow.client == filterClient && current.Client == filterClient {
			c.flow.teamHistory = history
		}
		c.mgr.mu.Unlock()
	}
	m.OnTeam = func(id string) { c.runTeam(win, id, "inspect", "") }
	m.OnCloseTeam = func() {
		c.mgr.mu.Lock()
		c.stopProcessLocked()
		c.flow.teamID = ""
		c.flow.team = nil
		c.flow.detail = nil
		c.flow.assetDiff = nil
		c.flow.process = nil
		c.flow.detailGeneration++
		c.mgr.mu.Unlock()
	}
	m.OnControl = func(action, workID string) { c.runTeam(win, m.TeamID, action, workID) }
	m.OnStopProject = func(id string) { c.runStop(win, id) }
	m.OnNextWork = func(cursor string) { c.runTeam(win, m.TeamID, "next", cursor) }
	m.OnPreviousWork = func() { c.runTeam(win, m.TeamID, "previous", "") }
	m.OnInspect = func(workID, memberID string) { c.runDetail(win, m.TeamID, "detail", workID, memberID) }
	m.OnCloseDetail = func() {
		c.mgr.mu.Lock()
		c.stopProcessLocked()
		c.flow.detail = nil
		c.flow.assetDiff = nil
		c.flow.process = nil
		c.flow.detailGeneration++
		c.mgr.mu.Unlock()
	}
	m.OnStopProcess = func() { c.mgr.mu.Lock(); c.stopProcessLocked(); c.mgr.mu.Unlock() }
	m.OnProcess = func(earlier bool) {
		if earlier {
			c.runDetail(win, m.TeamID, "earlier", "", "")
			return
		}
		c.watchProcess(win, m.TeamID)
	}
	m.OnRetryDraft = func() { c.runDetail(win, m.TeamID, "retry-draft", "", "") }
	m.OnSummaryRetryDraft = func() { c.runRecoveryDraft(win, m.TeamID, "summary") }
	m.OnHandoverRetryDraft = func() { c.runRecoveryDraft(win, m.RequestProjectID, "handover") }
	m.OnRequests = func(id string) { c.runRequests(win, id, "list", "current") }
	m.OnRequestList = func(kind string) { c.runRequests(win, m.RequestProjectID, "list", kind) }
	m.OnRequestGet = func(id string) { c.runRequests(win, m.RequestProjectID, "get", id) }
	m.OnRequestQueue = func(id, action string) {
		card := m.RequestCard
		c.mgr.mu.Lock()
		if saved, ok := c.flow.requestCards[requestCardKey(m.RequestProjectID, id)]; ok {
			copy := saved
			card = &copy
		}
		c.mgr.mu.Unlock()
		change, err := QueueChange(m.RequestPage, card, m.RequestQueue, id, action)
		if err != nil {
			current := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			if queueOwner.Matches(&c.flow, current.Client) {
				c.flow.err = err.Error()
			}
			c.mgr.mu.Unlock()
			return
		}
		submission := queueOwner
		submission.Change = change
		c.runRequests(win, m.RequestProjectID, "queue", id, submission)
	}
	m.OnRequestPage = func(direction int) {
		action := "next"
		if direction < 0 {
			action = "previous"
		}
		c.runRequests(win, m.RequestProjectID, action, "")
	}
	m.OnRequestArchive = func(id string, archive bool) {
		action := "restore"
		if archive {
			action = "archive"
		}
		c.runRequests(win, m.RequestProjectID, action, id)
	}
	m.OnBackRequests = func() {
		c.mgr.mu.Lock()
		c.flow.requestCard = nil
		c.flow.assetDiff = nil
		c.flow.assetNote = ""
		c.flow.requestGeneration++
		c.mgr.mu.Unlock()
	}
	m.OnCloseRequests = func() {
		c.mgr.mu.Lock()
		c.closeVisualResultsLocked()
		c.flow.requestProjectID = ""
		c.flow.requestCard = nil
		c.flow.assetDiff = nil
		c.flow.assetNote = ""
		c.flow.requestGeneration++
		c.mgr.mu.Unlock()
	}
	m.OnArtifact = func(file projectflow.Artifact, open func(string)) {
		c.runAsset(win, m.RequestProjectID, "file", file, "", open)
	}
	m.OnArtifactTarget = func(file projectflow.Artifact, open func(projectflow.FileTarget)) {
		c.runAssetForCard(win, m.RequestProjectID, "file", file, "", nil, nil, nil, open)
	}
	m.OnWorkDiff = func(workID string) {
		if (m.RequestProjectID == "" || observerTeam) && m.TeamID != "" && m.Detail != nil {
			c.runDetail(win, m.TeamID, "diff", workID, "")
			return
		}
		c.runAsset(win, m.RequestProjectID, "diff", projectflow.Artifact{}, workID, nil)
	}
	m.OnRequestDiff = func(requestID string) {
		c.runRequestChanges(win, m.RequestProjectID, requestID, queueOwner)
	}
	m.OnRevertRequest = func(requestID string, revision int) {
		c.revertRequest(win, m.RequestProjectID, requestID, revision, queueOwner)
	}
	m.OnRevertAskDraft = func() { c.runRecoveryDraft(win, m.RequestProjectID, "revert") }
	m.OnProjectDirectory = func() { c.runAsset(win, m.RequestProjectID, "directory", projectflow.Artifact{}, "", nil) }
	m.OnProjectDirectoryFor = func(id string) { c.runAsset(win, id, "directory", projectflow.Artifact{}, "", nil, queueOwner) }
	m.OnRequestWorkDetail = func(requestID, workID string) {
		c.runRequestWorkDetail(win, m.RequestProjectID, requestID, workID, queueOwner)
	}
	m.OnCloseDiff = func() { c.mgr.mu.Lock(); c.flow.assetDiff = nil; c.mgr.mu.Unlock() }
	m.OnArchiveProject = func(id string) { c.runLifecycle(win, id, "prepare") }
	m.OnRestoreProject = func(id string) { c.runLifecycle(win, id, "restore") }
	m.OnConfirmArchive = func() {
		if m.ArchiveCandidate != nil {
			c.runLifecycle(win, m.ArchiveCandidate.ID, "archive")
		}
	}
	m.OnCancelArchive = func() { c.mgr.mu.Lock(); c.flow.archiveCandidate = nil; c.mgr.mu.Unlock() }
	c.bindHistory(win, &m)
	c.bindMaterials(win, &m)
	c.bindActivity(win, &m)
	c.bindDeliveries(win, &m)
	c.bindDocuments(win, &m)
	c.bindHistorical(win, &m)
	c.bindMaterialCollection(win, &m)
	c.bindRequestReverts(win, &m)
	c.bindRequestCardActions(win, &m)
	return m
}

func (c *Controller) runAction(win any, action, arg, title string) {
	if win == nil {
		return
	}
	snap := c.mgr.hostSnapshot()
	cfgStore := c.mgr.configStore()
	sessions := c.mgr.sessionsStore()
	lc := c.mgr.lifecycleContext()

	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if f.busy {
		c.mgr.mu.Unlock()
		return
	}
	if client == nil || f.client != client {
		if action == "create" {
			f.directoryIntent, f.createAdoption = nil, nil
		}
		f.err = i18n.T(i18n.KeyCmdHostIsNotConnectedRefreshWhenReady)
		c.mgr.mu.Unlock()
		return
	}
	if action != "list" && !f.listing.Available {
		if action == "create" {
			f.directoryIntent, f.createAdoption = nil, nil
		}
		f.err = i18n.T(i18n.KeyCmdProjectHostIsUnavailable)
		c.mgr.mu.Unlock()
		return
	}
	workspace := currentWorkspace(cfgStore)
	f.busy = true
	if action == "list" {
		f.loadAttempted = true
	}
	f.err = ""
	f.generation++
	generation := f.generation
	navigation := f.navigation
	adoption := f.createAdoption
	if action == "create" {
		f.createAdoption = nil
		f.createError = ""
	}
	c.mgr.mu.Unlock()

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
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
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
		c.mgr.updateWindow(win, func() {
			current := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			f := &c.flow
			if current.Client != client || f.client != client || f.generation != generation {
				c.mgr.mu.Unlock()
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
			applyErr := ""
			pendingCreate := ""
			if action == "list" && f.pendingCreate != "" {
				if acceptedList && listing.Available {
					pendingCreate = f.pendingCreate
				} else {
					f.directoryIntent, f.createAdoption = nil, nil
				}
				f.pendingCreate = ""
			}
			c.mgr.mu.Unlock()

			if action == "create" && lead != "" && stillOpen && adoption != nil && sessions != nil && sessions.SelectedSessionID() == selected {
				if c.mgr.deps.ApplyWorkspaceAdoption != nil && !c.mgr.deps.ApplyWorkspaceAdoption(adoption, sessions, selected, lead) {
					applyErr = "Lead 已有未发送草稿；原草稿已保留，未覆盖。"
				}
			}
			if action == "create" || applyErr != "" {
				c.mgr.mu.Lock()
				if c.flow.client == client && c.flow.generation == generation {
					if applyErr != "" {
						c.flow.err = applyErr
					}
					if action == "create" {
						c.flow.createError = c.flow.err
					}
				}
				c.mgr.mu.Unlock()
			}
			if pendingCreate != "" {
				c.runAction(win, "create", pendingCreate, "")
			}
			if lead != "" && stillOpen && sessions != nil && sessions.SelectedSessionID() == selected {
				sessions.RefreshList(c.mgr.lifecycleContext())
				sessions.Select(lead)
			}
		})
	}()
}

// CreateFromDirectory initiates Project creation from a registered workspace directory click.
func (m *Manager) CreateFromDirectory(win any, path string) {
	m.CreateIntent(win, path, false, true)
}

// CreateWithDraft initiates Project creation while preserving eligible unsent draft state for the new Lead.
func (m *Manager) CreateWithDraft(win any, path string) {
	m.CreateIntent(win, path, true, true)
}

// CreateIntent prepares and runs Project creation on the primary controller.
func (m *Manager) CreateIntent(win any, path string, carry, registered bool) {
	c := m.primary
	c.Model(win)
	sessions := m.sessionsStore()
	var adoption any
	if carry && sessions != nil && m.deps.CaptureWorkspaceAdoption != nil {
		adoption = m.deps.CaptureWorkspaceAdoption(sessions)
	}
	m.mu.Lock()
	f := &c.flow
	if f.busy && f.loaded {
		m.mu.Unlock()
		return
	}
	f.expanded = true
	f.createAdoption = adoption
	f.directoryIntent = nil
	if registered {
		f.directoryIntent = &directoryIntent{client: f.client, path: path}
	}
	if !f.loaded {
		f.pendingCreate = path
		busy := f.busy
		m.mu.Unlock()
		if !busy {
			c.runAction(win, "list", "", "")
		}
		return
	}
	m.mu.Unlock()
	c.runAction(win, "create", path, "")
}
