package files

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/desktop/workspacefs"
)

func resourceModel() (Model, *FileTab) {
	path, preview, draft, problem, status := "a.txt", "saved", "saved", "", ""
	root := filepath.Clean(filepath.FromSlash("/remote"))
	tab := &FileTab{Path: path, Preview: preview, Draft: draft, readID: 1}
	tabs := &Tabs{binding: "host", boundRoot: root, Workspaces: map[string]*WorkspaceTabs{root: {Open: []*FileTab{tab}, Active: path}}}
	return Model{BindingID: "host", WorkspaceRoot: root, SessionID: "s", Tabs: tabs, SelectedFile: &path, Preview: &preview, EditDraft: &draft, PreviewErr: &problem, EditStatus: &status}, tab
}
func TestResourceObservationRefreshesOnlyChangedCleanContent(t *testing.T) {
	m, tab := resourceModel()
	tab.Status = "keep status"
	applyResourceObservation(m, tab, textReadResult{text: "saved"})
	if tab.Status != "keep status" {
		t.Fatal("noisy signal mutated unchanged content")
	}
	applyResourceObservation(m, tab, textReadResult{text: "external", remote: true})
	if tab.Preview != "external" || tab.Draft != "external" || tab.externalChanged {
		t.Fatal("clean content not refreshed")
	}
	*m.EditDraft = "unsaved"
	capture(m)
	applyResourceObservation(m, tab, textReadResult{text: "external again", remote: true})
	if tab.Preview != "external" || tab.Draft != "unsaved" || !tab.externalChanged || !tab.readBlocked {
		t.Fatal("dirty baseline/draft lost or saving unblocked")
	}
	applyResourceObservation(m, tab, textReadResult{err: errors.New("deleted")})
	if tab.Draft != "unsaved" || tab.Preview != "external" {
		t.Fatal("delete erased draft")
	}
}
func TestResourceObservationDisabledRetainsLoadedContent(t *testing.T) {
	m, tab := resourceModel()
	tab.AutoRefreshDisabled = true
	applyResourceObservation(m, tab, textReadResult{text: "external"})
	if tab.Preview != "saved" || !tab.externalChanged {
		t.Fatal("disabled resource auto-refreshed")
	}
	tab.AutoRefreshDisabled = false
	applyResourceObservation(m, tab, textReadResult{text: "external"})
	if tab.Preview != "external" || tab.externalChanged {
		t.Fatal("reenable failed")
	}
}
func TestResourceWatchUsesOwnedReaderAndCancelsOnPause(t *testing.T) {
	m, tab := resourceModel()
	tab.Remote = true // The completed read selected the Host, not a local editor.
	signal := make(chan func(), 1)
	stopped := make(chan struct{})
	notified := make(chan struct{}, 16)
	var content atomic.Value
	content.Store("saved")
	m.notify = func() {
		select {
		case notified <- struct{}{}:
		default:
		}
	}
	m.ReadTextPageContext = func(ctx context.Context, path string, offset int64) (workspacefs.TextPage, error) {
		return workspacefs.TextPage{Text: content.Load().(string), EOF: true, Version: "content"}, nil
	}
	m.WatchRemoteContext = func(ctx context.Context, dir string, changed func()) error {
		signal <- changed
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	}
	observeFileResource(m)
	var changed func()
	select {
	case changed = <-signal:
	case <-time.After(time.Second):
		t.Fatal("watch not started")
	}
	select {
	case <-notified:
	case <-time.After(time.Second):
		t.Fatal("initial read missing")
	}
	observeFileResource(m)
	content.Store("changed")
	changed()
	select {
	case <-notified:
	case <-time.After(time.Second):
		t.Fatal("change read missing")
	}
	observeFileResource(m)
	if tab.Preview != "changed" {
		t.Fatal("watch did not refresh resource")
	}
	m.Tabs.Pause()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("hidden pane leaked watch")
	}
	content.Store("late")
	changed()
	observe := m.Tabs.resourceObserver
	if observe != nil || tab.Preview != "changed" {
		t.Fatal("late result changed hidden resource")
	}
}
func TestResourceTableIgnoresVersionOnlyNoise(t *testing.T) {
	page := &workspacefs.XLSXPage{Text: "cell", Version: "v1", Request: workspacefs.XLSXPageRequest{Sheet: "second", Version: "v1"}}
	tab := &FileTab{Path: "a.xlsx", tablePage: page}
	next := *page
	next.Version = "v2"
	next.Request.Version = "v2"
	if resourceContentChanged(tab, textReadResult{table: &next}) {
		t.Fatal("metadata-only version refreshed table")
	}
	next.Text = "new cell"
	if !resourceContentChanged(tab, textReadResult{table: &next}) {
		t.Fatal("table content change missed")
	}
}

func TestResourcePagedTextKeepsLoadedRangeAndIgnoresMetadata(t *testing.T) {
	tab := &FileTab{Path: "large.txt", Preview: "first\nsecond", Draft: "first\nsecond", Truncated: true, textPaged: true, textPage: workspacefs.TextPage{Next: 2, Version: "old", EOF: false}}
	calls := 0
	read := func(offset int64) (workspacefs.TextPage, error) {
		calls++
		if offset == 0 {
			return workspacefs.TextPage{Text: "first", Next: 1, Version: "new"}, nil
		}
		return workspacefs.TextPage{Text: "second", Separator: "\n", Next: 2, Version: "new"}, nil
	}
	page, text, err := observeTextPages(context.Background(), read, 2)
	if err != nil || text != "first\nsecond" || calls != 2 {
		t.Fatalf("range %q calls%d err%v", text, calls, err)
	}
	if resourceContentChanged(tab, textReadResult{text: text, page: &page, truncated: true}) {
		t.Fatal("version-only text noise counted as content change")
	}
	page.Text = "changed"
	if !resourceContentChanged(tab, textReadResult{text: "first\nchanged", page: &page, truncated: true}) {
		t.Fatal("loaded second-page change missed")
	}
}
func TestResourceAutoRefreshPreferencePersistsWithoutWatcher(t *testing.T) {
	m, tab := resourceModel()
	tab.AutoRefreshDisabled = true
	tab.externalChanged = true
	data, err := m.Tabs.Export()
	if err != nil {
		t.Fatal(err)
	}
	var restored Tabs
	if err = restored.Restore(data); err != nil {
		t.Fatal(err)
	}
	got := restored.Workspaces[filepath.Clean(m.WorkspaceRoot)].Open[0]
	if !got.AutoRefreshDisabled || got.externalChanged || restored.resourceObserver != nil {
		t.Fatal("lost preference or persisted observation authority")
	}
}
func TestResourceReaderHostFailureNeverFallsBackLocal(t *testing.T) {
	m, tab := resourceModel()
	tab.Remote = true
	wanted := errors.New("Host read failed")
	m.ReadRemoteContext = func(context.Context, string) (string, error) { return "", wanted }
	got := resourceReader(m, tab)(context.Background())
	if !errors.Is(got.err, wanted) || !got.remote {
		t.Fatal("Host authority lost")
	}
}
