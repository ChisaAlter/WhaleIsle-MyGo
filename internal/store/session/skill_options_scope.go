package session

import (
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

type skillOptionsScope struct {
	client            *dsh.Client
	generation, epoch uint64
	sessionID, cwd    string
}

func (s *Store) captureSkillOptionsScopeLocked() skillOptionsScope {
	return skillOptionsScope{s.client, s.eventsGeneration, s.skillOptionsEpoch, s.selectedID, s.cwd}
}
func (s *Store) skillOptionsScopeCurrentLocked(owner skillOptionsScope) bool {
	return owner.client != nil && owner == s.captureSkillOptionsScopeLocked()
}
func (s *Store) currentSkillOptionsLocked() []SkillOption {
	if !s.skillOptionsScopeCurrentLocked(s.skillOptionsOwner) {
		return nil
	}
	return append([]SkillOption(nil), s.skillOptions...)
}
func (s *Store) invalidateSkillOptionsLocked() {
	s.invalidateViewLocked()
	s.skillOptionsEpoch++
	s.skillOptions = nil
	s.skillOptionsOwner = skillOptionsScope{}
	s.skillsErr = nil
	// Same-session preset changes also invalidate the full inventory and any
	// response/edit target captured under its previous version.
	s.skillCatalogVersion++
	s.skillCatalog = SkillCatalog{}
	s.skillCatalogClient = nil
	s.skillCatalogScope = api.SkillInventoryScope{}
	s.skillCatalogSession, s.skillCatalogCwd = "", ""
	s.skillCatalogLoading = false
	s.skillRemovalSequence++
	s.skillRemoval = SkillRemovalReview{}
}
