package usage

import "fmt"

// StatisticsOwner changes on reattachment even if a disk overview is restored.
func (s *Store) StatisticsOwner() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("%p/%p/%d", s, s.client, s.statsEpoch)
}
