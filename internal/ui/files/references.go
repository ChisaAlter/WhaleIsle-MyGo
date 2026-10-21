package files

import (
	"whaleisle/internal/ui/filelink"
)

func workspaceReference(m Model, path string) filelink.Reference {
	return filelink.Reference{BindingID: m.BindingID, SessionID: m.SessionID, WorkspaceRoot: m.WorkspaceRoot, Path: path}
}
