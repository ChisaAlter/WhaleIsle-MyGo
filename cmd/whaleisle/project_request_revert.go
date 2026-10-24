package main

import (
	"context"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

type projectRequestRevision struct {
	id       string
	revision int
}

type projectRequestRevertState struct {
	client     *dsh.Client
	navigation uint64
	projectID  string
	errors     map[projectRequestRevision]string
}

func (s *projectController) requestRevertStateLocked() *projectRequestRevertState {
	f := s.projectFlow
	if f.client == nil || f.requestProjectID == "" || f.requestCards == nil {
		return nil
	}
	state := f.requestReverts
	if state == nil || state.client != f.client || state.navigation != f.navigation || state.projectID != f.requestProjectID {
		state = &projectRequestRevertState{client: f.client, navigation: f.navigation, projectID: f.requestProjectID, errors: make(map[projectRequestRevision]string)}
		f.requestReverts = state
	}
	return state
}

func (s *projectController) requestRevertCardLocked(state *projectRequestRevertState, key projectRequestRevision) (projectflow.RequestCard, bool) {
	f := s.projectFlow
	if state == nil || f.requestReverts != state || s.dshClient != state.client || f.client != state.client || f.navigation != state.navigation || f.requestProjectID != state.projectID || !f.visible() || !f.listing.Available {
		return projectflow.RequestCard{}, false
	}
	card, ok := f.requestCards[projectRequestCardKey(state.projectID, key.id)]
	if !ok || card.ID != key.id || card.ProjectID != state.projectID || card.Revision != key.revision {
		return projectflow.RequestCard{}, false
	}
	for _, project := range f.listing.Projects {
		if project.ID == state.projectID {
			return card, true
		}
	}
	return projectflow.RequestCard{}, false
}

func (s *projectController) requestRevertAllowedLocked(card projectflow.RequestCard) bool {
	if card.Changes == nil || !card.Changes.Revertible || card.Changes.RevertedAt != nil {
		return false
	}
	for _, project := range s.projectFlow.listing.Projects {
		if project.ID == card.ProjectID {
			return project.Lifecycle == "ready" && !project.Archiving
		}
	}
	return false
}

func (s *projectController) bindProjectRequestReverts(win *mygo.Window, m *projects.Model) {
	s.mu.Lock()
	state := s.requestRevertStateLocked()
	s.mu.Unlock()
	m.RequestRevert = func(id string, revision int) projects.RequestRevertState {
		key := projectRequestRevision{id, revision}
		s.mu.Lock()
		card, known := s.requestRevertCardLocked(state, key)
		view := projects.RequestRevertState{Known: known, Allowed: known && s.requestRevertAllowedLocked(card)}
		if state != nil && s.projectFlow.requestReverts == state {
			view.Error = state.errors[key]
		}
		s.mu.Unlock()
		if known && win != nil {
			view.Confirm = func() { s.revertProjectRequestOwned(win, state, key) }
			view.Ask = func() {
				s.mu.Lock()
				current, allowed := s.requestRevertCardLocked(state, key)
				s.mu.Unlock()
				if allowed {
					s.runProjectRecoveryDraft(win, current.ProjectID, "revert", current)
				}
			}
		}
		return view
	}
}

// A card's read-only changes route is available from the authoritative list;
// it need not first become the controller's currently opened detail card.
func (s *projectController) runProjectRequestChanges(win *mygo.Window, projectID, requestID string, owner projectQueueSubmission) {
	if win == nil {
		return
	}
	s.mu.Lock()
	f, client := s.projectFlow, s.dshClient
	if !owner.matches(f, client) || f.busy || !f.visible() || !f.listing.Available || f.requestProjectID != projectID {
		s.mu.Unlock()
		return
	}
	_, known := f.requestCards[projectRequestCardKey(projectID, requestID)]
	known = known || f.requestCard != nil && f.requestCard.ID == requestID && f.requestCard.ProjectID == projectID
	for _, card := range f.requestPage.Items {
		known = known || card.ID == requestID && card.ProjectID == projectID
	}
	for _, project := range f.listing.Projects {
		if project.ID == projectID && project.Queue != nil && project.Queue.Active != nil {
			known = known || project.Queue.Active.ID == requestID
		}
	}
	if !known {
		s.mu.Unlock()
		return
	}
	f.busy, f.err, f.requestRevertError = true, "", ""
	f.generation++
	f.requestGeneration++
	generation, navigation, requestGeneration := f.generation, f.navigation, f.requestGeneration
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		card, err := rpc.Request(ctx, projectID, requestID)
		var changes projectflow.RequestChanges
		if err == nil {
			changes, err = rpc.RequestChanges(ctx, projectID, requestID)
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if f.navigation != navigation || f.requestGeneration != requestGeneration || f.requestProjectID != projectID {
				return
			}
			if err != nil {
				f.err = err.Error()
				return
			}
			f.requestCard = &card
			s.rememberProjectRequestCardLocked(f.requestCard)
			f.assetDiff = &projectflow.WorkDiff{Requirement: &changes}
			f.assetNote = ""
		})
	}()
	win.Update(func() {})
}

// The confirmation retains the displayed revision; conflict checks belong to
// the Host, which rejects a revert that would overwrite newer edits.
func (s *projectController) revertProjectRequest(win *mygo.Window, projectID, requestID string, revision int, owner projectQueueSubmission) {
	s.mu.Lock()
	state := s.requestRevertStateLocked()
	valid := state != nil && owner.Client == state.client && owner.Navigation == state.navigation && projectID == state.projectID
	s.mu.Unlock()
	if valid {
		s.revertProjectRequestOwned(win, state, projectRequestRevision{requestID, revision})
	}
}

func (s *projectController) revertProjectRequestOwned(win *mygo.Window, state *projectRequestRevertState, key projectRequestRevision) {
	if win == nil {
		return
	}
	s.mu.Lock()
	f, client := s.projectFlow, s.dshClient
	card, known := s.requestRevertCardLocked(state, key)
	if !known || f.busy || !s.requestRevertAllowedLocked(card) {
		s.mu.Unlock()
		return
	}
	projectID, requestID, revision := card.ProjectID, key.id, key.revision
	f.busy, f.requestRevertError = true, ""
	delete(state.errors, key)
	f.generation++
	generation := f.generation
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		err := rpc.RevertRequest(ctx, projectID, requestID, revision)
		var fresh projectflow.RequestCard
		committed := err == nil
		if committed {
			fresh, err = rpc.Request(ctx, projectID, requestID)
		}
		win.Update(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				return
			}
			f.busy = false
			if _, current := s.requestRevertCardLocked(state, key); !current {
				return
			}
			if err != nil {
				f.requestRevertError = err.Error()
				state.errors[key] = err.Error()
				if committed {
					delete(f.requestCards, projectRequestCardKey(projectID, requestID))
					if f.requestCard != nil && f.requestCard.ID == requestID {
						f.requestCard = nil
					}
					f.err = i18n.T(i18n.KeyCmdChangesWereRevertedButRefreshingFailed) + " " + err.Error()
				}
				return
			}
			f.requestCard = &fresh
			delete(state.errors, key)
			s.rememberProjectRequestCardLocked(f.requestCard)
			for i := range f.requestPage.Items {
				if f.requestPage.Items[i].ID == requestID {
					f.requestPage.Items[i] = fresh
				}
			}
			f.assetDiff = nil
			f.assetNote = i18n.T(i18n.KeyCmdThisRequirementSChangesWereReverted)
		})
	}()
	win.Update(func() {})
}
