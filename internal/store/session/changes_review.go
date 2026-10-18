package session

// ChangesReviewPath validates the recorded declaration and summary before a
// caller uses the existing Host-scoped Files preview. It grants no local path
// authority and does not change native-open or approval policy.
func (s *Store) ChangesReviewPath(scope TurnFileScope, target TurnFileTarget) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if target.Kind != "changes" || target.Index < 0 {
		return "", false
	}
	return s.turnFileTargetLocked(scope, target)
}
