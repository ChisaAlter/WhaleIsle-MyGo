package session

import "whaleisle/internal/dsh/api"

// Apply the existing session/delete reply atomically, before repainting the
// archive roster. This also covers a missed or delayed $events notification.
func (s *Store) applyDeleteResultLocked(out api.SessionDeleteValue) {
	for _, id := range out.DeletedSessionIds {
		m := lifecycleMutation{kind: "api-session/deleted", sessionID: id}
		if s.listLoading {
			s.listMutations = append(s.listMutations, m)
		}
		s.applyLifecycleMutationLocked(m, true)
	}
	s.archivedSessionIDs = append([]string(nil), out.ArchivedSessionIds...)
}
