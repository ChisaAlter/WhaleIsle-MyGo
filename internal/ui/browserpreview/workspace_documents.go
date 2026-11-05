package browserpreview

import (
	"context"
	"fmt"
	"path/filepath"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pdfpreview"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

const (
	DocumentImage     = "image"
	DocumentPDF       = "pdf"
	DocumentHTML      = "html"
	DocumentMedia     = "media"
	DocumentOfficePDF = "office-pdf"
	DocumentHost      = "host-document"
)

// DocumentSource describes a resource, never a temporary server URL or lease.
// Every restored occurrence obtains new bytes under its original source scope.
type DocumentSource struct {
	DeclaredDelivery bool                    `json:"declaredDelivery,omitempty"`
	declaredDelivery *declaredDocumentPermit `json:"-"`
	// The durable marker prevents recovery from borrowing the ordinary Host
	// loader. Its live permit is deliberately private and absent from JSON.
	LocalDelivery    bool                 `json:"localDelivery,omitempty"`
	localDelivery    *localDocumentPermit `json:"-"`
	NavigationPaneID string               `json:"navigationPaneId,omitempty"`
	Authority        string               `json:"authority"`
	SessionID        string               `json:"sessionId"`
	ViewSessionID    string               `json:"viewSessionId,omitempty"`
	AddressKind      string               `json:"addressKind,omitempty"`
	ParentSessionID  string               `json:"parentSessionId,omitempty"`
	Mode             string               `json:"mode,omitempty"`
	Workspace        string               `json:"workspace"`
	Path             string               `json:"path"`
	Kind             string               `json:"kind"`
}

// BindLocalDeliveryDocument issues only an in-memory permission for this exact
// resource and this owner. Copying or serializing a descriptor grants no access.
func BindLocalDeliveryDocument(source DocumentSource, valid func() bool) DocumentSource {
	source.LocalDelivery = true
	source.localDelivery = nil
	source.localDelivery = &localDocumentPermit{source: source, valid: valid}
	return source
}

type localDocumentPermit struct {
	source  DocumentSource
	valid   func() bool
	revoked atomic.Bool
}

func (source DocumentSource) LocalDeliveryValid() bool {
	permit := source.localDelivery
	if !source.LocalDelivery || permit == nil || permit.revoked.Load() {
		return false
	}
	copy := source
	copy.localDelivery = nil
	if copy != permit.source {
		return false
	}
	if permit.valid == nil || !permit.valid() {
		permit.revoked.Store(true)
		return false
	}
	return !permit.revoked.Load()
}

func localDocumentPermissionError() string {
	return browserChromeLabel("本地交付文件的预览权限已失效；请从当前会话的原交付卡重新打开。", "Local delivery preview permission ended; reopen the file from its original delivery card in the current session.")
}

type DocumentResult struct {
	Viewer   *desktop.HostOptions
	Media    *mediapreview.Document
	PDF      *pdfpreview.Document
	HTML     *htmldocument.Document
	Err      error
	Valid    func() bool
	Complete func(error)
	Canceled func()
}

func (r DocumentResult) close() {
	if r.Media != nil {
		r.Media.Close()
	}
	if r.PDF != nil {
		r.PDF.Close()
	}
	if r.HTML != nil {
		r.HTML.Close()
	}
}

type documentLoader struct {
	ready func(DocumentSource) bool
	read  func(context.Context, DocumentSource) DocumentResult
}
type documentRead struct{ cancel context.CancelFunc }

func (b *Controller) SetDocumentLoader(ready func(DocumentSource) bool, read func(context.Context, DocumentSource) DocumentResult) {
	b.documentLoader = documentLoader{ready: ready, read: read}
}

func (b *Controller) OpenDocumentSource(source DocumentSource, floating ...bool) {
	float := len(floating) > 0 && floating[0]
	// A fresh local gesture has a fresh permit. Reuse only a currently valid
	// occurrence of the exact public descriptor; never resurrect an old owner.
	if source.LocalDelivery {
		if source.LocalDeliveryValid() {
			for i, tab := range b.Tabs {
				if tab.source == nil || !sameLocalDocumentDescriptor(*tab.source, source) || !tab.source.LocalDeliveryValid() {
					continue
				}
				b.Hide()
				b.clearSourceDocument(tab)
				tab.source, tab.sourcePending, tab.floatAfterRead, tab.Draft = &source, true, float, source.Path
				tab.Error = ""
				tab.ActionMessage = browserChromeLabel("正在读取本地交付文件…", "Reading local delivery file…")
				b.Active = i
				if b.documentLoader.ready == nil || !b.documentLoader.ready(source) {
					b.denyLocalDocument(tab)
				}
				b.update()
				return
			}
		}
	}
	// Ordinary document behavior retains its original equality/focus policy.
	for i, tab := range b.Tabs {
		if !source.LocalDelivery && tab.source != nil && *tab.source == source {
			b.Select(i)
			tab.floatAfterRead = float
			if float && tab.sourceRead != nil {
				tab.sourceRead.cancel()
				tab.sourceRead = nil
			}
			if float || (tab.Document == nil && tab.HTML == nil && tab.Media == nil) {
				b.retryDocument(tab)
			}
			return
		}
	}
	b.Hide()
	b.next++
	tab := &Tab{ID: b.next, source: &source, sourcePending: true, floatAfterRead: float, Draft: source.Path}
	b.Tabs = append(b.Tabs, tab)
	b.Active = len(b.Tabs) - 1
	// OpenDocumentSource is the deferred WorkspacePanes route's consumption
	// point. Check again before any source bytes or native viewer can be used.
	if source.LocalDelivery || source.AddressKind == "local-delivery" {
		if !source.LocalDeliveryValid() || b.documentLoader.ready == nil || !b.documentLoader.ready(source) {
			b.denyLocalDocument(tab)
		} else {
			tab.ActionMessage = browserChromeLabel("正在读取本地交付文件…", "Reading local delivery file…")
		}
	}
	b.update()
}

func (b *Controller) pauseDocumentReads() {
	for _, tab := range b.Tabs {
		if tab.sourceRead != nil {
			tab.sourceRead.cancel()
			tab.sourceRead = nil
			tab.sourcePending = true
		}
	}
}

func sameLocalDocumentDescriptor(a, b DocumentSource) bool {
	if !a.LocalDelivery || !b.LocalDelivery {
		return false
	}
	a.localDelivery, b.localDelivery = nil, nil
	return a == b
}

// Clear the previous read and immutable viewer before rebinding a live gesture.
// Its late completion no longer matches sourceRead and cannot paint old bytes.
func (b *Controller) clearSourceDocument(tab *Tab) {
	if tab.sourceRead != nil {
		tab.sourceRead.cancel()
		tab.sourceRead = nil
	}
	if b.floatingDocument != nil && b.floatingDocument.tab == tab {
		b.stopFloatingDocument()
	}
	if tab.Host != nil {
		_ = tab.Host.Close()
		tab.Host = nil
	}
	if tab.Document != nil {
		tab.Document.Close()
		tab.Document = nil
	}
	if tab.HTML != nil {
		b.cancelInteractivePreviews()
		tab.HTML.Close()
		tab.HTML = nil
	}
	if tab.Media != nil {
		tab.Media.Close()
		tab.Media = nil
	}
}

// denyLocalDocument also runs for an already displayed source, before the next
// owner frame can show it again. Buffered native content loses its host as well.
func (b *Controller) denyLocalDocument(tab *Tab) {
	b.clearSourceDocument(tab)
	tab.sourcePending = false
	tab.ActionMessage = ""
	tab.Error = localDocumentPermissionError()
}

func (b *Controller) resumeDocument(tab *Tab) {
	if tab.source != nil && tab.source.DeclaredDelivery && !tab.source.DeclaredDeliveryValid() {
		b.denyDeclaredDocument(tab)
		return
	}
	if tab.source != nil && tab.source.LocalDelivery && !tab.source.LocalDeliveryValid() {
		b.denyLocalDocument(tab)
		return
	}
	if tab.source == nil || !tab.sourcePending || tab.sourceRead != nil {
		return
	}
	loader := b.documentLoader
	if loader.ready == nil || loader.read == nil || !loader.ready(*tab.source) {
		if tab.source.LocalDelivery {
			b.denyLocalDocument(tab)
			return
		}
		tab.ActionMessage = browserChromeLabel("切回文档所属会话和工作区后，将重新读取源文件恢复预览。", "Return to the document’s session and workspace to reread its source and restore the preview.")
		if tab.source.AddressKind == "subagent" {
			tab.ActionMessage = browserChromeLabel("请在原 Agents 面板重新进入文档所属子会话；若原面板已关闭，请从新的来源面板重新打开此文件。", "Re-enter the document’s child session in its original Agents panel. If that panel is closed, reopen the file from a new source panel.")
		}
		return
	}
	source := *tab.source
	ctx, cancel := context.WithCancel(context.Background())
	request := &documentRead{cancel: cancel}
	tab.sourceRead = request
	tab.sourcePending = false
	tab.Error = ""
	tab.ActionMessage = browserChromeLabel("正在从源文件恢复文档预览…", "Restoring document preview from its source…")
	go func() {
		result := loader.read(ctx, source)
		if ctx.Err() != nil {
			result.close()
			if result.Canceled != nil {
				result.Canceled()
			}
			return
		}
		b.parent.Update(func() { b.finishDocumentRead(tab, request, ctx, source, loader, result) })
	}()
}

func (b *Controller) finishDocumentRead(tab *Tab, request *documentRead, ctx context.Context, source DocumentSource, loader documentLoader, result DocumentResult) {
	if tab.sourceRead != request || ctx.Err() != nil {
		result.close()
		if result.Canceled != nil {
			result.Canceled()
		}
		return
	}
	tab.sourceRead = nil
	request.cancel()
	if (source.LocalDelivery && !source.LocalDeliveryValid()) || loader.ready == nil || !loader.ready(source) || (result.Valid != nil && !result.Valid()) {
		result.close()
		if source.LocalDelivery {
			b.denyLocalDocument(tab)
		} else {
			tab.sourcePending = true
			tab.ActionMessage = browserChromeLabel("文档所属会话已改变；切回后继续恢复。", "The document’s session changed; switch back to resume restoration.")
		}
		return
	}
	tab.ActionMessage = ""
	err := result.Err
	if err == nil {
		if b.floatingDocument != nil && b.floatingDocument.tab == tab {
			b.stopFloatingDocument()
		}
		if tab.Host != nil {
			_ = tab.Host.Close()
			tab.Host = nil
		}
		if tab.Document != nil {
			tab.Document.Close()
			tab.Document = nil
		}
		if tab.HTML != nil {
			tab.HTML.Close()
			tab.HTML = nil
		}
		if tab.Media != nil {
			tab.Media.Close()
			tab.Media = nil
		}
		switch {
		case result.Viewer != nil:
			opts := *result.Viewer
			opts.Parent, opts.OnChange, opts.OnFocus, opts.OnFailure = b.parent, b.update, b.focus, b.pageFailure(tab)
			tab.Host, err = desktop.NewHost(opts)
			if err == nil && b.current() == tab {
				b.showHost(tab.Host)
			}
		case result.PDF != nil:
			err = b.openPDFTab(tab, result.PDF)
		case result.HTML != nil:
			err = b.openHTMLTab(tab, result.HTML)
		case result.Media != nil:
			err = b.openSourceMediaTab(tab, result.Media)
		default:
			err = fmt.Errorf("%s", browserChromeLabel("文档读取未返回预览内容", "Document read returned no preview content"))
		}
	} else {
		result.close()
	}
	if err != nil {
		tab.Error = err.Error()
	}
	if err == nil && source.LocalDelivery && !source.LocalDeliveryValid() {
		b.denyLocalDocument(tab)
		err = fmt.Errorf("%s", tab.Error)
	}
	if err == nil && tab.floatAfterRead && b.current() == tab {
		tab.floatAfterRead = false
		b.openFloatingDocument(tab)
	}
	if result.Complete != nil {
		result.Complete(err)
	}
}

// A source media result uses the existing isolated LocalMediaURL viewer, while
// retaining the loading/error/source tab instead of creating an unrelated tab.
func (b *Controller) openSourceMediaTab(tab *Tab, doc *mediapreview.Document) error {
	if !doc.Valid() {
		doc.Close()
		return fmt.Errorf("media preview expired")
	}
	tab.Media, tab.Draft = doc, doc.Name
	h, err := desktop.NewHost(desktop.HostOptions{Parent: b.parent, URL: doc.URL(), LocalMediaURL: doc.URL(), Title: doc.Name, TopInset: chrome.TitleBarHeight + BrowserChromeHeight, Untrusted: true, OnChange: b.update, OnFailure: b.pageFailure(tab), OnFocus: b.focus, OnClosed: doc.Close})
	if err != nil {
		doc.Close()
		tab.Media = nil
		tab.Error = err.Error()
		return err
	}
	tab.Host = h
	if b.current() == tab {
		b.showHost(h)
	}
	if b.parent != nil {
		go func() { <-doc.Done(); b.parent.Update(func() { b.PruneMediaDocuments() }) }()
	}
	b.update()
	return nil
}

func (b *Controller) retryDocument(tab *Tab) {
	if tab.sourceRead != nil {
		return
	}
	tab.Error, tab.ActionMessage = "", ""
	tab.sourcePending = true
	if tab == b.current() {
		b.resumeDocument(tab)
	}
	b.update()
}

func (b *Controller) documentSourceControls(c *ui.Context, tab *Tab) {
	if tab.source.DeclaredDelivery {
		b.documentHeader(c, tab, filepath.Base(tab.source.Path), nil, false)
		return
	}
	ui.Text(c, filepath.Base(tab.source.Path)).FontSize(theme.InterfaceSize(12)).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(tab.source.Path)
	if ui.Button(c, browserChromeLabel("重新读取", "Reread")).Disabled(tab.sourceRead != nil).Clicked() {
		b.retryDocument(tab)
	}
}

// An expired lease is discarded, but the original resource descriptor remains.
// The loader, not an expired capability URL, decides when it can be read again.
func (b *Controller) releaseSourceDocument(tab *Tab) {
	if tab.source == nil {
		return
	}
	if tab.source.LocalDelivery && !tab.source.LocalDeliveryValid() {
		b.denyLocalDocument(tab)
		return
	}
	b.capturePDFPosition(tab)
	if tab.HTML != nil {
		b.cancelInteractivePreviews()
	}
	tab.Document, tab.HTML, tab.Host = nil, nil, nil
	tab.Error, tab.ActionMessage = "", ""
	tab.sourcePending = true
}
