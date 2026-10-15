package main

import (
	"context"
	"time"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/store/session"
)

// openLocalHTML is reached only by an explicit Files/confirmed delivery gesture.
// It reads the saved file; selecting another file or previewing never saves,
// discards or publishes the Files editor's per-tab draft.
func (s *mainState) openLocalHTML(sessions *session.Store, sid, root, path string) {
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
	sessions.SetLastAction("正在打开已保存的 HTML 快照，编辑草稿保持不变…")
	go func() {
		var doc *htmldocument.Document
		var err error
		if sourceErr == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			var data []byte
			data, err = mediapreview.ReadSnapshot(ctx, reader, sid, path, htmldocument.MaxBytes, valid)
			if err == nil {
				doc, err = htmldocument.OpenBytes(path, data, valid)
			}
		} else {
			doc, err = htmldocument.OpenWorkspace(root, path, valid)
		}
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
				sessions.SetLastAction("HTML 预览失败，可在 Files 使用系统打开：" + err.Error())
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
				sessions.SetLastAction("HTML 预览不可用，请在预览标签使用系统打开")
			} else {
				sessions.SetLastAction("静态 HTML 快照已打开；" + doc.Resources.Summary() + "；源文件与编辑草稿保持不变")
			}
		})
	}()
}
