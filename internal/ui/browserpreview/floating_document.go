package browserpreview

import (
	"errors"
	"time"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pdfpreview"
	desktop "whaleisle/internal/desktop/webview"
)

const floatingDocumentChromeHeight = 40

type floatingDocumentWindow struct {
	tab    *Tab
	pdf    *pdfpreview.Document
	html   *htmldocument.Document
	media  *mediapreview.Document
	host   *desktop.Host
	win    *mygo.Window
	done   chan struct{}
	closed bool
	status string
}

// A floating document borrows the already-loaded, immutable capability. The
// source tab remains its sole owner. Never reopen its filesystem path, transfer
// a Host cookie, or release its lease when only this borrowed view closes.
func floatingDocumentOptions(t *Tab) (desktop.HostOptions, error) {
	opts := desktop.HostOptions{TopInset: floatingDocumentChromeHeight, Untrusted: true}
	count := 0
	if t != nil {
		if t.Document != nil {
			count++
		}
		if t.HTML != nil {
			count++
		}
		if t.Media != nil {
			count++
		}
	}
	if count != 1 {
		return opts, errors.New("请先打开 PDF、静态 HTML 或音视频文件预览")
	}
	if t.Document != nil {
		if !t.Document.Valid() {
			return opts, errors.New("PDF 预览已过期；请重新打开文件")
		}
		opts.Title, opts.URL = t.Document.Name, t.Document.ViewerURL()
		opts.LocalPDFViewerURL = opts.URL
		opts.URL = t.pdfViewerURL(opts.URL)
	} else if t.Media != nil {
		if !t.Media.Valid() {
			return opts, errors.New("媒体预览已过期；请重新打开文件")
		}
		opts.Title, opts.URL = t.Media.Name, t.Media.URL()
		opts.LocalMediaURL = opts.URL
	} else {
		if !t.HTML.Valid() {
			return opts, errors.New("HTML 预览已过期；请重新打开文件")
		}
		opts.Title, opts.URL = t.HTML.Name, t.HTML.URL()
		opts.LocalHTMLURL = opts.URL
	}
	return opts, nil
}

func (b *Controller) openFloatingDocument(t *Tab) {
	if b.parent == nil || t == nil || b.current() != t || t.Host == nil || !t.Host.Alive() {
		return
	}
	opts, err := floatingDocumentOptions(t)
	if err != nil {
		t.Error = err.Error()
		b.update()
		return
	}
	b.stopFloatingDocument()
	f := &floatingDocumentWindow{tab: t, pdf: t.Document, html: t.HTML, media: t.Media, done: make(chan struct{})}
	b.floatingDocument = f
	w := mygo.NewWindow(mygo.WindowOptions{
		Title: "浮动预览 · " + opts.Title, Width: 720, Height: 520, MinWidth: 320, MinHeight: 220,
		Parent: b.parent, AlwaysOnTop: true, SkipTaskbar: true, DisableMinimize: true, DisableMaximize: true, DisableFullScreen: true,
		Content: ui.View(func(c *ui.Context) {
			fonttheme.ApplyInterface(c)
			ui.Column(c).Fill().Children(func() {
				ui.Row(c).Height(floatingDocumentChromeHeight).Padding(6, 10).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, truncate(opts.Title, 40)+" · 只读快照").FontSize(fonttheme.InterfaceSize(12)).Grow(1).Tooltip(opts.Title)
					if t.source != nil && ui.Button(c, "刷新").Disabled(t.sourceRead != nil).Clicked() {
						t.floatAfterRead = true
						b.retryDocument(t)
					}
					if t.Error != "" {
						ui.Text(c, t.Error).Tooltip(t.Error)
					}
					if ui.Button(c, "关闭").Clicked() {
						t.floatAfterRead = false
						b.stopFloatingDocument()
					}
				})
				if f.status != "" {
					ui.Text(c, f.status).FontSize(fonttheme.InterfaceSize(12)).Padding(12)
				}
			})
		}),
	})
	f.win = w
	if w == nil || w.IsDestroyed() {
		b.stopFloatingDocument()
		t.Error = "浮动预览窗口创建失败"
		b.update()
		return
	}
	w.OnClosed(func() {
		if b.floatingDocument == f {
			t.floatAfterRead = false
			b.stopFloatingDocument()
		} else {
			f.close()
		}
	})
	w.OnHide(func() {
		if f.host != nil {
			f.host.SetVisible(false)
		}
	})
	w.OnShow(func() {
		if f.host != nil && !f.closed {
			f.host.SetVisible(true)
		}
	})
	opts.Parent = w
	// Host failure closes only its borrowed child, never the source document.
	opts.OnClosed = func() {
		if !f.closed {
			f.status = "浮窗预览已停止；请关闭后重新打开。"
			w.Invalidate()
		}
	}
	h, err := desktop.NewHost(opts)
	if err != nil {
		b.stopFloatingDocument()
		t.Error = "浮动预览不可用：" + err.Error()
		b.update()
		return
	}
	f.host = h
	if !b.floatingDocumentCurrent(f) {
		b.stopFloatingDocument()
		return
	}
	w.ShowInactive()
	h.SetVisible(true)
	b.update()
	go b.watchFloatingDocument(f, w)
}

func (b *Controller) floatingDocumentCurrent(f *floatingDocumentWindow) bool {
	if f == nil || f.closed || b.floatingDocument != f || f.tab == nil || f.tab.Document != f.pdf || f.tab.HTML != f.html || f.tab.Media != f.media {
		return false
	}
	found := false
	for _, tab := range b.Tabs {
		if tab == f.tab {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	_, err := floatingDocumentOptions(f.tab)
	return err == nil
}
func (b *Controller) pruneFloatingDocument() {
	if f := b.floatingDocument; f != nil && !b.floatingDocumentCurrent(f) {
		b.stopFloatingDocument()
	}
}
func (b *Controller) stopFloatingDocument() {
	f := b.floatingDocument
	if f == nil {
		return
	}
	if f.pdf != nil && f.host != nil && f.tab != nil && f.tab.Document == f.pdf {
		if p, ok := pdfPositionFromURL(f.pdf.ViewerURL(), f.host.State().URL); ok {
			f.tab.pdfPosition = &p
			if f.tab.Host != nil && f.tab.Host.Alive() {
				_ = f.tab.Host.Navigate(f.tab.pdfViewerURL(f.pdf.ViewerURL()))
			}
		}
	}
	b.floatingDocument = nil
	f.close()
	b.update()
}
func (f *floatingDocumentWindow) close() {
	if f.closed {
		return
	}
	f.closed = true
	if f.done != nil {
		close(f.done)
	}
	if f.host != nil {
		_ = f.host.Close()
	}
	if f.win != nil && !f.win.IsDestroyed() {
		f.win.Destroy()
	}
}

// The existing frame sweeps handle Host/session changes immediately. This
// bounded watcher also clears expired capabilities while the owner is idle.
// All document validity and UI access stays on the UI thread, one queued check
// at a time; no document bytes, filesystem reads, or Host requests are involved.
func (b *Controller) watchFloatingDocument(f *floatingDocumentWindow, w *mygo.Window) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-f.done:
			return
		case <-ticker.C:
		}
		applied := make(chan struct{})
		w.Update(func() {
			defer close(applied)
			if b.floatingDocument == f {
				b.pruneFloatingDocument()
			}
		})
		select {
		case <-f.done:
			return
		case <-applied:
		}
	}
}
