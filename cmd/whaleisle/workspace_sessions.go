package main

import (
	"encoding/json"
	"fmt"

	"whaleisle/internal/recovery"
	"whaleisle/internal/ui/frame"
)

// The configured workspace scopes session IDs across different Host projects.
// Unlike the binding token, this key is stable across reconnects and restarts.
func workspaceSessionKey(scope, workspace, session string) string {
	raw, _ := json.Marshal([3]string{scope, workspace, session})
	return string(raw)
}

type workspaceSessionLayout struct {
	restoreReadScope string
	restoreReadRoot  string
	Panes            *frame.WorkspacePanes `json:"-"`
	Saved            json.RawMessage       `json:"panes"`
	Open             bool                  `json:"open"`
	Tab              string                `json:"tab"`
	Right            float32               `json:"right"`
	Tree             float32               `json:"tree"`
}

type workspaceSessionLayouts struct {
	Version   int                                `json:"version"`
	BySession map[string]*workspaceSessionLayout `json:"bySession"`
	active    string
}

func (s *mainState) workspaceProjectionLocked() frame.Model {
	return frame.Model{Sizes: &s.layout, FileTabs: &s.fileTabs,
		FileExpanded: &s.fileExpanded, SelectedFile: &s.selectedFile,
		FileFilter: &s.fileFilter, FilePreview: &s.filePreview,
		FilePreviewErr: &s.filePreviewErr, FileEditDraft: &s.fileEditDraft,
		FileEditStatus: &s.fileEditStatus, FileStatNote: &s.fileStatNote,
		FileEditing: &s.fileEditing, FileEditViewMode: &s.fileEditViewMode,
		FileEditSplit: &s.fileEditSplit, FilePreviewList: &s.filePreviewList,
		RightPaneTab: &s.rightPaneTab, DiffPath: &s.diffPath,
		DiffText: &s.diffText, DiffErr: &s.diffErr, GitWorkbench: &s.gitWorkbench, WorkspaceRoot: s.gitWorkbench.Root}
}

func (s *mainState) captureWorkspaceSessionLocked() {
	if s.workspaceSessions.BySession == nil || s.workspaceSessions.active == "" {
		return
	}
	entry := s.workspaceSessions.BySession[s.workspaceSessions.active]
	if entry == nil {
		return
	}
	entry.Panes = s.layout.Workspace
	entry.Panes.CaptureSessionProjection(s.workspaceProjectionLocked())
	entry.Open, entry.Tab = s.filesPaneOpen, s.rightPaneTab
	entry.Right, entry.Tree = s.layout.Right, s.layout.RightTree
}

// Called under state.mu before frame-local adapters are copied or rebound.
func (s *mainState) selectWorkspaceSessionLocked() {
	scope, workspace, session := "", "", ""
	if s.cfgStore != nil {
		workspace = s.cfgStore.Get().Workspace
	}
	if s.sessions != nil {
		session = s.sessions.Snapshot().SelectedID
		scope = s.sessions.WorkspaceLayoutScope()
	}
	s.restoreSavedAgentReadersLocked(scope, workspace)
	s.switchWorkspaceSessionLocked(scope, workspace, session)
}

func (s *mainState) switchWorkspaceSessionLocked(scope, workspace, session string) {
	if s.layout.Workspace == nil {
		s.layout.Workspace = &frame.WorkspacePanes{}
	}
	key := workspaceSessionKey(scope, workspace, session)
	layouts := &s.workspaceSessions
	if layouts.BySession == nil {
		layouts.Version = 1
		layouts.BySession = map[string]*workspaceSessionLayout{key: {Panes: s.layout.Workspace}}
		layouts.active = key
		s.captureWorkspaceSessionLocked()
		return
	}
	if layouts.active == key {
		return
	}
	s.captureWorkspaceSessionLocked()
	s.layout.Workspace.SuspendSession()
	s.hostTerminalUI.Observe(&s.hostTerminals, "", false)
	entry := layouts.BySession[key]
	if entry == nil && scope != "" {
		legacyKey := workspaceSessionKey("legacy", workspace, session)
		if legacy := layouts.BySession[legacyKey]; legacy != nil {
			entry = legacy
			layouts.BySession[key] = entry
			delete(layouts.BySession, legacyKey)
		}
	}
	if entry == nil {
		entry = &workspaceSessionLayout{Panes: &frame.WorkspacePanes{}, Tab: "files", Right: s.layout.Right, Tree: s.layout.RightTree}
		layouts.BySession[key] = entry
	}
	layouts.active = key
	s.layout.Workspace = entry.Panes
	s.filesPaneOpen, s.rightPaneTab = entry.Open, entry.Tab
	s.layout.Right, s.layout.RightTree = entry.Right, entry.Tree
	s.layout.Workspace.PublishSessionProjection(s.workspaceProjectionLocked())
	s.fileOpenWithFor, s.fileOpenWithApps = "", nil
}

func (s *mainState) restoreWorkspaceSessions(saved recovery.State) error {
	raw := saved.Layout.WorkspacePanes
	var envelope struct {
		BySession json.RawMessage `json:"bySession"`
	}
	if len(raw) != 0 {
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return err
		}
	}
	layouts := workspaceSessionLayouts{Version: 1, BySession: map[string]*workspaceSessionLayout{}}
	if len(envelope.BySession) != 0 {
		if err := json.Unmarshal(raw, &layouts); err != nil {
			return err
		}
		if layouts.BySession == nil {
			return fmt.Errorf("missing session layouts")
		}
		if layouts.Version != 1 {
			return fmt.Errorf("unsupported session layout version %d", layouts.Version)
		}
		for key, entry := range layouts.BySession {
			if entry == nil {
				return fmt.Errorf("invalid session layout %q", key)
			}
			panes, err := frame.RestoreWorkspacePanes(entry.Saved)
			if err != nil {
				return fmt.Errorf("session layout %q: %w", key, err)
			}
			if panes == nil {
				panes = &frame.WorkspacePanes{}
			}
			entry.Panes = panes
			var route [3]string
			if json.Unmarshal([]byte(key), &route) == nil && route[0] != "" && route[0] != "legacy" {
				entry.restoreReadScope, entry.restoreReadRoot = route[0], route[2]
			}
			if !finiteSize(entry.Right, 300, 3000) {
				entry.Right = s.layout.Right
			}
			if !finiteSize(entry.Tree, 60, 2000) {
				entry.Tree = s.layout.RightTree
			}
		}
	} else {
		panes, err := frame.RestoreWorkspacePanes(raw)
		if err != nil {
			return err
		}
		if panes == nil || panes.Root == nil {
			panes = &frame.WorkspacePanes{}
			// Pre-pane snapshots stored editor drafts only in State.Files.
			panes.CaptureSessionProjection(s.workspaceProjectionLocked())
		}
		l := saved.Layout
		layouts.BySession[workspaceSessionKey("legacy", saved.Workspace, saved.SelectedSession)] = &workspaceSessionLayout{Panes: panes, Open: l.RightOpen, Tab: l.RightTab, Right: s.layout.Right, Tree: s.layout.RightTree}
	}
	s.workspaceSessions = layouts
	// No layout is active until the actual selected session is known. The
	// recovered session's tree cannot leak into the temporary empty selection.
	s.layout.Workspace = &frame.WorkspacePanes{}
	return nil
}

func (s *mainState) exportWorkspaceSessionsLocked() json.RawMessage {
	if s.workspaceSessions.BySession == nil {
		return s.layout.Workspace.Export()
	}
	s.captureWorkspaceSessionLocked()
	for _, entry := range s.workspaceSessions.BySession {
		entry.Saved = entry.Panes.Export()
	}
	raw, _ := json.Marshal(s.workspaceSessions)
	return raw
}

func (s *mainState) disposeWorkspaceSessions() {
	if s.workspaceSessions.BySession == nil {
		s.layout.Workspace.DisposeContent()
		return
	}
	for _, entry := range s.workspaceSessions.BySession {
		entry.Panes.DisposeContent()
	}
}

// Only records that arrived with a durable Host scope are eligible. Legacy
// layout migration cannot retroactively prove the saved child belonged here.
func (s *mainState) restoreSavedAgentReadersLocked(scope, workspace string) {
	if s.sessions == nil || scope == "" {
		return
	}
	for key, entry := range s.workspaceSessions.BySession {
		if entry == nil || entry.restoreReadScope != scope || entry.restoreReadRoot == "" {
			continue
		}
		var route [3]string
		if json.Unmarshal([]byte(key), &route) != nil || route[0] != scope || route[1] != workspace || route[2] != entry.restoreReadRoot {
			continue
		}
		entry.Panes.RestoreRetainedAgentReaders(s.sessions, scope, entry.restoreReadRoot)
	}
}
