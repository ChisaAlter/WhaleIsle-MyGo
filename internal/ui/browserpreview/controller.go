package browserpreview

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pdfpreview"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
)

// Controller owns isolated native pages. It is used only from the UI thread;
// page state is owned by Host and its native navigation callbacks.
type Controller struct {
	frameContext              *ui.Context
	mini                      *browserMiniPlayer
	miniGeometry              mygo.Rectangle
	onMiniOpen, onMiniRestore func()
	discovery                 *localDiscovery
	developerTools            func() bool
	workspaceOwner            func() WorkspaceOwner
	documentLoader            documentLoader
	onFocus                   func()
	onFailure                 func(desktop.PageFailure)
	docked                    bool
	floatingDocument          *floatingDocumentWindow
	pip                       *browserPictureInPicture
	recording                 *browserRecording
	rendered                  bool
	actionTarget              func() ActionTarget
	operation                 *previewOperation
	viewport                  *mygo.Rectangle
	Tabs                      []*Tab
	Active                    int
	next                      int
	parent                    *mygo.Window
}
type Tab struct {
	toolsExpanded  bool
	pdfPosition    *PDFReadingPosition
	pdfPageDraft   string
	workspace      browserWorkspace
	source         *DocumentSource
	sourcePending  bool
	floatAfterRead bool
	sourceRead     *documentRead
	tools          browserTools
	Document       *pdfpreview.Document
	HTML           *htmldocument.Document
	Media          *mediapreview.Document
	interactive    *interactivePreview

	ActionMessage string
	ArtifactPath  string
	RecordingPath string
	ID            int
	Draft, Error  string
	lastURL       string
	resumeURL     string
	Host          *desktop.Host
}

const BrowserChromeHeight = 104
const BrowserResourceChromeHeight = 68

func NewController(parent *mygo.Window) *Controller { return &Controller{parent: parent} }
func (b *Controller) current() *Tab {
	if b.Active < 0 || b.Active >= len(b.Tabs) {
		return nil
	}
	return b.Tabs[b.Active]
}

// RefreshCurrent follows the current resource's own toolbar: authenticated
// document reread, restored browser address, or an existing native page reload.
func (b *Controller) RefreshCurrent() {
	t := b.current()
	if t == nil {
		return
	}
	if t.source != nil {
		b.retryDocument(t)
		return
	}
	b.CancelAction()
	t.Error, t.ActionMessage = "", ""
	if t.resumeURL != "" {
		b.restoreBrowserAddress(t)
	} else if t.Host != nil {
		t.Host.Reload()
	}
}
func (b *Controller) update() {
	if b.parent != nil {
		b.parent.Update(func() {})
	}
}
func (b *Controller) NewTab(raw string) {
	b.Hide()
	b.next++
	workspace, _ := b.browserWorkspace()
	b.Tabs = append(b.Tabs, &Tab{ID: b.next, workspace: workspace})
	b.Active = len(b.Tabs) - 1
	if raw != "" {
		b.Navigate(raw)
	}
	b.update()
}
func (b *Controller) Navigate(raw string) {
	b.CancelAction()
	if b.current() == nil {
		b.NewTab("")
	}
	t := b.current()
	workspace, scopeErr := b.browserWorkspace()
	if scopeErr != nil {
		t.Error = scopeErr.Error()
		b.update()
		return
	}
	if t.source != nil || t.Document != nil || t.HTML != nil || t.Media != nil || t.workspace.needsNewTab(workspace) {
		// Existing tabs keep their cookies/history in their original partition.
		b.NewTab("")
		t = b.current()
	}
	t.workspace = workspace
	t.Draft = raw
	normalized := NormalizeURL(raw)
	if normalized == "" {
		t.Error = browserChromeLabel("请输入有效的 HTTP(S) 地址", "Enter a valid HTTP(S) address")
		b.update()
		return
	}
	t.Draft = normalized
	t.Error = ""
	t.ActionMessage = ""
	if t.Host != nil && t.Host.Alive() {
		if state := t.Host.State(); state.URL == normalized && !state.Loading && state.Error == "" {
			// ElectronWebViewImpl.loadUrl reloads an already observed address;
			// submitting it again must not append a duplicate history entry.
			t.Host.Reload()
		} else if err := t.Host.Navigate(normalized); err != nil {
			t.Error = err.Error()
		}
		b.showHost(t.Host)
		return
	}
	h, err := desktop.NewHost(desktop.HostOptions{Parent: b.parent, URL: normalized, Title: "Browser", BrowserProfile: true, BrowserPartition: t.workspace.key, TopInset: chrome.TitleBarHeight + BrowserChromeHeight, Untrusted: true, OnChange: b.update, OnFocus: b.focus, OnFailure: b.pageFailure(t)})
	if err != nil {
		t.Error = err.Error()
		b.update()
		return
	}
	t.Host = h
	t.resumeURL = ""
	t.tools.settings.Scheme = "system"
	b.showHost(h)
	b.update()
}
func (b *Controller) Hide() {
	b.stopLocalDiscovery()
	b.pauseDocumentReads()
	b.CancelAction()
	b.cancelInteractivePreviews()
	for _, t := range b.Tabs {
		if t.Host != nil {
			t.Host.SetVisible(false)
		}
	}
}
func (b *Controller) Select(index int) {
	if index < 0 || index >= len(b.Tabs) {
		return
	}
	b.Hide()
	b.Active = index
	b.resumeCurrent()
	if t := b.current(); t.Host != nil {
		b.showHost(t.Host)
		if t.source == nil && t.Document == nil && t.HTML == nil && t.Media == nil {
			t.Draft = t.Host.State().URL
		}
	}
	b.update()
}
func (b *Controller) CloseTab(index int) {
	if index < 0 || index >= len(b.Tabs) {
		return
	}
	b.Hide()
	if b.mini != nil && b.mini.tab == b.Tabs[index] {
		b.CloseMiniPlayer()
	}
	if b.floatingDocument != nil && b.floatingDocument.tab == b.Tabs[index] {
		b.stopFloatingDocument()
	}
	if d := b.Tabs[index].Document; d != nil {
		d.Close()
	}
	if d := b.Tabs[index].Media; d != nil {
		d.Close()
	}
	if d := b.Tabs[index].HTML; d != nil {
		d.Close()
	}
	if h := b.Tabs[index].Host; h != nil {
		_ = h.Close()
	}
	b.Tabs = append(b.Tabs[:index], b.Tabs[index+1:]...)
	if index < b.Active {
		b.Active--
	}
	if b.Active >= len(b.Tabs) {
		b.Active = len(b.Tabs) - 1
	}
	if len(b.Tabs) == 0 {
		b.NewTab("")
	} else {
		b.Select(b.Active)
	}
}
func (b *Controller) Close() {
	b.CloseMiniPlayer()
	b.stopLocalDiscovery()
	b.pauseDocumentReads()
	b.stopFloatingDocument()
	b.stopPictureInPicture()
	b.CancelAction()
	b.cancelInteractivePreviews()
	for _, t := range b.Tabs {
		if t.Media != nil {
			t.Media.Close()
		}
		if t.HTML != nil {
			t.HTML.Close()
		}
		if t.Document != nil {
			t.Document.Close()
		}
		if t.Host != nil {
			_ = t.Host.Close()
		}
	}
	b.Tabs = nil
	b.Active = 0
}
func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return string(r)
}
