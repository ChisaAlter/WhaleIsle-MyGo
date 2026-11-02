package frame

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceDeliveryPreviewProjectionUsesFilesOccurrenceAndPreservesRoutes(t *testing.T) {
	for _, start := range []string{"diff", "browser"} {
		for _, mode := range []string{"text", "document", "dirty same", "dirty other", "failure"} {
			t.Run(start+"/"+mode, func(t *testing.T) {
				oldLocale := i18n.Locale()
				i18n.SetLocale(i18n.LocaleEN)
				defer i18n.SetLocale(oldLocale)
				root := t.TempDir()
				path := filepath.Join(root, "saved.txt")
				if mode == "document" {
					path = filepath.Join(root, "saved.pdf")
				}
				if err := os.WriteFile(path, []byte("saved source"), 0600); err != nil {
					t.Fatal(err)
				}
				w := &WorkspacePanes{}
				w.syncRoute("files")
				filePane := w.Root
				startPane := w.selectKind(filePane, start)
				if mode == "dirty same" || mode == "dirty other" {
					draftPath := filepath.Base(path)
					if mode == "dirty other" {
						draftPath = "different.txt"
					}
					filePane.Files = &WorkspaceFileState{Selected: draftPath, Preview: "saved", Draft: "unsaved", Editing: true, Tabs: files.Tabs{Workspaces: map[string]*files.WorkspaceTabs{root: {Active: draftPath, Open: []*files.FileTab{{Path: draftPath, Preview: "saved", Draft: "unsaved", Editing: true}}}}}}
				}
				w.BeginContentFrame("binding", root)
				var adapter WorkspaceFileState
				adapter.Tabs.Preferences = &files.PreviewPreferences{}
				sizes := DefaultLayout()
				sizes.Workspace = w
				route := start
				store := session.New()
				store.Select("session")
				m := Model{Store: store, Sizes: &sizes, RightPaneTab: &route, WorkspaceRoot: root, FileBindingID: "binding"}
				m = adapter.model(m)
				calls, documentCalls := 0, 0
				var gotErr error
				m.OnPreviewDeliverable = func(path string) error {
					calls++
					if mode == "failure" {
						route = "browser"
						return errors.New("preview denied")
					}
					fm := files.Model{Tabs: m.FileTabs, WorkspaceRoot: root, BindingID: "binding", SessionID: "session", SelectedFile: m.SelectedFile, Preview: m.FilePreview, PreviewErr: m.FilePreviewErr, Editing: m.FileEditing, EditDraft: m.FileEditDraft, EditStatus: m.FileEditStatus, StatNote: m.FileStatNote}
					fm.OpenDeliveryDocument = func(abs string, kind files.DeliveryPreviewKind) error {
						documentCalls++
						if kind != files.DeliveryPreviewPDF || abs != path {
							t.Error("incorrect document")
						}
						source := browserpreview.BindLocalDeliveryDocument(browserpreview.DocumentSource{LocalDelivery: true, AddressKind: "local-delivery", Path: abs, Kind: browserpreview.DocumentPDF}, func() bool { return false })
						w.OpenDocument(source, func() *browserpreview.Controller { return browserpreview.NewController(nil) })
						route = "browser"
						return nil
					}
					err := files.PreviewDeliveryPath(fm, path)
					if err == nil && documentCalls == 0 {
						route = "files"
					}
					return err
				}
				tt := ui.NewTester(func(c *ui.Context) {
					local := w.beginFileProjection(m)
					defer w.finishFileProjection(local)
					if ui.Button(c, "Preview delivery").Clicked() {
						gotErr = local.OnPreviewDeliverable(path)
					}
					if w.pending != nil {
						action := w.pending
						w.pending = nil
						action()
					}
				}, 600, 300)
				if err := tt.Click("Preview delivery"); err != nil {
					t.Fatal(err)
				}
				if filePane.Files == nil {
					t.Fatal("Files state not selected")
				}
				defer filePane.Files.Tabs.Dispose()
				if mode == "dirty same" || mode == "failure" {
					if gotErr == nil || w.Active != startPane.ID || route != start || w.route != start {
						t.Fatalf("failure changed route err=%v active=%s route=%s", gotErr, w.Active, route)
					}
				} else if mode == "document" {
					if gotErr != nil || documentCalls != 1 || w.Root.find(w.Active).Kind != "browser" || route != "browser" || w.pending != nil {
						t.Fatalf("document route lost err=%v calls=%d route=%s", gotErr, documentCalls, route)
					}
				} else {
					if gotErr != nil || w.Active != filePane.ID || route != "files" || filePane.Files.Selected != filepath.Base(path) {
						t.Fatalf("first-frame Files preview rejected/wrong owner err=%v active=%s route=%s selected=%s", gotErr, w.Active, route, filePane.Files.Selected)
					}
				}
				if mode == "dirty same" {
					if calls != 0 || !strings.Contains(gotErr.Error(), "unsaved") {
						t.Fatal("same-file dirty gate bypassed")
					}
				}
				if mode == "dirty other" {
					tabs := filePane.Files.Tabs.Workspaces[root]
					if tabs == nil || len(tabs.Open) != 2 || tabs.Open[0].Draft != "unsaved" {
						t.Fatal("other file's draft changed")
					}
				}
			})
		}
	}
}

func TestWorkspaceDeliveryPreviewProjectionRejectsOnlySameFileAcrossSplit(t *testing.T) {
	for _, otherPath := range []string{"same.txt", "other.txt"} {
		t.Run(otherPath, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "same.txt")
			if err := os.WriteFile(path, []byte("saved"), 0600); err != nil {
				t.Fatal(err)
			}
			w := &WorkspacePanes{}
			left := w.leaf("files")
			right := w.leaf("diff")
			w.Root = &WorkspacePane{ID: "split", First: left, Second: right, Axis: "horizontal", Ratio: .5}
			w.Active, w.route = right.ID, "diff"
			left.Files = &WorkspaceFileState{Selected: otherPath, Preview: "saved", Draft: "unsaved", Editing: true, Tabs: files.Tabs{Workspaces: map[string]*files.WorkspaceTabs{root: {Active: otherPath, Open: []*files.FileTab{{Path: otherPath, Preview: "saved", Draft: "unsaved", Editing: true}}}}}}
			w.BeginContentFrame("binding", root)
			var adapter WorkspaceFileState
			sizes := DefaultLayout()
			sizes.Workspace = w
			route := "diff"
			m := adapter.model(Model{Sizes: &sizes, RightPaneTab: &route, WorkspaceRoot: root, FileBindingID: "binding"})
			calls := 0
			var result error
			m.OnPreviewDeliverable = func(path string) error {
				calls++
				fm := files.Model{Tabs: m.FileTabs, WorkspaceRoot: root, BindingID: "binding", SelectedFile: m.SelectedFile, Preview: m.FilePreview, PreviewErr: m.FilePreviewErr, Editing: m.FileEditing, EditDraft: m.FileEditDraft}
				err := files.PreviewDeliveryPath(fm, path)
				if err == nil {
					route = "files"
				}
				return err
			}
			tt := ui.NewTester(func(c *ui.Context) {
				local := w.beginFileProjection(m)
				defer w.finishFileProjection(local)
				if ui.Button(c, "Preview").Clicked() {
					result = local.OnPreviewDeliverable(path)
				}
			}, 600, 300)
			if err := tt.Click("Preview"); err != nil {
				t.Fatal(err)
			}
			w.Root.eachResource(func(n *WorkspacePane) {
				if n.Files != nil {
					defer n.Files.Tabs.Dispose()
				}
			})
			if otherPath == "same.txt" {
				if result == nil || calls != 0 || w.Active != right.ID || route != "diff" {
					t.Fatalf("cross-split same draft bypassed err=%v calls=%d active=%s route=%s", result, calls, w.Active, route)
				}
			} else {
				if result != nil || calls != 1 || w.Root.find(w.Active).Kind != "files" || route != "files" {
					t.Fatalf("unrelated draft blocked preview err=%v calls=%d route=%s", result, calls, route)
				}
			}
			if left.Files.Draft != "unsaved" || left.Files.Tabs.Workspaces[root].Open[0].Draft != "unsaved" {
				t.Fatal("cross-split draft changed")
			}
		})
	}
}

func TestWorkspaceDeliveryPreviewRestoresFilesFloatOnlyAfterSuccess(t *testing.T) {
	for _, start := range []string{"diff", "browser"} {
		for _, mode := range []string{"success", "failure", "dirty same"} {
			t.Run(start+"/"+mode, func(t *testing.T) {
				root := t.TempDir()
				path := filepath.Join(root, "saved.txt")
				if err := os.WriteFile(path, []byte("saved source"), 0600); err != nil {
					t.Fatal(err)
				}
				w := &WorkspacePanes{}
				w.syncRoute("files")
				filePane := w.Root
				startPane := w.selectKind(filePane, start)
				draftPath := "other.txt"
				if mode == "dirty same" {
					draftPath = "saved.txt"
				}
				filePane.Files = &WorkspaceFileState{Selected: draftPath, Preview: "saved", Draft: "unsaved", Editing: true, Tabs: files.Tabs{Workspaces: map[string]*files.WorkspaceTabs{root: {Active: draftPath, Open: []*files.FileTab{{Path: draftPath, Preview: "saved", Draft: "unsaved", Editing: true}}}}}}
				if !w.FloatWorkspaceTab(filePane.ID) {
					t.Fatal("float failed")
				}
				w.MinimizeWorkspaceFloat(filePane.ID, true)
				w.activateDockTab(startPane.ID)
				before := *filePane.Float
				w.BeginContentFrame("binding", root)
				var adapter WorkspaceFileState
				sizes := DefaultLayout()
				sizes.Workspace = w
				route := start
				m := adapter.model(Model{Sizes: &sizes, RightPaneTab: &route, WorkspaceRoot: root, FileBindingID: "binding"})
				calls := 0
				var result error
				m.OnPreviewDeliverable = func(path string) error {
					calls++
					if mode == "failure" {
						route = "files"
						return errors.New("preview unavailable")
					}
					err := files.PreviewDeliveryPath(files.Model{Tabs: m.FileTabs, WorkspaceRoot: root, BindingID: "binding", SelectedFile: m.SelectedFile, Preview: m.FilePreview, PreviewErr: m.FilePreviewErr, Editing: m.FileEditing, EditDraft: m.FileEditDraft}, path)
					if err == nil {
						route = "files"
					}
					return err
				}
				tt := ui.NewTester(func(c *ui.Context) {
					local := w.beginFileProjection(m)
					defer w.finishFileProjection(local)
					ui.Column(c).Fill().Children(func() {
						if ui.Button(c, "Preview").Clicked() {
							result = local.OnPreviewDeliverable(path)
						}
						if !filePane.Float.Minimized {
							ui.Text(c, "Visible Files preview")
						}
					})
				}, 700, 400)
				if tt.HasText("Visible Files preview") {
					t.Fatal("minimized Files body rendered before open")
				}
				if err := tt.Click("Preview"); err != nil {
					t.Fatal(err)
				}
				defer filePane.Files.Tabs.Dispose()
				if mode == "success" {
					if result != nil || calls != 1 || filePane.Float.Minimized || !w.visible("files") || w.Active != filePane.ID || route != "files" || !tt.HasText("Visible Files preview") || filePane.Files.Selected != "saved.txt" {
						t.Fatalf("successful preview stayed invisible result=%v active=%s route=%s float=%+v", result, w.Active, route, filePane.Float)
					}
				} else {
					if result == nil || !filePane.Float.Minimized || w.visible("files") || w.Active != startPane.ID || route != start || tt.HasText("Visible Files preview") {
						t.Fatalf("failure changed Files presentation result=%v active=%s route=%s float=%+v", result, w.Active, route, filePane.Float)
					}
				}
				if filePane.Float.X != before.X || filePane.Float.Y != before.Y || filePane.Float.Width != before.Width || filePane.Float.Height != before.Height {
					t.Fatal("preview changed saved float geometry")
				}
				if filePane.Files.Tabs.Workspaces[root].Open[0].Draft != "unsaved" {
					t.Fatal("unrelated/same-file draft was overwritten")
				}
			})
		}
	}
}

func TestWorkspaceDeliveryPreviewFirstFrameRecoveredDraftIsBoundBeforeGate(t *testing.T) {
	for _, previewPath := range []string{"restored.txt", "other.txt"} {
		t.Run(previewPath, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, previewPath)
			if err := os.WriteFile(path, []byte("saved source"), 0600); err != nil {
				t.Fatal(err)
			}
			w := &WorkspacePanes{}
			w.syncRoute("diff")
			diff := w.Root
			// Recovery has only the legacy main adapter. BeginContentFrame runs
			// before the first projection creates any pane's Files state.
			adapter := WorkspaceFileState{Selected: "restored.txt", Preview: "saved", Draft: "unsaved", Editing: true, Tabs: files.Tabs{Workspaces: map[string]*files.WorkspaceTabs{root: {Active: "restored.txt", Open: []*files.FileTab{{Path: "restored.txt", Preview: "saved", Draft: "unsaved", Editing: true}}}}}}
			w.BeginContentFrame("binding", root)
			if diff.Files != nil {
				t.Fatal("fixture prematurely made Files state")
			}
			sizes := DefaultLayout()
			sizes.Workspace = w
			route := "diff"
			m := adapter.model(Model{Sizes: &sizes, RightPaneTab: &route, WorkspaceRoot: root, FileBindingID: "binding"})
			calls := 0
			var result error
			m.OnPreviewDeliverable = func(path string) error {
				calls++
				err := files.PreviewDeliveryPath(files.Model{Tabs: m.FileTabs, WorkspaceRoot: root, BindingID: "binding", SelectedFile: m.SelectedFile, Preview: m.FilePreview, PreviewErr: m.FilePreviewErr, Editing: m.FileEditing, EditDraft: m.FileEditDraft}, path)
				if err == nil {
					route = "files"
				}
				return err
			}
			tt := ui.NewTester(func(c *ui.Context) {
				local := w.beginFileProjection(m)
				defer w.finishFileProjection(local)
				if ui.Button(c, "Preview recovered").Clicked() {
					result = local.OnPreviewDeliverable(path)
				}
			}, 700, 400)
			if err := tt.Click("Preview recovered"); err != nil {
				t.Fatal(err)
			}
			w.Root.eachResource(func(n *WorkspacePane) {
				if n.Files != nil {
					t.Cleanup(n.Files.Tabs.Dispose)
				}
			})
			if previewPath == "restored.txt" {
				if result == nil || calls != 0 || w.Active != diff.ID || route != "diff" {
					t.Fatalf("first-frame same-file recovery gate bypassed err=%v calls=%d route=%s", result, calls, route)
				}
			} else {
				if result != nil || calls != 1 || w.Root.find(w.Active).Kind != "files" || route != "files" {
					t.Fatalf("unrelated recovered draft blocked preview err=%v calls=%d route=%s", result, calls, route)
				}
			}
			if diff.Files.previewBinding != "binding" || diff.Files.previewRoot != root || diff.Files.Draft != "unsaved" || diff.Files.Tabs.Workspaces[root].Open[0].Draft != "unsaved" {
				t.Fatal("binding discarded or ignored recovered draft")
			}
		})
	}
}
