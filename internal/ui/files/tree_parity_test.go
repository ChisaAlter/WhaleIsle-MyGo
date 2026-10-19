package files

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/filelink"
	"whaleisle/internal/ui/i18n"
)

func TestNativeFileTreeRowsKeepHeightIndentAndOpaqueSelection(t *testing.T) {
	expanded := map[string]bool{}
	selected := "deep/file.txt"
	m := Model{BindingID: "fixture", WorkspaceRoot: t.TempDir(), Expanded: &expanded, SelectedFile: &selected}
	ink, muted, accent := ui.RGB(20, 20, 20), ui.RGB(90, 90, 90), ui.RGB(65, 118, 230)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
			rowDir(c, m, "one", "First folder", 0, ink, muted, accent)
			rowDir(c, m, "one/two", "Nested folder", 1, ink, muted, accent)
			rowDir(c, m, "one/two/three/four", "Deep folder", 3, ink, muted, accent)
			rowFile(c, m, selected, "MMMM selected file.txt", 3, ink, muted, accent)
		})
	}, 300, 600)
	for _, name := range []string{"Folder one", "Folder one/two", "Folder one/two/three/four", "File " + selected} {
		r, ok := tt.Find(name)
		if !ok || math.Abs(float64(r.H-28)) > .1 {
			t.Fatalf("row %q has incorrect natural height: %+v", name, r)
		}
	}
	first, _ := tt.Find("First folder")
	nested, _ := tt.Find("Nested folder")
	deep, _ := tt.Find("Deep folder")
	if math.Abs(float64(nested.X-first.X-12)) > .1 || math.Abs(float64(deep.X-first.X-36)) > .1 {
		t.Fatalf("tree nesting changed vertical padding instead of horizontal inset: first=%+v nested=%+v deep=%+v", first, nested, deep)
	}
	text, _ := tt.Find("MMMM selected file.txt")
	image := tt.Image()
	dark := 0
	for y := int(text.Y); y < int(text.Y+text.H); y++ {
		for x := int(text.X); x < int(text.X+text.W); x++ {
			pixel := image.RGBAAt(x, y)
			if pixel.R < 80 && pixel.G < 80 && pixel.B < 80 {
				dark++
			}
		}
	}
	if dark < 5 {
		t.Fatal("selected row faded its text with the background")
	}
	if tt.HasText(label("复制引用", "Copy reference")) {
		t.Fatal("copy-reference action still occupies every file row")
	}
	if err := tt.Click("Folder one"); err != nil {
		t.Fatal(err)
	}
	if !expanded["one"] {
		t.Fatal("folder row did not expand")
	}
	tt.Key(0, ui.KeyEnter)
	if expanded["one"] {
		t.Fatal("keyboard activation did not collapse folder")
	}
}

func TestNativeFileTreeExternalOpenUsesClickedPathWithoutSelecting(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "clicked.txt"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	selected := "other.txt"
	opened := ""
	m := Model{WorkspaceRoot: root, SelectedFile: &selected, OnOpenDoc: func(path string, _ workspacefs.DocKind, inWebView bool) {
		if inWebView {
			t.Error("system opener selected webview")
		}
		opened = path
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			rowFile(c, m, "clicked.txt", "clicked.txt", 0, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(65, 118, 230))
		})
	}, 300, 200)
	if err := tt.RightClick("File clicked.txt"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeyFilesOpenExternal)); err != nil {
		t.Fatal(err)
	}
	if opened != filepath.Join(root, "clicked.txt") || selected != "other.txt" {
		t.Fatalf("menu used selected file instead of clicked file: opened=%q selected=%q", opened, selected)
	}
}

func TestNativeFileTreeReferenceAndContextMenuKeepPathIdentity(t *testing.T) {
	expanded := map[string]bool{}
	selected := ""
	var referenced filelink.Reference
	m := Model{BindingID: "fixture", SessionID: "session", WorkspaceRoot: t.TempDir(), Expanded: &expanded, SelectedFile: &selected, CanAddToChat: true, OnReferenceFile: func(value filelink.Reference) error { referenced = value; return nil }}
	path := "folder/" + strings.Repeat("long-name-", 12) + ".txt"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { rowFile(c, m, path, path, 2, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(65, 118, 230)) })
	}, 280, 200)
	row, _ := tt.Find("File " + path)
	tt.Move(row.X+20, row.Y+14)
	button, ok := tt.Find("Reference file " + path)
	if !ok || button.X+button.W > row.X+row.W || button.W != 20 || button.H != 20 {
		t.Fatalf("long file name displaced reference action: %+v row=%+v", button, row)
	}
	if err := tt.Click("Reference file " + path); err != nil {
		t.Fatal(err)
	}
	if referenced.Path != path || referenced.BindingID != "fixture" || referenced.SessionID != "session" || selected != "" {
		t.Fatal("reference selected file or lost initiating identity")
	}
	if err := tt.RightClick("File " + path); err != nil {
		t.Fatal(err)
	}
	if tt.Menu() == nil {
		t.Fatal("file context menu missing")
	}
	tt.CloseMenu()
}

func TestNativeFileTreeEmptyAndErrorStatesStayDistinct(t *testing.T) {
	for _, failed := range []bool{false, true} {
		root := t.TempDir()
		m := Model{BindingID: "fixture", WorkspaceRoot: root, Tabs: &Tabs{Tree: &workspacefs.DirectoryTree{}}}
		if failed {
			m.WorkspaceRoot = filepath.Join(root, "missing")
			m.ListRemoteContext = func(context.Context, string) ([]workspacefs.Entry, error) {
				return nil, errors.New("fixture directory failure")
			}
		}
		m.Tabs.Tree.Ensure(m.BindingID, m.WorkspaceRoot)
		tt := ui.NewTester(func(c *ui.Context) {
			paintDir(c, m, m.WorkspaceRoot, "", 0, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(65, 118, 230))
		}, 400, 300)
		deadline := time.Now().Add(2 * time.Second)
		for m.Tabs.Tree.Get("", m.ListRemoteContext, func() {}).Busy {
			if time.Now().After(deadline) {
				t.Fatal("isolated directory read did not settle")
			}
			time.Sleep(time.Millisecond * 5)
			tt.Frame()
		}
		tt.Frame()
		empty := tt.HasText(label("此文件夹为空", "This folder is empty"))
		if failed {
			if empty || !tt.HasText("fixture directory failure") {
				t.Fatal("directory error was disguised as empty")
			}
		} else if !empty {
			t.Fatal("ready empty directory had no feedback")
		}
		m.Tabs.Tree.Cancel()
	}
}
