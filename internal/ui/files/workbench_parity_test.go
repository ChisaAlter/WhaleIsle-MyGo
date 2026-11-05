package files

import (
	"context"
	"errors"
	"github.com/egoist/mygo/ui"
	"testing"
	"time"
	"whaleisle/internal/desktop/workspacefs"
)

func TestHostPresentedPathDoesNotRequireDesktopMount(t *testing.T) {
	for _, tt := range []struct{ root, value, want string }{
		{"/remote/work", "/remote/work/src/main.go", "src/main.go"},
		{"/remote/work", "src/../report.md", "report.md"},
		{`C:\Remote\Work`, `c:\remote\work\src\main.go`, "src/main.go"},
		{"/", "/report.md", "report.md"},
	} {
		got, ok := hostPresentedPath(tt.root, tt.value)
		if !ok || got != tt.want {
			t.Fatalf("%q %q -> %q %v", tt.root, tt.value, got, ok)
		}
	}
	for _, raw := range []string{"../private.md", "/other/report.md", "https://example.test/report.md", `C:\unrelated\file.md`} {
		if _, ok := hostPresentedPath("/remote/work", raw); ok {
			t.Fatalf("claimed foreign target %q", raw)
		}
	}
	var selected, preview, draft, problem string
	tabs := &Tabs{}
	m := Model{Tabs: tabs, WorkspaceRoot: "/unmounted/host/work", BindingID: "host/session", SessionID: "session", SelectedFile: &selected, Preview: &preview, EditDraft: &draft, PreviewErr: &problem, ReadRemoteContext: func(context.Context, string) (string, error) { return "saved Host text", nil }}
	t.Cleanup(tabs.Dispose)
	if ok, err := OpenPresentedPath(m, "src/main.go:12"); !ok || err != nil || selected != "src/main.go" {
		t.Fatalf("Host entry did not reach Files: %v %v %q", ok, err, selected)
	}
}

func TestUnselectedFileRowOffersExplicitEditor(t *testing.T) {
	selected := "current.txt"
	var opened string
	m := Model{WorkspaceRoot: "/host/work", SelectedFile: &selected, EditorApps: []OpenWithApp{{ID: "code", Name: "Visual Studio Code"}},
		ListRemoteContext: func(context.Context, string) ([]workspacefs.Entry, error) { return nil, nil },
		OnOpenWithEditor: func(path, id string, line int) {
			if id != "code" || line != 0 {
				t.Errorf("wrong editor position %s %d", id, line)
			}
			opened = path
		}}
	tt := ui.NewTester(func(c *ui.Context) { ui.Button(c, "unselected row").ContextMenu(treeFileMenu(m, "other.txt")) }, 400, 100)
	if err := tt.RightClick("unselected row"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Visual Studio Code"); err != nil {
		t.Fatal(err)
	}
	if opened != "/host/work/other.txt" || selected != "current.txt" {
		t.Fatal(opened, selected)
	}
}

func TestHostImageByteReaderIsCapturedAndUsed(t *testing.T) {
	m, tab, notify := imageRouteModel("/unmounted/host/work")
	t.Cleanup(m.Tabs.Dispose)
	data := imageRoutePNG(t, 4, 3)
	m.ReadImageContext = func(ctx context.Context, path string, limit int64) ([]byte, error) {
		if path != "same.png" || limit != 32<<20 {
			return nil, errors.New("wrong captured image request")
		}
		return data, nil
	}
	if p := awaitImageRoute(t, m, tab, notify); p.err != nil || p.bitmap == nil {
		t.Fatalf("Host image: %+v", p)
	}
}

func TestMarkdownImageUsesSourceDirectoryAndCancelsOnPause(t *testing.T) {
	for _, tt := range []struct{ ref, want string }{{"../images/a.png", "images/a.png"}, {"images/a.png?rev=2", "docs/images/a.png"}, {"a%20b.png", "docs/a b.png"}} {
		if got, err := markdownImagePath("docs/readme.md", tt.ref); err != nil || got != tt.want {
			t.Fatalf("%q -> %q %v", tt.ref, got, err)
		}
	}
	for _, ref := range []string{"../../secret.png", "/secret.png", "file:///secret.png", "https://user:password@example.test/a.png"} {
		if _, err := markdownImagePath("docs/readme.md", ref); err == nil {
			t.Fatal(ref)
		}
	}
	tab := &FileTab{Path: "docs/readme.md"}
	started, canceled := make(chan struct{}), make(chan struct{})
	m := Model{WorkspaceRoot: "/unmounted/host/work", Tabs: &Tabs{}, BindingID: "host/session", ReadImageContext: func(ctx context.Context, path string, _ int64) ([]byte, error) {
		if path != "images/a.png" {
			return nil, errors.New("wrong image resource")
		}
		close(started)
		<-ctx.Done()
		close(canceled)
		return nil, ctx.Err()
	}}
	w := tabsFor(m)
	w.Open = []*FileTab{tab}
	w.Active = tab.Path
	q := &imageReadQueue{}
	tab.markdownImages = map[string]*markdownImageRead{"../images/a.png": {queue: q, owner: imageOwner(m, tab)}}
	q.request(imageOwner(m, tab), markdownImageLoader(m, tab, "../images/a.png"))
	<-started
	m.Tabs.Pause()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("hidden tab retained image request")
	}
	if tab.markdownImages != nil {
		t.Fatal("image resources retained after pause")
	}
}
