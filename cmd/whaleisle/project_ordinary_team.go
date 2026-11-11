package main

import (
	"context"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
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

func (s *mainState) ordinaryTeamModel(win *mygo.Window) projects.OrdinaryTeamModel {
	s.mu.Lock()
	sessions := s.sessions
	s.mu.Unlock()
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if sessions == nil {
		return projects.OrdinaryTeamModel{}
	}
	sid := sessions.SelectedSessionID()
	s.mu.Lock()
	t := s.projectFlow.ordinaryTeam
	initial := t == nil || t.client != client || t.sessionID != sid
	if initial {
		t = &ordinaryTeamState{client: client, sessionID: sid}
		s.projectFlow.ordinaryTeam = t
	}
	m := projects.OrdinaryTeamModel{SessionID: sid, Team: t.team, Loading: t.loading, Error: t.err, State: &t.view}
	if !t.lastRead.IsZero() {
		m.NextRefresh = t.lastRead.Add(5 * time.Second)
	}
	s.mu.Unlock()
	if m.Team != nil {
		snapshot := sessions.RenderSnapshot()
		team := *m.Team
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
		m.Team = &team
	}
	m.OnRefresh = func() { s.refreshOrdinaryTeam(win, t) }
	m.OnOpenMember = func(id string) {
		s.dshMu.Lock()
		currentClient := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		valid := s.projectFlow.ordinaryTeam == t && currentClient == t.client
		s.mu.Unlock()
		if !valid || sessions.SelectedSessionID() != sid {
			return
		}
		snap := sessions.Snapshot()
		nav := sessions.SyncAgentNavigation(snap.PlanHistoryBinding)
		for _, row := range nav.Rows {
			if row.Address.ParentSessionID == sid && row.Address.ChildSessionID == id && row.CanOpen() {
				sessions.OpenAgentNavigation(nav.Token, row.Address)
				if win != nil {
					win.Update(func() {})
				}
				return
			}
		}
		s.mu.Lock()
		if s.projectFlow.ordinaryTeam == t {
			t.err = "成员目录尚未就绪，请刷新后重试。"
		}
		s.mu.Unlock()
	}
	if initial && client != nil && sid != "" {
		s.refreshOrdinaryTeam(win, t)
		s.mu.Lock()
		m.Loading = t.loading
		if !t.lastRead.IsZero() {
			m.NextRefresh = t.lastRead.Add(5 * time.Second)
		}
		s.mu.Unlock()
	}
	return m
}

func (s *mainState) refreshOrdinaryTeam(win *mygo.Window, t *ordinaryTeamState) {
	s.dshMu.Lock()
	currentClient := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	if t == nil || t.client == nil || t.loading || s.projectFlow.ordinaryTeam != t || currentClient != t.client || t.sessionID == "" {
		s.mu.Unlock()
		return
	}
	t.read++
	read := t.read
	t.loading = true
	t.err = ""
	t.lastRead = time.Now()
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		client := projectflow.Client{Origin: t.client.Endpoint.Origin, Cookie: t.client.Endpoint.Cookie, HTTP: t.client.HTTP}
		team, err := client.SessionTeam(ctx, t.sessionID)
		s.dshMu.Lock()
		currentClient := s.dshClient
		s.dshMu.Unlock()
		s.mu.Lock()
		if s.projectFlow.ordinaryTeam != t || currentClient != t.client || t.read != read {
			s.mu.Unlock()
			return
		}
		t.loading = false
		if err != nil {
			t.err = err.Error()
		} else {
			t.team = team
		}
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}()
}
