package dirpick

import (
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/i18n"
)

func fixtureBrowser() BrowserState {
	parent := &api.DirectoryListing{Path: "/home", Home: "/home", Crumbs: []api.DirectoryListingCrumbsItem{{Name: "/", Path: "/"}, {Name: "home", Path: "/home"}}, Entries: []api.DirectoryListingCrumbsItem{{Name: "Project root", Path: "/home/project"}, {Name: "Other folder", Path: "/home/other"}, {Name: ".private", Path: "/home/.private", Hidden: true}}}
	child := &api.DirectoryListing{Path: "/home/project", Home: "/home", Crumbs: []api.DirectoryListingCrumbsItem{{Name: "/", Path: "/"}, {Name: "home", Path: "/home"}, {Name: "project", Path: "/home/project"}}}
	for index := 0; index < 40; index++ {
		child.Entries = append(child.Entries, api.DirectoryListingCrumbsItem{Name: fmt.Sprintf("Child %02d", index), Path: fmt.Sprintf("/home/project/%02d", index)})
	}
	selected := parent.Entries[0]
	return BrowserState{Parent: parent, Selected: &selected, Child: child}
}

func TestMillerPaneGeometryAndPinnedFooter(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, size := range [][2]int{{680, 500}, {372, 252}} {
		s := fixtureBrowser()
		var picked string
		tt := ui.NewTester(func(c *ui.Context) { View(c, Model{State: &s, OnPick: func(path string) { picked = path }}) }, size[0], size[1])
		footer, ok := tt.Find("Open")
		if !ok || footer.Y+footer.H > float32(size[1]) {
			t.Fatalf("Open clipped at %v: %+v", size, footer)
		}
		cancel, ok := tt.Find("Cancel")
		if !ok || cancel.Y+cancel.H > float32(size[1]) {
			t.Fatalf("Cancel clipped at %v: %+v", size, cancel)
		}
		right, ok := tt.Find("Child 00")
		if !ok || right.X < 0 || right.X+right.W > float32(size[0]) {
			t.Fatalf("child pane not scrolled into narrow viewport: %+v", right)
		}
		if size[0] == 680 {
			left, ok := tt.Find("Project root")
			if !ok || math.Abs(float64(left.W-right.W)) > 1 || left.W < 248 || right.X <= left.X+left.W {
				t.Fatalf("Miller columns not equal around divider: left=%+v right=%+v", left, right)
			}
		}
		tt.Scroll(right.X+right.W/2, right.Y+right.H/2, 0, 2000)
		last, ok := tt.Find("Child 39")
		if !ok || last.Y+last.H > footer.Y {
			t.Fatalf("child list cannot scroll independently: %+v footer=%+v", last, footer)
		}
		after, _ := tt.Find("Open")
		if after != footer {
			t.Fatal("scroll moved pinned footer")
		}
		if err := tt.Click("Open"); err != nil {
			t.Fatal(err)
		}
		if picked != "/home/project" {
			t.Fatalf("Open did not adopt selection: %q", picked)
		}
		name := "directory-picker-miller-wide.png"
		if size[0] < 680 {
			name = "directory-picker-miller-short.png"
		}
		saveSecondarySnapshot(t, tt, name)
	}
}

func TestHiddenToggleSelectionAndPrefixFiltering(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	s := fixtureBrowser()
	s.Selected = nil
	s.Child = nil
	var selected string
	m := Model{State: &s, OnSelect: func(entry api.DirectoryListingCrumbsItem, right bool) {
		if right {
			t.Fatal("solo pane reported right pick")
		}
		selected = entry.Path
	}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 680, 500)
	if tt.HasText(".private") {
		t.Fatal("hidden entry visible by default")
	}
	if err := tt.Click("Show hidden files"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(".private") {
		t.Fatal("hidden toggle did not reveal Host-flagged entry")
	}
	if err := tt.Click(".private"); err != nil {
		t.Fatal(err)
	}
	if selected != "/home/.private" {
		t.Fatal("row pick changed Host path")
	}
	if err := tt.Click("Show hidden files"); err != nil {
		t.Fatal(err)
	}
	draft := "/home/.pr"
	s.PathDraft = &draft
	tt.Frame()
	if !tt.HasText(".private") || tt.HasText("Other folder") {
		t.Fatal("dot-led prefix did not reveal/filter hidden row")
	}
	draft = "/home/no-match"
	tt.Frame()
	if tt.HasText(".private") || !tt.HasText("Other folder") {
		t.Fatal("prefix miss did not restore ordinary visibility")
	}
	s = fixtureBrowser()
	draft = "/home/project/Child 03"
	s.PathDraft = &draft
	tt.Frame()
	if !tt.HasText("Project root") || !tt.HasText("Other folder") || !tt.HasText("Child 03") || tt.HasText("Child 02") {
		t.Fatal("prefix filtered stale left pane or failed to filter final pane")
	}
}

func TestNestedCreateRawNameAndParentInertness(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	s := fixtureBrowser()
	var parent, name string
	closed, navigated, picked := 0, 0, 0
	m := Model{State: &s, OnCreate: func(p, n string) { parent, name = p, n }, OnClose: func() { closed++ }, OnNavigate: func(string) { navigated++ }, OnPick: func(string) { picked++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 680, 500)
	if err := tt.Click("New folder"); err != nil {
		t.Fatal(err)
	}
	if s.FolderDraft == nil || !tt.HasText("Folder name") {
		t.Fatal("nested create dialog did not open")
	}
	input, ok := tt.Find("Folder name")
	if !ok || math.Abs(float64(input.H-44)) > 1 || input.W > 332 {
		t.Fatalf("nested input geometry differs: %+v", input)
	}
	if err := tt.Click("Folder name"); err != nil {
		t.Fatal(err)
	}
	tt.Type(" child with spaces ")
	tt.Key(0, ui.KeyEnter)
	if parent != "/home/project" || name != " child with spaces " {
		t.Fatalf("create trimmed or retargeted input: parent=%q name=%q", parent, name)
	}
	s.Creating = true
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if s.FolderDraft == nil {
		t.Fatal("Escape dismissed in-flight create")
	}
	for _, label := range []string{"Home", "Open"} {
		_ = tt.Click(label)
	}
	if closed+navigated+picked != 0 {
		t.Fatal("nested create left parent controls active")
	}
	settleDialogReveal(tt)
	saveSecondarySnapshot(t, tt, "directory-picker-create.png")
	s.Creating = false
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if s.FolderDraft != nil {
		t.Fatal("Escape did not cancel editable nested dialog")
	}
}

func TestLoadingCanBeSupersededButCannotAdoptOrCreate(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	s := fixtureBrowser()
	s.SlowScan = true
	selected, navigated, picked, closed := 0, 0, 0, 0
	m := Model{State: &s, Loading: true, OnNavigate: func(string) { navigated++ }, OnSelect: func(api.DirectoryListingCrumbsItem, bool) { selected++ }, OnPick: func(string) { picked++ }, OnClose: func() { closed++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 680, 500)
	if !tt.HasText("Loading\u2026") {
		t.Fatal("missing delayed floating loading indicator")
	}
	_ = tt.Click("Home")
	_ = tt.Click("Child 00")
	_ = tt.Click("Open")
	_ = tt.Click("New folder")
	_ = tt.Click("Cancel")
	if navigated != 1 || selected != 1 || picked != 0 || closed != 1 || s.FolderDraft != nil {
		t.Fatalf("loading policy incorrect: nav=%d select=%d pick=%d close=%d", navigated, selected, picked, closed)
	}
	m.Loading = false
	m.Busy = true
	tt.Frame()
	_ = tt.Click("Home")
	_ = tt.Click("Open")
	if navigated != 1 || picked != 0 {
		t.Fatal("owner busy did not freeze parent")
	}
}

func TestDirectoryErrorToastRetryAndEmptyData(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	s := fixtureBrowser()
	retries := 0
	m := Model{State: &s, Err: "Host denied this scan", OnRetry: func() { retries++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 680, 500)
	if !tt.HasText(m.Err) || !tt.HasText("Child 00") {
		t.Fatal("failure lost visible panes or notification")
	}
	if err := tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	if retries != 1 {
		t.Fatal("toast retry did not reach owner")
	}
	s = BrowserState{Parent: &api.DirectoryListing{Path: "/empty"}}
	m.Err = ""
	tt.Frame()
	if !tt.HasText("Open") || !tt.HasText("Edit path") || tt.HasText("Child 00") {
		t.Fatal("empty filesystem level lost browse controls")
	}
	saveSecondarySnapshot(t, tt, "directory-picker-empty.png")
}

// settleDialogReveal lets a nested card finish its entrance, so the frame that
// follows shows the settled surface rather than the first frame of the fade.
func settleDialogReveal(tt *ui.Tester) {
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(2 * time.Second))
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

func TestPathEditorFocusLeavingCardCancelsDraft(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	s := fixtureBrowser()
	draft := "/home/project/"
	s.PathDraft = &draft
	s.FocusPath = true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Box(c).Height(420).FillWidth().Children(func() { View(c, Model{State: &s, OnCancelPath: func() { s.PathDraft = nil }}) })
			ui.Button(c, "Outside card")
		})
	}, 680, 500)
	if err := tt.Click("Outside card"); err != nil {
		t.Fatal(err)
	}
	if s.PathDraft != nil {
		t.Fatal("focus escaping the path editor's card retained draft")
	}
}

func TestFailedPathRetryRetainsFocusButOutsideAndEscapeCancel(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, action := range []string{"Retry", "Tab Retry", "New error Tab Retry", "Tab Retry Escape", "Tab Retry outside", "Outside card", "Escape"} {
		t.Run(action, func(t *testing.T) {
			s := fixtureBrowser()
			draft := "/home/project/missing"
			s.PathDraft, s.FocusPath = &draft, true
			retried, canceled := 0, 0
			retriedOwner := ""
			errText := "Directory is not available"
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					ui.Button(c, "Outside card").Height(32)
					ui.Box(c).Height(420).FillWidth().Children(func() {
						owner := errText
						View(c, Model{State: &s, Err: errText,
							OnRetry:      func() { retried++; retriedOwner = owner },
							OnCancelPath: func() { canceled++; s.PathDraft = nil; errText = "" },
						})
					})
				})
			}, 680, 500)
			if action == "New error Tab Retry" {
				// A replacement dialog error must replace the old action owner.
				s = fixtureBrowser()
				draft, errText = "/home/project/new-missing", "Replacement directory failure"
				s.PathDraft, s.FocusPath = &draft, true
				tt.Frame()
			}
			if action == "Tab Retry" || action == "New error Tab Retry" || action == "Tab Retry Escape" || action == "Tab Retry outside" {
				for n := 0; n < 128 && !tt.Focused("Retry") && s.PathDraft != nil; n++ {
					tt.Key(0, ui.KeyTab)
				}
				if !tt.Focused("Retry") {
					t.Fatalf("Tab could not reach owned retry: canceled=%d draft=%v", canceled, s.PathDraft)
				}
				if action == "Tab Retry Escape" {
					tt.Key(0, ui.KeyEscape)
				} else if action == "Tab Retry outside" {
					if err := tt.Click("Outside card"); err != nil {
						t.Fatal(err)
					}
				} else {
					tt.Key(0, ui.KeyEnter)
				}
			} else if action == "Escape" {
				tt.Key(0, ui.KeyEscape)
			} else if err := tt.Click(action); err != nil {
				t.Fatal(err)
			}
			if action == "Retry" || action == "Tab Retry" || action == "New error Tab Retry" {
				if retried != 1 || retriedOwner != errText || canceled != 0 || s.PathDraft == nil || *s.PathDraft != draft || !tt.Focused("Edit path") {
					t.Fatalf("recovery pointer canceled draft: retry=%d cancel=%d draft=%v", retried, canceled, s.PathDraft)
				}
			} else if canceled != 1 || retried != 0 || s.PathDraft != nil {
				t.Fatalf("genuine cancellation was suppressed: retry=%d cancel=%d draft=%v", retried, canceled, s.PathDraft)
			}
		})
	}
}
