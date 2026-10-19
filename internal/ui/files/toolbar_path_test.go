package files

import (
	"context"
	"github.com/egoist/mygo/ui"
	"slices"
	"testing"
	"whaleisle/internal/ui/i18n"
)

func TestToolbarHostPathAndReveal(t *testing.T) {
	m, tab := resourceModel()
	m.WorkspaceRoot = `C:\remote\project`
	tab.readFullPath = ""
	m.ReadRemoteContext = func(context.Context, string) (string, error) { return "", nil }
	var revealed string
	m.OnReveal = func(path string) { revealed = path }
	if got := toolbarAbsolutePath(m, "folder/a.txt"); got != "C:/remote/project/folder/a.txt" {
		t.Fatal(got)
	}
	revealToolbarPath(m, "folder/a.txt", "")
	if revealed != "C:/remote/project/folder/a.txt" {
		t.Fatal(revealed)
	}
	var opened string
	withToolbarPath(m, "folder/a.txt", "", func(path string) { opened = path })
	if opened != "C:/remote/project/folder/a.txt" {
		t.Fatal("Open With target", opened)
	}
	revealed = ""
	revealToolbarPath(m, "../outside", "")
	if revealed != "" {
		t.Fatal("escaped workspace")
	}
}
func TestToolbarNarrowActionsRemainVisible(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(old)
	m, tab := resourceModel()
	tab.Path = "folder/a.txt"
	*m.SelectedFile = tab.Path
	tabsFor(m).Active = tab.Path
	m.OnReveal = func(string) {}
	m.ReadRemoteContext = func(context.Context, string) (string, error) { return "", nil }
	tt := ui.NewTester(func(c *ui.Context) {
		previewPane(c, m, c.Theme().Text, c.Theme().Text, c.Theme().Border, c.Theme().Background)
	}, 360, 600)
	for _, text := range []string{"Copy absolute path", "Copy relative path", "Reload", i18n.T(i18n.KeyFilesReveal)} {
		if err := tt.RightClick("File preview toolbar"); err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(tt.Menu(), text) {
			t.Fatalf("inaccessible context action %s: %v", text, tt.Menu())
		}
		tt.CloseMenu()
	}
	if rect, ok := tt.Find("Preview in separate window"); !ok || rect.X < 0 || rect.X+rect.W > 361 {
		t.Fatalf("floating action displaced: %+v", rect)
	}
}
