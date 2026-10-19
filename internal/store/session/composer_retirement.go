package session

import (
	"context"
	"errors"
)

// Attachment sends retain their detached files until the provisional echo has
// handed off to a durable message or queued inbox entry, not just an RPC receipt.
type composerRetirement struct {
	done    chan struct{}
	settled bool
	err     error
}

func (t *ComposerTarget) settleRetirementLocked(id string, err error) {
	r := t.retirements[id]
	if r == nil || r.settled {
		return
	}
	r.settled, r.err = true, err
	close(r.done)
}

// A durably deleted Session cannot be reopened to complete these handoffs.
// Release only its unsettled echoes, as Session.dispose does. Ordinary removed
// notifications retain reloadable Session state and do not take this path.
// Observed messages/queued entries already left composerPending and retain
// their successful settlement; deletion must never turn them into a resend.
func (s *Store) retireDeletedSessionSubmissionsLocked(sessionID string) bool {
	retired := false
	for id, pending := range s.composerPending {
		if pending.SessionID != sessionID || !pending.target.currentLocked() {
			continue
		}
		pending.target.settleRetirementLocked(id, errors.New("session deleted; send outcome unconfirmed, draft and attachments retained; check the conversation before retrying"))
		delete(s.composerPending, id)
		retired = true
	}
	return retired
}

// AwaitSubmissionRetirement is called only for an attachment-bearing detached
// send after Submit succeeds. Observation may precede the RPC reply; its latched
// result cannot be changed by a later cancellation or another submission.
func (t *ComposerTarget) AwaitSubmissionRetirement(ctx context.Context, id string) error {
	s := t.store
	s.mu.Lock()
	r := t.retirements[id]
	s.mu.Unlock()
	if r == nil {
		return errors.New("attachment submission has no retirement owner; check the conversation before retrying")
	}
	select {
	case <-r.done:
		s.mu.Lock()
		err := r.err
		delete(t.retirements, id)
		s.mu.Unlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
