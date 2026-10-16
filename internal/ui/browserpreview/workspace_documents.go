package browserpreview

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/pdfpreview"
	"whaleisle/internal/ui/theme"
)

const (
	DocumentPDF       = "pdf"
	DocumentHTML      = "html"
	DocumentOfficePDF = "office-pdf"
)

// DocumentSource describes a resource, never a temporary server URL or lease.
// Every restored occurrence obtains new bytes under its original source scope.
type DocumentSource struct {
	Authority       string `json:"authority"`
	SessionID       string `json:"sessionId"`
	ViewSessionID   string `json:"viewSessionId,omitempty"`
	AddressKind     string `json:"addressKind,omitempty"`
	ParentSessionID string `json:"parentSessionId,omitempty"`
	Mode            string `json:"mode,omitempty"`
	Workspace       string `json:"workspace"`
	Path            string `json:"path"`
	Kind            string `json:"kind"`
}

type DocumentResult struct {
	PDF      *pdfpreview.Document
	HTML     *htmldocument.Document
	Err      error
	Valid    func() bool
	Complete func(error)
	Canceled func()
}

func (r DocumentResult) close() {
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

func (b *Controller) OpenDocumentSource(source DocumentSource) {
	// Reopening one resource focuses its occurrence in this pane only.
	for i, tab := range b.Tabs {
		if tab.source != nil && *tab.source == source {
			b.Select(i)
			if tab.Document == nil && tab.HTML == nil {
				b.retryDocument(tab)
			}
			return
		}
	}
	b.Hide()
	b.next++
	b.Tabs = append(b.Tabs, &Tab{ID: b.next, source: &source, sourcePending: true, Draft: source.Path})
	b.Active = len(b.Tabs) - 1
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

func (b *Controller) resumeDocument(tab *Tab) {
	if tab.source == nil || !tab.sourcePending || tab.sourceRead != nil {
		return
	}
	loader := b.documentLoader
	if loader.ready == nil || loader.read == nil || !loader.ready(*tab.source) {
		tab.ActionMessage = "切回文档所属会话和工作区后，将重新读取源文件恢复预览。"
		if tab.source.AddressKind == "subagent" {
			tab.ActionMessage = "请在 Agents 面板重新进入文档所属子会话；已有源绑定恢复后将重新读取，无需再次选择文件。"
		}
		return
	}
	source := *tab.source
	ctx, cancel := context.WithCancel(context.Background())
	request := &documentRead{cancel: cancel}
	tab.sourceRead = request
	tab.sourcePending = false
	tab.Error = ""
	tab.ActionMessage = "正在从源文件恢复文档预览…"
	go func() {
		result := loader.read(ctx, source)
		if ctx.Err() != nil {
			result.close()
			if result.Canceled != nil {
				result.Canceled()
			}
			return
		}
		b.parent.Update(func() {
			if tab.sourceRead != request || ctx.Err() != nil {
				result.close()
				if result.Canceled != nil {
					result.Canceled()
				}
				return
			}
			tab.sourceRead = nil
			cancel()
			if !loader.ready(source) || (result.Valid != nil && !result.Valid()) {
				result.close()
				tab.sourcePending = true
				tab.ActionMessage = "文档所属会话已改变；切回后继续恢复。"
				return
			}
			tab.ActionMessage = ""
			err := result.Err
			if err == nil {
				switch {
				case result.PDF != nil:
					err = b.openPDFTab(tab, result.PDF)
				case result.HTML != nil:
					err = b.openHTMLTab(tab, result.HTML)
				default:
					err = fmt.Errorf("文档读取未返回预览内容")
				}
			} else {
				result.close()
			}
			if err != nil {
				tab.Error = err.Error()
			}
			if result.Complete != nil {
				result.Complete(err)
			}
		})
	}()
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
	ui.Text(c, filepath.Base(tab.source.Path)).FontSize(theme.InterfaceSize(12)).Grow(1).Tooltip(tab.source.Path)
	if ui.Button(c, "重新读取").Disabled(tab.sourceRead != nil).Clicked() {
		b.retryDocument(tab)
	}
}

// An expired lease is discarded, but the original resource descriptor remains.
// The loader, not an expired capability URL, decides when it can be read again.
func (b *Controller) releaseSourceDocument(tab *Tab) {
	if tab.source == nil {
		return
	}
	tab.capturePDFPosition()
	if tab.HTML != nil {
		b.cancelInteractivePreviews()
	}
	tab.Document, tab.HTML, tab.Host = nil, nil, nil
	tab.Error, tab.ActionMessage = "", ""
	tab.sourcePending = true
}
