package main

import (
	"context"
	"fmt"
	"strings"

	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/dsh"
)

// The caller captures the client and session from the same frame as
// FileBindingID. Never resolve a newly selected Host or Session on a worker.
func workspacePreviewStat(ctx context.Context, client *dsh.Client, sessionID, rel string) string {
	st, err := workspacefs.StatRemoteScoped(ctx, client, sessionID, rel)
	if err != nil {
		return ""
	}
	note := strings.TrimSpace(st.Version)
	if st.Bytes != nil {
		if note != "" {
			note += " · "
		}
		note += fmt.Sprintf("%.0f B", *st.Bytes)
	}
	if note == "" {
		note = strings.TrimSpace(st.AbsolutePath)
	}
	return note
}
