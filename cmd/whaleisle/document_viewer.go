package main

import (
	"errors"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) openDocumentViewer(client *dsh.Client, sessionID, path string) (files.DocumentSurface, error) {
	s.dshMu.Lock()
	current := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	win, status, sessions := s.win, s.bootStatus, s.sessions
	s.mu.Unlock()
	if client == nil || client != current || win == nil || status.State != harness.Ready || sessions == nil || sessions.SelectedSessionID() != sessionID || client.Endpoint.Origin != status.Endpoint.Origin || client.Endpoint.Cookie != status.Endpoint.Cookie {
		return nil, errors.New(i18n.T(i18n.KeyCmdDocumentViewerOwnerChanged))
	}
	target, err := desktopwebview.DocumentURL(client.Endpoint.Origin, sessionID, path)
	if err != nil {
		return nil, err
	}
	return desktopwebview.NewHost(desktopwebview.HostOptions{
		Parent: win, URL: target, Title: path, Untrusted: true,
		SessionOrigin: client.Endpoint.Origin, SessionCookie: client.Endpoint.Cookie,
		// A document preview is not the Harness view: no microphone, no camera;
		// notifications follow the shell gate.
		PermissionPolicy: desktopwebview.PermissionPolicy{
			NotificationsAllowed: systemNotificationsSupported,
		}.Decide,
		ShortcutPreferences:   s.pageShortcutPreferences(),
		OnShortcutCloseWindow: s.queueProductShortcutClose,
		OnChange:              func() { win.Update(func() {}) },
		OnFailure:             s.onEmbeddedPageFailure,
	})
}
