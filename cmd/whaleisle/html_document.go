package main

import (
	"whaleisle/internal/store/session"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
)

// Only an explicit Files or confirmed-delivery gesture reaches this opener.
// It previews the saved source; editor drafts remain owned by their file pane.
func (s *mainState) openLocalHTML(sessions *session.Store, sid, root, path string) {
	s.openWorkspaceDocument(sessions, sid, root, path, browserpreviewui.DocumentHTML)
}
