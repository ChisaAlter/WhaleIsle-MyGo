package main

import (
	"path/filepath"
	"strings"

	"whaleisle/internal/store/session"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
)

// Admission still requires the rendered child navigation token and full address.
// Only the descriptor survives restart; recovery reacquires the live binding
// after the user explicitly enters this child again, without selecting it.
func (s *mainState) openAgentDocument(sessions *session.Store, token uint64, address session.AgentSessionAddress, path string) {
	s.openAgentPaneDocument("", sessions, token, address, path)
}

func (s *mainState) openAgentPaneDocument(paneID string, sessions *session.Store, token uint64, address session.AgentSessionAddress, path string) {
	if sessions == nil { return }
	_, valid, err := sessions.AgentMediaPreviewSource(token, address)
	if err != nil || !valid() { return }
	snap := sessions.Snapshot()
	kind := browserpreviewui.DocumentHTML
	if strings.EqualFold(filepath.Ext(path), ".pdf") { kind = browserpreviewui.DocumentPDF }
	source := browserpreviewui.DocumentSource{
		NavigationPaneID: paneID,
		Authority: s.browserWorkspaceOwner().Authority,
		SessionID: address.ChildSessionID, ViewSessionID: snap.SelectedID,
		AddressKind: address.Kind, ParentSessionID: address.ParentSessionID, Mode: address.Mode,
		Workspace: snap.SelectedCwd, Path: path, Kind: kind,
	}
	s.openWorkspaceDocumentSource(sessions, source)
}
