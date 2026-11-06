package pluginui

import (
	"context"
	"errors"
	"testing"
)

func TestCancelledWorkspaceActionDoesNotReachRenderer(t *testing.T) {
	subject := Subject{Kind: "navigation"}
	state := Snapshot{Connected: true, Page: 1, Revision: 1, Subject: &subject,
		WorkspaceNavigation: []WorkspaceNavigation{{SessionID: "owned", Owner: "plugin", CanArchive: true}}}
	// The valid capability intentionally has no socket: a cancelled operation
	// must return before allocating or writing any renderer command.
	o := &Owner{selected: subject, state: state, page: &bridgePage{id: 1}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := o.InvokeWorkspace(ctx, state, "archive", "owned", "", "plugin"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled archive returned %v", err)
	}
	if o.workspaceOps.next != 0 || len(o.workspaceOps.pending) != 0 {
		t.Fatal("cancelled archive was scheduled")
	}
}
