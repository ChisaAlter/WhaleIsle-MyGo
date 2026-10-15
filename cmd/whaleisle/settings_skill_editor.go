package main

import (
	"context"
	"strings"
	"time"

	"whaleisle/internal/store/session"
)

func (s *mainState) cancelSettingsSkillEdit() {
	s.mu.Lock()
	if s.settingsSkillEditorSaving {
		s.mu.Unlock()
		return
	}
	s.settingsSkillInspectVersion++
	s.settingsSkillEditTarget = session.SkillEditTarget{}
	s.settingsSkillEditWritable = false
	s.settingsSkillEditName = ""
	s.settingsSkillEditDesc = ""
	s.settingsSkillEditWhen = ""
	s.settingsSkillEditContent = ""
	s.settingsSkillEditGroups = ""
	s.settingsSkillsDetail = ""
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) inspectSettingsSkill(name string) {
	s.mu.Lock()
	if s.settingsSkillEditorSaving {
		s.mu.Unlock()
		return
	}
	source, win := s.sessions, s.win
	s.settingsSkillInspectVersion++
	ticket := s.settingsSkillInspectVersion
	s.settingsSkillEditName = strings.TrimSpace(name)
	s.settingsSkillsDetail = "Loading skill…"
	s.settingsSkillEditWritable = false
	s.mu.Unlock()
	if source == nil {
		return
	}
	target, err := source.CaptureCatalogSkillEdit(strings.TrimSpace(name))
	if err != nil {
		s.mu.Lock()
		s.settingsSkillsDetail = err.Error()
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
		return
	}
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		loaded, err := source.LoadCatalogSkillEdit(ctx, target)
		s.mu.Lock()
		if s.settingsSkillInspectVersion != ticket || s.sessions != source {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.settingsSkillsDetail = err.Error()
		} else {
			fields := loaded.Fields()
			s.settingsSkillEditTarget = loaded
			s.settingsSkillEditName = strings.TrimSpace(name)
			s.settingsSkillEditDesc = fields.Description
			s.settingsSkillEditWhen = fields.WhenToUse
			s.settingsSkillEditContent = fields.Content
			s.settingsSkillEditGroups = strings.Join(fields.Groups, ", ")
			s.settingsSkillEditModel = fields.ModelInvocable
			s.settingsSkillEditUser = fields.UserInvocable
			s.settingsSkillEditWritable = loaded.Writable()
			s.settingsSkillsDetail = loaded.Label()
			if !loaded.Writable() {
				body := fields.Content
				if len(body) > 800 {
					body = body[:800] + "…"
				}
				s.settingsSkillsDetail += "\n" + body
			}
		}
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}()
}

func (s *mainState) saveSettingsSkillEdit() {
	s.mu.Lock()
	if s.settingsSkillEditorSaving || !s.settingsSkillEditWritable {
		s.mu.Unlock()
		return
	}
	source, win := s.sessions, s.win
	target := s.settingsSkillEditTarget
	ticket := s.settingsSkillInspectVersion
	fields := session.SkillEditorFields{Description: s.settingsSkillEditDesc, WhenToUse: s.settingsSkillEditWhen, Content: s.settingsSkillEditContent, Groups: strings.Split(s.settingsSkillEditGroups, ","), ModelInvocable: s.settingsSkillEditModel, UserInvocable: s.settingsSkillEditUser}
	if source == nil {
		s.mu.Unlock()
		return
	}
	s.settingsSkillEditorSaving = true
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := source.SaveCatalogSkillEditor(ctx, target, fields)
		cancel()
		s.mu.Lock()
		s.settingsSkillEditorSaving = false
		if s.settingsSkillInspectVersion != ticket || s.sessions != source {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.settingsSkillsLastAction = err.Error()
		} else {
			s.settingsSkillsLastAction = "Skill saved: " + s.settingsSkillEditName
			s.settingsSkillEditWritable = false
			s.settingsSkillEditName = ""
			s.settingsSkillEditTarget = session.SkillEditTarget{}
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
