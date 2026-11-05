package main

import (
	"context"
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

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

func (s *mainState) bindProjectTeamModel(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	client := s.projectFlow.client
	s.mu.Unlock()
	m.ProjectTeam = func(id string) projects.ProjectTeamModel { return s.projectTeamModel(win, client, id) }
}

func (s *mainState) projectTeamModel(win *mygo.Window, client *dsh.Client, id string) projects.ProjectTeamModel {
	s.mu.Lock()
	f := &s.projectFlow
	var project projectflow.Project
	if client != nil && s.dshClient == client && f.client == client && f.listing.Available {
		for _, candidate := range f.listing.Projects {
			if candidate.ID == id && candidate.CoordinatorSessionID != "" {
				project = candidate
				break
			}
		}
	}
	if project.ID == "" {
		s.mu.Unlock()
		return projects.ProjectTeamModel{}
	}
	if f.projectTeams == nil {
		f.projectTeams = map[string]*projectTeamState{}
	}
	t := f.projectTeams[id]
	if t == nil || t.client != client || t.coordinator != project.CoordinatorSessionID {
		t = &projectTeamState{client: client, projectID: id, coordinator: project.CoordinatorSessionID}
		f.projectTeams[id] = t
	}
	readDue := !t.loading && (t.lastRead.IsZero() || !time.Now().Before(t.lastRead.Add(5*time.Second)))
	m := projects.ProjectTeamModel{
		Project: project, SessionID: project.CoordinatorSessionID, State: &t.view,
		Overview: t.overview, Loading: t.loading || readDue, Error: t.err,
		ActivityStale: t.overview == nil || t.err != "" || t.loading || readDue || !f.loaded,
		NextRefresh:   t.lastRead.Add(5 * time.Second),
	}
	sessions := s.sessions
	s.mu.Unlock()
	if sessions != nil {
		snapshot := sessions.RenderSnapshot()
		m.SessionID = snapshot.SelectedID
		// Runtime list/follow updates override the older Team enrichment in both
		// directions: an explicit inactive member cannot stay cached as running.
		if m.Overview != nil && m.Overview.Team != nil {
			overview := *m.Overview
			team := *overview.Team
			team.Members = append([]projectflow.TeamMember(nil), team.Members...)
			status := make(map[string]bool, len(snapshot.Items))
			for _, item := range snapshot.Items {
				status[item.SessionID] = item.Running
			}
			for i := range team.Members {
				if running, known := status[team.Members[i].ID]; known {
					team.Members[i].Running = running
				}
			}
			overview.Team, overview.Project = &team, project
			m.Overview = &overview
			m.CountKnown = !m.ActivityStale && team.StatusKnown && team.Failure == "" && snapshot.ClientReady && snapshot.HistoryReady && !snapshot.HistoryLoading && !snapshot.ListLoading && snapshot.ListErr == ""
		}
	}
	if m.Overview != nil && m.Overview.Team != nil {
		jobs := map[string]bool{}
		for _, id := range project.Activity.RunningJobSessionIDs {
			jobs[id] = true
		}
		seen := map[string]bool{}
		for _, member := range m.Overview.Team.Members {
			if member.Role != "lead" && (member.Running || jobs[member.ID]) {
				seen[member.ID] = true
			}
		}
		m.RunningCount = len(seen)
	}
	m.OnRefresh = func() { s.refreshProjectTeamAction(win, t) }
	if readDue {
		s.refreshProjectTeamAction(win, t)
	}
	return m
}

func (s *mainState) refreshProjectTeamAction(win *mygo.Window, t *projectTeamState) {
	s.mu.Lock()
	f := &s.projectFlow
	if win == nil || t == nil || t.client == nil || t.loading || f.projectTeams[t.projectID] != t || s.dshClient != t.client || f.client != t.client || !f.listing.Available {
		s.mu.Unlock()
		return
	}
	t.read++
	read := t.read
	t.loading, t.err, t.lastRead = true, "", time.Now()
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: t.client.Endpoint.Origin, Cookie: t.client.Endpoint.Cookie, HTTP: t.client.HTTP}
		overview, err := rpc.TeamOverview(ctx, t.projectID)
		if err == nil && overview.Project.CoordinatorSessionID != t.coordinator {
			err = fmt.Errorf("Project coordinator changed; reopen the Team from its Project conversation")
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := &s.projectFlow
			if f.projectTeams[t.projectID] != t || s.dshClient != t.client || f.client != t.client || t.read != read {
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
