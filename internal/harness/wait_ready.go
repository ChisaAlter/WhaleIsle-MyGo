package harness

import (
	"context"
	"errors"
)

// RestartAndWait owns one explicit restart and waits for that attempt's Ready.
// A newer restart/stop cannot accidentally satisfy the waiting settings switch.
func (s *Supervisor) RestartAndWait(ctx context.Context) error {
	s.Stop()
	target, issued := s.startOwned()
	if !issued {
		return errors.New("another owner started the runtime before Project restart")
	}
	return s.waitReady(ctx, target)
}

func (s *Supervisor) waitReady(ctx context.Context, target int) error {
	for {
		s.mu.Lock()
		st := s.status
		if s.gen != target {
			s.mu.Unlock()
			return errors.New("runtime changed while awaiting Project restart")
		}
		if st.Generation == target {
			switch st.State {
			case Ready:
				s.mu.Unlock()
				return nil
			case Failed:
				s.mu.Unlock()
				if st.Err != nil {
					return st.Err
				}
				return errors.New("Harness restart failed")
			case Stopped:
				s.mu.Unlock()
				return errors.New("Harness restart stopped")
			}
		}
		if s.statusChanged == nil {
			s.statusChanged = make(chan struct{})
		}
		changed := s.statusChanged
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}
