package frame

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/i18n"
)

// A temporary loopback follow stream supplies real, scoped session history.
// The card and its preview authority are projected by the production store.
func responsiveDeliveryStore(t *testing.T) (*session.Store, session.TurnFileScope, session.TurnFileTarget) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/present.host":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"fixture","available":false,"fileManager":null}`))
		case dsh.MuxPath:
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer conn.CloseNow()
			for {
				var frame dsh.ClientFrame
				if wsjson.Read(r.Context(), conn, &frame) != nil {
					return
				}
				if frame.Type == dsh.FrameOpen && frame.Endpoint == "session/follow" {
					value := json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"deliverables/presented","seq":12,"data":{"turn":4,"callId":"delivery","files":[{"path":"report.pdf","description":"Delivered report"}]}}},{"type":"event","event":{"type":"assistant/message","seq":13,"data":{"turn":4,"message":{"id":"done","content":[{"type":"text","text":"Report ready"}]}}}},{"type":"event","event":{"type":"turn/end","seq":14,"data":{"turn":4}}}],"hasMore":false}`)
					if wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: frame.StreamID, Value: value}) != nil {
						return
					}
				}
			}
		default:
			http.NotFound(w, r)
		}
	}))
	store := session.New()
	store.Select("delivery-session")
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	store.Attach(client, nil)
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	store.AttachMux(mux)
	ctx, cancel := context.WithCancel(context.Background())
	store.FollowSelected(ctx)
	t.Cleanup(func() {
		cancel()
		store.Detach()
		_ = mux.Close()
		server.Close()
	})

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		snap := store.Snapshot()
		files := session.TurnDeliveries(snap.Messages)[4].Files
		if snap.HistoryReady && snap.TurnFiles.Scope.Binding != "" && len(files) == 1 {
			target := files[0].Target
			if path, ok := store.TurnFilePreviewPath(snap.TurnFiles.Scope, target); ok && path == "report.pdf" {
				return store, snap.TurnFiles.Scope, target
			}
		}
		select {
		case <-deadline.C:
			t.Fatalf("loopback delivery history did not become actionable: %+v", snap)
		case <-time.After(time.Millisecond):
		}
	}
}

// Tester.Texts includes clipped nodes, so require a nonempty clipped rectangle
// as well as the actual browser error; an off-screen zero-width node is not UI.
func requireResponsiveDeliveryVisible(t *testing.T, tt *ui.Tester, want string) {
	t.Helper()
	for _, text := range tt.Texts() {
		if strings.Contains(text, want) {
			if r, ok := tt.Find(text); ok && r.W > 0 && r.H > 0 {
				return
			}
		}
	}
	t.Fatalf("%q is not visible in the frame; texts=%q", want, tt.Texts())
}

func TestDeliveryResponsiveFramePreview(t *testing.T) {
	previousLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previousLocale)

	for _, size := range []struct{ width, height int }{{600, 400}, {600, 900}, {1200, 900}} {
		width, height := size.width, size.height
		t.Run(fmt.Sprintf("%dx%d_DIP", width, height), func(t *testing.T) {
			store, scope, target := responsiveDeliveryStore(t)
			root := t.TempDir()
			sizes := DefaultLayout()
			sizes.Bottom, sizes.Right = 0, 540
			w := &WorkspacePanes{}
			w.syncRoute("files")
			sizes.Workspace = w
			route, open := "files", false
			controller := browserpreview.NewController(nil)
			defer controller.Close()
			readyChecks, reads, calls := 0, 0, 0
			controller.SetDocumentLoader(func(source browserpreview.DocumentSource) bool {
				readyChecks++
				if !source.LocalDeliveryValid() {
					t.Error("accepted delivery lost its local permit before readiness")
				}
				return false
			}, func(context.Context, browserpreview.DocumentSource) browserpreview.DocumentResult {
				reads++
				t.Error("not-ready document started a read")
				return browserpreview.DocumentResult{}
			})
			create := func() *browserpreview.Controller { return controller }
			preview := func(path string) error {
				calls++
				if path != "report.pdf" {
					return fmt.Errorf("unexpected delivery path %q", path)
				}
				source := browserpreview.BindLocalDeliveryDocument(browserpreview.DocumentSource{
					Authority: "fixture", SessionID: scope.SessionID, ViewSessionID: scope.SessionID,
					AddressKind: "local-delivery", Workspace: root, Path: filepath.Join(root, path), Kind: browserpreview.DocumentPDF,
				}, func() bool {
					current, ok := store.TurnFilePreviewPath(scope, target)
					return ok && current == path
				})
				if !source.LocalDeliveryValid() {
					return errors.New("fixture delivery permit invalid")
				}
				// Match main's accepted document callback: schedule the document,
				// and publish these local route values for the next frame.
				w.OpenDocument(source, create)
				open, route = true, "browser"
				return nil
			}
			tt := ui.NewTester(func(c *ui.Context) {
				// Match main's persistent NavigationShell plus inner frame with
				// ExternalNavigation; both receive the live expanded state.
				NavigationShell(c, Model{Store: store, Sizes: &sizes, RightPaneOpen: open}, func() {
					View(c, Model{Store: store, Sizes: &sizes, ExternalNavigation: true, WorkspaceRoot: root, FileBindingID: "fixture-binding",
						RightPaneOpen: open, RightPaneTab: &route, NewWorkspaceBrowser: create, OnPreviewDeliverable: preview, OnToggleTools: func() { open = !open }})
				})
			}, width, height)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			requireResponsiveDeliveryVisible(t, tt, "Delivered report")
			if calls != 0 || open || w.Fullscreen != "" {
				t.Fatal("rendering the delivery card opened a preview")
			}
			if err := tt.Click("Preview report.pdf"); err != nil {
				t.Fatalf("click delivery card: %v; texts=%q", err, tt.Texts())
			}
			// Exercise the real next-frame owner/route projection twice.
			for frame := 0; frame < 2; frame++ {
				tt.Frame()
				if calls != 1 || !open || route != "browser" || w.pending != nil {
					t.Fatalf("frame %d lost preview route: calls=%d open=%v route=%q pending=%v", frame, calls, open, route, w.pending != nil)
				}
				active := w.Root.find(w.Active)
				if active == nil || active.Kind != "browser" || active.browser != controller {
					t.Fatalf("frame %d lost the addressed browser occurrence: active=%+v", frame, active)
				}
				if w.Fullscreen != "" {
					t.Fatalf("frame %d presentation changed persisted fullscreen=%q", frame, w.Fullscreen)
				}
				if width == 600 {
					header, ok := tt.Find("Workspace pane header")
					if !ok || header.X != 0 || header.W != float32(width) {
						t.Fatalf("auto preview did not cover viewport: %+v", header)
					}
				}
				requireResponsiveDeliveryVisible(t, tt, "Local delivery preview permission ended")
			}
			if readyChecks == 0 || reads != 0 || len(controller.Tabs) != 1 || controller.Tabs[0].Host != nil || controller.Tabs[0].Error == "" {
				t.Fatalf("not-ready document did not terminate without a WebView: ready=%d reads=%d tabs=%+v", readyChecks, reads, controller.Tabs)
			}
			if sizes.Right != 540 || sizes.Sidebar != SidebarDefault {
				t.Fatalf("preview changed saved layout preferences: right=%v sidebar=%v", sizes.Right, sizes.Sidebar)
			}
			if width == 600 {
				// A resize releases only derived fullscreen; the same document is
				// visible in its dock without changing saved mode or tab identity.
				tt.SetSize(1200, height)
				requireResponsiveDeliveryVisible(t, tt, "Local delivery preview permission ended")
				header, ok := tt.Find("Workspace pane header")
				if !ok || header.W >= 1200 || w.Fullscreen != "" {
					t.Fatalf("resize retained auto fullscreen: %+v mode=%q", header, w.Fullscreen)
				}
				tt.SetSize(600, height)
				requireResponsiveDeliveryVisible(t, tt, "Local delivery preview permission ended")
				if err := tt.Click("Exit fullscreen"); err != nil {
					t.Fatal(err)
				}
				if open || w.Fullscreen != "" || tt.HasText("Local delivery preview permission ended") || len(controller.Tabs) != 1 {
					t.Fatal("auto exit did not collapse and retain document")
				}
				open = true
				tt.Frame()
				requireResponsiveDeliveryVisible(t, tt, "Local delivery preview permission ended")
				if w.Fullscreen != "" || len(controller.Tabs) != 1 {
					t.Fatal("reopening changed mode/document identity")
				}
			}
		})
	}
}

func TestDeliveryResponsiveFramePreviewErrorPreservesRoute(t *testing.T) {
	previousLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previousLocale)

	for _, width := range []int{600, 1200} {
		for _, initial := range []struct {
			name       string
			open       bool
			fullscreen bool
		}{
			{name: "closed"},
			{name: "closed_with_retained_fullscreen", fullscreen: true},
			{name: "open", open: true},
		} {
			// An already-open phone rightbar covers the conversation, so its
			// delivery card cannot be clicked until the user collapses it.
			if width < int(workspaceAutoFullscreenWidth) && initial.open {
				continue
			}
			t.Run(fmt.Sprintf("%d_DIP/%s", width, initial.name), func(t *testing.T) {
				store, scope, target := responsiveDeliveryStore(t)
				root := t.TempDir()
				sizes := DefaultLayout()
				sizes.Bottom, sizes.Right = 0, 540
				w := &WorkspacePanes{}
				w.syncRoute("diff")
				sizes.Workspace = w
				if initial.fullscreen {
					w.Fullscreen = w.Active
				}
				originalActive, originalFullscreen := w.Active, w.Fullscreen
				route, open, calls := "diff", initial.open, 0
				controller := browserpreview.NewController(nil)
				defer controller.Close()
				create := func() *browserpreview.Controller { return controller }
				preview := func(path string) error {
					calls++
					source := browserpreview.BindLocalDeliveryDocument(browserpreview.DocumentSource{
						SessionID: scope.SessionID, AddressKind: "local-delivery", Workspace: root,
						Path: filepath.Join(root, path), Kind: browserpreview.DocumentPDF,
					}, func() bool {
						current, ok := store.TurnFilePreviewPath(scope, target)
						return ok && current == path
					})
					// Even a callback that dirties its local route and queues an
					// open before reporting failure must be rolled back.
					w.OpenDocument(source, create)
					route = "browser"
					return errors.New("Delivery preview rejected")
				}
				tt := ui.NewTester(func(c *ui.Context) {
					View(c, Model{Store: store, Sizes: &sizes, WorkspaceRoot: root, FileBindingID: "fixture-binding",
						RightPaneOpen: open, RightPaneTab: &route, NewWorkspaceBrowser: create, OnPreviewDeliverable: preview})
				}, width, 900)
				tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
				if err := tt.Click("Preview report.pdf"); err != nil {
					t.Fatalf("click delivery card: %v; texts=%q", err, tt.Texts())
				}
				for frame := 0; frame < 2; frame++ {
					tt.Frame()
					if calls != 1 || open != initial.open || route != "diff" || w.route != "diff" || w.Active != originalActive || w.Fullscreen != originalFullscreen || w.pending != nil || len(controller.Tabs) != 0 {
						t.Fatalf("frame %d failure changed route/fullscreen: calls=%d open=%v route=%q workspaceRoute=%q active=%q fullscreen=%q pending=%v tabs=%d", frame, calls, open, route, w.route, w.Active, w.Fullscreen, w.pending != nil, len(controller.Tabs))
					}
					requireResponsiveDeliveryVisible(t, tt, "Could not preview file; try again")
				}
			})
		}
	}
}
