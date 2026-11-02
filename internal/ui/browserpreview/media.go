package browserpreview

import (
	"errors"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/mediapreview"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
)

// OpenMedia owns one already-authorized immutable snapshot, not a Host client
// or local path. Native controls do not autoplay and expose no application JS.
func (b *Controller) OpenMedia(doc *mediapreview.Document, floating bool) error {
	if doc == nil || !doc.Valid() {
		return errors.New("media preview expired")
	}
	b.Hide()
	b.next++
	t := &Tab{ID: b.next, Media: doc, Draft: doc.Name}
	b.Tabs = append(b.Tabs, t)
	b.Active = len(b.Tabs) - 1
	h, err := desktop.NewHost(desktop.HostOptions{Parent: b.parent, URL: doc.URL(), LocalMediaURL: doc.URL(), Title: doc.Name, TopInset: chrome.TitleBarHeight + BrowserChromeHeight, Untrusted: true, OnChange: b.update, OnClosed: doc.Close})
	if err != nil {
		doc.Close()
		t.Error = browserChromeLabel("音视频预览不可用：", "Media preview unavailable: ") + err.Error()
	} else {
		t.Host = h
		b.showHost(h)
		if floating {
			b.openFloatingDocument(t)
		}
	}
	b.update()
	// Expiry also stops a buffered player while no owner frame is repainting.
	if b.parent != nil {
		go func() { <-doc.Done(); b.parent.Update(func() { b.PruneMediaDocuments() }) }()
	}
	return err
}
func (b *Controller) PruneMediaDocuments() {
	b.pruneFloatingDocument()
	for _, t := range b.Tabs {
		if t.Media == nil || t.Media.Valid() {
			continue
		}
		if t.Host != nil && t.Host.Alive() {
			_ = t.Host.Close()
		}
		t.Media.Close()
		if t.source != nil && t.source.LocalDelivery {
			if !t.source.LocalDeliveryValid() {
				b.denyLocalDocument(t)
			} else {
				b.clearSourceDocument(t)
				t.sourcePending = true
				t.Error = ""
				t.ActionMessage = browserChromeLabel("媒体快照已结束，正在重新读取原交付文件…", "Media snapshot ended; rereading the original delivery file…")
			}
			continue
		}
		if t.Error == "" {
			t.Error = browserChromeLabel("媒体预览已过期、关闭或 Host/会话改变；请从 Files 重新打开。", "Media preview expired, closed or changed Host/session; reopen it from Files.")
		}
	}
}
func (b *Controller) mediaControls(c *ui.Context, t *Tab) {
	ui.Text(c, t.Media.Name).FontSize(fonttheme.InterfaceSize(12)).Grow(1)
	if ui.Button(c, browserChromeLabel("浮动预览", "Floating preview")).Disabled(t.Host == nil || !t.Host.Alive()).Clicked() {
		b.openFloatingDocument(t)
	}
	ui.Text(c, browserChromeLabel("≤32 MiB · 30 分钟 · 播放支持取决于 WebView2", "≤32 MiB · 30 minutes · Playback support depends on WebView2")).FontSize(fonttheme.InterfaceSize(10))
}
