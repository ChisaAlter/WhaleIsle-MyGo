package config

// HasAutoDownloadUpdates distinguishes an accepted preference, including false,
// from an older configuration that predates the native update-center migration.
func (s *Store) HasAutoDownloadUpdates() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, present := s.raw["autoDownloadUpdates"]
	return present
}
