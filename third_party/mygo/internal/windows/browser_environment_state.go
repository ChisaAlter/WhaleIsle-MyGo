package windows

// browserEnvironmentOwners is confined to the main thread. A retired generation
// cannot be reused by a late environment-creation callback or a new window.
type browserEnvironmentOwners[T comparable] struct {
	members map[T]struct{}
	closed  bool
}

func (s *browserEnvironmentOwners[T]) add(owner T) bool {
	if s.closed {
		return false
	}
	if s.members == nil {
		s.members = make(map[T]struct{})
	}
	s.members[owner] = struct{}{}
	return true
}

// remove returns true only when this removal retires the last live owner.
func (s *browserEnvironmentOwners[T]) remove(owner T) bool {
	if s.closed {
		return false
	}
	if _, ok := s.members[owner]; !ok {
		return false
	}
	delete(s.members, owner)
	if len(s.members) != 0 {
		return false
	}
	s.closed = true
	return true
}
