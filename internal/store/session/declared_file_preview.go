package session

// DeclaredTurnFileTarget admits only a path explicitly published by a durable
// delivery in this exact current Host/session scope. It does not resolve paths
// in the desktop filesystem and is not an Open on Host action.
func (s *Store) DeclaredTurnFileTarget(scope TurnFileScope, path string) (TurnFileTarget, bool) {
	if s == nil || path == "" {
		return TurnFileTarget{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if scope.Binding == "" || scope != s.turnFilesScopeLocked() {
		return TurnFileTarget{}, false
	}
	var selected TurnFileTarget
	found := false
	for _, delivery := range TurnDeliveries(s.messages) {
		for _, file := range delivery.Files {
			if file.Path == path && (!found || file.Target.Seq > selected.Seq || file.Target.Seq == selected.Seq && file.Target.Index > selected.Index) {
				selected, found = file.Target, true
			}
		}
	}
	return selected, found
}
