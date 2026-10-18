package main

import (
	"context"
	"time"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
)

func (s *mainState) openMediaFile(sessions *session.Store, expectedClient *dsh.Client, sid, root, path string, floating bool) {
	if sessions == nil || expectedClient == nil || sid == "" {
		return
	}
	if root == "" || !mediapreview.ValidPath(path) {
		sessions.SetLastAction("音视频预览需要当前会话工作区和明确支持的单文件路径；不接受 URL 或路径遍历。")
		return
	}
	client, sourceValid, sourceErr := sessions.MediaPreviewSource(sid)
	s.mu.Lock()
	win, host := s.win, s.bootStatus
	s.mediaPreviewRequest++
	request := s.mediaPreviewRequest
	s.mu.Unlock()
	if win == nil || sourceErr != nil || client != expectedClient {
		sessions.SetLastAction("媒体预览需要当前会话的 Host 文件读取服务；不会把 Host 路径当成本机路径打开。")
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
	sessions.SetLastAction("正在读取当前 Host/会话的单个音视频快照（最大 32 MiB）…")
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
				message := "媒体预览失败；当前 Host/backend 必须支持有界字节读取。可在 Files 显式外部打开。"
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
				sessions.SetLastAction("媒体预览不可用，可在 Files 显式外部打开：" + err.Error())
			} else {
				sessions.SetLastAction("只读音视频快照已打开。请点击播放器播放；编解码支持取决于 WebView2，30 分钟后过期。")
			}
		})
	}()
}
