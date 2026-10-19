package hostterminal

// Occurrences share the authenticated transport, but own independent follow
// duplex streams, acknowledgements and local attachment IDs. The Host's
// per-attachment writable state controls input when two panes show one terminal.
func (s *Store) Occurrence(key, source, session string) *Store {
	s.mu.Lock()
	if child := s.occurrences[key]; child != nil {
		s.mu.Unlock()
		return child
	}
	if s.occurrences == nil {
		s.occurrences = make(map[string]*Store)
	}
	child := &Store{}
	s.occurrences[key] = child
	owner, remote, ready, notify := s.owner, s.remote, s.ready, s.notify
	preference, save := s.shellPreference, s.saveShellPreference
	origin := s.occurrences[source]
	s.mu.Unlock()
	// Only an explicit duplicate/source pane may inherit a live identity.
	// Fresh tabs and saved descriptors must not adopt another pane's process
	// merely because the shared transport currently has an active terminal.
	var seed View
	if origin != nil {
		seed = origin.Snapshot(session)
	}
	child.bind(owner, remote, ready, notify)
	child.ConfigureShellPreference(preference, save)
	child.Show(session, true)
	child.mu.Lock()
	st := child.sessionLocked(session)
	st.items, st.environment = append([]Info(nil), seed.Items...), seed.Environment
	for _, info := range st.items {
		child.entryLocked(terminalKey{session, info.ID}).info = info
	}
	if seed.ActiveID != "" {
		e := child.entryLocked(terminalKey{session, seed.ActiveID})
		e.info, e.desired = seed.Info, true
		st.active = seed.ActiveID
		child.startHoldLocked(e)
	}
	child.mu.Unlock()
	return child
}
func (s *Store) occurrenceStores() []*Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	children := make([]*Store, 0, len(s.occurrences))
	for _, child := range s.occurrences {
		children = append(children, child)
	}
	return children
}
func (s *Store) bindOccurrences(owner any, remote Remote, ready bool, notify func()) {
	for _, child := range s.occurrenceStores() {
		child.bind(owner, remote, ready, notify)
	}
}
func (s *Store) transportOccurrences(owner any, ready bool) {
	for _, child := range s.occurrenceStores() {
		child.transport(owner, ready)
	}
}

// ExistingOccurrence never creates a pane while reconciling hidden content.
func (s *Store) ExistingOccurrence(key string) *Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.occurrences[key]
}
