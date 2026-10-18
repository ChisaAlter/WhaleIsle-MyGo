package browserpreview

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmldocument"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
)

// OpenHTML owns only the selected file's immutable snapshot. It does not prepare
// a visual reply, capture a bitmap, serve sibling assets, or expose a Host bridge.
func (b *Controller) OpenHTML(doc *htmldocument.Document) error {
	b.Hide()
	b.next++
	t := &Tab{ID: b.next, HTML: doc, Draft: doc.Name}
	b.Tabs = append(b.Tabs, t)
	b.Active = len(b.Tabs) - 1
	return b.openHTMLTab(t, doc)
}

func (b *Controller) openHTMLTab(t *Tab, doc *htmldocument.Document) error {
	t.HTML, t.Draft = doc, doc.Name
	h, err := desktop.NewHost(desktop.HostOptions{Parent: b.parent, URL: doc.URL(), LocalHTMLURL: doc.URL(), Title: doc.Name, TopInset: chrome.TitleBarHeight + BrowserChromeHeight, Untrusted: true, OnChange: b.update, OnFocus: b.focus, OnClosed: doc.Close})
	if err != nil {
		doc.Close()
		if t.source != nil {
			t.HTML = nil
		}
		t.Error = browserChromeLabel("HTML 预览不可用，请使用系统打开 / HTML preview unavailable; use system open", "HTML preview unavailable; open externally")
	} else {
		t.Host = h
		if b.current() == t {
			b.showHost(h)
		}
	}
	b.update()
	return err
}

// PruneHTMLDocuments is called with the existing PDF lease sweep on every frame.
func (b *Controller) PruneHTMLDocuments() {
	b.pruneFloatingDocument()
	for _, t := range b.Tabs {
		if t.HTML == nil || t.HTML.Valid() {
			continue
		}
		if t.Host != nil && t.Host.Alive() {
			_ = t.Host.Close()
		}
		t.HTML.Close()
		if t.source != nil {
			b.releaseSourceDocument(t)
			continue
		}
		if t.Error == "" {
			t.Error = browserChromeLabel("HTML 预览已关闭、过期或所属会话改变；请重新打开文件，或使用系统打开。", "HTML preview closed, expired or changed session; reopen the file or open externally.")
		}
	}
}
func (b *Controller) htmlControls(c *ui.Context, t *Tab, dark bool) {
	ui.Text(c, t.HTML.Name).FontSize(fonttheme.InterfaceSize(12)).Grow(1)
	if ui.Button(c, browserChromeLabel("浮动预览", "Floating preview")).Disabled(t.Host == nil || !t.Host.Alive()).Clicked() {
		b.openFloatingDocument(t)
	}
	if t.HTML.CanRunScripts() {
		b.interactiveHTMLControls(c, t, dark)
	}
	if ui.Button(c, browserChromeLabel("系统打开", "Open externally")).Clicked() {
		path, err := t.HTML.ExternalPath()
		if err == nil {
			err = mygo.Shell.OpenPath(path)
		}
		if err != nil {
			t.Error = err.Error()
		}
	}
	if ui.Button(c, browserChromeLabel("复制文件路径", "Copy file path")).Clicked() {
		path, err := t.HTML.ExternalPath()
		if err != nil {
			t.Error = err.Error()
		} else {
			mygo.Clipboard.WriteText(path)
		}
	}
}
