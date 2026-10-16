package main

import (
	"context"
	"fmt"
	"time"
	"whaleisle/internal/desktop/mediapreview"

	"whaleisle/internal/desktop/pdfpreview"
	"whaleisle/internal/store/session"
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
	reader, sourceValid, sourceErr := sessions.MediaPreviewSource(sid)
	valid := func() bool {
		s.mu.Lock()
		st := s.bootStatus
		current := !s.quitting && s.sessions == sessions && s.win == win && st.Generation == host.Generation && st.State == host.State && st.Endpoint.Origin == host.Endpoint.Origin && st.Endpoint.Cookie == host.Endpoint.Cookie
		s.mu.Unlock()
		snap := sessions.Snapshot()
		return current && (sourceErr != nil || sourceValid()) && snap.SelectedID == sid && snap.SelectedCwd == root
	}
	if !valid() {
		return
	}
	if convert {
		s.setOfficePreviewMessage(sessions, sid, path, "正在转换 Office 文档并准备 PDF 预览…")
	} else {
		sessions.SetLastAction("正在准备 PDF 只读预览…")
	}
	go func() {
		var doc *pdfpreview.Document
		var err error
		var office *session.OfficePDFDocument
		if convert {
			// The Host returns the source Office path and complete PDF bytes.
			// Never open its source or private conversion directory locally.
			office, err = sessions.RenderOfficePDFBytesForSession(context.Background(), sid, path)
			if err == nil && valid() {
				doc, err = pdfpreview.OpenBytes(officePreviewName(path), office.Data, valid)
			} else if err == nil {
				err = fmt.Errorf("Office conversion session changed")
			}
		} else {
			if sourceErr == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				var data []byte
				data, err = mediapreview.ReadSnapshot(ctx, reader, sid, path, pdfpreview.MaxBytes, valid)
				if err == nil {
					doc, err = pdfpreview.OpenBytes(path, data, valid)
				}
			} else {
				doc, err = pdfpreview.OpenWorkspace(root, path, valid)
			}
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
				if convert {
					s.setOfficePreviewMessage(sessions, sid, path, officePreviewFailure(err))
				} else {
					sessions.SetLastAction(err.Error())
				}
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
			if err := s.browserController.OpenPDF(doc); err != nil {
				if office != nil {
					s.setOfficePreviewMessage(sessions, sid, path, "Office 已转换，但 PDF 阅读器无法打开："+err.Error())
				} else {
					sessions.SetLastAction(err.Error())
				}
			} else {
				if office != nil {
					s.setOfficePreviewMessage(sessions, sid, path, officePreviewReady(office.MissingFonts))
				} else {
					sessions.SetLastAction("PDF 只读预览标签已打开；原文件和编辑草稿保持不变")
				}
			}
		})
	}()
}
