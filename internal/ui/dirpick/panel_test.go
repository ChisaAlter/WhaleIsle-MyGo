package dirpick

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/i18n"
)

func TestDirPickView(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	name := ""
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Listing: api.DirectoryListing{
				Path: "/home",
				Crumbs: []api.DirectoryListingCrumbsItem{
					{Name: "home", Path: "/home"},
				},
				Entries: []api.DirectoryListingCrumbsItem{
					{Name: "proj", Path: "/home/proj"},
				},
			},
			ThemeID:    "system",
			SystemDark: false,
			NewName:    &name,
			OnNavigate: func(string) {},
			OnPick:     func(string) {},
			OnCreate:   func(string, string) {},
			OnClose:    func() {},
		})
	}, 480, 360)
	for _, label := range []string{
		i18n.T(i18n.KeyDirPickTitle),
		"home",
		"proj",
		i18n.T(i18n.KeyDirPickSelect),
		i18n.T(i18n.KeyDirPickSelectHere),
		i18n.T(i18n.KeyDirPickCreate),
	} {
		if !tt.HasText(label) {
			t.Fatalf("missing %q; texts=%q", label, tt.Texts())
		}
	}
}

func TestFolderListScrollsWithoutMovingFooter(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	entries := make([]api.DirectoryListingCrumbsItem, 40)
	for index := range entries {
		entries[index] = api.DirectoryListingCrumbsItem{Name: fmt.Sprintf("Folder %02d", index), Path: fmt.Sprintf("/host/%02d", index)}
	}
	var picked string
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Listing: api.DirectoryListing{Path: "/host", Entries: entries}, OnNavigate: func(string) {}, OnPick: func(path string) { picked = path }})
	}, 300, 480)
	footer, ok := tt.Find(i18n.T(i18n.KeyDirPickSelectHere))
	if !ok || footer.Y+footer.H > 480 {
		t.Fatalf("selection footer unavailable: %+v", footer)
	}
	first, ok := tt.Find("Folder 00")
	if !ok {
		t.Fatal("missing first folder")
	}
	tt.Scroll(first.X+first.W/2, first.Y+first.H/2, 0, 2000)
	last, ok := tt.Find("Folder 39")
	if !ok || last.Y < first.Y || last.Y+last.H > footer.Y {
		t.Fatalf("last folder not reachable above footer: last=%+v footer=%+v", last, footer)
	}
	after, _ := tt.Find(i18n.T(i18n.KeyDirPickSelectHere))
	if after != footer {
		t.Fatalf("footer moved during list scroll: before=%+v after=%+v", footer, after)
	}
	saveSecondarySnapshot(t, tt, "directory-picker-normal.png")
	if err := tt.Click(i18n.T(i18n.KeyDirPickSelectHere)); err != nil {
		t.Fatal(err)
	}
	if picked != "/host" {
		t.Fatalf("picked=%q", picked)
	}
}

func TestFolderNavigationAndCreationKeepHostPaths(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	name := " new folder "
	var navigated, parent, created string
	listing := api.DirectoryListing{Path: "/host/current", Home: "/host/home", Truncated: true, Crumbs: []api.DirectoryListingCrumbsItem{{Name: strings.Repeat("deep crumb ", 20), Path: "/host"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Listing: listing, NewName: &name, OnNavigate: func(path string) { navigated = path }, OnCreate: func(path, value string) { parent, created = path, value }})
	}, 280, 360)
	if !tt.HasText("Only part of this folder is shown.") {
		t.Fatal("truncated listing looks complete")
	}
	if err := tt.Click("Home"); err != nil {
		t.Fatal(err)
	}
	if navigated != "/host/home" {
		t.Fatalf("home target=%q", navigated)
	}
	if err := tt.Click(i18n.T(i18n.KeyDirPickNewName)); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if parent != "/host/current" || created != "new folder" || name != "" {
		t.Fatalf("create=(%q,%q), draft=%q", parent, created, name)
	}
	created = ""
	tt.Key(0, ui.KeyEnter)
	if created != "" {
		t.Fatal("blank create was submitted")
	}
}

func TestFolderActionsRemainVisibleInShortWindow(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	entries := make([]api.DirectoryListingCrumbsItem, 40)
	for index := range entries {
		entries[index] = api.DirectoryListingCrumbsItem{Name: fmt.Sprintf("Folder %02d", index), Path: fmt.Sprintf("/host/%02d", index)}
	}
	name := ""
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Listing: api.DirectoryListing{Path: "/host", Entries: entries, Truncated: true}, NewName: &name, OnCreate: func(string, string) {}, OnPick: func(string) {}})
	}, 280, 320)
	for _, label := range []string{i18n.T(i18n.KeyDirPickCreate), i18n.T(i18n.KeyDirPickSelectHere)} {
		bounds, ok := tt.Find(label)
		if !ok || bounds.Y < 0 || bounds.Y+bounds.H > 320 {
			t.Fatalf("short-window action %q is clipped: %+v", label, bounds)
		}
	}
	saveSecondarySnapshot(t, tt, "directory-picker-short.png")
}

func TestFolderPendingStateBlocksActionsButKeepsClose(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	name := "new"
	actions, closed := 0, 0
	m := Model{Loading: true, Listing: api.DirectoryListing{Path: "/host", Home: "/home", Entries: []api.DirectoryListingCrumbsItem{{Name: "Child", Path: "/host/child"}}}, NewName: &name,
		OnNavigate: func(string) { actions++ }, OnPick: func(string) { actions++ }, OnCreate: func(string, string) { actions++ }, OnClose: func() { closed++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 360, 400)
	for _, label := range []string{"Home", "Child", i18n.T(i18n.KeyDirPickSelectHere), i18n.T(i18n.KeyDirPickCreate)} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if actions != 0 {
		t.Fatalf("pending list invoked %d actions", actions)
	}
	if !tt.HasText("Loading folders…") {
		t.Fatal("missing pending state")
	}
	if err := tt.Click(i18n.T(i18n.KeyDirPickClose)); err != nil {
		t.Fatal(err)
	}
	if closed != 1 {
		t.Fatalf("Close unavailable during loading: count=%d", closed)
	}
	m.Loading = false
	m.Busy = true
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeyDirPickSelectHere)); err != nil {
		t.Fatal(err)
	}
	if actions != 0 {
		t.Fatal("busy selection invoked callback")
	}
}

func TestFolderErrorRetryPreservesListingAndRespectsPendingState(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	retries := 0
	m := Model{Err: "The Host could not read this folder", Listing: api.DirectoryListing{Path: "/host/good", Entries: []api.DirectoryListingCrumbsItem{{Name: "Retained folder", Path: "/host/good/retained"}}}, OnRetry: func() { retries++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 360, 320)
	if !tt.HasText(m.Err) || !tt.HasText("Retained folder") {
		t.Fatal("retry surface lost the error or prior listing")
	}
	if err := tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	if retries != 1 {
		t.Fatalf("retry callback count=%d", retries)
	}
	for _, loading := range []bool{true, false} {
		m.Loading, m.Busy = loading, !loading
		tt.Frame()
		if err := tt.Click("Retry"); err != nil {
			t.Fatal(err)
		}
		if retries != 1 {
			t.Fatalf("pending retry invoked callback: Loading=%t Busy=%t", m.Loading, m.Busy)
		}
	}
}

func saveSecondarySnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	encodeErr := png.Encode(file, tt.Image())
	closeErr := file.Close()
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
