package mcpconfig

import "context"

// OperationContext owns foreground actions for this attached Host. Hiding the
// settings page stops health polling without interrupting an OAuth browser flow.
func (c *Controller) OperationContext() context.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.operationContext == nil {
		c.operationContext, c.operationCancel = context.WithCancel(context.Background())
	}
	return c.operationContext
}

// CancelOperations retires this Host permanently, including actions whose
// callbacks were queued before the controller was replaced.
func (c *Controller) CancelOperations() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.operationContext == nil {
		c.operationContext, c.operationCancel = context.WithCancel(context.Background())
	}
	c.operationCancel()
}
