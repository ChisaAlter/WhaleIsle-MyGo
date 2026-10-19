package main

import (
	"context"
	"strings"
	"time"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/settings"
)

type settingsSkillCreateState struct {
	open, busy bool
	draft      settings.SkillCreateDraft
	err        string
	target     session.SkillTarget
	source     *session.Store
}

func (s *mainState) openSettingsSkillCreate() {
	s.mu.Lock()
	if s.settingsSkillCreate.busy {
		s.mu.Unlock()
		return
	}
	state := settingsSkillCreateState{open: true, source: s.sessions}
	state.draft = settings.SkillCreateDraft{Root: session.SkillRootUser, ModelInvocable: true, UserInvocable: true}
	if state.source == nil {
		state.err = "Runtime disconnected"
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
	return &settings.SkillCreateAccess{Open: create.open, Busy: create.busy, Draft: &create.draft, Error: create.err, OnSave: s.saveSettingsSkillCreate, OnCancel: s.cancelSettingsSkillCreate}
}

func (s *mainState) cancelSettingsSkillCreate() {
	s.mu.Lock()
	if !s.settingsSkillCreate.busy {
		s.settingsSkillCreate = settingsSkillCreateState{}
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
	create.busy, create.err = true, ""
	create.draft = draft
	source, target, win := create.source, create.target, s.win
	target.Root = draft.Root
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		fields := session.SkillEditorFields{Description: draft.Description, WhenToUse: draft.WhenToUse, Content: draft.Content, Groups: strings.Split(draft.Groups, ","), ModelInvocable: draft.ModelInvocable, UserInvocable: draft.UserInvocable}
		err := source.CreateSkillInventoryFields(ctx, target, draft.Name, fields)
		cancel()
		s.mu.Lock()
		s.settingsSkillCreate.busy = false
		if err != nil {
			s.settingsSkillCreate.err = err.Error()
		} else {
			s.settingsSkillCreate = settingsSkillCreateState{}
			s.settingsSkillsLastAction = "Skill created: " + draft.Name
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
