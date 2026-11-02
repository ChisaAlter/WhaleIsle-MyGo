package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/i18n"
)

func deliveryAuditModel(t *testing.T, files map[string][]byte) Model {
	t.Helper()
	root := t.TempDir()
	for path, contents := range files {
		if err := os.WriteFile(filepath.Join(root, path), contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	selected, preview, draft, problem, status, editing := "", "", "", "", "", false
	tabs := &Tabs{}
	tabs.Bind("fixture-host/session", root)
	t.Cleanup(tabs.Dispose)
	return Model{Tabs: tabs, BindingID: "fixture-host/session", SessionID: "session", WorkspaceRoot: root,
		SelectedFile: &selected, Preview: &preview, EditDraft: &draft, PreviewErr: &problem, EditStatus: &status, Editing: &editing}
}

func TestDeliveryPreviewPreservesCommittedNativeDraftBeforePaint(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"note.txt": []byte("saved")})
	*m.SelectedFile, *m.Preview, *m.EditDraft, *m.Editing = "note.txt", "saved", "saved", true
	m.Tabs.Root = filepath.Clean(m.WorkspaceRoot)
	rememberTab(m, "note.txt")
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { sourceTextArea(c, m, m.EditDraft, false, "editor", ui.RGB(0, 0, 0)).AutoFocus() })
	}, 320, 180)
	client := tester.NativeTextInputClient()
	if client == nil {
		t.Fatal("native text fixture did not focus")
	}
	client.ReplaceText(&ui.TextInputRange{End: ui.UTF16Len("saved")}, "unpainted draft")
	if *m.EditDraft != "saved" {
		t.Fatal("fixture already consumed a new paint")
	}
	if err := PreviewDeliveryPath(m, "note.txt"); err == nil {
		t.Fatal("preview accepted an unpainted committed draft")
	}
	if selectedTab(m).Draft != "unpainted draft" || !*m.Editing {
		t.Fatal("preview changed or discarded the committed draft")
	}
	if data, err := os.ReadFile(filepath.Join(m.WorkspaceRoot, "note.txt")); err != nil || string(data) != "saved" {
		t.Fatalf("disk changed: %q %v", data, err)
	}
}

func TestDeliveryPreviewLeavesDifferentDirtyTabIntact(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"dirty.txt": []byte("saved"), "target.txt": []byte("target")})
	*m.SelectedFile, *m.Preview, *m.EditDraft, *m.Editing = "dirty.txt", "saved", "unsaved", true
	m.Tabs.Root = filepath.Clean(m.WorkspaceRoot)
	rememberTab(m, "dirty.txt")
	if err := PreviewDeliveryPath(m, "target.txt"); err != nil {
		t.Fatal(err)
	}
	dirty := findTab(tabsFor(m), "dirty.txt")
	if dirty == nil || dirty.Draft != "unsaved" || dirty.Preview != "saved" || !dirty.Editing {
		t.Fatalf("other tab draft changed: %+v", dirty)
	}
}

// Successful admission currently means only that a tab was selected. These
// assertions require an actual usable preview and deliberately catch a silent
// PDF/HTML/media fallback with no external viewer installed.
func TestDeliveryPreviewShowsRequestedDocumentWithoutOptionalViewer(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	for _, tc := range []struct{ name, body string }{
		{"page.pdf", "%PDF-1.4\nfixture"},
		{"page.html", "<!doctype html><title>Rendered fixture</title><p>Visible fixture</p>"},
		{"clip.mp4", "synthetic media bytes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := deliveryAuditModel(t, map[string][]byte{tc.name: []byte(tc.body)})
			opened := 0
			m.OnOpenDoc = func(string, workspacefs.DocKind, bool) { opened++ }
			m.OnOpenMedia = func(string, bool) { opened++ }
			// Optional embedded Host viewer unavailable: do not create a WebView.
			m.OpenDocumentViewer = func(string) (DocumentSurface, error) { return nil, errors.New("fixture viewer unavailable") }
			err := PreviewDeliveryPath(m, tc.name)
			if err != nil {
				return
			} // A visible explicit failure is acceptable.
			ui.NewTester(func(c *ui.Context) {
				previewPane(c, m, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(0, 0, 0), ui.RGB(255, 255, 255))
			}, 480, 400)
			tab := selectedTab(m)
			if opened == 0 && (tab == nil || !tab.document.enabled) {
				t.Errorf("returned success without starting a %s viewer; selected=%q previewErr=%q", tc.name, *m.SelectedFile, *m.PreviewErr)
			}
		})
	}
}

func TestDeliveryPreviewAsyncFailureIsNotSuccess(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"broken.xlsx": []byte("invalid workbook")})
	notified := make(chan struct{}, 2)
	m.notify = func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	}
	called := make(chan struct{}, 1)
	m.ReadTableContext = func(context.Context, string, workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
		called <- struct{}{}
		return nil, errors.New("Host reader must not be used")
	}
	err := PreviewDeliveryPath(m, "broken.xlsx")
	if err != nil {
		return
	}
	select {
	case <-notified:
	case <-time.After(3 * time.Second):
		t.Fatal("asynchronous workbook read did not complete")
	}
	syncTextRead(m)
	select {
	case <-called:
		t.Fatal("local workbook preview invoked the Host reader")
	default:
	}
	if !strings.Contains(*m.PreviewErr, "zip: not a valid zip file") || !selectedTab(m).readBlocked || selectedTab(m).readLoading {
		t.Fatalf("accepted local read did not publish terminal parse failure: %q", *m.PreviewErr)
	}
	tester := ui.NewTester(func(c *ui.Context) {
		previewPane(c, m, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(0, 0, 0), ui.RGB(255, 255, 255))
	}, 480, 400)
	if !tester.HasText(*m.PreviewErr) {
		t.Fatal("local parse error was not visible in Files")
	}
}

// Invalid text must settle into a visible read error, not a blank success. The
// immediate nil return is only acceptance; this exercises its eventual outcome.
func TestDeliveryPreviewBinaryTextReportsReadFailure(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"bad.txt": {'x', 0, 'y'}})
	notified := make(chan struct{}, 2)
	m.notify = func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	}
	if err := PreviewDeliveryPath(m, "bad.txt"); err != nil {
		return
	}
	select {
	case <-notified:
	case <-time.After(3 * time.Second):
		t.Fatal("text read did not finish")
	}
	syncTextRead(m)
	if *m.PreviewErr == "" || !selectedTab(m).readBlocked {
		t.Fatalf("binary text was not shown as a read failure: error=%q", *m.PreviewErr)
	}
}

func TestDeliveryPreviewInvalidImageReportsDecodeFailure(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"bad.png": []byte("not an image")})
	notified := make(chan struct{}, 2)
	m.notify = func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	}
	if err := PreviewDeliveryPath(m, "bad.png"); err != nil {
		return
	}
	tab := selectedTab(m)
	if tab == nil {
		t.Fatal("preview selected no tab")
	}
	image := readImagePreview(m, tab)
	if image.loading {
		select {
		case <-notified:
		case <-time.After(3 * time.Second):
			t.Fatal("image decode did not finish")
		}
		image = readImagePreview(m, tab)
	}
	if image.err == nil || image.bitmap != nil {
		t.Fatal("invalid image was not reported as a decode failure")
	}
}

// Block the single Files read worker until after a validated local file is
// removed. The captured Host reader returns distinctly different bytes. A local
// delivery preview must keep its local source policy when it eventually reads.
func TestDeliveryPreviewDoesNotFallbackToHostAfterLocalReadFailure(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"note.txt": []byte("local data")})
	queue := &textReadQueue{}
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{}, 2)
	var once sync.Once
	defer once.Do(func() { close(release) })
	queue.request(textReadOwner{path: "blocked"}, func(context.Context) textReadResult {
		close(started)
		<-release
		return textReadResult{}
	})
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("blocking fixture did not start")
	}
	m.Tabs.textReader = queue
	m.notify = func() {
		select {
		case done <- struct{}{}:
		default:
		}
	}
	calls := make(chan struct{}, 1)
	m.ReadTextPageContext = func(context.Context, string, int64) (workspacefs.TextPage, error) {
		calls <- struct{}{}
		return workspacefs.TextPage{Text: "different Host data", EOF: true}, nil
	}
	err := PreviewDeliveryPath(m, "note.txt")
	if err != nil {
		return
	}
	if err := os.Remove(filepath.Join(m.WorkspaceRoot, "note.txt")); err != nil {
		t.Fatal(err)
	}
	once.Do(func() { close(release) })
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("delivery text read did not finish")
	}
	syncTextRead(m)
	select {
	case <-calls:
		t.Fatalf("local delivery read silently used Host data: preview=%q error=%q", *m.Preview, *m.PreviewErr)
	default:
	}
	if *m.PreviewErr == "" {
		t.Fatal("local read failure was not shown")
	}
}

func TestDeliveryPreviewUsesLocalWorkbookForLocalScope(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"book.xlsx": []byte("local file")})
	notified := make(chan struct{}, 2)
	m.notify = func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	}
	calls := make(chan struct{}, 1)
	m.ReadTableContext = func(context.Context, string, workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
		calls <- struct{}{}
		return nil, errors.New("remote test reader called")
	}
	err := PreviewDeliveryPath(m, "book.xlsx")
	if err == nil {
		select {
		case <-notified:
		case <-time.After(3 * time.Second):
			t.Fatal("workbook read did not finish")
		}
	}
	select {
	case <-calls:
		t.Fatal("local delivery preview selected the Host workbook reader instead of the validated local file")
	default:
	}
}
