package browserpreview

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pdfpreview"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
)

// OpenPDF owns the document lease until its tab/window closes. A native viewer
// failure leaves a clear system-open action, never an arbitrary file: fallback.
func (b *Controller) OpenPDF(doc *pdfpreview.Document) error {
	b.Hide()
	b.next++
	t := &Tab{ID: b.next, Document: doc, Draft: doc.Name}
	b.Tabs = append(b.Tabs, t)
	b.Active = len(b.Tabs) - 1
	return b.openPDFTab(t, doc)
}

func (b *Controller) openPDFTab(t *Tab, doc *pdfpreview.Document) error {
	t.Document, t.Draft = doc, doc.Name
	h, err := desktop.NewHost(desktop.HostOptions{Parent: b.parent, URL: doc.URL(), LocalPDFURL: doc.URL(), Title: doc.Name, TopInset: chrome.TitleBarHeight + BrowserChromeHeight, Untrusted: true, OnChange: b.update, OnFocus: b.focus, OnClosed: doc.Close})
	if err != nil {
		doc.Close()
		if t.source != nil { t.Document = nil }
		t.Error = err.Error()
	} else {
		t.Host = h
		if b.current() == t { b.showHost(h) }
	}
	b.update()
	return err
}

// Called on every owning-window frame, including session/Host changes.
func (b *Controller) PrunePDFDocuments() {
	b.pruneFloatingDocument()
	for _, t := range b.Tabs {
		if t.Document == nil || t.Document.Valid() {
			continue
		}
		if t.Host != nil && t.Host.Alive() {
			_ = t.Host.Close()
		}
		t.Document.Close()
		if t.source != nil {
			b.releaseSourceDocument(t)
			continue
		}
		if t.Error == "" {
			t.Error = "PDF 预览已关闭、过期或所属会话改变；请重新打开文件，或使用系统打开。"
		}
	}
}
func (b *Controller) documentControls(c *ui.Context, t *Tab) {
	ui.Text(c, t.Document.Name).FontSize(fonttheme.InterfaceSize(12)).Grow(1)
	if ui.Button(c, "浮动预览").Disabled(t.Host == nil || !t.Host.Alive()).Clicked() {
		b.openFloatingDocument(t)
	}
	if ui.Button(c, "系统打开").Clicked() {
		path, err := t.Document.ExternalPath()
		if err == nil {
			err = mygo.Shell.OpenPath(path)
		}
		if err != nil {
			t.Error = err.Error()
		}
	}
	if ui.Button(c, "复制文件路径").Clicked() {
		path, err := t.Document.ExternalPath()
		if err != nil {
			t.Error = err.Error()
		} else {
			mygo.Clipboard.WriteText(path)
		}
	}
}
