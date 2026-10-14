package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"whaleisle/internal/desktop/pdfpreview"
	"whaleisle/internal/desktop/workspacefs"
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
	if sessions == nil || sid == "" {
		return
	}
	s.mu.Lock()
	win, host := s.win, s.bootStatus
	s.pdfPreviewRequest++
	request := s.pdfPreviewRequest
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
	sessions.SetLastAction("正在准备 PDF 只读预览…")
	go func() {
		var doc *pdfpreview.Document
		var err error
		if convert {
			// Only this input in this Session may request conversion. The returned path
			// grants exactly one snapshot, never its temp directory or any sibling file.
			_, err = workspacefs.JoinRoot(root, path)
			if err == nil && (!filepath.IsAbs(root) || strings.TrimSpace(path) == "") {
				err = fmt.Errorf("Office PDF requires the active session workspace")
			}
			if err == nil {
				var generated string
				generated, err = sessions.RenderOfficePDFForSession(context.Background(), sid, path)
				if err == nil && valid() {
					doc, err = pdfpreview.OpenGenerated(generated, valid)
				} else if err == nil {
					err = fmt.Errorf("Office conversion session changed")
				}
			}
		} else {
			doc, err = pdfpreview.OpenWorkspace(root, path, valid)
		}
		win.Update(func() {
			s.mu.Lock()
			same := s.pdfPreviewRequest == request
			s.mu.Unlock()
			if !same || !valid() {
				if doc != nil {
					doc.Close()
				}
				return
			}
			if err != nil {
				sessions.SetLastAction(err.Error())
				return
			}
			s.closeWebViewHost()
			s.mu.Lock()
			s.browserPreviewOpen = true
			s.webviewOpen = false
			s.mu.Unlock()
			if s.browserController == nil {
				s.browserController = browserpreviewui.NewController(win)
			}
			if err := s.browserController.OpenPDF(doc); err != nil {
				sessions.SetLastAction(err.Error())
			} else {
				sessions.SetLastAction("PDF 只读预览标签已打开；原文件和编辑草稿保持不变")
			}
		})
	}()
}
