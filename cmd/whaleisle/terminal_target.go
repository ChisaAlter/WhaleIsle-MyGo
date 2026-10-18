package main

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo"
	desktopwebview "whaleisle/internal/desktop/webview"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/frame"
	termui "whaleisle/internal/ui/terminal"
)

// terminalTargetHandler is created on the UI thread for the currently painted
// native terminal. Its cwd is permanently captured from that instance, never
// substituted with a newly selected session or a shell-reported directory.
func (s *mainState) terminalTargetHandler(win *mygo.Window, m frame.Model, reveal func(string)) func(string) error {
	panel, sessions := m.Terminal, m.Store
	if panel == nil || sessions == nil {
		return nil
	}
	cwd := panel.Workspace()
	before := sessions.Snapshot()
	workspace := sessions.WorkspaceCwd()
	s.mu.Lock()
	host := s.bootStatus
	s.mu.Unlock()
	valid := func() bool {
		s.mu.Lock()
		current := !s.quitting && s.win == win && s.sessions == sessions && s.terminalPanel == panel &&
			s.bootStatus.Generation == host.Generation && s.bootStatus.State == host.State &&
			s.bootStatus.Endpoint.Origin == host.Endpoint.Origin && s.bootStatus.Endpoint.Cookie == host.Endpoint.Cookie
		s.mu.Unlock()
		if !current || win == nil || win.IsDestroyed() || panel.Workspace() != cwd || m.WorkspaceRoot != cwd {
			return false
		}
		now := sessions.Snapshot()
		if now.SelectedID != before.SelectedID || now.SelectedCwd != before.SelectedCwd || sessions.WorkspaceCwd() != workspace {
			return false
		}
		s.dshMu.Lock()
		client := s.dshClient
		s.dshMu.Unlock()
		return fmt.Sprintf("%p\x00%s", client, now.SelectedID) == m.FileBindingID
	}
	return func(selection string) error {
		return termui.OpenSelectionTarget(selection, cwd, termui.SelectionTargetActions{
			Valid: valid,
			OpenBrowser: func(raw string) error {
				normalized := browserpreviewui.NormalizeURL(raw)
				if normalized == "" {
					return errors.New("Invalid HTTP(S) URL")
				}
				if s.browserController == nil {
					s.browserController = s.newBrowserController(win)
				}
				s.browserController.Navigate(normalized)
				reveal("browser")
				return nil
			},
			OpenExternal: func(raw string) error {
				normalized := browserpreviewui.NormalizeURL(raw)
				if normalized == "" {
					return errors.New("Invalid HTTP(S) URL")
				}
				return desktopwebview.OpenExternal(normalized)
			},
			OpenPath: func(path string, line int) error {
				model := files.Model{
					BindingID: m.FileBindingID, SessionID: before.SelectedID, WorkspaceRoot: cwd,
					Tabs: m.FileTabs, SelectedFile: m.SelectedFile,
					Preview: m.FilePreview, PreviewErr: m.FilePreviewErr,
					Editing: m.FileEditing, EditDraft: m.FileEditDraft, EditStatus: m.FileEditStatus,
					StatNote: m.FileStatNote, EditViewMode: m.FileEditViewMode, EditSplit: m.FileEditSplit,
				}
				handled, err := files.OpenPresentedPath(model, path)
				if err != nil {
					return err
				}
				if !handled {
					return errors.New("Path is unavailable or outside this terminal's local workspace")
				}
				if line > 0 {
					files.RequestLine(model, line)
				} // Deferred by F05 until the text read completes.
				if m.FileTabs != nil {
					m.FileTabs.Plans.ShowFiles(before.SelectedID)
				}
				reveal("files")
				return nil
			},
		})
	}
}
