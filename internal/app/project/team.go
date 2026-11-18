package projectctl

import (
	"context"
	"fmt"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

type ordinaryTeamState struct {
	client    *dsh.Client
	sessionID string
	team      *projectflow.TeamProjection
	loading   bool
	err       string
	read      uint64
	lastRead  time.Time
	view      projects.OrdinaryTeamViewState
}

// This viewing owner is separate from the Project overview/modal controller.
// A session's header and retained Team Sidebar share it without changing the
// current overview section, workstream page, or any editable Project draft.
type projectTeamState struct {
	client                 *dsh.Client
	projectID, coordinator string
	view                   projects.TeamViewState
	overview               *projectflow.Overview
	loading                bool
	err                    string
	read                   uint64
	lastRead               time.Time
}

func applySessionRunningStatus(members []projectflow.TeamMember, items []session.Item) []projectflow.TeamMember {
	out := append([]projectflow.TeamMember(nil), members...)
	status := make(map[string]bool, len(items))
	for _, item := range items {
		status[item.SessionID] = item.Running
	}
	for i := range out {
		if running, known := status[out[i].ID]; known {
			out[i].Running = running
		}
	}
	return out
}

// HasOrdinaryTeam reports whether an ordinary session team snapshot state is currently cached.
func (m *Manager) HasOrdinaryTeam() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ordinaryTeam != nil
}

// OrdinaryTeamModel builds the read/refresh model for an ordinary session's Team view.
func (m *Manager) OrdinaryTeamModel(win any) projects.OrdinaryTeamModel {
	sessions := m.sessionsStore()
	snap := m.hostSnapshot()
	client := snap.Client
	if sessions == nil {
		return projects.OrdinaryTeamModel{}
	}
	sid := sessions.SelectedSessionID()
	var firstRead uint64
	m.mu.Lock()
	t := m.ordinaryTeam
	if t == nil || t.client != client || t.sessionID != sid {
		t = &ordinaryTeamState{client: client, sessionID: sid}
		m.ordinaryTeam = t
		if client != nil && sid != "" {
			// Claim the first read in this critical section, before the model
			// reads the flag. Starting it after the model was assembled let a
			// fast read finish in between, and the frame then carried
			// Loading=false with the nil Team it snapshotted first, the
			// unavailable flash this model exists to avoid.
			firstRead, _ = m.beginOrdinaryTeamReadLocked(t, client)
		}
	}
	model := projects.OrdinaryTeamModel{
		SessionID: sid,
		Team:      t.team,
		Loading:   t.loading,
		Error:     viewError(t.err),
		State:     &t.view,
	}
	if !t.lastRead.IsZero() {
		model.NextRefresh = t.lastRead.Add(5 * time.Second)
	}
	m.mu.Unlock()
	if firstRead != 0 {
		go m.readOrdinaryTeam(win, t, firstRead)
	}

	if model.Team != nil {
		snapshot := sessions.RenderSnapshot()
		team := *model.Team
		team.Members = applySessionRunningStatus(team.Members, snapshot.Items)
		model.Team = &team
	}
	model.OnRefresh = func() { m.refreshOrdinaryTeam(win, t) }
	model.OnOpenMember = func(id string) {
		nowSnap := m.hostSnapshot()
		m.mu.Lock()
		valid := m.ordinaryTeam == t && nowSnap.Client == t.client
		m.mu.Unlock()
		if !valid || sessions.SelectedSessionID() != sid {
			return
		}
		fullSnap := sessions.Snapshot()
		nav := sessions.SyncAgentNavigation(fullSnap.PlanHistoryBinding)
		for _, row := range nav.Rows {
			if row.Address.ParentSessionID == sid && row.Address.ChildSessionID == id && row.CanOpen() {
				sessions.OpenAgentNavigation(nav.Token, row.Address)
				if win != nil {
					m.updateWindow(win, func() {})
				}
				return
			}
		}
		m.mu.Lock()
		if m.ordinaryTeam == t {
			t.err = "成员目录尚未就绪，请刷新后重试。"
		}
		m.mu.Unlock()
	}
	return model
}

// refreshOrdinaryTeam starts one read of the selected session's team.
func (m *Manager) refreshOrdinaryTeam(win any, t *ordinaryTeamState) {
	snap := m.hostSnapshot()
	m.mu.Lock()
	read, ok := m.beginOrdinaryTeamReadLocked(t, snap.Client)
	m.mu.Unlock()
	if !ok {
		return
	}
	go m.readOrdinaryTeam(win, t, read)
}

// beginOrdinaryTeamReadLocked marks t loading and returns its read generation.
// The caller holds the manager mutex, so the flag is set before any frame can
// snapshot the state and before the read asks its dependencies for anything.
func (m *Manager) beginOrdinaryTeamReadLocked(t *ordinaryTeamState, client *dsh.Client) (uint64, bool) {
	if t == nil || t.client == nil || t.loading || m.ordinaryTeam != t || client != t.client || t.sessionID == "" {
		return 0, false
	}
	t.read++
	t.loading = true
	t.err = ""
	t.lastRead = time.Now()
	return t.read, true
}

// readOrdinaryTeam performs one read on a worker and commits it only while the
// state it read is still the current one.
func (m *Manager) readOrdinaryTeam(win any, t *ordinaryTeamState, read uint64) {
	ctx, cancel := context.WithTimeout(m.lifecycleContext(), 30*time.Second)
	defer cancel()
	rpc := projectRPC(t.client)
	team, err := rpc.SessionTeam(ctx, t.sessionID)
	nowSnap := m.hostSnapshot()
	m.mu.Lock()
	if m.ordinaryTeam != t || nowSnap.Client != t.client || t.read != read {
		m.mu.Unlock()
		return
	}
	t.loading = false
	if err != nil {
		t.err = err.Error()
	} else {
		t.team = team
	}
	m.mu.Unlock()
	if win != nil {
		m.updateWindow(win, func() {})
	}
}

func (m *Manager) bindTeamModel(win any, model *projects.Model) {
	m.mu.Lock()
	client := m.primary.flow.client
	m.mu.Unlock()
	model.ProjectTeam = func(id string) projects.ProjectTeamModel { return m.TeamModel(win, client, id) }
}

// TeamModel builds the shared read-only TeamAction / TeamSidebar model for a Project.
func (m *Manager) TeamModel(win any, client *dsh.Client, id string) projects.ProjectTeamModel {
	snap := m.hostSnapshot()
	sessions := m.sessionsStore()

	m.mu.Lock()
	f := &m.primary.flow
	var project projectflow.Project
	if client != nil && snap.Client == client && f.client == client && f.listing.Available {
		for _, candidate := range f.listing.Projects {
			if candidate.ID == id && candidate.CoordinatorSessionID != "" {
				project = candidate
				break
			}
		}
	}
	if project.ID == "" {
		m.mu.Unlock()
		return projects.ProjectTeamModel{}
	}
	if m.projectTeams == nil {
		m.projectTeams = map[string]*projectTeamState{}
	}
	t := m.projectTeams[id]
	if t == nil || t.client != client || t.coordinator != project.CoordinatorSessionID {
		t = &projectTeamState{client: client, projectID: id, coordinator: project.CoordinatorSessionID}
		m.projectTeams[id] = t
	}
	readDue := !t.loading && (t.lastRead.IsZero() || !time.Now().Before(t.lastRead.Add(5*time.Second)))
	model := projects.ProjectTeamModel{
		Project:       project,
		SessionID:     project.CoordinatorSessionID,
		State:         &t.view,
		Overview:      t.overview,
		Loading:       t.loading || readDue,
		Error:         viewError(t.err),
		ActivityStale: t.overview == nil || t.err != "" || t.loading || readDue || !f.loaded,
		NextRefresh:   t.lastRead.Add(5 * time.Second),
	}
	m.mu.Unlock()

	if sessions != nil {
		snapshot := sessions.RenderSnapshot()
		model.SessionID = snapshot.SelectedID
		// Runtime list/follow updates override the older Team enrichment in both
		// directions: an explicit inactive member cannot stay cached as running.
		if model.Overview != nil && model.Overview.Team != nil {
			overview := *model.Overview
			team := *overview.Team
			team.Members = applySessionRunningStatus(team.Members, snapshot.Items)
			overview.Team, overview.Project = &team, project
			model.Overview = &overview
			model.CountKnown = !model.ActivityStale && team.StatusKnown && team.Failure == "" && snapshot.ClientReady && snapshot.HistoryReady && !snapshot.HistoryLoading && !snapshot.ListLoading && snapshot.ListErr == ""
		}
	}
	if model.Overview != nil && model.Overview.Team != nil {
		jobs := map[string]bool{}
		for _, jobID := range project.Activity.RunningJobSessionIDs {
			jobs[jobID] = true
		}
		seen := map[string]bool{}
		for _, member := range model.Overview.Team.Members {
			if member.Role != "lead" && (member.Running || jobs[member.ID]) {
				seen[member.ID] = true
			}
		}
		model.RunningCount = len(seen)
	}
	model.OnRefresh = func() { m.refreshTeamAction(win, t) }
	if readDue {
		m.refreshTeamAction(win, t)
	}
	return model
}

func (m *Manager) refreshTeamAction(win any, t *projectTeamState) {
	snap := m.hostSnapshot()
	lc := m.lifecycleContext()
	m.mu.Lock()
	f := &m.primary.flow
	if win == nil || t == nil || t.client == nil || t.loading || m.projectTeams[t.projectID] != t || snap.Client != t.client || f.client != t.client || !f.listing.Available {
		m.mu.Unlock()
		return
	}
	t.read++
	read := t.read
	t.loading, t.err, t.lastRead = true, "", time.Now()
	m.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(t.client)
		overview, err := rpc.TeamOverview(ctx, t.projectID)
		if err == nil && overview.Project.CoordinatorSessionID != t.coordinator {
			err = fmt.Errorf("Project coordinator changed; reopen the Team from its Project conversation")
		}
		m.updateWindow(win, func() {
			nowSnap := m.hostSnapshot()
			m.mu.Lock()
			defer m.mu.Unlock()
			f := &m.primary.flow
			if m.projectTeams[t.projectID] != t || nowSnap.Client != t.client || f.client != t.client || t.read != read {
				return
			}
			t.loading, t.lastRead = false, time.Now()
			if err != nil {
				t.err = err.Error()
				return
			}
			for i, project := range f.listing.Projects {
				if project.ID == t.projectID && project.CoordinatorSessionID == t.coordinator {
					t.overview = &overview
					f.listing.Projects[i] = overview.Project
					return
				}
			}
		})
	}()
}

func (c *Controller) runTeam(win any, id, action, arg string) {
	if win == nil || id == "" {
		return
	}
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if f.busy {
		c.mgr.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available {
		f.err = i18n.T(i18n.KeyCmdProjectHostIsUnavailableRefreshToRetry)
		c.mgr.mu.Unlock()
		return
	}
	known := false
	for _, p := range f.listing.Projects {
		if p.ID == id {
			known = true
		}
	}
	if !known {
		f.err = i18n.T(i18n.KeyCmdThisProjectIsNotInTheCurrentListRefr)
		c.mgr.mu.Unlock()
		return
	}
	c.stopProcessLocked()
	if f.teamID != id {
		f.teamQuery = ""
		f.team = nil
		f.teamNote = ""
	}
	f.detail = nil
	f.process = nil
	f.detailGeneration++
	f.teamID = id
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	prior := f.team
	cursors := []string{""}
	if prior != nil && len(prior.CurrentCursors) > 0 {
		cursors = append([]string(nil), prior.CurrentCursors...)
	}
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		var out projectflow.Overview
		var err error
		var note string
		switch action {
		case "inspect", "next", "previous":
			if action == "next" {
				if prior == nil || prior.WorkPage.CurrentNextCursor == nil || arg == "" || arg != *prior.WorkPage.CurrentNextCursor {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectTeamRefreshPaging))
					break
				}
				cursors = append(cursors, arg)
			}
			if action == "previous" {
				if prior == nil || len(cursors) < 2 {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectWorkNoPreviousPage))
					break
				}
				cursors = cursors[:len(cursors)-1]
			}
			out, err = rpc.TeamOverviewPage(ctx, id, cursors[len(cursors)-1])
		case "stop", "pause", "resume":
			if prior == nil || prior.Team == nil || prior.Team.Failure != "" || prior.Project.Lifecycle != "ready" {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectTeamRefreshControl))
				break
			}
			note, err = rpc.Control(ctx, id, action, arg)
			if err == nil {
				out, err = rpc.TeamOverviewPage(ctx, id, cursors[len(cursors)-1])
			}
		default:
			err = fmt.Errorf("unsupported Project Team action")
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if nowSnap.Client != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.teamID != id {
				return
			}
			if err != nil {
				f.err = err.Error()
				// Keep the last readable page and its navigation on a failed refresh.
				return
			}
			if out.WorkPage.CurrentCursor != nil && *out.WorkPage.CurrentCursor == cursors[len(cursors)-1] {
				out.CurrentCursors = cursors
			}
			f.team = &out
			f.teamNote = note
			for i, p := range f.listing.Projects {
				if p.ID == out.Project.ID {
					f.listing.Projects[i] = out.Project
				}
			}
		})
	}()
}

// The input/approval stop action must not navigate to a Team observer. It
// submits one Project-level stop and refreshes authoritative activity in place.
func (c *Controller) runStop(win any, id string) {
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	known := false
	for _, p := range f.listing.Projects {
		if p.ID == id && p.Lifecycle == "ready" && !p.Archiving {
			known = true
			break
		}
	}
	if win == nil || client == nil || f.client != client || f.busy || !f.listing.Available || !known {
		c.mgr.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		note, err := rpc.Control(ctx, id, "stop", "")
		var listing projectflow.Listing
		var refreshErr error
		if err == nil {
			listing, refreshErr = rpc.List(ctx)
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if nowSnap.Client != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if err != nil {
				f.err = err.Error()
				return
			}
			f.lifecycleNote = note
			if refreshErr != nil {
				f.refreshNote = errcopy.Under(localizedText("已停止；刷新失败", "Stopped; refresh failed"), refreshErr)
				return
			}
			f.listing = listing
			f.loaded = true
		})
	}()
}

func (c *Controller) runLifecycle(win any, id, action string) {
	if win == nil || id == "" {
		return
	}
	snap := c.mgr.hostSnapshot()
	lc := c.mgr.lifecycleContext()
	c.mgr.mu.Lock()
	f := &c.flow
	client := snap.Client
	if f.busy {
		c.mgr.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available {
		f.err = i18n.T(i18n.KeyCmdProjectHostUnavailableRefreshToRetry)
		c.mgr.mu.Unlock()
		return
	}
	var selected projectflow.Project
	for _, p := range f.listing.Projects {
		if p.ID == id {
			selected = p
			break
		}
	}
	if selected.ID == "" || selected.Archiving {
		f.err = i18n.T(i18n.KeyCmdProjectUnavailableOrArchivingRefreshTo)
		c.mgr.mu.Unlock()
		return
	}
	if action == "archive" && (f.archiveCandidate == nil || f.archiveCandidate.ID != id) {
		c.mgr.mu.Unlock()
		return
	}
	if action == "restore" && selected.Lifecycle != "archived" {
		c.mgr.mu.Unlock()
		return
	}
	if action == "prepare" && selected.Lifecycle != "ready" {
		c.mgr.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	f.lifecycleNote = ""
	f.generation++
	generation := f.generation
	c.mgr.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(lc, 30*time.Second)
		defer cancel()
		rpc := projectRPC(client)
		var err error
		var candidate *projectflow.Project
		var result projectflow.LifecycleResult
		var listing projectflow.Listing
		var refreshErr error
		mutated := false
		if action == "prepare" {
			var fresh projectflow.Project
			fresh, err = rpc.LifecycleProject(ctx, id)
			if err == nil {
				if fresh.Archiving || fresh.Lifecycle != "ready" {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectLifecycleChanged))
				} else if fresh.HasActiveWork() || fresh.Activity.State == "" {
					candidate = &fresh
				} else {
					result, err = rpc.SetProjectLifecycle(ctx, id, "archive")
					mutated = err == nil
				}
			}
		} else if action == "archive" || action == "restore" {
			result, err = rpc.SetProjectLifecycle(ctx, id, action)
			mutated = err == nil
		} else {
			err = fmt.Errorf("unsupported Project lifecycle action")
		}
		if mutated {
			listing, refreshErr = rpc.List(ctx)
		}
		c.mgr.updateWindow(win, func() {
			nowSnap := c.mgr.hostSnapshot()
			c.mgr.mu.Lock()
			defer c.mgr.mu.Unlock()
			f := &c.flow
			if nowSnap.Client != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if err != nil {
				f.err = err.Error()
				return
			}
			if candidate != nil {
				f.archiveCandidate = candidate
				return
			}
			f.archiveCandidate = nil
			if mutated {
				if refreshErr == nil {
					f.listing = listing
					f.loaded = true
				} else {
					for i, p := range f.listing.Projects {
						if p.ID == id {
							f.listing.Projects[i] = result.Project
						}
					}
					f.err = errcopy.Under(i18n.T(i18n.KeyCmdProjectStateUpdatedButListRefreshFaile), refreshErr)
				}
				// Clear stale snapshots whose controls depend on Project lifecycle.
				if f.teamID == id {
					f.teamID = ""
					f.team = nil
					f.detail = nil
					f.process = nil
					f.detailGeneration++
				}
				if f.requestProjectID == id {
					f.requestProjectID = ""
					f.requestCard = nil
					f.requestGeneration++
				}
				if result.Project.Lifecycle == "archived" {
					f.lifecycleNote = i18n.T(i18n.KeyCmdProjectArchivedWorkHasStoppedConversat)
				} else {
					f.lifecycleNote = i18n.T(i18n.KeyCmdProjectRestoredSendANewInstructionInT)
				}
			}
		})
	}()
}
