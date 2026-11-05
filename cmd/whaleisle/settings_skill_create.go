package main

import (
	"context"
	"fmt"
	"time"
	"whaleisle/internal/ui/i18n"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/settings"
)

type settingsSkillCreateState struct {
	open, busy bool
	draft      settings.SkillCreateDraft
	err        string
	target     session.SkillTarget
	source     *session.Store
	epoch      uint64
	cancel     context.CancelFunc
}

func (s *mainState) invalidateSettingsSkillCreateLocked() {
	if s.settingsSkillCreate.cancel != nil {
		s.settingsSkillCreate.cancel()
	}
	s.settingsSkillCreate = settingsSkillCreateState{epoch: s.settingsSkillCreate.epoch + 1}
}

// The Host detach path also calls this when the Settings page is not rendered.
func (s *mainState) invalidateSettingsSkillCreate() {
	s.mu.Lock()
	s.invalidateSettingsSkillCreateLocked()
	s.mu.Unlock()
}

func (s *mainState) settingsSkillCreateCurrentLocked() bool {
	create := &s.settingsSkillCreate
	if create.source == nil || s.sessions != create.source {
		return false
	}
	current, err := create.source.SkillInventoryTarget(session.SkillRootUser)
	return err == nil && create.target.SameScope(current)
}

func (s *mainState) openSettingsSkillCreate() {
	s.mu.Lock()
	if s.settingsSkillCreate.busy {
		s.mu.Unlock()
		return
	}
	state := settingsSkillCreateState{open: true, source: s.sessions, epoch: s.settingsSkillCreate.epoch + 1}
	state.draft = settings.SkillCreateDraft{Root: session.SkillRootUser, ModelInvocable: true, UserInvocable: true}
	if state.source == nil {
		state.err = i18n.T(i18n.KeyCmdRuntimeDisconnected)
	} else {
		var err error
		state.target, err = state.source.SkillInventoryTarget(state.draft.Root)
		if err != nil {
			state.err = err.Error()
		}
	}
	s.settingsSkillCreate = state
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) settingsSkillCreateAccess() *settings.SkillCreateAccess {
	s.mu.Lock()
	defer s.mu.Unlock()
	create := &s.settingsSkillCreate
	if create.open && create.source != nil && !s.settingsSkillCreateCurrentLocked() {
		s.invalidateSettingsSkillCreateLocked()
		create = &s.settingsSkillCreate
	}
	return &settings.SkillCreateAccess{Open: create.open, Busy: create.busy, Draft: &create.draft, Error: create.err, OnSave: s.saveSettingsSkillCreate, OnCancel: s.cancelSettingsSkillCreate}
}

func (s *mainState) cancelSettingsSkillCreate() {
	s.mu.Lock()
	if !s.settingsSkillCreate.busy {
		s.invalidateSettingsSkillCreateLocked()
	}
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) saveSettingsSkillCreate(draft settings.SkillCreateDraft) {
	s.mu.Lock()
	create := &s.settingsSkillCreate
	if !create.open || create.busy || create.source == nil || s.sessions != create.source {
		s.mu.Unlock()
		return
	}
	if !s.settingsSkillCreateCurrentLocked() {
		s.invalidateSettingsSkillCreateLocked()
		s.mu.Unlock()
		return
	}
	create.busy, create.err = true, ""
	draft.Groups = append([]string(nil), draft.Groups...)
	create.draft = draft
	source, target, win := create.source, create.target, s.win
	epoch := create.epoch
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	create.cancel = cancel
	target.Root = draft.Root
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		fields := session.SkillEditorFields{Description: draft.Description, WhenToUse: draft.WhenToUse, Content: draft.Content, Groups: draft.Groups, ModelInvocable: draft.ModelInvocable, UserInvocable: draft.UserInvocable}
		err := source.CreateSkillInventoryFields(ctx, target, draft.Name, fields)
		cancel()
		s.mu.Lock()
		if s.settingsSkillCreate.epoch != epoch {
			s.mu.Unlock()
			return
		}
		if !s.settingsSkillCreateCurrentLocked() {
			s.invalidateSettingsSkillCreateLocked()
			s.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
			return
		}
		s.settingsSkillCreate.busy = false
		s.settingsSkillCreate.cancel = nil
		if err != nil {
			s.settingsSkillCreate.err = err.Error()
		} else {
			s.invalidateSettingsSkillCreateLocked()
			s.settingsSkillsLastAction = fmt.Sprintf(i18n.T(i18n.KeyCmdSkillCreated), draft.Name)
		}
		s.mu.Unlock()
		if err == nil {
			source.RefreshSkills(context.Background())
		}
		if win != nil {
			win.Update(func() {})
		}
	}()
}
