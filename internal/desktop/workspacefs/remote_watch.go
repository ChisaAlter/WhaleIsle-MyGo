package workspacefs

import (
	"context"
	"encoding/json"

	"whaleisle/internal/dsh"
)

// WatchRemoteChanges opens workspaceFiles/changes and invokes onFrame for each payload.
// Cancels when ctx ends.
func WatchRemoteChanges(ctx context.Context, mux *dsh.Mux, path string, onFrame func(json.RawMessage)) error {
	if mux == nil || onFrame == nil {
		return ErrNotFile
	}
	if err := mux.Dial(ctx); err != nil {
		return err
	}
	args := map[string]any{"path": path}
	stream, err := mux.Open(ctx, "workspaceFiles/changes", map[string]any{"args": args})
	if err != nil {
		return err
	}
	defer stream.Cancel()
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			return err
		}
		onFrame(raw)
	}
}
