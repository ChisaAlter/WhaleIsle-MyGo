package browserpreview

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pdfpreview"
	"whaleisle/internal/ui/i18n"
)

func localDeliveryBrowserSource(valid func() bool) DocumentSource {
	return BindLocalDeliveryDocument(DocumentSource{Authority: "local-owner", SessionID: "same", ViewSessionID: "same", AddressKind: "local-delivery", Workspace: "/workspace", Path: "/workspace/saved.pdf", Kind: DocumentPDF}, valid)
}

func TestDocumentLocalDeliveryPermitCannotPersistChangeResourceOrRevive(t *testing.T) {
	live := true
	source := localDeliveryBrowserSource(func() bool { return live })
	if !source.LocalDeliveryValid() {
		t.Fatal("live permit rejected")
	}
	for _, field := range []string{"path", "workspace", "session", "authority", "kind", "address"} {
		changed := source
		switch field {
		case "path":
			changed.Path += "2"
		case "workspace":
			changed.Workspace += "2"
		case "session":
			changed.SessionID += "2"
		case "authority":
			changed.Authority += "2"
		case "kind":
			changed.Kind = DocumentHTML
		case "address":
			changed.AddressKind = "session"
		}
		if changed.LocalDeliveryValid() {
			t.Fatalf("permit authorized changed %s", field)
		}
	}
	raw, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "localDocumentPermit") || strings.Contains(string(raw), "valid") {
		t.Fatal("permit serialized")
	}
	var restored DocumentSource
	if err = json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.LocalDelivery || restored.LocalDeliveryValid() {
		t.Fatal("serialized descriptor recreated permission")
	}
	live = false
	if source.LocalDeliveryValid() {
		t.Fatal("revocation ignored")
	}
	live = true
	if source.LocalDeliveryValid() {
		t.Fatal("old permit revived")
	}
}

func TestDocumentLocalDeliveryPendingRouteRechecksBeforeRead(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	live := true
	source := localDeliveryBrowserSource(func() bool { return live })
	b := NewController(nil)
	reads := 0
	b.SetDocumentLoader(func(s DocumentSource) bool { return s.LocalDeliveryValid() }, func(context.Context, DocumentSource) DocumentResult {
		reads++
		return DocumentResult{Err: errors.New("must not read")}
	})
	// Simulate the already-authorized WorkspacePanes closure waiting until its
	// next frame. Owner revokes between scheduling and route consumption.
	pending := func() { b.OpenDocumentSource(source) }
	live = false
	pending()
	b.resumeCurrent()
	tab := b.current()
	if tab == nil || tab.source == nil || tab.Error == "" || tab.sourcePending || tab.sourceRead != nil || reads != 0 || tab.Host != nil {
		t.Fatalf("stale pending route tab=%+v reads=%d", tab, reads)
	}
	if !strings.Contains(tab.Error, "permission ended") {
		t.Fatal(tab.Error)
	}
	// Its terminal error is painted in a real document container, not returned
	// as a misleading success or redirected to an external opener.
	tt := ui.NewTester(func(c *ui.Context) { b.View(c, "", false, nil) }, 700, 500)
	found := false
	for _, text := range tt.Texts() {
		if strings.Contains(text, "permission ended") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no visible terminal error: %q", tt.Texts())
	}
}

func TestDocumentLocalDeliveryLoadingAndFinalFailureHaveOwnedTab(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	source := localDeliveryBrowserSource(func() bool { return true })
	b := NewController(nil)
	b.SetDocumentLoader(func(s DocumentSource) bool { return s.LocalDeliveryValid() }, func(context.Context, DocumentSource) DocumentResult {
		return DocumentResult{Err: errors.New("source failed")}
	})
	b.OpenDocumentSource(source)
	tab := b.current()
	if tab == nil || !tab.sourcePending || tab.Error != "" || !strings.Contains(tab.ActionMessage, "Reading local") {
		t.Fatalf("no loading document tab: %+v", tab)
	}
	// Exercise the UI-thread completion path directly; no native guest, product
	// window, real profile or active Host is created for this pure test.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := &documentRead{cancel: cancel}
	tab.sourceRead = request
	tab.sourcePending = false
	completion := 0
	b.finishDocumentRead(tab, request, ctx, source, b.documentLoader, DocumentResult{Err: errors.New("exact local source failed"), Valid: source.LocalDeliveryValid, Complete: func(err error) {
		completion++
		if err == nil {
			t.Error("nil final failure")
		}
	}})
	if len(b.Tabs) != 1 || b.current() != tab || tab.Error != "exact local source failed" || tab.ActionMessage != "" || tab.sourceRead != nil || completion != 1 || tab.Host != nil {
		t.Fatalf("final failure tab=%+v completion=%d", tab, completion)
	}
}

func TestDocumentLocalDeliveryLateResultAndVisibleLeaseRevoked(t *testing.T) {
	live := true
	source := localDeliveryBrowserSource(func() bool { return live })
	b := NewController(nil)
	b.SetDocumentLoader(func(s DocumentSource) bool { return s.LocalDeliveryValid() }, nil)
	b.OpenDocumentSource(source)
	tab := b.current()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := &documentRead{cancel: cancel}
	tab.sourceRead = request
	tab.sourcePending = false
	live = false
	b.finishDocumentRead(tab, request, ctx, source, b.documentLoader, DocumentResult{Valid: source.LocalDeliveryValid})
	if tab.Error == "" || tab.sourcePending || tab.sourceRead != nil || tab.Host != nil {
		t.Fatal("late result resurrected stale source")
	}
	live = true
	b.retryDocument(tab)
	if tab.Error == "" || tab.sourcePending || tab.sourceRead != nil {
		t.Fatal("retry revived revoked token")
	}
}

func TestDocumentLocalDeliveryRestoredDescriptorCannotBorrowOrdinaryLoader(t *testing.T) {
	source := localDeliveryBrowserSource(func() bool { return true })
	b := NewController(nil)
	b.OpenDocumentSource(source)
	raw, err := json.Marshal(b.WorkspaceSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	var saved WorkspaceSnapshot
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	restored := NewController(nil)
	ready, reads := 0, 0
	restored.SetDocumentLoader(func(DocumentSource) bool { ready++; return true }, func(context.Context, DocumentSource) DocumentResult { reads++; return DocumentResult{} })
	restored.RestoreWorkspace(&saved)
	restored.resumeCurrent()
	tab := restored.current()
	if tab == nil || tab.Error == "" || tab.sourcePending || tab.sourceRead != nil || ready != 0 || reads != 0 {
		t.Fatalf("restored local source borrowed ordinary loader: %+v ready=%d reads=%d", tab, ready, reads)
	}
}

func TestDocumentLocalDeliveryRepeatedGestureRebindsOneOwnedTab(t *testing.T) {
	oldLive, newLive := true, true
	old := localDeliveryBrowserSource(func() bool { return oldLive })
	fresh := localDeliveryBrowserSource(func() bool { return newLive })
	b := NewController(nil)
	b.SetDocumentLoader(func(s DocumentSource) bool { return s.LocalDeliveryValid() }, nil)
	b.OpenDocumentSource(old)
	tab := b.current()
	doc, err := pdfpreview.OpenBytes("saved.pdf", []byte("%PDF-1.7\n%%EOF"), old.LocalDeliveryValid)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	tab.Document = doc
	tab.Error = "old viewer error"
	tab.sourcePending = false
	b.OpenDocumentSource(fresh, true)
	if len(b.Tabs) != 1 || b.current() != tab || tab.source.localDelivery != fresh.localDelivery || tab.source.localDelivery == old.localDelivery || tab.Document != nil || tab.HTML != nil || tab.Media != nil || tab.Host != nil || !tab.sourcePending || !tab.floatAfterRead || tab.Error != "" || doc.Valid() {
		t.Fatalf("duplicate gesture kept old owner/bytes or added tab: %+v", tab)
	}
	oldLive = false
	if !tab.source.LocalDeliveryValid() {
		t.Fatal("reused tab still depended on the old permit")
	}
	newLive = false
	b.resumeCurrent()
	if tab.Error == "" || tab.sourcePending {
		t.Fatal("new owner revocation did not deny reused tab")
	}
}

func TestDocumentLocalDeliveryDoesNotReuseRevokedOrChangedDescriptors(t *testing.T) {
	for _, change := range []string{"revoked old", "revoked new", "path", "workspace", "authority", "session", "view", "kind", "address", "mode", "parent", "pane"} {
		t.Run(change, func(t *testing.T) {
			oldLive, newLive := true, true
			old := localDeliveryBrowserSource(func() bool { return oldLive })
			b := NewController(nil)
			b.SetDocumentLoader(func(s DocumentSource) bool { return s.LocalDeliveryValid() }, nil)
			b.OpenDocumentSource(old)
			original := b.current()
			public := old
			public.localDelivery = nil
			switch change {
			case "revoked old":
				oldLive = false
			case "revoked new":
				newLive = false
			case "path":
				public.Path += "-other"
			case "workspace":
				public.Workspace += "-other"
			case "authority":
				public.Authority += "-other"
			case "session":
				public.SessionID += "-other"
			case "view":
				public.ViewSessionID += "-other"
			case "kind":
				public.Kind = DocumentHTML
			case "address":
				public.AddressKind = "other"
			case "mode":
				public.Mode = "child"
			case "parent":
				public.ParentSessionID = "parent"
			case "pane":
				public.NavigationPaneID = "pane"
			}
			fresh := BindLocalDeliveryDocument(public, func() bool { return newLive })
			b.OpenDocumentSource(fresh)
			if len(b.Tabs) != 2 || b.current() == original || original.source.localDelivery != old.localDelivery || b.current().source.localDelivery != fresh.localDelivery {
				t.Fatal("changed/revoked source borrowed original tab")
			}
			if change == "revoked old" {
				if original.source.LocalDeliveryValid() {
					t.Fatal("old permit resurrected")
				}
				oldLive = true
				if original.source.LocalDeliveryValid() {
					t.Fatal("new gesture revived old permit")
				}
			}
			if change == "revoked new" && b.current().Error == "" {
				t.Fatal("revoked incoming source was treated as ready")
			}
		})
	}
}

func TestDocumentLocalDeliveryRepeatedGestureCancelsInflightAndRejectsLateResult(t *testing.T) {
	old := localDeliveryBrowserSource(func() bool { return true })
	fresh := localDeliveryBrowserSource(func() bool { return true })
	b := NewController(nil)
	b.SetDocumentLoader(func(s DocumentSource) bool { return s.LocalDeliveryValid() }, nil)
	b.OpenDocumentSource(old)
	tab := b.current()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := &documentRead{cancel: cancel}
	tab.sourceRead = request
	tab.sourcePending = false
	b.OpenDocumentSource(fresh)
	if ctx.Err() != context.Canceled || tab.sourceRead != nil || !tab.sourcePending || len(b.Tabs) != 1 || tab.source.localDelivery != fresh.localDelivery {
		t.Fatal("repeated gesture did not replace old inflight source")
	}
	late, err := pdfpreview.OpenBytes("saved.pdf", []byte("%PDF-1.7\n%%EOF"), old.LocalDeliveryValid)
	if err != nil {
		t.Fatal(err)
	}
	defer late.Close()
	canceled := 0
	b.finishDocumentRead(tab, request, ctx, old, b.documentLoader, DocumentResult{PDF: late, Valid: old.LocalDeliveryValid, Canceled: func() { canceled++ }})
	if late.Valid() || canceled != 1 || tab.Document != nil || tab.source.localDelivery != fresh.localDelivery || !tab.sourcePending || tab.Error != "" {
		t.Fatal("late old read altered freshly rebound tab")
	}
}

func documentMediaFixture(t *testing.T, valid func() bool) *mediapreview.Document {
	t.Helper()
	data := make([]byte, 42)
	copy(data, "fLaC")
	data[7] = 34
	doc, err := mediapreview.Open("saved.flac", data, valid)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(doc.Close)
	return doc
}

func TestDocumentLocalDeliveryMediaExpiryReturnsToOwnedSource(t *testing.T) {
	for _, mode := range []string{"valid source", "revoked source", "ordinary media"} {
		t.Run(mode, func(t *testing.T) {
			live := true
			source := localDeliveryBrowserSource(func() bool { return live })
			public := source
			public.Kind = DocumentMedia
			public.Path = "/workspace/saved.flac"
			source = BindLocalDeliveryDocument(public, func() bool { return live })
			b := NewController(nil)
			tab := &Tab{ID: 1, source: &source, Draft: source.Path}
			b.Tabs = []*Tab{tab}
			b.Active = 0
			valid := source.LocalDeliveryValid
			if mode == "ordinary media" {
				tab.source = nil
				valid = func() bool { return true }
			}
			doc := documentMediaFixture(t, valid)
			tab.Media = doc
			if mode == "revoked source" {
				live = false
			} else {
				doc.Close()
			}
			b.PruneMediaDocuments()
			switch mode {
			case "valid source":
				if tab.Media != nil || tab.Host != nil || !tab.sourcePending || tab.sourceRead != nil || tab.Error != "" || tab.ActionMessage == "" || !tab.source.LocalDeliveryValid() {
					t.Fatalf("expired source stuck behind dead media controls: %+v", tab)
				}
			case "revoked source":
				if tab.Media != nil || tab.Host != nil || tab.sourcePending || tab.Error == "" || tab.source.LocalDeliveryValid() {
					t.Fatalf("revoked source resurrected after expiry: %+v", tab)
				}
			case "ordinary media":
				if tab.Media != doc || tab.Error == "" || tab.sourcePending {
					t.Fatal("ordinary non-source media behavior changed")
				}
			}
		})
	}
}
