package main

import (
	"context"
	"strings"
	"time"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"whaleisle/internal/projectflow"
)

// The explicit recovery button revalidates saved evidence, then submits through
// the original Lead composer so ordinary send failure/recovery remains visible.
func (s *projectController) runProjectRecoveryDraft(win *mygo.Window, id, kind string, requestCards ...projectflow.RequestCard) {
	if win == nil || id == "" {
		return
	}
	s.mu.Lock()
	f := s.projectFlow
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
		f.err = i18n.T(i18n.KeyCmdRefreshOrRestoreTheOriginalProjectFirs)
		s.mu.Unlock()
		return
	}
	selectedWork, selectedCard := f.detail, f.requestCard
	var revertState *projectRequestRevertState
	var revertKey projectRequestRevision
	switch kind {
	case "summary":
		if f.teamID != id || f.team == nil || f.team.Project.CoordinatorSessionID != project.CoordinatorSessionID || selectedWork == nil || selectedWork.Workstream.SummaryFailure == nil {
			s.mu.Unlock()
			return
		}
	case "handover":
		if len(requestCards) == 1 {
			card := requestCards[0]
			selectedCard = &card
		}
		revertState = s.requestRevertStateLocked()
		if selectedCard == nil || selectedCard.ProjectID != id || selectedCard.HandoverError == "" || selectedCard.AuthorityError != "" || revertState == nil {
			s.mu.Unlock()
			return
		}
		revertKey = projectRequestRevision{selectedCard.ID, selectedCard.Revision}
		cached, known := s.requestRevertCardLocked(revertState, revertKey)
		if !known || cached.HandoverError != selectedCard.HandoverError || cached.AuthorityError != "" {
			s.mu.Unlock()
			return
		}
		selectedCard = &cached
	case "revert":
		if len(requestCards) > 1 {
			s.mu.Unlock()
			return
		}
		if len(requestCards) == 1 {
			card := requestCards[0]
			selectedCard = &card
		}
		revertState = s.requestRevertStateLocked()
		if selectedCard == nil || selectedCard.ProjectID != id || revertState == nil {
			s.mu.Unlock()
			return
		}
		revertKey = projectRequestRevision{selectedCard.ID, selectedCard.Revision}
		cached, known := s.requestRevertCardLocked(revertState, revertKey)
		if !known || !s.requestRevertAllowedLocked(cached) || !strings.Contains(revertState.errors[revertKey], "pluginrpc: revert_conflict:") {
			s.mu.Unlock()
			return
		}
		selectedCard = &cached
	default:
		s.mu.Unlock()
		return
	}
	f.busy = true
	f.err = ""
	f.generation++
	generation := f.generation
	navigation := f.navigation
	detailGeneration := f.detailGeneration
	s.mu.Unlock()
	selectedSession := sessions.SelectedSessionID()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var draft string
		var err error
		if kind == "summary" {
			draft, err = rpc.PrepareSummaryRetry(ctx, project, *selectedWork)
		} else if kind == "revert" {
			draft, err = rpc.PrepareRevertRequest(ctx, *selectedCard)
		} else {
			draft, err = rpc.PrepareHandoverRetry(ctx, project, *selectedCard)
		}
		win.Update(func() {
			s.mu.Lock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || s.sessions != sessions || f.generation != generation {
				s.mu.Unlock()
				return
			}
			f.busy = false
			if kind == "summary" && (f.teamID != id || f.team == nil || f.team.Project.CoordinatorSessionID != project.CoordinatorSessionID || f.detail != selectedWork || f.detailGeneration != detailGeneration) {
				s.mu.Unlock()
				return
			}
			if kind == "revert" || kind == "handover" {
				if _, known := s.requestRevertCardLocked(revertState, revertKey); !known {
					s.mu.Unlock()
					return
				}
			}
			if err != nil {
				f.err = err.Error()
				s.mu.Unlock()
				return
			}
			if !f.visible() || f.navigation != navigation || sessions.SelectedSessionID() != selectedSession {
				f.err = i18n.T(i18n.KeyCmdNavigationChangedTheRecoveryDraftWasNo)
				s.mu.Unlock()
				return
			}
			target := s.composerDraftLocked(project.CoordinatorSessionID)
			if target.Text != "" || len(target.Attachments) > 0 || target.Sending || target.Preparing > 0 || target.EditSeq > 0 || target.EditSessionID != "" {
				f.err = i18n.T(i18n.KeyCmdTheLeadHasADraftOrPendingSendItWas)
				s.mu.Unlock()
				return
			}
			notice := i18n.T(i18n.KeyCmdSendingTheRecoveryRequestToTheOriginal)
			if kind == "summary" {
				f.teamNote = notice
			} else {
				f.assetNote = notice
			}
			s.mu.Unlock()
			sessions.Select(project.CoordinatorSessionID)
			if submitErr := submitProjectRecoveryDraft(target, draft, func() {
				snap := sessions.Snapshot()
				s.submitComposer(win, sessions, target, project.CoordinatorSessionID, snap.ScratchCwd, 0, false, snap.ComposerHostGeneration)
			}); submitErr != nil {
				s.mu.Lock()
				s.projectFlow.err = submitErr.Error()
				s.mu.Unlock()
			}
		})
	}()
}
