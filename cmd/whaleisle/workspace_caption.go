package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/frame"
)

// Global main panels retain AppFrame's terminal drawer and main-view cwd. The
// right dock has no active Session surface here, so its toggle has no target.
func (s *mainState) bindGlobalWorkspace(win *mygo.Window, m *frame.Model, root string, dark bool) {
	s.ensureTerminal(root, dark)
	s.mu.Lock()
	m.Terminal, m.TerminalShellsNote = s.terminalPanel, s.terminalShellsNote
	m.FileTabs, m.SelectedFile = &s.fileTabs, &s.selectedFile
	m.FilePreview, m.FilePreviewErr = &s.filePreview, &s.filePreviewErr
	m.FileEditing, m.FileEditDraft, m.FileEditStatus = &s.fileEditing, &s.fileEditDraft, &s.fileEditStatus
	m.FileStatNote, m.FileEditViewMode, m.FileEditSplit = &s.fileStatNote, &s.fileEditViewMode, &s.fileEditSplit
	s.mu.Unlock()
	invalidate := func() {
		if win != nil {
			win.Invalidate()
		}
	}
	m.OnToggleTerminal = func() { s.toggleTerminal(root, dark); invalidate() }
	m.OnRestartTerminal = func() { s.restartTerminal(root, dark); invalidate() }
	m.OnCloseTerminal = func() {
		s.mu.Lock()
		s.terminalSessions.CloseWorkspace(root)
		s.terminalPanel = nil
		s.terminalVisible, s.layout.Bottom = false, 0
		s.mu.Unlock()
		invalidate()
	}
	if m.Store != nil {
		snap := m.Store.Snapshot()
		s.dshMu.Lock()
		client := s.dshClient
		s.dshMu.Unlock()
		m.FileBindingID = fmt.Sprintf("%p\x00%s", client, snap.SelectedID)
		s.mu.Lock()
		draft := s.composerDraftLocked(snap.SelectedID)
		s.mu.Unlock()
		m.CanAddSelectionToChat = snap.SelectedID != "" && !draft.Sending
		m.OnAddSelectionToChat = func(text string) error { return appendSelectionToComposer(m.Store, snap.SelectedID, draft, text) }
		m.OnOpenTerminalTarget = s.terminalTargetHandler(win, *m, func(tab string) {
			s.mu.Lock()
			s.filesPaneOpen, s.rightPaneTab = true, tab
			s.mu.Unlock()
			invalidate()
		})
	}
}

func (s *mainState) observeWorkspaceOpenInApp() *frame.WorkspaceOpenInAppState {
	s.mu.Lock()
	if s.workspaceOpenInApp == nil {
		store := s.cfgStore
		choice := ""
		if store != nil {
			choice = store.Get().GoOpenInAppChoice
		}
		s.workspaceOpenInApp = frame.NewWorkspaceOpenInAppState(choice, func(choice string) error {
			if store == nil {
				return errors.New("desktop preferences are unavailable")
			}
			return store.Update(func(cfg *config.Config) { cfg.GoOpenInAppChoice = choice })
		})
	}
	value, sessions := s.workspaceOpenInApp, s.sessions
	s.mu.Unlock()
	value.ObserveHost(sessions)
	return value
}

func (s *mainState) closeWorkspaceOpenInApp() {
	s.mu.Lock()
	value := s.workspaceOpenInApp
	s.mu.Unlock()
	if value != nil {
		value.Close()
	}
}

func managedCaptionSession(snapshot session.ViewState) bool {
	for _, item := range snapshot.Items {
		if item.SessionID == snapshot.SelectedID {
			return item.Presentation != nil && item.Presentation.Composer == "managed"
		}
	}
	return false
}

func (s *mainState) bindWorkspaceTrailing(win *mygo.Window, m *frame.Model, badge *chrome.GitBadge, status *desktopgit.RepositoryStatus, busy bool, root, message string) {
	if m == nil || m.Store == nil {
		return
	}
	owner, snapshot := m.Store, m.Store.Snapshot()
	generation, selected, cwd := snapshot.ComposerHostGeneration, snapshot.SelectedID, snapshot.SelectedCwd
	current := func() bool {
		latest := owner.Snapshot()
		return latest.ComposerHostGeneration == generation && latest.SelectedID == selected && latest.SelectedCwd == cwd
	}
	s.mu.Lock()
	settings := s.settings
	report, reportRoot := s.gitPRReport, s.gitPRRoot
	statusCurrent := s.gitCheckedRoot == root && s.gitRepositoryStatus == status
	terminalOpen, surfacesOpen := s.terminalVisible, s.filesPaneOpen
	s.mu.Unlock()
	if m.WorkspacePage {
		surfacesOpen = false
	}
	hideGit := false
	if settings != nil {
		for _, namespace := range settings.Snapshot().Namespaces {
			if namespace.NS != "ui-git" {
				continue
			}
			var preference struct {
				TitlebarGit *bool `json:"titlebarGit"`
			}
			if json.Unmarshal(namespace.Value, &preference) == nil && preference.TitlebarGit != nil {
				hideGit = !*preference.TitlebarGit
			}
			break
		}
	}
	changeRequestOpen := reportRoot == root && report.Current != nil && strings.EqualFold(report.Current.State, "OPEN")
	diffModel := diffsurface.Model{Workbench: m.GitWorkbench, WorkspaceRoot: root, ThemeID: m.ThemeID, SystemDark: m.SystemDark, CaptionAdmission: current}
	preferences := m.TitlebarPreferences
	toggleSurfaces, toggleTerminal := m.OnToggleTools, m.OnToggleTerminal
	m.WorkspaceTrailing = func(c *ui.Context, density string) {
		model := chrome.WorkspaceTrailingModel{
			Git: badge, Status: status, WorkspaceRoot: cwd, Density: density,
			GitLoaded: status != nil && statusCurrent && cwd == root, GitBusy: busy || m.GitWorkbench != nil && m.GitWorkbench.Operation != nil,
			ManagedSession: managedCaptionSession(snapshot), HideGit: hideGit, ChangeRequestOpen: changeRequestOpen,
			ShowTerminal: !preferences.HideTerminalToggle, TerminalAvailable: len(snapshot.Workspaces) != 0,
			TerminalOpen: terminalOpen, ShowSurfaces: !preferences.HideSurfacesToggle, SurfacesOpen: surfacesOpen,
			OnToggleTerminal: toggleTerminal, OnToggleSurfaces: toggleSurfaces,
			OnGitMenuOpen: func() {
				if current() {
					s.refreshGit(root)
					s.refreshPR(root, false)
				}
			},
			OnGitAction: func(action string) {
				if !current() || m.GitWorkbench == nil {
					return
				}
				if action == "pull" || action == "open_pr" {
					if badge != nil {
						if action == "pull" && badge.OnPull != nil {
							badge.OnPull()
						}
						if action == "open_pr" && badge.OnPRView != nil {
							badge.OnPRView()
						}
					}
					return
				}
				if !diffsurface.CaptionAction(c, diffModel, message, action, m.WorkspacePage) {
					return
				}
				if action == "init" || m.WorkspacePage {
					if win != nil {
						win.Invalidate()
					}
					return
				}
				s.mu.Lock()
				s.filesPaneOpen, s.rightPaneTab, s.gitPROpen = true, "diff", false
				s.mu.Unlock()
				if win != nil {
					win.Invalidate()
				}
			},
			RenderBranches: func(c *ui.Context, anchor ui.Element, open *bool) {
				if !current() {
					*open = false
				}
				diffsurface.CaptionBranches(c, diffModel, anchor, open)
			},
		}
		chrome.WorkspaceTrailing(c, model)
		// Review retirement is independent of the visible Git button. A global
		// modal stays the same seat when navigating back to an open diff.
		diffsurface.CaptionReview(c, diffModel)
		if !m.RightPaneOpen || m.RightPaneTab == nil || *m.RightPaneTab != "diff" || m.GitWorkbench.CaptionReviewOpen() {
			diffsurface.CaptionActivity(c, diffModel)
		}
	}
}
