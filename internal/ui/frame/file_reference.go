package frame

import (
	"errors"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/filelink"
)

func insertWorkspaceReference(m Model, ref filelink.Reference) error {
	if m.Store == nil || m.ComposerState == nil {
		return errors.New("select a conversation before adding a file")
	}
	binding, sessionID, root := m.Store.WorkspaceFileReferenceOrigin()
	if !m.CanAddSelectionToChat || sessionID == "" || root != m.WorkspaceRoot || binding != m.FileBindingID {
		return errors.New("select the file's conversation and workspace before adding it")
	}
	return composer.InsertWorkspaceReference(m.ComposerState, ref, binding, sessionID, root)
}
