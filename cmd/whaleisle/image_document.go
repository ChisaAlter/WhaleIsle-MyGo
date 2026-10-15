package main

import (
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/store/session"
)

// openLocalImage follows the existing session-bound static snapshot lease.
// It is reached only by the Files image compatibility button.
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
	sessions.SetLastAction("正在校验并打开隔离图片快照，编辑草稿保持不变…")
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
				sessions.SetLastAction("隔离图片预览未打开：" + err.Error())
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
				sessions.SetLastAction("当前系统的隔离图片 WebView 不可用；图片未在兼容预览中显示")
			} else {
				sessions.SetLastAction("隔离图片快照已打开；" + doc.Resources.Summary() + "；无脚本/网络/应用桥接。AVIF 显示取决于当前 WebView 编解码器，源文件与草稿不变")
			}
		})
	}()
}

func supportsImagePreview(path string) bool { return imagepreview.Supports(path) }
