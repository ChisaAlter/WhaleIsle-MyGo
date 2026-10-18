package browserpreview

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/pdfpreview"
)

// Synthetic fixtures only. No native window, renderer, external document or
// user's workspace/Host is opened by these ownership tests.
func TestFloatingDocumentBorrowAndOrigin(t *testing.T) {
	for _, kind := range []string{"pdf", "html"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			name, text := "snapshot.pdf", "%PDF-1.4\n%%EOF\n"
			if kind == "html" {
				name, text = "snapshot.html", "<!doctype html><p>Static fixture</p>"
			}
			if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			var current atomic.Bool
			current.Store(true)
			tab := &Tab{ID: 1}
			var valid func() bool
			var closeSource func()
			if kind == "pdf" {
				doc, err := pdfpreview.OpenWorkspace(root, name, current.Load)
				if err != nil {
					t.Fatal(err)
				}
				tab.Document = doc
				valid = doc.Valid
				closeSource = doc.Close
			} else {
				doc, err := htmldocument.OpenWorkspace(root, name, current.Load)
				if err != nil {
					t.Fatal(err)
				}
				tab.HTML = doc
				valid = doc.Valid
				closeSource = doc.Close
			}
			defer closeSource()
			opts, err := floatingDocumentOptions(tab)
			if err != nil {
				t.Fatal(err)
			}
			if opts.URL == "" || opts.Title != name || !opts.Untrusted || opts.SessionOrigin != "" || opts.SessionCookie != "" || len(opts.TrustedOrigins) != 0 || opts.OnClosed != nil {
				t.Fatal("borrowed preview must not acquire Host authority or close the source lease")
			}
			if kind == "pdf" && (opts.LocalPDFViewerURL != opts.URL || opts.LocalHTMLURL != "") {
				t.Fatal("PDF capability not pinned")
			}
			if kind == "html" && (opts.LocalHTMLURL != opts.URL || opts.LocalPDFURL != "") {
				t.Fatal("HTML capability not pinned")
			}
			b := &Controller{Tabs: []*Tab{tab}}
			attach := func() *floatingDocumentWindow {
				f := &floatingDocumentWindow{tab: tab, pdf: tab.Document, html: tab.HTML, done: make(chan struct{})}
				b.floatingDocument = f
				return f
			}
			f := attach()
			if !b.floatingDocumentCurrent(f) {
				t.Fatal("active source refused")
			}
			b.stopFloatingDocument()
			b.stopFloatingDocument()
			if b.floatingDocument != nil || !f.closed || !valid() {
				t.Fatal("closing borrowed view destroyed source lease")
			}
			select {
			case <-f.done:
			default:
				t.Fatal("watcher not cancelled")
			}

			f = attach()
			b.Tabs = append(b.Tabs, &Tab{ID: 2})
			b.Active = 1
			b.pruneFloatingDocument()
			if b.floatingDocument != f {
				t.Fatal("ordinary tab selection revoked independent preview")
			}
			b.Tabs = b.Tabs[1:]
			b.pruneFloatingDocument()
			if b.floatingDocument != nil || !valid() {
				t.Fatal("removed source tab was not detached safely")
			}

			b.Tabs = []*Tab{tab}
			b.Active = 0
			f = attach()
			pdf, html := tab.Document, tab.HTML
			tab.Document, tab.HTML = nil, nil
			b.pruneFloatingDocument()
			if b.floatingDocument != nil || !f.closed || !valid() {
				t.Fatal("replaced capability retained old floating view")
			}
			tab.Document, tab.HTML = pdf, html
			f = attach()
			current.Store(false)
			b.pruneFloatingDocument()
			if b.floatingDocument != nil || !f.closed || valid() {
				t.Fatal("Host/session revocation left floating snapshot visible")
			}
			if _, err := floatingDocumentOptions(tab); err == nil {
				t.Fatal("revoked capability reopened")
			}
		})
	}
}

func TestFloatingDocumentRefusesUnscopedViews(t *testing.T) {
	for _, tab := range []*Tab{nil, {}, {Document: &pdfpreview.Document{}, HTML: &htmldocument.Document{}}} {
		if _, err := floatingDocumentOptions(tab); err == nil {
			t.Fatal("ambiguous/non-document view accepted")
		}
	}
	b := NewController(nil)
	b.openFloatingDocument(&Tab{})
	if b.floatingDocument != nil {
		t.Fatal("missing native parent opened a view")
	}
}
