package browserpreview

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pdfpreview"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
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
	h, err := desktop.NewHost(desktop.HostOptions{Parent: b.parent, URL: t.pdfViewerURL(doc.ViewerURL()), LocalPDFViewerURL: doc.ViewerURL(), Title: doc.Name, TopInset: chrome.TitleBarHeight + BrowserChromeHeight, Untrusted: true, OnChange: b.update, OnFailure: b.pageFailure(t), OnFocus: b.focus, OnClosed: doc.Close})
	if err != nil {
		doc.Close()
		if t.source != nil {
			t.Document = nil
		}
		t.Error = err.Error()
	} else {
		t.Host = h
		if b.current() == t {
			b.showHost(h)
		}
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
		b.capturePDFPosition(t)
		if t.Host != nil && t.Host.Alive() {
			_ = t.Host.Close()
		}
		t.Document.Close()
		if t.source != nil {
			b.releaseSourceDocument(t)
			continue
		}
		if t.Error == "" {
			t.Error = browserChromeLabel("PDF 预览已关闭、过期或所属会话改变；请重新打开文件，或使用系统打开。", "PDF preview closed, expired or changed session; reopen the file or use system open.")
		}
	}
}
func (b *Controller) documentControls(c *ui.Context, t *Tab) {
	b.documentHeader(c, t, t.Document.Name, t.Document.ExternalPath, false)
}
