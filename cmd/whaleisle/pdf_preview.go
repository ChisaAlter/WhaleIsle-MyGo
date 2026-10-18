package main

import (
	"whaleisle/internal/store/session"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
)

func (s *mainState) openLocalPDF(sessions *session.Store, sid, root, path string) {
	s.startPDFPreview(sessions, sid, root, path, false)
}

func (s *mainState) openOfficePDF(sessions *session.Store, sid, root, path string) {
	s.startPDFPreview(sessions, sid, root, path, true)
}

func (s *mainState) startPDFPreview(sessions *session.Store, sid, root, path string, convert bool) {
	kind := browserpreviewui.DocumentPDF
	if convert { kind = browserpreviewui.DocumentOfficePDF }
	s.openWorkspaceDocument(sessions, sid, root, path, kind)
}
