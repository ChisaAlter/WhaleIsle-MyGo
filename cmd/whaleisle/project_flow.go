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
	archiveCandidate              *projectflow.Project
	lifecycleNote                 string
	assetDiff                     *projectflow.WorkDiff
	assetNote                     string
	requestProjectID, requestKind string
	requestPage                   projectflow.RequestPage
	requestCard                   *projectflow.RequestCard
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

func (s *mainState) projectModel(win *mygo.Window) projects.Model {
	s.mu.Lock()
	f := &s.projectFlow
	if f.client != s.dshClient {
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
	listing := f.listing
	listing.Projects = append([]projectflow.Project(nil), listing.Projects...)
	m := projects.Model{ArchiveCandidate: f.archiveCandidate, LifecycleNote: f.lifecycleNote, AssetDiff: f.assetDiff, AssetNote: f.assetNote, RequestProjectID: f.requestProjectID, RequestKind: f.requestKind, RequestPage: f.requestPage, RequestCard: f.requestCard, RequestPageNumber: len(f.requestCursors), RequestLoaded: f.requestLoaded, Detail: f.detail, Process: f.process, TeamID: f.teamID, Team: f.team, TeamNote: f.teamNote, Expanded: f.expanded, Busy: f.busy, Loaded: f.loaded, Listing: listing, Error: f.err, Workspace: workspace, RenameID: f.renameID, RenameDraft: &f.renameDraft}
	for _, p := range listing.Projects {
		if p.ID == f.requestProjectID {
			m.RequestMutable = p.Lifecycle == "ready" && !p.Archiving
		}
	}
	sessions := s.sessions
	s.mu.Unlock()
	if sessions != nil {
		m.SelectedSession = sessions.Snapshot().SelectedID
	}
	m.OnToggle = func() {
		s.mu.Lock()
		s.projectFlow.expanded = !s.projectFlow.expanded
		s.projectFlow.navigation++
		load := s.projectFlow.expanded && !s.projectFlow.loaded
		s.mu.Unlock()
		if load {
			s.runProjectAction(win, "list", "", "")
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
	m.OnTeam = func(id string) { s.runProjectTeam(win, id, "inspect", "") }
	m.OnCloseTeam = func() {
		s.mu.Lock()
		s.projectFlow.teamID = ""
		s.projectFlow.team = nil
		s.projectFlow.detail = nil
		s.projectFlow.process = nil
		s.projectFlow.detailGeneration++
		s.mu.Unlock()
	}
	m.OnControl = func(action, workID string) { s.runProjectTeam(win, m.TeamID, action, workID) }
	m.OnNextWork = func(cursor string) { s.runProjectTeam(win, m.TeamID, "next", cursor) }
	m.OnInspect = func(workID, memberID string) { s.runProjectDetail(win, m.TeamID, "detail", workID, memberID) }
	m.OnCloseDetail = func() {
		s.mu.Lock()
		s.projectFlow.detail = nil
		s.projectFlow.process = nil
		s.projectFlow.detailGeneration++
		s.mu.Unlock()
	}
	m.OnProcess = func(earlier bool) {
		action := "process"
		if earlier {
			action = "earlier"
		}
		s.runProjectDetail(win, m.TeamID, action, "", "")
	}
	m.OnRetryDraft = func() { s.runProjectDetail(win, m.TeamID, "retry-draft", "", "") }
	m.OnRequests = func(id string) { s.runProjectRequests(win, id, "list", "current") }
	m.OnRequestList = func(kind string) { s.runProjectRequests(win, m.RequestProjectID, "list", kind) }
	m.OnRequestGet = func(id string) { s.runProjectRequests(win, m.RequestProjectID, "get", id) }
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
		s.runProjectAsset(win, m.RequestProjectID, "diff", projectflow.Artifact{}, workID, nil)
	}
	m.OnProjectDirectory = func() { s.runProjectAsset(win, m.RequestProjectID, "directory", projectflow.Artifact{}, "", nil) }
	m.OnCloseDiff = func() { s.mu.Lock(); s.projectFlow.assetDiff = nil; s.mu.Unlock() }
	m.OnArchiveProject = func(id string) { s.runProjectLifecycle(win, id, "prepare") }
	m.OnRestoreProject = func(id string) { s.runProjectLifecycle(win, id, "restore") }
	m.OnConfirmArchive = func() {
		if m.ArchiveCandidate != nil {
			s.runProjectLifecycle(win, m.ArchiveCandidate.ID, "archive")
		}
	}
	m.OnCancelArchive = func() { s.mu.Lock(); s.projectFlow.archiveCandidate = nil; s.mu.Unlock() }
	return m
}
func (s *mainState) runProjectAction(win *mygo.Window, action, arg, title string) {
	if win == nil {
		return
	}
	s.mu.Lock()
	f := &s.projectFlow
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
			f := &s.projectFlow
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
			stillOpen := f.expanded && f.navigation == navigation
			s.mu.Unlock()
			if lead != "" && stillOpen && sessions != nil && sessions.Snapshot().SelectedID == selected {
				sessions.RefreshList(context.Background())
				sessions.Select(lead)
			}
		})
	}()
}
