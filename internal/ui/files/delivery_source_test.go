package files

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
)

func deliveryNotification(m *Model) <-chan struct{} {
	done := make(chan struct{}, 16)
	m.notify = func() {
		select {
		case done <- struct{}{}:
		default:
		}
	}
	return done
}

func awaitDeliveryRead(t *testing.T, m Model, done <-chan struct{}) {
	t.Helper()
	if tab := selectedTab(m); tab == nil || !tab.readLoading {
		t.Fatal("no accepted asynchronous read")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("local preview read did not settle")
	}
	syncTextRead(m)
	if selectedTab(m).readLoading {
		t.Fatal("completion left the preview loading")
	}
}

func deliveryHostTraps(m *Model) *atomic.Int32 {
	calls := &atomic.Int32{}
	m.ReadRemote = func(string) (string, error) { calls.Add(1); return "Host bytes", nil }
	m.ReadRemoteContext = func(context.Context, string) (string, error) { calls.Add(1); return "Host bytes", nil }
	m.ReadTextPageContext = func(context.Context, string, int64) (workspacefs.TextPage, error) {
		calls.Add(1)
		return workspacefs.TextPage{Text: "Host bytes", EOF: true}, nil
	}
	m.ReadTableContext = func(context.Context, string, workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
		calls.Add(1)
		return nil, errors.New("Host table trap")
	}
	m.StatRemote = func(string) string { calls.Add(1); return "Host stat" }
	m.StatRemoteContext = func(context.Context, string) string { calls.Add(1); return "Host stat" }
	m.WatchRemoteContext = func(context.Context, string, func()) error { calls.Add(1); return errors.New("Host watch trap") }
	m.OpenDocumentViewer = func(string) (DocumentSurface, error) {
		calls.Add(1)
		return &fixtureDocumentSurface{recommended: true}, nil
	}
	return calls
}

func TestDeliveryDocumentRoutesExactFileAndPropagatesUnavailable(t *testing.T) {
	for _, tc := range []struct {
		path string
		kind DeliveryPreviewKind
	}{{"page.pdf", DeliveryPreviewPDF}, {"page.html", DeliveryPreviewHTML}, {"clip.mp4", DeliveryPreviewMedia}} {
		t.Run(tc.path, func(t *testing.T) {
			m := deliveryAuditModel(t, map[string][]byte{tc.path: []byte("document fixture"), "draft.txt": []byte("saved")})
			*m.SelectedFile, *m.Preview, *m.EditDraft, *m.Editing = "draft.txt", "saved", "unsaved", true
			m.Tabs.Root = filepath.Clean(m.WorkspaceRoot)
			rememberTab(m, "draft.txt")
			calls := 0
			m.OnOpenDoc = func(string, workspacefs.DocKind, bool) { t.Error("default/Host opener invoked") }
			m.OnOpenMedia = func(string, bool) { t.Error("Host media opener invoked") }
			m.OpenDeliveryDocument = func(path string, kind DeliveryPreviewKind) error {
				calls++
				if path != filepath.Join(m.WorkspaceRoot, tc.path) || kind != tc.kind {
					t.Fatalf("wrong addressed route %q %q", path, kind)
				}
				return nil
			}
			if err := PreviewDeliveryPath(m, tc.path); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || *m.SelectedFile != "draft.txt" || *m.EditDraft != "unsaved" || !*m.Editing {
				t.Fatal("route changed a native draft or never opened")
			}
			want := errors.New("fixture local viewer unavailable")
			m.OpenDeliveryDocument = func(string, DeliveryPreviewKind) error { return want }
			if err := PreviewDeliveryPath(m, tc.path); !errors.Is(err, want) {
				t.Fatalf("viewer rejection lost: %v", err)
			}
		})
	}
}

func deliveryWorkbookBytes(t *testing.T) []byte {
	t.Helper()
	var body bytes.Buffer
	z := zip.NewWriter(&body)
	for _, part := range [][2]string{
		{"xl/workbook.xml", `<workbook xmlns:r="urn:r"><sheets><sheet name="Saved" r:id="r1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<Relationships><Relationship Id="r1" Type="urn:worksheet/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`},
		{"xl/worksheets/sheet1.xml", `<worksheet><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>local saved cell</t></is></c></row><row r="61"><c r="A61" t="inlineStr"><is><t>local next page</t></is></c></row></sheetData></worksheet>`},
	} {
		w, err := z.Create(part[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write([]byte(part[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func TestDeliveryWorkbookLoadsNativeGridAndPagesOnlyLocal(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"book.xlsx": deliveryWorkbookBytes(t)})
	done := deliveryNotification(&m)
	hostCalls := deliveryHostTraps(&m)
	if err := PreviewDeliveryPath(m, "book.xlsx"); err != nil {
		t.Fatal(err)
	}
	if !selectedTab(m).readLoading || previewEditable(m, "book.xlsx") {
		t.Fatal("workbook did not enter readonly loading")
	}
	awaitDeliveryRead(t, m, done)
	tab := selectedTab(m)
	if tab.tablePage == nil || tab.Error != "" || !strings.Contains(tab.tablePage.Text, "local saved cell") || tab.Remote {
		t.Fatalf("local workbook not loaded: %+v", tab)
	}
	tester := ui.NewTester(func(c *ui.Context) {
		previewPane(c, m, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(0, 0, 0), ui.RGB(255, 255, 255))
	}, 720, 520)
	if !tester.HasText("local saved cell") {
		t.Fatal("native workbook grid never displayed saved cell")
	}
	req := tab.tablePage.Request
	req.Row = 60
	requestTablePage(m, req)
	awaitDeliveryRead(t, m, done)
	if tab.tablePage == nil || !strings.Contains(tab.tablePage.Text, "local next page") {
		t.Fatal("local workbook next page missing")
	}
	read := resourceReader(m, tab)(context.Background())
	if read.err != nil || read.remote || read.table == nil {
		t.Fatalf("observation source changed: %+v", read)
	}
	if hostCalls.Load() != 0 {
		t.Fatalf("local workbook invoked Host %d times", hostCalls.Load())
	}
}

func TestDeliveryTextLoadingReadOnlyReloadAndReplacement(t *testing.T) {
	for _, change := range []string{"rename", "replace"} {
		t.Run(change, func(t *testing.T) {
			m := deliveryAuditModel(t, map[string][]byte{"note.txt": []byte("local saved text")})
			done := deliveryNotification(&m)
			calls := deliveryHostTraps(&m)
			if err := PreviewDeliveryPath(m, "note.txt"); err != nil {
				t.Fatal(err)
			}
			tester := ui.NewTester(func(c *ui.Context) {
				previewPane(c, m, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(0, 0, 0), ui.RGB(255, 255, 255))
			}, 480, 300)
			if !tester.HasText(label("正在读取…", "Loading…")) {
				t.Fatal("accepted read did not show loading")
			}
			awaitDeliveryRead(t, m, done)
			if *m.Preview != "local saved text" || previewEditable(m, "note.txt") || explicitSaveDisabled(m) == false {
				t.Fatal("saved text not readonly")
			}
			path := filepath.Join(m.WorkspaceRoot, "note.txt")
			if err := os.Rename(path, filepath.Join(m.WorkspaceRoot, "renamed.txt")); err != nil {
				t.Fatal(err)
			}
			if change == "replace" {
				if err := os.WriteFile(path, []byte("replacement must not render"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			reloadSelectedFile(m)
			awaitDeliveryRead(t, m, done)
			if *m.PreviewErr == "" || strings.Contains(*m.Preview, "replacement") || !selectedTab(m).readBlocked {
				t.Fatal("missing/replaced local file accepted")
			}
			observation := resourceReader(m, selectedTab(m))(context.Background())
			if observation.err == nil || observation.remote {
				t.Fatal("observation read replacement or Host")
			}
			if calls.Load() != 0 {
				t.Fatalf("local read invoked Host %d times", calls.Load())
			}
		})
	}
}

func TestDeliveryTextPagingAndHostSwitchRetainLocalAuthority(t *testing.T) {
	body := strings.Repeat("local line\n", workspacefs.TextPreviewLimit/11+4)
	m := deliveryAuditModel(t, map[string][]byte{"large.txt": []byte(body)})
	done := deliveryNotification(&m)
	calls := deliveryHostTraps(&m)
	if err := PreviewDeliveryPath(m, "large.txt"); err != nil {
		t.Fatal(err)
	}
	awaitDeliveryRead(t, m, done)
	tab := selectedTab(m)
	if !tab.textPaged || tab.textPage.EOF {
		t.Fatal("fixture did not enter local paging")
	}
	beginNextTextPage(m)
	awaitDeliveryRead(t, m, done)
	if tab.Preview != body || !tab.textPage.EOF {
		t.Fatal("local text pages changed bytes")
	}
	m.Tabs.Bind("replacement-host/session", m.WorkspaceRoot)
	m.BindingID = "replacement-host/session"
	syncDeliveryPreview(m)
	syncTextRead(m)
	if *m.PreviewErr == "" || *m.Preview != "" || tab.readLoading || tab.readResume {
		t.Fatal("Host switch retained/restarted old delivery")
	}
	if calls.Load() != 0 {
		t.Fatalf("local paging invoked Host %d times", calls.Load())
	}
}

func TestDeliveryWatchInvalidatesAfterLateBaseline(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"note.txt": []byte("original")})
	done := deliveryNotification(&m)
	calls := deliveryHostTraps(&m)
	if err := PreviewDeliveryPath(m, "note.txt"); err != nil {
		t.Fatal(err)
	}
	awaitDeliveryRead(t, m, done)
	read := resourceReader(m, selectedTab(m))
	if initial := read(context.Background()); initial.err != nil || initial.remote || initial.text != "original" {
		t.Fatalf("initial observation did not read local content: %+v", initial)
	}
	local, err := deliveryReadModel(m, selectedTab(m))
	if err != nil {
		t.Fatal(err)
	}
	// The observer's initial read may complete before its watcher goroutine
	// starts. Establish that ordering explicitly, without scheduler timing.
	path := filepath.Join(m.WorkspaceRoot, "note.txt")
	if err := os.Rename(path, filepath.Join(m.WorkspaceRoot, "renamed.txt")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changed := make(chan struct{}, 1)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		_ = local.WatchRemoteContext(ctx, "", func() {
			select {
			case changed <- struct{}{}:
			default:
			}
		})
	}()
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("late watcher baseline did not invalidate the earlier read")
	}
	if observation := read(context.Background()); observation.err == nil || observation.remote {
		t.Fatalf("watch invalidation did not reveal the missing local file: %+v", observation)
	}
	if calls.Load() != 0 {
		t.Fatal("local observation used a Host watch/reader")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("local watcher did not stop on cancellation")
	}
}

func TestDeliveryObserverUsesLocalSourceAndCancelsOnPause(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"note.txt": []byte("original")})
	done := deliveryNotification(&m)
	calls := deliveryHostTraps(&m)
	if err := PreviewDeliveryPath(m, "note.txt"); err != nil {
		t.Fatal(err)
	}
	awaitDeliveryRead(t, m, done)
	observeFileResource(m)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("local observer did not read")
	}
	observeFileResource(m)
	path := filepath.Join(m.WorkspaceRoot, "note.txt")
	if err := os.Rename(path, filepath.Join(m.WorkspaceRoot, "renamed.txt")); err != nil {
		t.Fatal(err)
	}
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for !selectedTab(m).readBlocked {
		select {
		case <-done:
			// A ready or initial-read notification may predate the rename.
			// Only the applied observation can establish the resulting state.
			observeFileResource(m)
		case <-deadline.C:
			t.Fatal("local observer missed renamed file")
		}
	}
	if *m.EditStatus == "" || *m.Preview != "original" {
		t.Fatal("failed observation hid error or erased saved content")
	}
	if calls.Load() != 0 {
		t.Fatal("local observation used a Host watch/reader")
	}
	m.Tabs.Pause()
	if m.Tabs.resourceObserver != nil {
		t.Fatal("hidden Files pane retained an observer")
	}
}

func TestDeliveryHostSwitchRetiresQueuedLocalRead(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"note.txt": []byte("original")})
	done := deliveryNotification(&m)
	started, release, retired := make(chan struct{}), make(chan struct{}), make(chan struct{})
	queue := &textReadQueue{}
	queue.request(textReadOwner{path: "blocker"}, func(context.Context) textReadResult {
		close(started)
		<-release
		close(retired)
		return textReadResult{text: "stale"}
	})
	<-started
	m.Tabs.textReader = queue
	if err := PreviewDeliveryPath(m, "note.txt"); err != nil {
		close(release)
		t.Fatal(err)
	}
	m.Tabs.Bind("other-host/session", m.WorkspaceRoot)
	m.BindingID = "other-host/session"
	close(release)
	<-retired
	syncDeliveryPreview(m)
	syncTextRead(m)
	if *m.Preview != "" || *m.PreviewErr == "" || selectedTab(m).readLoading {
		t.Fatal("stale accepted read survived Host switch")
	}
	select {
	case <-done:
		t.Fatal("retired read published completion")
	default:
	}
}

func TestDeliveryReadDeniedRemainsLocalAndShowsFailure(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"note.txt": []byte("original")})
	done := deliveryNotification(&m)
	calls := deliveryHostTraps(&m)
	if err := PreviewDeliveryPath(m, "note.txt"); err != nil {
		t.Fatal(err)
	}
	awaitDeliveryRead(t, m, done)
	path := filepath.Join(m.WorkspaceRoot, "note.txt")
	if runtime.GOOS == "windows" {
		// Windows does not enforce POSIX read bits. Removing the file's read
		// capability as a regular document (directory replacement) exercises the
		// same terminal local-open failure without modifying any real ACL.
		if err := os.Rename(path, filepath.Join(m.WorkspaceRoot, "original.txt")); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := os.Chmod(path, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0600) })
		if f, err := os.Open(path); err == nil {
			_ = f.Close()
			t.Skip("current account bypasses file permission bits")
		}
	}
	reloadSelectedFile(m)
	awaitDeliveryRead(t, m, done)
	if *m.PreviewErr == "" || !selectedTab(m).readBlocked || calls.Load() != 0 {
		t.Fatal("local open denial was hidden or switched reader")
	}
}

func TestDeliveryOpenedHandleRejectsChangedFileAndWorkspace(t *testing.T) {
	m := deliveryAuditModel(t, map[string][]byte{"note.txt": []byte("original")})
	done := deliveryNotification(&m)
	if err := PreviewDeliveryPath(m, "note.txt"); err != nil {
		t.Fatal(err)
	}
	awaitDeliveryRead(t, m, done)
	source := selectedTab(m).deliverySource
	root, file, info, err := source.open()
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(source.file, info) {
		t.Fatal("opened unchecked handle")
	}
	_ = file.Close()
	_ = root.Close()
	path := filepath.Join(m.WorkspaceRoot, "note.txt")
	if err := os.Rename(path, filepath.Join(m.WorkspaceRoot, "original.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if data, err := source.readBytes(context.Background(), 64); err == nil {
		t.Fatalf("changed file read as declaration: %q", data)
	}
	copy := *source
	copy.rootFile = source.file
	if _, err := copy.readBytes(context.Background(), 64); err == nil {
		t.Fatal("changed root identity accepted")
	}
}
