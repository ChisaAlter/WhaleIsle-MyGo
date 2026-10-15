package main

import (
	"context"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/projectflow"
)

// runProjectRecoveryDraft only fills an empty original Lead composer after the
// Host revalidates saved evidence. The user still sends the request explicitly.
func (s *mainState) runProjectRecoveryDraft(win *mygo.Window, id, kind string) {
	if win == nil || id == "" {
		return
	}
	s.mu.Lock()
	f := &s.projectFlow
	client := s.dshClient
	sessions := s.sessions
	if f.busy || client == nil || f.client != client || !f.listing.Available || sessions == nil {
		s.mu.Unlock()
		return
	}
	var project projectflow.Project
	for _, p := range f.listing.Projects {
		if p.ID == id {
			project = p
			break
		}
	}
	if project.ID == "" || project.CoordinatorSessionID == "" || project.Lifecycle != "ready" || project.Archiving {
		f.err = resumeText("请先刷新或恢复原 Project。", "Refresh or restore the original Project first.")
		s.mu.Unlock()
		return
	}
	selectedWork, selectedCard := f.detail, f.requestCard
	switch kind {
	case "summary":
		if f.teamID != id || f.team == nil || f.team.Project.CoordinatorSessionID != project.CoordinatorSessionID || selectedWork == nil || selectedWork.Workstream.SummaryFailure == nil {
			s.mu.Unlock()
			return
		}
	case "handover":
		if f.requestProjectID != id || selectedCard == nil || selectedCard.ProjectID != id || selectedCard.HandoverError == "" {
			s.mu.Unlock()
			return
		}
	default:
		s.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	navigation := f.navigation
	detailGeneration, requestGeneration := f.detailGeneration, f.requestGeneration
	s.mu.Unlock()
	selectedSession := sessions.Snapshot().SelectedID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var draft string
		var err error
		if kind == "summary" {
			draft, err = rpc.PrepareSummaryRetry(ctx, project, *selectedWork)
		} else {
			draft, err = rpc.PrepareHandoverRetry(ctx, project, *selectedCard)
		}
		win.Update(func() {
			s.mu.Lock()
			f := &s.projectFlow
			if s.dshClient != client || f.client != client || s.sessions != sessions || f.generation != generation {
				s.mu.Unlock()
				return
			}
			f.busy = false
			if kind == "summary" && (f.teamID != id || f.team == nil || f.team.Project.CoordinatorSessionID != project.CoordinatorSessionID || f.detail != selectedWork || f.detailGeneration != detailGeneration) ||
				kind == "handover" && (f.requestProjectID != id || f.requestCard != selectedCard || f.requestGeneration != requestGeneration) {
				s.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				s.mu.Unlock()
				return
			}
			if !f.expanded || f.navigation != navigation || sessions.Snapshot().SelectedID != selectedSession {
				f.err = resumeText("导航已改变，尚未写入恢复草稿。", "Navigation changed; the recovery draft was not inserted.")
				s.mu.Unlock()
				return
			}
			target := s.composerDraftLocked(project.CoordinatorSessionID)
			if target.Text != "" || len(target.Attachments) > 0 || target.Sending || target.Preparing > 0 || target.EditSeq > 0 || target.EditSessionID != "" {
				f.err = resumeText("Lead 中已有草稿或正在发送，已保留。请先处理现有草稿。", "The Lead has a draft or pending send; it was preserved. Finish that draft first.")
				s.mu.Unlock()
				return
			}
			target.Text = draft
			notice := resumeText("已将重新汇总指令放入原 Lead 草稿，尚未发送或重新执行工作。", "Recovery request placed in the original Lead draft. It has not been sent or rerun any work.")
			if kind == "summary" {
				f.teamNote = notice
			} else {
				f.assetNote = notice
			}
			s.mu.Unlock()
			sessions.Select(project.CoordinatorSessionID)
		})
	}()
}
