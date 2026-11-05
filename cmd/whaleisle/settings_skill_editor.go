package main

import (
	"context"
	"fmt"
	"strings"
	"time"
	"whaleisle/internal/ui/i18n"

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
	s.settingsSkillEditGroups = nil
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
	s.settingsSkillsDetail = i18n.T(i18n.KeyCmdSkillLoading)
	s.settingsSkillsLastAction = ""
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
			s.settingsSkillEditGroups = append([]string(nil), fields.Groups...)
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

func (s *mainState) reloadSettingsSkillEdit() {
	s.mu.Lock()
	if s.settingsSkillEditorSaving || !s.settingsSkillEditWritable || s.sessions == nil {
		s.mu.Unlock()
		return
	}
	source, target, win := s.sessions, s.settingsSkillEditTarget, s.win
	s.settingsSkillInspectVersion++
	ticket := s.settingsSkillInspectVersion
	s.settingsSkillEditorSaving = true
	s.settingsSkillsLastAction = ""
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		loaded, err := source.ReloadCatalogSkillEdit(ctx, target)
		s.mu.Lock()
		if s.settingsSkillInspectVersion != ticket || s.sessions != source {
			s.mu.Unlock()
			return
		}
		s.settingsSkillEditorSaving = false
		if err != nil {
			s.settingsSkillsLastAction = err.Error()
		} else {
			s.settingsSkillEditTarget = loaded
			fields := loaded.Fields()
			s.settingsSkillsDetail = loaded.Label() + "\n\n" + pluginCopy("已重新核对当前文件，下面保留你的草稿。再次保存将写入这份草稿。当前文件：", "The current file was rechecked. Your draft is retained below; Save writes that draft. Current file:") +
				fmt.Sprintf("\n%s: %s\n%s: %s\n%s: %s\n%s: %t · %s: %t\n\n%s", pluginCopy("描述", "Description"), fields.Description, pluginCopy("使用时机", "When to use"), fields.WhenToUse, pluginCopy("分组", "Groups"), strings.Join(fields.Groups, ", "), pluginCopy("模型可调用", "Model invocable"), fields.ModelInvocable, pluginCopy("用户可调用", "User invocable"), fields.UserInvocable, fields.Content)
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
	fields := session.SkillEditorFields{Description: s.settingsSkillEditDesc, WhenToUse: s.settingsSkillEditWhen, Content: s.settingsSkillEditContent, Groups: append([]string(nil), s.settingsSkillEditGroups...), ModelInvocable: s.settingsSkillEditModel, UserInvocable: s.settingsSkillEditUser}
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
			s.settingsSkillsLastAction = fmt.Sprintf(i18n.T(i18n.KeyCmdSkillSaved), s.settingsSkillEditName)
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
