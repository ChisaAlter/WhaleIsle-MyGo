package session

// SelectionSummary is the part of the view the shell compares on every render
// pass. Each field matches the Snapshot() field of the same name (Selected*
// describe the listed row of SelectedID), but it is read from the live state
// under the store lock without projecting or cloning the transcript, so it is
// cheap enough to call several times per frame.
type SelectionSummary struct {
	SelectedID                string
	SelectedCwd               string
	SelectedComposer          string
	ComposerHostGeneration    uint64
	ComposerSessionGeneration uint64
	WorkspaceHostGeneration   uint64
	ClientReady               bool
	ListLoading               bool
	ListErr                   string
	HasWorkspaces             bool
	// SelectedListed is false when SelectedID is empty or absent from the list.
	SelectedListed  bool
	SelectedTitle   string
	SelectedRunning bool
}

func (s *Store) SelectionSummary() SelectionSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := SelectionSummary{
		SelectedID:                s.selectedID,
		ComposerHostGeneration:    s.composerHostGeneration,
		ComposerSessionGeneration: s.followGeneration,
		WorkspaceHostGeneration:   s.eventsGeneration,
		ClientReady:               s.client != nil,
		ListLoading:               s.listLoading,
		HasWorkspaces:             len(s.workspaceMemberships) != 0,
	}
	if s.listErr != nil {
		out.ListErr = s.listErr.Error()
	}
	for _, it := range s.items {
		if it.SessionID == s.selectedID {
			out.SelectedCwd = it.Cwd
			if it.Presentation != nil {
				out.SelectedComposer = it.Presentation.Composer
			}
			if it.SessionID != "" {
				out.SelectedListed, out.SelectedTitle, out.SelectedRunning = true, it.Title, it.Running
			}
			break
		}
	}
	return out
}

// ComposerSessionGeneration returns the current follow generation without building a view.
func (s *Store) ComposerSessionGeneration() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.followGeneration
}

// PromptBusy reports whether a prompt is busy without building a view.
func (s *Store) PromptBusy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.promptBusy || s.composerPromptBusyLocked()
}

// HasListedItems reports whether the store has any listed sessions without building a view.
func (s *Store) HasListedItems() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items) > 0
}

// ClientReady reports whether a Host client is attached without building a view.
func (s *Store) ClientReady() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client != nil
}

// ComposerHostGeneration returns the current host generation without building a view.
func (s *Store) ComposerHostGeneration() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.composerHostGeneration
}

// WorkspaceHostGeneration returns the current workspace/events host generation without building a view.
func (s *Store) WorkspaceHostGeneration() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.eventsGeneration
}

// PlanHistoryBinding returns the current plan history binding without building a view.
func (s *Store) PlanHistoryBinding() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.planHistoryBinding
}

// ScratchCwd returns the current scratch working directory without building a view.
func (s *Store) ScratchCwd() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scratchCwd
}

// SelectedModel returns the currently selected model label without building a view.
func (s *Store) SelectedModel() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	selection, known := s.currentModelSelectionLocked()
	if !known {
		return ""
	}
	return labelForSelection(selection, s.modelOptions)
}

// TurnFilesScope returns the current turn-file scope without cloning turn-file results or transcript state.
func (s *Store) TurnFilesScope() TurnFileScope {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turnFilesScopeLocked()
}

// HasListedSession reports whether id is present in the current session list without cloning the view.
func (s *Store) HasListedSession(id string) bool {
	if id == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		if it.SessionID == id {
			return true
		}
	}
	return false
}

// ListedItems returns a shallow copy of the current session list without projecting the transcript.
func (s *Store) ListedItems() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Item(nil), s.items...)
}

// WorkspaceNavigationPending reports whether a workspace navigation request is
// recorded, without projecting the view. False guarantees that
// Snapshot().WorkspaceNavigation is nil; true still needs Snapshot() for the
// validated request.
func (s *Store) WorkspaceNavigationPending() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.workspaceNavigation.view.Mode != ""
}
