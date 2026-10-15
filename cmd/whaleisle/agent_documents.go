package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pdfpreview"
	"whaleisle/internal/store/session"
)

// openAgentDocument uses the shared PDF/static-HTML browser tabs with a child
// source lease. It never selects that child or installs a parent workspace root.
func (s *mainState) openAgentDocument(sessions *session.Store, token uint64, address session.AgentSessionAddress, path string) {
	if sessions == nil {
		return
	}
	reader, sourceValid, err := sessions.AgentMediaPreviewSource(token, address)
	if err != nil {
		return
	}
	s.mu.Lock()
	win := s.win
	s.pdfPreviewRequest++
	s.htmlDocumentRequest++
	pdfRequest, htmlRequest := s.pdfPreviewRequest, s.htmlDocumentRequest
	s.mu.Unlock()
	if win == nil {
		return
	}
	valid := func() bool {
		s.mu.Lock()
		current := !s.quitting && s.sessions == sessions && s.win == win
		s.mu.Unlock()
		return current && sourceValid()
	}
	pdf := strings.EqualFold(filepath.Ext(path), ".pdf")
	limit := htmldocument.MaxBytes
	if pdf {
		limit = pdfpreview.MaxBytes
	}
	sessions.SetAgentResourceStatus(token, address, path, true, nil)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		data, err := mediapreview.ReadSnapshot(ctx, reader, address.ChildSessionID, path, limit, valid)
		var pdfDoc *pdfpreview.Document
		var htmlDoc *htmldocument.Document
		if err == nil {
			if pdf {
				pdfDoc, err = pdfpreview.OpenBytes(path, data, valid)
			} else {
				htmlDoc, err = htmldocument.OpenBytes(path, data, valid)
			}
		}
		closeDocument := func() {
			if pdfDoc != nil {
				pdfDoc.Close()
			}
			if htmlDoc != nil {
				htmlDoc.Close()
			}
		}
		win.Update(func() {
			s.mu.Lock()
			same := s.pdfPreviewRequest == pdfRequest && s.htmlDocumentRequest == htmlRequest
			s.mu.Unlock()
			if !valid() || !same {
				closeDocument()
				sessions.SetAgentResourceStatus(token, address, path, false, errors.New("document preview replaced; reopen to retry"))
				return
			}
			if err != nil {
				sessions.SetAgentResourceStatus(token, address, path, false, err)
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
			if pdf {
				err = s.browserController.OpenPDF(pdfDoc)
			} else {
				err = s.browserController.OpenHTML(htmlDoc)
			}
			if err != nil {
				closeDocument()
			}
			sessions.SetAgentResourceStatus(token, address, path, false, err)
		})
	}()
}
