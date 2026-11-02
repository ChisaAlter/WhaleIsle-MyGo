package main

import (
	"context"
	"fmt"
	"time"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) openMediaFile(sessions *session.Store, expectedClient *dsh.Client, sid, root, path string, floating bool) {
	if sessions == nil || expectedClient == nil || sid == "" {
		return
	}
	if root == "" || !mediapreview.ValidPath(path) {
		sessions.SetLastAction(i18n.T(i18n.KeyCmdMediaPathRequired))
		return
	}
	client, sourceValid, sourceErr := sessions.MediaPreviewSource(sid)
	s.mu.Lock()
	win, host := s.win, s.bootStatus
	s.mediaPreviewRequest++
	request := s.mediaPreviewRequest
	s.mu.Unlock()
	if win == nil || sourceErr != nil || client != expectedClient {
		sessions.SetLastAction(i18n.T(i18n.KeyCmdMediaHostServiceRequired))
		return
	}
	valid := func() bool {
		s.mu.Lock()
		st := s.bootStatus
		current := !s.quitting && s.sessions == sessions && s.win == win && st.Generation == host.Generation && st.State == host.State && st.Endpoint.Origin == host.Endpoint.Origin && st.Endpoint.Cookie == host.Endpoint.Cookie
		s.mu.Unlock()
		snap := sessions.Snapshot()
		return current && sourceValid() && snap.SelectedID == sid && snap.SelectedCwd == root
	}
	if !valid() {
		return
	}
	sessions.SetLastAction(i18n.T(i18n.KeyCmdMediaReadingSnapshot))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		readValid := func() bool {
			s.mu.Lock()
			same := s.mediaPreviewRequest == request
			s.mu.Unlock()
			return same && valid()
		}
		data, err := mediapreview.Read(ctx, client, sid, path, readValid)
		win.Update(func() {
			if !readValid() {
				return
			}
			// Publish the capability only on the still-current UI thread. A
			// discarded callback after window destruction must own no listener.
			var doc *mediapreview.Document
			if err == nil {
				doc, err = mediapreview.Open(path, data, valid)
			}
			if err != nil || doc == nil {
				message := i18n.T(i18n.KeyCmdMediaBoundedReadFailed)
				if err != nil {
					message += " " + err.Error()
				}
				sessions.SetLastAction(message)
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
			if err = s.browserController.OpenMedia(doc, floating); err != nil {
				sessions.SetLastAction(fmt.Sprintf(i18n.T(i18n.KeyCmdMediaPreviewUnavailable), err.Error()))
			} else {
				sessions.SetLastAction(i18n.T(i18n.KeyCmdMediaSnapshotOpened))
			}
		})
	}()
}
