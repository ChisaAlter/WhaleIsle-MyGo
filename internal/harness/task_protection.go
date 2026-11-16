package harness

import (
	"context"
	"errors"

	"whaleisle/internal/desktop/taskcontrol"
)

// ProtectTaskAction binds the control exchange to this exact supervised Host.
// Tokens stay in process memory and the child environment, never Status/logs.
func (s *Supervisor) ProtectTaskAction(ctx context.Context, operation string, localRunning func() int, confirm func(taskcontrol.Inspection) (bool, error), commit func() error) (bool, error) {
	if !s.protectionMu.TryLock() {
		return false, errors.New("another protected lifecycle action is pending")
	}
	defer s.protectionMu.Unlock()
	s.mu.Lock()
	gen, st, token := s.gen, s.status, s.taskControlToken
	// While stopChild is still waiting for a cancelled or stopped child to exit,
	// hostAbsent must remain false so callers retain normal protection until exit.
	hostAbsent := (st.State == Stopped || st.State == Failed) && s.cmd == nil && s.exited == nil && s.stoppingChildren == 0
	s.mu.Unlock()
	checkedCommit := func() error {
		if !s.current(gen) {
			return errors.New("runtime changed during confirmation; retry the action")
		}
		return commit()
	}
	if hostAbsent {
		local := 0
		if localRunning != nil {
			local = localRunning()
		}
		if local > 0 {
			approved, err := confirm(taskcontrol.Inspection{OK: true, LocalRunning: local})
			if err != nil {
				return false, err
			}
			if !approved {
				return false, taskcontrol.ErrCancelled
			}
		}
		if err := checkedCommit(); err != nil {
			return false, err
		}
		return true, nil
	}
	client := taskcontrol.New(st.Endpoint.Origin, token)
	client.LocalRunning = localRunning
	return client.Coordinate(ctx, operation, confirm, checkedCommit)
}
