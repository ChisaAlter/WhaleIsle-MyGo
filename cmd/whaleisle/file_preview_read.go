package main

import (
	"context"
	"fmt"
	"strings"

	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/dsh"
)

// The caller captures the client from the same frame as FileBindingID. Never
// look up a newly selected Host when a queued text-preview read finally runs.
func workspacePreviewStat(ctx context.Context, client *dsh.Client, rel string) string {
	st, err := workspacefs.StatRemote(ctx, client, rel)
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
