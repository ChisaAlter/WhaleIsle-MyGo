package main

import (
	"fmt"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// openLocalImage follows the existing session-bound static snapshot lease.
// Files routes GIF/SVG/ICO/AVIF here automatically on supported platforms;
// the explicit reopen action uses the same isolated snapshot path.
func (s *mainState) openLocalImage(sessions *session.Store, sid, root, path string) {
	if sessions == nil || sid == "" {
		return
	}
	s.mu.Lock()
	win, host := s.win, s.bootStatus
	s.htmlDocumentRequest++
	request := s.htmlDocumentRequest
	s.mu.Unlock()
	if win == nil {
		return
	}
	valid := func() bool {
		s.mu.Lock()
		st := s.bootStatus
		current := !s.quitting && s.sessions == sessions && s.win == win && st.Generation == host.Generation && st.State == host.State && st.Endpoint.Origin == host.Endpoint.Origin && st.Endpoint.Cookie == host.Endpoint.Cookie
		s.mu.Unlock()
		snap := sessions.Snapshot()
		return current && snap.SelectedID == sid && snap.SelectedCwd == root
	}
	if !valid() {
		return
	}
	sessions.SetLastAction(i18n.T(i18n.KeyCmdImageSnapshotPreparing))
	go func() {
		doc, err := htmldocument.OpenImageWorkspace(root, path, valid)
		win.Update(func() {
			s.mu.Lock()
			same := s.htmlDocumentRequest == request
			s.mu.Unlock()
			if !same || !valid() {
				if doc != nil {
					doc.Close()
				}
				return
			}
			if err != nil {
				s.mu.Lock()
				s.filesPaneOpen = true
				s.rightPaneTab = "files"
				s.mu.Unlock()
				sessions.SetLastAction(fmt.Sprintf(i18n.T(i18n.KeyCmdImageSnapshotFailed), err.Error()))
				return
			}
			s.closeWebViewHost()
			s.mu.Lock()
			s.browserPreviewOpen = true
			s.webviewOpen = false
			s.mu.Unlock()
			if s.browserController == nil {
				s.browserController = s.newBrowserController(win)
			}
			if err := s.browserController.OpenHTML(doc); err != nil {
				sessions.SetLastAction(i18n.T(i18n.KeyCmdImagePreviewUnavailable))
			} else {
				sessions.SetLastAction(fmt.Sprintf(i18n.T(i18n.KeyCmdImageSnapshotOpened), doc.Resources.Summary()))
			}
		})
	}()
}

func supportsImagePreview(path string) bool { return imagepreview.Supports(path) }
