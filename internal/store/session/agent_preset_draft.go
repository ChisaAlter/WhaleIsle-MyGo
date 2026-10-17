package session

import "whaleisle/internal/dsh"

type AgentPresetSelectionView struct {
	ID, Label, Error string
	Busy, CanSelect  bool
}
type agentPresetDraftState struct {
	client     *dsh.Client
	epoch      uint64
	staged     map[string]string
	session    string
	generation uint64
	busy       bool
	error      string
}

func (s *Store) presetStateLocked() *agentPresetDraftState {
	if s.presetDraft.client != s.client || s.presetDraft.epoch != s.composerHostGeneration {
		s.presetDraft = agentPresetDraftState{client: s.client, epoch: s.composerHostGeneration, staged: map[string]string{}}
	}
	return &s.presetDraft
}
func (s *Store) blankPresetTargetLocked(id string) bool {
	if id == "" {
		return true
	}
	for _, it := range s.items {
		if it.SessionID == id {
			return it.Blank && !it.Running && it.Origin != "subagent"
		}
	}
	return false
}
func (s *Store) stagedPresetLocked(cwd string) string {
	return s.presetStateLocked().staged[filepathClean(cwd)]
}
func (s *Store) AgentPresetSelection() AgentPresetSelectionView {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.presetStateLocked()
	v := AgentPresetSelectionView{CanSelect: s.client != nil && s.blankPresetTargetLocked(s.selectedID)}
	if s.selectedID == "" {
		v.ID = p.staged[filepathClean(s.cwd)]
		v.Label = labelForAgentPresetID(v.ID, s.agentPresetOptions)
	}
	if p.session == s.selectedID && p.generation == s.followGeneration {
		v.Busy = p.busy
		v.Error = p.error
	}
	return v
}
