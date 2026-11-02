package frame

import (
	"context"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceDocumentOpenRestoresMinimizedBrowserAndConsumesPending(t *testing.T) {
	for _, minimized := range []bool{true, false} {
		t.Run(map[bool]string{true: "minimized", false: "already visible"}[minimized], func(t *testing.T) {
			oldLocale := i18n.Locale()
			i18n.SetLocale(i18n.LocaleEN)
			defer i18n.SetLocale(oldLocale)
			w := &WorkspacePanes{}
			w.syncRoute("files")
			files := w.Root
			browser := w.selectKind(files, "browser")
			controller := browserpreview.NewController(nil)
			browser.browser = controller
			readyChecks := 0
			controller.SetDocumentLoader(func(browserpreview.DocumentSource) bool { readyChecks++; return false }, func(context.Context, browserpreview.DocumentSource) browserpreview.DocumentResult {
				t.Error("not-ready source read")
				return browserpreview.DocumentResult{}
			})
			if !w.FloatWorkspaceTab(browser.ID) {
				t.Fatal("float failed")
			}
			w.MinimizeWorkspaceFloat(browser.ID, minimized)
			w.activateDockTab(files.ID)
			before := *browser.Float
			source := browserpreview.BindLocalDeliveryDocument(browserpreview.DocumentSource{Authority: "owner", AddressKind: "local-delivery", SessionID: "s", Workspace: "root", Path: "saved.pdf", Kind: browserpreview.DocumentPDF}, func() bool { return true })
			w.OpenDocument(source, func() *browserpreview.Controller { t.Fatal("replaced retained controller"); return nil })
			if w.pending == nil || w.Active != files.ID || browser.Float.Minimized != minimized || len(controller.Tabs) != 0 {
				t.Fatal("open mutated painted layout before pending")
			}
			// Run the real deferred route through ui.Tester's frame rather than
			// calling a copied implementation of the layout transition.
			tt := ui.NewTester(func(c *ui.Context) {
				if w.pending != nil {
					action := w.pending
					w.pending = nil
					action()
				}
				if browser.Float != nil && !browser.Float.Minimized {
					controller.View(c, "", false, nil)
				}
			}, 900, 600)
			if browser.Float.Minimized || !w.visible("browser") || w.Active != browser.ID || w.route != "browser" || w.focusPending != browser.ID || w.pending != nil || w.Root.find(files.ID) != files || browser.browser != controller || readyChecks == 0 || len(controller.Tabs) != 1 {
				t.Fatalf("pending route not visible/selected: float=%+v active=%s ready=%d tabs=%d", browser.Float, w.Active, readyChecks, len(controller.Tabs))
			}
			if browser.Float.X != before.X || browser.Float.Y != before.Y || browser.Float.Width != before.Width || browser.Float.Height != before.Height {
				t.Fatal("explicit open changed preferred float geometry")
			}
			found := false
			for _, text := range tt.Texts() {
				if strings.Contains(text, "permission ended") {
					found = true
				}
			}
			if !found {
				t.Fatalf("restored container did not paint final error: %q", tt.Texts())
			}
			route := "files"
			prepareWorkspace(Model{Sizes: &LayoutSizes{Workspace: w}, RightPaneTab: &route})
			if browser.Float.Minimized || w.Active != browser.ID {
				t.Fatal("next frame focus lost restored document")
			}
		})
	}
}

func TestWorkspaceDocumentFloatRestoreDoesNotGrantRevokedSource(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	files := w.Root
	browser := w.selectKind(files, "browser")
	controller := browserpreview.NewController(nil)
	browser.browser = controller
	ready, reads := 0, 0
	controller.SetDocumentLoader(func(browserpreview.DocumentSource) bool { ready++; return true }, func(context.Context, browserpreview.DocumentSource) browserpreview.DocumentResult {
		reads++
		return browserpreview.DocumentResult{}
	})
	w.FloatWorkspaceTab(browser.ID)
	w.MinimizeWorkspaceFloat(browser.ID, true)
	w.activateDockTab(files.ID)
	// General selection does not restore a minimized browser. Only the explicit
	// document path changes this presentation state.
	w.activateDockTab(browser.ID)
	if !browser.Float.Minimized {
		t.Fatal("ordinary activation unexpectedly restored float")
	}
	w.activateDockTab(files.ID)
	live := true
	source := browserpreview.BindLocalDeliveryDocument(browserpreview.DocumentSource{Authority: "owner", SessionID: "s", AddressKind: "local-delivery", Path: "old.pdf", Kind: browserpreview.DocumentPDF}, func() bool { return live })
	w.OpenDocument(source, func() *browserpreview.Controller { t.Fatal("replaced controller"); return nil })
	live = false
	tt := ui.NewTester(func(c *ui.Context) {
		if w.pending != nil {
			action := w.pending
			w.pending = nil
			action()
		}
		if !browser.Float.Minimized {
			controller.View(c, "", false, nil)
		}
	}, 900, 600)
	if browser.Float.Minimized || w.Active != browser.ID || len(controller.Tabs) != 1 || controller.Tabs[0].Error == "" || ready != 0 || reads != 0 || source.LocalDeliveryValid() {
		t.Fatalf("revoked route gained access: float=%+v ready=%d reads=%d", browser.Float, ready, reads)
	}
	live = true
	tt.Frame()
	if source.LocalDeliveryValid() || ready != 0 || reads != 0 || controller.Tabs[0].Error == "" {
		t.Fatal("restoring container revived source permission")
	}
}
