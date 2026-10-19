package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
)

// Temporary files and in-memory Host callbacks only. This exercises the actual
// Files async read, source buffer, observation and serial SaveTextChecked path.
func TestDelimitedDesktopSourceEditSaveRoundTrip(t *testing.T) {
	for _, tc := range []struct{ path, original, external, edited string }{
		{"data.csv", "name,value\r\n\"comma, name\",0012\r\n", "name,value\r\nexternal,0013\r\n", "name,value\r\n\"edited, name\",0014\r\n"},
		{"data.TSV", "name\tvalue\n\"two\nlines\"\t0012\n", "name\tvalue\nexternal\t0013\n", "name\tvalue\n\"edited\nlines\"\t0014\n"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			root := t.TempDir()
			full := filepath.Join(root, tc.path)
			if err := os.WriteFile(full, []byte(tc.original), 0600); err != nil {
				t.Fatal(err)
			}
			var selected, preview, draft, problem, status string
			editing := false
			notified := make(chan struct{}, 8)
			tabs := &Tabs{}
			m := Model{BindingID: "desktop-fixture/session", SessionID: "session", WorkspaceRoot: root,
				Tabs: tabs, SelectedFile: &selected, Preview: &preview, EditDraft: &draft,
				PreviewErr: &problem, EditStatus: &status, Editing: &editing,
				notify: func() { notified <- struct{}{} }}
			tabs.Bind(m.BindingID, root)
			t.Cleanup(tabs.Dispose)
			var hostCalls, writes atomic.Int32
			m.ReadTextPageContext = func(context.Context, string, int64) (workspacefs.TextPage, error) {
				hostCalls.Add(1)
				return workspacefs.TextPage{Text: "WRONG_HOST_SOURCE", EOF: true}, nil
			}
			m.ReadTableContext = func(context.Context, string, workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
				hostCalls.Add(1)
				return nil, errors.New("desktop delimited source reached a table reader")
			}
			m.OnSaveEdit = func(path, text string) error {
				writes.Add(1)
				if path != tc.path {
					return errors.New("save crossed file ownership")
				}
				return workspacefs.WriteFileText(full, text)
			}
			selectFile(m, tc.path)
			awaitFileTextRead(t, m, notified)
			tab := selectedTab(m)
			if preview != tc.original || tab == nil || tab.tablePage != nil || tab.Remote || !previewEditable(m, tc.path) {
				t.Fatalf("CSV/TSV did not retain editable source: %q / %q", preview, problem)
			}
			// The production watcher supplies invalidation only. A refresh must
			// continue the local source even though frame-bound Host readers exist.
			if err := os.WriteFile(full, []byte(tc.external), 0600); err != nil {
				t.Fatal(err)
			}
			observation := resourceReader(m, tab)(context.Background())
			if observation.err != nil || observation.remote || observation.text != tc.external {
				t.Fatalf("observation changed source: %+v", observation)
			}
			applyResourceObservation(m, tab, observation)
			if preview != tc.external || tab.Remote || tab.readBlocked {
				t.Fatal("local refresh blocked desktop saving")
			}
			editing = true
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					sourceTextArea(c, m, m.EditDraft, false, "editor", ui.RGB(0, 0, 0)).AutoFocus()
				})
			}, 500, 260)
			client := tester.NativeTextInputClient()
			if client == nil {
				t.Fatal("existing native source editor did not own input")
			}
			rangeAll := ui.TextInputRange{End: ui.UTF16Len(tc.external)}
			client.ReplaceText(&rangeAll, tc.edited)
			// Explicit Save captures committed native text before another frame.
			if err := requestSaveDraft(m); err != nil {
				t.Fatal(err)
			}
			deadline := time.NewTimer(3 * time.Second)
			defer deadline.Stop()
			for snapshot := tab.save.Snapshot(); snapshot.Running || snapshot.Pending; snapshot = tab.save.Snapshot() {
				select {
				case <-notified:
				case <-deadline.C:
					t.Fatal("fixture save did not settle")
				}
			}
			syncSaveResults(m)
			data, err := os.ReadFile(full)
			if err != nil || string(data) != tc.edited || preview != tc.edited || draft != tc.edited || tab.Dirty() {
				t.Fatalf("save changed source text or lost editor input: disk=%q preview=%q draft=%q err=%v", data, preview, draft, err)
			}
			if writes.Load() != 1 || hostCalls.Load() != 0 {
				t.Fatalf("wrong route: desktop writes=%d Host reads=%d", writes.Load(), hostCalls.Load())
			}
		})
	}
}

func TestBinaryWorkbookRenderedPageCannotSaveOverFile(t *testing.T) {
	for _, path := range []string{"book.xls", "book.XLSX"} {
		t.Run(path, func(t *testing.T) {
			root := t.TempDir()
			full := filepath.Join(root, path)
			original := []byte("BINARY_WORKBOOK_CANARY")
			if err := os.WriteFile(full, original, 0600); err != nil {
				t.Fatal(err)
			}
			tab := &FileTab{Path: path, Preview: "rendered page", Draft: "edited page"}
			tabs := &Tabs{Workspaces: map[string]*WorkspaceTabs{root: {Open: []*FileTab{tab}, Active: path}}}
			t.Cleanup(tabs.Dispose)
			writes := 0
			m := Model{WorkspaceRoot: root, Tabs: tabs, SelectedFile: &path, Preview: &tab.Preview, EditDraft: &tab.Draft,
				OnSaveEdit: func(string, string) error { writes++; return nil }}
			if err := requestSaveDraft(m); err == nil || tab.save != nil || writes != 0 {
				t.Fatal("rendered workbook page entered the text writer")
			}
			data, err := os.ReadFile(full)
			if err != nil || string(data) != string(original) {
				t.Fatal("read-only workbook bytes changed")
			}
		})
	}
}

func TestDelimitedDefaultPreviewDoesNotCreateFallbackSurface(t *testing.T) {
	for _, path := range []string{"data.csv", "data.TSV"} {
		t.Run(path, func(t *testing.T) {
			m := previewViewModel("name,value\nsource,0012\n")
			tab := selectedTab(m)
			*m.SelectedFile, tab.Path = path, path
			tabsFor(m).Active = path
			m.SessionID = "session"
			editing := false
			m.Editing = &editing
			calls := 0
			surface := &workbookSurfaceFixture{fixtureDocumentSurface: fixtureDocumentSurface{recommended: true}, workbook: true}
			m.OpenDocumentViewer = func(opened string) (DocumentSurface, error) {
				calls++
				if opened != path {
					t.Fatalf("viewer crossed tab ownership: %q", opened)
				}
				return surface, nil
			}
			t.Cleanup(m.Tabs.Dispose)
			// Render the complete production preview entry, including its toolbar
			// prepare and the surface-before-grid/source dispatch.
			preview := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					previewPane(c, m, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(180, 180, 180), ui.RGB(255, 255, 255))
				})
			}, 600, 360)
			preview.Frame()
			if calls != 0 || tab.document.enabled || tab.document.surface != nil || tab.document.attempted || !previewEditable(m, path) {
				t.Fatal("default CSV/TSV navigation created a readonly fallback surface")
			}
			// The existing explicit chooser may still open the original table
			// viewer. Returning to Auto must immediately restore native source.
			chooser := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).Children(func() { paintDocumentViewerButton(c, m) })
			}, 600, 100)
			if err := chooser.Click(label("选择预览器", "Choose viewer")); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || !tab.document.enabled || tab.document.choice != 2 {
				t.Fatal("explicit document viewer did not open lazily")
			}
			chooser.Frame()
			if calls != 1 {
				t.Fatal("prepare duplicated the explicitly opened surface")
			}
			if err := chooser.Click(label("自动选择", "Auto viewer")); err != nil {
				t.Fatal(err)
			}
			preview.Frame()
			if tab.document.enabled || surface.visible || calls != 1 || !previewEditable(m, path) {
				t.Fatal("Auto kept the delimited fallback instead of desktop source")
			}
		})
	}
}
