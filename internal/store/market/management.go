package market

// Confirmation is ephemeral: no action or credential is restored after restart.
type managementState struct {
	filter         string
	selected       map[string]bool
	confirmation   string
	confirm        func()
	confirmRuntime uint64
}

func (s *Store) ManagementFilter() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.management.filter
}
func (s *Store) SetManagementFilter(filter string) {
	s.mu.Lock()
	s.management.filter = filter
	s.mu.Unlock()
	s.fire()
}
func (s *Store) UpdateSelected(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.management.selected[id]
}
func (s *Store) ToggleUpdateSelected(id string) {
	s.mu.Lock()
	if s.management.selected == nil {
		s.management.selected = map[string]bool{}
	}
	s.management.selected[id] = !s.management.selected[id]
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SelectAvailableUpdates(selectAll bool) {
	s.mu.Lock()
	s.management.selected = map[string]bool{}
	if selectAll && s.updatesCurrentLocked() {
		for _, row := range s.updates.Rows {
			if row.Available {
				s.management.selected[row.ID] = true
			}
		}
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) RequestConfirmation(description string, action func()) {
	s.mu.Lock()
	s.management.confirmation = description
	s.management.confirm = action
	s.management.confirmRuntime = s.generation
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Confirmation() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.management.confirmation
}
func (s *Store) ResolveConfirmation(accept bool) {
	s.mu.Lock()
	action := s.management.confirm
	current := s.management.confirmRuntime == s.generation
	s.clearConfirmationLocked()
	s.mu.Unlock()
	s.fire()
	if accept && current && action != nil {
		action()
	}
}

func (s *Store) clearConfirmationLocked() {
	s.management.confirm = nil
	s.management.confirmation = ""
	s.management.confirmRuntime = 0
}
