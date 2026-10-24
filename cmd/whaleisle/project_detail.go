package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

func (s *projectController) runProjectDetail(win *mygo.Window, id, action, workID, memberID string) {
	if win == nil || id == "" {
		return
	}
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	if f.busy {
		s.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available || f.teamID != id || f.team == nil || f.team.Team == nil {
		f.err = i18n.T(i18n.KeyCmdRefreshTheProjectTeamFirst)
		s.mu.Unlock()
		return
	}
	s.stopProjectProcessLocked()
	p := f.team.Project
	prior := f.detail
	oldProcess := f.process
	mux := s.dshMux
	sessions := s.sessions
	if action != "detail" && prior == nil {
		s.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	f.detailGeneration++
	detailGeneration := f.detailGeneration
	navigation := f.navigation
	s.mu.Unlock()
	selected := ""
	if sessions != nil {
		selected = sessions.Snapshot().SelectedID
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var detail projectflow.WorkDetail
		var diff projectflow.WorkDiff
		var process projectflow.ProcessSnapshot
		var draft string
		var err error
		switch action {
		case "detail":
			detail, err = rpc.WorkDetail(ctx, id, workID, memberID)
		case "diff":
			detail, err = rpc.WorkDetail(ctx, id, prior.Workstream.ID, "")
			if err == nil {
				if detail.Worker == nil || detail.Worker.Role == "readonly" || workID != detail.Workstream.ID {
					err = fmt.Errorf("refresh a writable original Project member before viewing its diff")
				} else {
					diff, err = rpc.WorkDiff(ctx, id, detail.Workstream.ID)
				}
			}
		case "process":
			detail = *prior
			process, err = projectflow.ReadProcess(ctx, mux, p, detail)
		case "earlier":
			detail = *prior
			if oldProcess == nil {
				err = fmt.Errorf("open member history before paging")
			} else {
				process, err = rpc.EarlierProcess(ctx, p, detail, *oldProcess)
			}
		case "retry-draft":
			detail, err = rpc.WorkDetail(ctx, id, prior.Workstream.ID, "")
			if err == nil {
				draft, err = projectflow.RetryDraft(p, detail)
			}
		default:
			err = fmt.Errorf("unsupported Project detail action")
		}
		win.Update(func() {
			s.mu.Lock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				s.mu.Unlock()
				return
			}
			f.busy = false
			if f.teamID != id || f.detailGeneration != detailGeneration {
				s.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				s.mu.Unlock()
				return
			}
			f.detail = &detail
			if action == "detail" {
				f.assetDiff = nil
				f.process = nil
			}
			if action == "diff" {
				f.assetDiff = &diff
			}
			if action == "process" || action == "earlier" {
				f.process = &process
			}
			if action != "retry-draft" {
				s.mu.Unlock()
				return
			}
			if !f.visible() || f.navigation != navigation || sessions == nil || sessions.Snapshot().SelectedID != selected {
				f.err = i18n.T(i18n.KeyCmdNavigationChangedTheRetryDraftWasNotI)
				s.mu.Unlock()
				return
			}
			target := s.composerDraftLocked(p.CoordinatorSessionID)
			if target.Text != "" || len(target.Attachments) > 0 || target.Sending || target.Preparing > 0 || target.EditSeq > 0 {
				f.err = i18n.T(i18n.KeyCmdTheLeadHasADraftOrPendingSendItWas)
				s.mu.Unlock()
				return
			}
			target.Text = draft
			f.teamNote = i18n.T(i18n.KeyCmdRetryRequestPlacedInTheOriginalLeadDr)
			s.mu.Unlock()
			sessions.Select(p.CoordinatorSessionID)
		})
	}()
}
