package workspacefs

import (
	"context"
	"encoding/json"
	"fmt"

	"whaleisle/internal/dsh"
)

// WatchRemoteDirectory uses the Host's workspace scope lookup, which is a
// Session ID on the wire. Paths are absolute against that captured workspace.
func WatchRemoteDirectory(ctx context.Context, mux *dsh.Mux, sessionID, path string, changed func()) error {
	if mux == nil || sessionID == "" || changed == nil {
		return ErrNotFile
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := mux.Dial(ctx); err != nil {
		return err
	}
	stream, err := mux.Open(ctx, "workspaceFiles/changes", map[string]any{"args": map[string]any{"workspaceFileScopeId": sessionID, "path": path}})
	if err != nil {
		return err
	}
	defer stream.Cancel()
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		var frame struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(raw, &frame); err != nil {
			return err
		}
		switch frame.Kind {
		case "ready", "change":
			changed()
		default:
			return fmt.Errorf("workspace directory watch: unknown frame %q", frame.Kind)
		}
	}
}
