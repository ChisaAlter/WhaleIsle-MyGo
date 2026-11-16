package main

import (
	projectctl "whaleisle/internal/app/project"
	"whaleisle/internal/ui/workspacepicker"
)

// Project's published directory contribution is row-only. State is a pure
// projection of its listing and directory/check cache, not a render-time RPC.
func projectWorkspaceDirectoryActions(available bool, state func(path string) (active, refused bool), run func(string)) []workspacepicker.DirectoryAction {
	return projectctl.WorkspaceDirectoryActions(available, state, run)
}
