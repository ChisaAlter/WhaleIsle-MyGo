package projectctl

import "context"

// readOperation identifies a cancellable presentation read. Its completion may
// release only its own admission; a stale callback cannot clear a newer action.
type readOperation struct{ cancel context.CancelFunc }

func (c *Controller) beginReadLocked(cancel context.CancelFunc) *readOperation {
	op := &readOperation{cancel: cancel}
	c.flow.readOperation = op
	c.flow.busy = true
	return op
}

func (c *Controller) finishReadLocked(op *readOperation) bool {
	if op == nil {
		return false
	}
	op.cancel()
	if c.flow.catalogRead == op {
		c.flow.catalogRead = nil
	}
	if c.flow.readOperation != op {
		return false
	}
	c.flow.readOperation = nil
	c.flow.busy = false
	return true
}
