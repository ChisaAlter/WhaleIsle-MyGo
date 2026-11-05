package main

import (
	"context"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/settings"
)

// Called synchronously by the rendered group gesture so both the visible
// membership and its catalog owner are fixed before the Store starts I/O.
func (s *mainState) setVisibleSkillGroupInvocation(source *session.Store, catalog session.SkillCatalog, rows []settings.SkillRow, enabled bool) {
	if source == nil {
		return
	}
	s.mu.Lock()
	current := s.sessions == source
	s.mu.Unlock()
	if !current {
		return
	}
	members := make([]session.SkillInvocationMember, 0, len(rows))
	for _, row := range rows {
		if row.Writable {
			members = append(members, session.SkillInvocationMember{Name: row.Name, Source: row.Source, Path: row.Path})
		}
	}
	if err := source.SetVisibleSkillGroupInvocation(context.Background(), catalog, members, enabled); err != nil {
		source.SetLastAction(err.Error())
	}
}
