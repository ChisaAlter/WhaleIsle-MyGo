package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/workspacepicker"
)

type workspaceAdoption struct {
	sourceID        string
	draft           *composer.DraftState
	snapshot        json.RawMessage
	revision        uint64
	carry           bool
	panelGeneration uint64
}

func (s *mainState) captureWorkspaceAdoption(sessions *session.Store) workspaceAdoption {
	snap := sessions.Snapshot()
	s.mu.Lock()
	d := s.composerDraftLocked(snap.SelectedID)
	op := workspaceAdoption{sourceID: snap.SelectedID, draft: d, snapshot: append(json.RawMessage(nil), d.SaveDraft()...), revision: workspaceSessionDraftRevision(d), panelGeneration: s.assistantConversationGeneration,
		carry: (snap.SelectedID == "" || snap.SelectedSummaryBlank && !snap.HasConversationContent()) && !d.Sending && d.Preparing == 0 && d.EditSeq == 0 && d.GroupReply == nil}
	s.mu.Unlock()
	return op
}

// On a successful directory commit, stage an eligible blank draft in the new
// Hero immediately. Creation failures leave that same draft visible for retry.
func (s *mainState) finishWorkspaceAdoption(win *mygo.Window, sessions *session.Store, row session.WorkspaceRow, adoption workspaceAdoption) {
	finish := func() {
		if sessions.SelectedSessionID() != "" || sessions.WorkspaceCwd() != row.Path {
			return
		}
		carried := false
		s.mu.Lock()
		if s.assistantConversationGeneration != adoption.panelGeneration || s.sessions != nil && s.sessions != sessions {
			s.mu.Unlock()
			return
		}
		if adoption.carry && s.composerDrafts[adoption.sourceID] == adoption.draft && workspaceSessionDraftRevision(adoption.draft) == adoption.revision && bytes.Equal(adoption.draft.SaveDraft(), adoption.snapshot) {
			if adoption.sourceID == "" || !workspaceSessionDraftOccupied(s.composerDrafts[""]) {
				carried = true
				s.composerDrafts[""] = adoption.draft
				adoption.draft.TargetID = ""
				if adoption.sourceID != "" {
					delete(s.composerDrafts, adoption.sourceID)
				}
			}
		}
		s.mu.Unlock()
		s.revealMainConversation()
		s.workspaceSelectionCommitted(win, sessions, row.Path)
		if win == nil {
			return
		}
		op, err := s.captureWorkspaceSessionOpen(win, sessions, sessions.ComposerHostGeneration(), row.WorkspaceID)
		if err != nil || op == nil {
			return
		}
		op.carryDraft = carried
		go func() {
			id, err := op.target.Create(context.Background())
			if !win.IsDestroyed() {
				win.Update(func() { op.finish(id, err) })
			}
		}()
	}
	if win != nil {
		if !win.IsDestroyed() {
			win.Update(finish)
		}
	} else {
		finish()
	}
}

// registerWorkspacePath is shared by local, native and remote directory flows.
func (s *mainState) registerWorkspacePath(win *mygo.Window, sessions *session.Store, path string) {
	if sessions == nil {
		return
	}
	adoption := s.captureWorkspaceAdoption(sessions)
	sessions.OpenWorkspacePicker()
	request := sessions.Snapshot().WorkspaceNavigation
	if request == nil || request.Mode != "pick" || request.Busy {
		return
	}
	sessions.RegisterWorkspace(context.Background(), request.Token, path, s.persistRegisteredWorkspace, func(row session.WorkspaceRow) {
		s.finishWorkspaceAdoption(win, sessions, row, adoption)
	})
}

// Saving config is the last fallible step before Store commits its selection.
// Do not acquire mainState.mu here: this callback runs under the Store lock.
func (s *mainState) persistRegisteredWorkspace(path string) error {
	if s.cfgStore == nil {
		return errors.New(i18n.T(i18n.KeyCmdWorkspaceSettingsUnavailable))
	}
	return s.cfgStore.Update(func(c *config.Config) { c.Workspace = path })
}
func (s *mainState) workspaceSelectionCommitted(win *mygo.Window, sessions *session.Store, path string) {
	s.mu.Lock()
	if sessions != nil && sessions.WorkspaceCwd() == path && sessions.SelectedSessionID() == "" {
		if home := s.composerDrafts[""]; home != nil && !home.Sending && home.Preparing == 0 {
			home.TargetID = ""
		}
	}
	s.mu.Unlock()
	s.revokePhoneGit()
	if win != nil {
		win.Update(func() {})
	}
}
func (s *mainState) workspaceNavigationView(c *ui.Context, win *mygo.Window, sessions *session.Store, themeID string, systemDark bool, onNoDirectory func()) {
	workspaceDeletionView(c, sessions, themeID, systemDark)
	workspacepicker.View(c, workspacepicker.Model{Store: sessions, ThemeID: themeID, SystemDark: systemDark, OnNoDirectory: onNoDirectory, DirectoryActions: s.workspaceDirectoryActions(win),
		Anchor: c.Resolve(s.workspacePickerAnchor),
		OnBrowse: func() {
			if request := sessions.Snapshot().WorkspaceNavigation; request != nil {
				sessions.CancelWorkspaceNavigation(request.Token)
			}
			s.openDirectoryPicker(sessions, win, "workspace")
		},
		OnChoose: func(token uint64, id string) {
			adoption := s.captureWorkspaceAdoption(sessions)
			if sessions.ChooseRegisteredWorkspace(token, id, s.persistRegisteredWorkspace) {
				s.finishWorkspaceAdoption(win, sessions, session.WorkspaceRow{WorkspaceID: id, Path: sessions.WorkspaceCwd()}, adoption)
			}
		},
		OnRegister: func(token uint64, path string) {
			adoption := s.captureWorkspaceAdoption(sessions)
			sessions.RegisterWorkspace(context.Background(), token, path, s.persistRegisteredWorkspace, func(row session.WorkspaceRow) { s.finishWorkspaceAdoption(win, sessions, row, adoption) })
		},
	})
}

func (s *mainState) projectDirectoryActions(win *mygo.Window) []workspacepicker.DirectoryAction {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	store, sessions := s.cfgStore, s.sessions
	f := &s.projectFlow
	available := client != nil && f.client == client && f.listing.Available
	active := map[string]bool{}
	for _, project := range f.listing.Projects {
		if project.Lifecycle != "archived" {
			active[projectDirectoryKey(project.CanonicalWorkingDirectory)] = true
		}
	}
	s.mu.Unlock()
	if !available || store != nil && !projectMountEnabled(store.Get(), projectDevelopmentOverride()) {
		return nil
	}
	controller := s.primaryProjectController()
	if sessions != nil {
		rows, _ := sessions.CachedWorkspaceChoices(client)
		for _, row := range rows {
			controller.projectDirectoryCheckReason(win, client, row.Path)
		}
	}
	s.mu.Lock()
	current := f.client == client && f.listing.Available
	refused := map[string]bool{}
	for key, check := range f.directoryChecks {
		refused[key] = check.reason != ""
	}
	s.mu.Unlock()
	return projectWorkspaceDirectoryActions(current, func(path string) (bool, bool) {
		key := projectDirectoryKey(path)
		return active[key], refused[key]
	}, func(path string) { s.createProjectFromDirectory(win, path) })
}

func workspaceDeletionView(c *ui.Context, sessions *session.Store, themeID string, systemDark bool) {
	if sessions == nil {
		return
	}
	deletion := sessions.SessionDeletion()
	if deletion.SessionID == "" {
		return
	}
	text := func(zh, en string) string {
		if i18n.Locale() == i18n.LocaleZH {
			return zh
		}
		return en
	}
	open := true
	colors := theme.Resolve(themeID).Active(systemDark)
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		chrome.DialogReveal(c, backdrop, panel)
		panel.Key("session-deletion-result").Width(420).MaxWidthPercent(94).Padding(24).Gap(16).Radius(24).Background(chrome.Color(colors.Field)).Children(func() {
			ui.Text(c, text("永久删除会话", "Permanently delete session")).FontSize(theme.InterfaceSize(17)).FontWeight(500)
			ui.Text(c, deletion.Title).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(14))
			ui.Text(c, sessionDeleteErrorText(deletion.Error, text)).FillWidth().MinHeight(40).FontSize(theme.InterfaceSize(13))
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				if ui.Button(c, text("关闭", "Close")).Disabled(deletion.Busy).Clicked() {
					sessions.DismissSessionDeletion(deletion.SessionID)
				}
				ui.Box(c).Grow(1)
				label := text("重试删除", "Retry deletion")
				if deletion.Busy {
					label = text("正在删除会话…", "Deleting session…")
				}
				if ui.Button(c, label).Disabled(deletion.Busy).Clicked() {
					sessions.DeleteSession(context.Background(), deletion.SessionID)
				}
			})
		})
	})
	if !open && !deletion.Busy {
		sessions.DismissSessionDeletion(deletion.SessionID)
	}
}

// sessionDeleteErrorText maps Host session/delete failures to the Electron
// delete.session.* copy. Unknown errors pass through untouched.
func sessionDeleteErrorText(raw string, text func(zh, en string) string) string {
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "live_unowned"), strings.Contains(lower, "cannot be safely"):
		return text("会话仍在运行且无法安全结束，请重试或重启后再删。", "The session is still running and cannot be stopped safely. Retry or restart, then delete again.")
	case strings.Contains(lower, "running"), strings.Contains(lower, "in progress") && !strings.Contains(lower, "delet"):
		return text("会话仍在运行，请先停止后再删除。", "The session is still running. Stop it before deleting.")
	case strings.Contains(lower, "not_archived"), strings.Contains(lower, "not archived"), strings.Contains(lower, "only archived"):
		return text("只能删除已归档的会话。", "Only archived sessions can be deleted.")
	case strings.Contains(lower, "not_found"), strings.Contains(lower, "not found"):
		return text("找不到该会话。", "The session could not be found.")
	case strings.Contains(lower, "partial"):
		return text("部分子会话已删除，但根会话仍在。请重试删除。", "Some child sessions were deleted but the root session remains. Retry the deletion.")
	case strings.Contains(lower, "incomplete"):
		return text("会话日志已删除，但归档状态未能完全更新。请刷新后检查。", "The session log was deleted but the archive state did not fully update. Refresh and check.")
	case strings.Contains(lower, "already"), strings.Contains(lower, "in progress"), strings.Contains(lower, "deleting"):
		return text("该会话正在删除中，请稍后再试。", "This session is already being deleted. Try again later.")
	}
	return raw
}
