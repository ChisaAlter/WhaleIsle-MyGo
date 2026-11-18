package processutil

import (
	"context"
	"os/exec"
	"time"
)

// PipeGrace bounds how long a wait for a command's output pipes may outlive the
// process or its context. A command that finishes writing exits before the
// grace expires, so only a descendant that inherited the pipe needs it.
const PipeGrace = 2 * time.Second

// CommandContext builds a context-bounded child whose wait ends with its
// context. Cancelling the context kills the child, but a descendant that
// inherited the output pipe keeps it open, and a wait for that output then
// blocks indefinitely: no error, no timeout, nothing the user can cancel.
// WaitDelay closes those pipes so the wait cannot outlive the context.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = PipeGrace
	return cmd
}
