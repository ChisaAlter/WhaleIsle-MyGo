package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/i18n"
)

func TestFileTreeVirtualizesLargeDirectory(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() {
		i18n.SetLocale(i18n.DefaultLocale)
		filesTreeVirtualRowHook = nil
	})

	root := t.TempDir()
	entries := make([]workspacefs.Entry, 150)
	for i := range entries {
		entries[i] = workspacefs.Entry{Name: fmt.Sprintf("file-%03d.txt", i)}
	}

	tabs := &Tabs{Tree: &workspacefs.DirectoryTree{}, Search: &workspacefs.Search{}}
	t.Cleanup(func() {
		tabs.Tree.Cancel()
		tabs.Search.Cancel()
	})
	filter := ""
	m := Model{
		BindingID:     "fixture",
		WorkspaceRoot: root,
		TreeOnly:      true,
		HostOnly:      true,
		Tabs:          tabs,
		Filter:        &filter,
		ThemeID:       "deepseek.light",
		ListRemoteContext: func(context.Context, string) ([]workspacefs.Entry, error) {
			return entries, nil
		},
	}

	built := map[int]string{}
	filesTreeVirtualRowHook = func(mode string, index int, key string) {
		if mode == "tree" {
			built[index] = key
		}
	}

	tt := ui.NewTester(func(c *ui.Context) {
		clear(built)
		View(c, m)
	}, 280, 320)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		clear(built)
		tt.Frame()
		if tt.HasText("file-000.txt") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !tt.HasText("file-000.txt") {
		t.Fatal("directory tree did not load initial file rows")
	}
	if len(built) == 0 || len(built) >= 30 {
		t.Fatalf("file tree built %d rows for 150-file directory, want bounded subset (<30)", len(built))
	}
	if tt.HasText("file-120.txt") {
		t.Fatal("off-screen file row should not be mounted in virtualized tree")
	}
}

func TestPreviewFindBarInputIsFlexibleInNarrowPane(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	root := t.TempDir()
	rel := "note.txt"
	if err := os.WriteFile(filepath.Join(root, rel), []byte("alpha beta alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	selected := rel
	tabs := &Tabs{
		Root: root,
		Workspaces: map[string]*WorkspaceTabs{
			root: {
				Active: rel,
				Open: []*FileTab{{
					Path:      rel,
					Preview:   "alpha beta alpha\n",
					findOpen:  true,
					findQuery: "alpha",
				}},
			},
		},
	}
	m := Model{
		WorkspaceRoot: root,
		SelectedFile:  &selected,
		Tabs:          tabs,
		ThemeID:       "deepseek.light",
	}

	const narrowW = 240
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintPreviewFindBar(c, m, rel, ui.RGB(200, 200, 200), ui.RGB(120, 120, 120))
		})
	}, narrowW, 80)
	tt.Frame()

	inputRect, ok := tt.Find("Find in file")
	if !ok {
		t.Fatal("find input missing")
	}
	if inputRect.W >= 200 {
		t.Fatalf("find input stayed rigid at %v DIP in %dpx narrow pane", inputRect.W, narrowW)
	}
	for _, btnLabel := range []string{"Previous match", "Next match", "Close find"} {
		r, found := tt.Find(btnLabel)
		if !found {
			t.Fatalf("missing find bar control %q", btnLabel)
		}
		if r.X+r.W > narrowW+0.5 {
			t.Fatalf("find bar control %q pushed outside narrow pane: %+v", btnLabel, r)
		}
	}
}
