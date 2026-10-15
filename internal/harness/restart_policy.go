package harness

import "time"

// RestartPolicy is one consistent snapshot for an ordinary failure decision.
// Changes do not cancel an already scheduled timer or restart a live child.
type RestartPolicy struct {
	AutoRestart bool
	MaxAttempts int
	BaseDelay   time.Duration
}

// RestartPolicy reads the current committed policy, falling back to the
// construction options for callers that do not supply ReadRestartPolicy.
// The provider runs outside s.mu so reading a config Store cannot introduce a
// supervisor/config lock-order dependency. Reading has no lifecycle effects.
func (s *Supervisor) RestartPolicy() RestartPolicy {
	s.mu.Lock()
	read := s.opts.ReadRestartPolicy
	policy := RestartPolicy{s.opts.AutoRestart, s.opts.MaxAttempts, s.opts.BaseDelay}
	s.mu.Unlock()
	if read != nil {
		return read()
	}
	return policy
}

func (p RestartPolicy) retryDelay(attempt int) (time.Duration, bool) {
	if !p.AutoRestart || attempt >= p.MaxAttempts {
		return 0, false
	}
	return p.BaseDelay << attempt, true
}
