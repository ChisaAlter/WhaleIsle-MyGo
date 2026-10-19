package session

import "context"

// HistorySkillReference captures both history and catalog ownership at the
// gesture. A returned file path is useful only while that same owner is current.
type HistorySkillReference struct {
	scope      TurnFileScope
	catalog    skillOptionsScope
	name, path string
	ready      bool
}

func (s *Store) historySkillRootLocked() bool {
	if identity := s.subagentIdentities[s.selectedID]; identity.value != nil {
		return false
	}
	for _, item := range s.items {
		if item.SessionID == s.selectedID && item.SubagentIdentity != nil {
			return false
		}
	}
	return true
}

func (s *Store) historySkillReferenceCurrentLocked(ref HistorySkillReference) bool {
	return ref.name != "" && ref.scope.Binding != "" && ref.scope == s.turnFilesScopeLocked() && s.skillOptionsScopeCurrentLocked(ref.catalog) && s.historySkillRootLocked()
}

func (s *Store) CaptureHistorySkillReference(scope TurnFileScope, name string) (HistorySkillReference, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ref := HistorySkillReference{scope: scope, catalog: s.captureSkillOptionsScopeLocked(), name: name}
	if !s.historySkillReferenceCurrentLocked(ref) {
		return HistorySkillReference{}, false
	}
	ref.ready = s.skillOptionsOwner == ref.catalog
	for _, option := range s.currentSkillOptionsLocked() {
		if option.Name == name {
			ref.path = option.Path
			break
		}
	}
	return ref, true
}

// ResolveHistorySkillReference reuses the waitable catalog reader on a cold
// click. No guessed SKILL.md path or child-to-root authority is introduced.
func (s *Store) ResolveHistorySkillReference(ctx context.Context, ref HistorySkillReference) (HistorySkillReference, bool) {
	s.mu.Lock()
	current := s.historySkillReferenceCurrentLocked(ref)
	s.mu.Unlock()
	if !current || ctx.Err() != nil {
		return HistorySkillReference{}, false
	}
	if !ref.ready {
		options := s.ListSkills(ctx)
		ref.ready = true
		for _, option := range options {
			if option.Name == ref.name {
				ref.path = option.Path
				break
			}
		}
	}
	_, ok := s.HistorySkillReferencePath(ref)
	return ref, ok && ctx.Err() == nil
}

// HistorySkillReferencePath is called again on the UI thread immediately before
// invoking the existing session-bound file preview callback.
func (s *Store) HistorySkillReferencePath(ref HistorySkillReference) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.historySkillReferenceCurrentLocked(ref) || !ref.ready || ref.path == "" {
		return "", false
	}
	for _, option := range s.currentSkillOptionsLocked() {
		if option.Name == ref.name && option.Path == ref.path {
			return ref.path, true
		}
	}
	return "", false
}
