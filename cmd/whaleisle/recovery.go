package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/egoist/mygo"
	"whaleisle/internal/recovery"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
)

func resumeText(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
}

// openResume runs after the ordinary layout defaults, before starting the Host.
// Loading a resume file never writes an editor buffer or executes a command.
func (s *mainState) openResume(dir string) {
	store, err := recovery.Open(dir)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.resumeError = err.Error()
		return
	}
	s.resumeStore = store
	saved := store.Snapshot()
	if _, err := os.Stat(store.Path()); err != nil {
		return
	}
	s.resumePendingSession = saved.SelectedSession
	s.resumeWorkspace = saved.Workspace
	s.composerDrafts = map[string]*composer.DraftState{}
	for key, raw := range saved.Drafts {
		draft, err := composer.RestoreDraftChecked(raw)
		if err != nil {
			s.resumeError = fmt.Sprintf("%s: %v", store.Path(), err)
			return
		}
		s.composerDrafts[key] = draft
	}
	if err := s.fileTabs.Restore(saved.Files); err != nil {
		s.resumeError = fmt.Sprintf("%s: %v", store.Path(), err)
		return
	}
	l := saved.Layout
	if finiteSize(l.TranscriptWidth, 1, 100000) {
		s.layout.TranscriptWidth = l.TranscriptWidth
	}
	// Older snapshots have no expanded preference: retain the configured width.
	s.layout.SidebarExpanded = s.layout.Sidebar
	if finiteSize(l.SidebarExpanded, 264, 420) {
		s.layout.SidebarExpanded = l.SidebarExpanded
	}
	if l.Sidebar == 0 {
		s.layout.Sidebar = 0
	} else if finiteSize(l.Sidebar, 264, 420) {
		s.layout.Sidebar = l.Sidebar
	}
	if finiteSize(l.Right, 300, 3000) {
		s.layout.Right = l.Right
	}
	if finiteSize(l.Bottom, 80, 1600) {
		s.terminalSavedBottom = l.Bottom
	}
	if l.TerminalOpen {
		s.terminalVisible = true
		s.layout.Bottom = s.terminalSavedBottom
		s.terminalResumePending = true
	}
	if finiteSize(l.RightTree, 60, 2000) {
		s.layout.RightTree = l.RightTree
	}
	s.filesPaneOpen = l.RightOpen
	if l.RightTab == "files" || l.RightTab == "diff" || l.RightTab == "agents" || l.RightTab == "browser" || l.RightTab == "host-terminal" {
		s.rightPaneTab = l.RightTab
	}
	if l.ChatTab == "chat" || l.ChatTab == "trajectory" {
		s.chatPaneTab = l.ChatTab
	}
	s.planBoardOpen = l.PlanOpen
	s.todoPanelCollapsed = l.TodoCollapsed
	if err := s.restoreWorkspaceSessions(saved); err != nil {
		s.resumeError = fmt.Sprintf("%s: %v", store.Path(), err)
		return
	}
	// Terminal height is remembered, but a previous shell command is never replayed.
}
func finiteSize(v, min, max float32) bool {
	return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) && v >= min && v <= max
}

// restoreResumeSelection waits for a successful list on the saved workspace.
// It does not send a prompt or create a missing session.
func (s *mainState) restoreResumeSelection() {
	s.mu.Lock()
	id, workspace, sessions := s.resumePendingSession, s.resumeWorkspace, s.sessions
	current := ""
	if s.cfgStore != nil {
		current = s.cfgStore.Get().Workspace
	}
	s.mu.Unlock()
	if id == "" || sessions == nil {
		return
	}
	snap := sessions.Snapshot()
	if !snap.ClientReady || snap.ListLoading || snap.ListErr != "" {
		return
	}
	// The initial empty snapshot arrives before RefreshList is scheduled.
	if len(snap.Items) == 0 {
		return
	}
	s.mu.Lock()
	s.resumePendingSession = ""
	s.mu.Unlock()
	if workspace != current {
		sessions.SetLastAction(resumeText("恢复的草稿已保留；工作区已改变，未自动打开旧会话。", "Recovered drafts were kept; the workspace changed, so the old session was not opened."))
		return
	}
	for _, item := range snap.Items {
		if item.SessionID == id {
			sessions.Select(id)
			return
		}
	}
	sessions.SetLastAction(resumeText("上次会话不在当前列表中；草稿已保留。", "The previous session is not in the current list; its draft was kept."))
}

func (s *mainState) saveResume() error {
	s.mu.Lock()
	interactions := s.interactions
	checkpoint := s.resumeCheckpoint
	win := s.win
	s.mu.Unlock()
	if checkpoint != nil {
		checkpoint.pause()
		defer checkpoint.resume()
	}
	// Flush outside mainState.mu: the draft writer owns a separate serial lock
	// and its resume callback must finish before the final combined snapshot.
	if interactions != nil {
		if err := interactions.FlushQuestionDraftPersistence(); err != nil {
			return err
		}
	}
	var next recovery.State
	var store *recovery.Store
	var err error
	capture := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		store = s.resumeStore
		next, err = s.captureResumeOwnedLocked()
	}
	// Protected quit can originate on a worker. Mutable native editor/pane
	// state is always captured on its UI owner, never by that worker.
	if win != nil {
		mygo.RunOnMain(capture)
	} else {
		capture() // Non-UI callers own their isolated state directly.
	}
	if err != nil {
		return err
	}
	err = store.Update(func(v *recovery.State) { applyResumeOwned(v, next) })
	if err == nil && checkpoint != nil {
		checkpoint.acceptFinal(next)
	}
	return err
}

// Called only on the native UI thread (or an isolated non-UI owner), with mu
// held. Each serialized payload is detached before a disk worker receives it.
func (s *mainState) captureResumeOwnedLocked() (recovery.State, error) {
	if s.resumeError != "" {
		return recovery.State{}, fmt.Errorf("%s", s.resumeError)
	}
	if s.resumeStore == nil {
		return recovery.State{}, fmt.Errorf("resume storage is not available")
	}
	next := recovery.State{Version: 1}
	if s.cfgStore != nil {
		next.Workspace = s.cfgStore.Get().Workspace
	}
	if s.sessions != nil {
		next.SelectedSession = s.sessions.Snapshot().SelectedID
	}
	if s.resumePendingSession != "" {
		next.SelectedSession = s.resumePendingSession
	}
	bottom := s.terminalSavedBottom
	if s.terminalVisible && s.layout.Bottom > 80 {
		bottom = s.layout.Bottom
	}
	next.Layout = recovery.Layout{TranscriptWidth: s.layout.TranscriptWidth, Sidebar: s.layout.Sidebar, SidebarExpanded: s.layout.SidebarExpanded, Right: s.layout.Right, Bottom: bottom, TerminalOpen: s.terminalVisible, RightTree: s.layout.RightTree, RightOpen: s.filesPaneOpen, RightTab: s.rightPaneTab, ChatTab: s.chatPaneTab, PlanOpen: s.planBoardOpen, TodoCollapsed: s.todoPanelCollapsed}
	next.Layout.WorkspacePanes = s.exportWorkspaceSessionsLocked()
	next.Drafts = make(map[string]json.RawMessage, len(s.composerDrafts))
	for key, draft := range s.composerDrafts {
		if draft != nil {
			next.Drafts[key] = draft.SaveDraft()
		}
	}
	files, err := s.fileTabs.Export()
	if err != nil {
		return recovery.State{}, err
	}
	next.Files = files
	return next, nil
}

// Question drafts have their own serialized writer. A combined UI checkpoint
// may never overwrite a newer question write with an earlier captured field.
func applyResumeOwned(v *recovery.State, next recovery.State) {
	v.Workspace, v.SelectedSession, v.Layout, v.Drafts, v.Files = next.Workspace, next.SelectedSession, next.Layout, next.Drafts, next.Files
}

// A normal quit must not silently discard an unsaved editor/composer. Default
// action is to keep the window open so the user can resolve the disk problem.
func (s *mainState) saveResumeBeforeClose(win *mygo.Window) bool {
	s.mu.Lock()
	hasFontEditor := s.appearanceEditor != nil
	s.mu.Unlock()
	if hasFontEditor {
		mygo.RunOnMain(func() { s.flushInactiveGeneralFontEdits(true) })
	}
	if err := s.saveResume(); err != nil {
		result, dialogErr := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning,
			Title:   resumeText("未能保存恢复状态", "Could not save resume state"),
			Message: strings.Join([]string{err.Error(), resumeText("原文件未被覆盖。请保留窗口、备份草稿并解决磁盘或多实例冲突后重试。直接退出可能丢失本次未保存内容。", "The original file was not overwritten. Keep this window open, back up drafts, and resolve disk or other-instance conflicts before retrying. Quitting now may lose unsaved changes.")}, "\n\n"),
			Buttons: []string{resumeText("保留窗口", "Keep window open"), resumeText("不保存并退出", "Quit without saving")}, DefaultButton: 0, CancelButton: 0})
		return dialogErr == nil && result.Button == 1
	}
	return true
}
