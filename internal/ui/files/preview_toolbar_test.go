package files

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/i18n"
)

func toolbarFixture(t *testing.T) (Model, *FileTab) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	m, tab := resourceModel()
	m.Editing = &tab.Editing
	m.Preview, m.EditDraft = &tab.Preview, &tab.Draft
	m.OnSaveEdit = func(string, string) error { return nil }
	m.WorkspaceRoot = t.TempDir()
	m.Tabs.Workspaces = nil
	tabsFor(m).Open, tabsFor(m).Active = []*FileTab{tab}, tab.Path
	return m, tab
}

func toolbarTester(m *Model, width float32) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintPreviewToolbar(c, *m, *m.SelectedFile, c.Theme().TextMuted, c.Theme().Border)
			ui.Box(c).Label("Resource body").Fill().MinHeight(0)
		})
	}, int(width), 300)
}

func TestPreviewToolbarLiteralSingleRowAndBranches(t *testing.T) {
	m, tab := toolbarFixture(t)
	for _, desktop := range []bool{true, false} {
		for _, width := range []float32{360, 451, 676} {
			m.SessionID = "session"
			if !desktop {
				m.SessionID = ""
			}
			m.WorkspaceRoot += strings.Repeat("long-workspace", 12)
			tt := toolbarTester(&m, width)
			header, ok := tt.Find("File preview toolbar")
			want := float32(38)
			if desktop {
				want = 36
			}
			body, _ := tt.Find("Resource body")
			if !ok || header.H != want || body.Y != header.Y+want {
				t.Fatalf("branch=%v width=%v header=%+v body=%+v", desktop, width, header, body)
			}
			floating, ok := tt.Find("Preview in separate window")
			if !ok || floating.W != 28 || floating.H != 28 || floating.X+floating.W > width {
				t.Fatalf("floating displaced: %+v", floating)
			}
			for _, caption := range []string{"Copy absolute path", "Copy relative path", "Copy draft", "Choose viewer", "Go to line", "Auto-refresh this file"} {
				if _, ok := tt.Find(caption); ok {
					t.Fatalf("permanent extra action %q", caption)
				}
			}
			before := desktopFileToolbar(m, tab.Path)
			tab.Editing = !tab.Editing
			if before != desktopFileToolbar(m, tab.Path) {
				t.Fatal("editor mode changed component identity")
			}
		}
	}
	for _, path := range []string{"a.docx", "a.XLSX", "/absolute.txt", "C:/absolute.txt", "../escape.txt"} {
		m.SessionID = "session"
		if desktopFileToolbar(m, path) {
			t.Fatalf("desktop file accepted %q", path)
		}
	}
}

func TestPreviewToolbarContextKeepsDraftWrapAndNavigation(t *testing.T) {
	m, tab := toolbarFixture(t)
	tab.Draft, tab.Editing = "draft\nsecond\nthird", true
	tt := toolbarTester(&m, 676)
	if err := tt.Click("Word wrap"); err != nil {
		t.Fatal(err)
	}
	if !previewPreferences(m).WordWrap || tab.Draft != "draft\nsecond\nthird" {
		t.Fatal("wrap changed draft")
	}
	if err := tt.RightClick("File preview toolbar"); err != nil {
		t.Fatal(err)
	}
	for _, caption := range []string{"Copy absolute path", "Copy relative path", "Copy draft", "Keep draft and preview", "Reload", "Auto-refresh this file", "Go to line…"} {
		if !slices.Contains(tt.Menu(), caption) {
			t.Fatalf("missing context action %s: %v", caption, tt.Menu())
		}
	}
	if err := tt.ChooseMenuItem("Keep draft and preview"); err != nil {
		t.Fatal(err)
	}
	if tab.Editing || tab.Draft != "draft\nsecond\nthird" || tab.Preview != "saved" {
		t.Fatal("preview discarded draft")
	}
	if _, ok := tt.Find(i18n.T(i18n.KeyFilesSave)); !ok {
		t.Fatal("dirty desktop draft lost Save after Keep draft")
	}
	if err := tt.RightClick("File preview toolbar"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Auto-refresh this file"); err != nil {
		t.Fatal(err)
	}
	if !tab.AutoRefreshDisabled {
		t.Fatal("checked context action did not persist preference")
	}
	if err := tt.RightClick("File preview toolbar"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Go to line…"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Line number"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("2")
	tt.Key(0, ui.KeyEscape)
	if tab.revealLine != 0 {
		t.Fatal("cancel navigated")
	}
	if err := tt.RightClick("File preview toolbar"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Go to line…"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Line number"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("2")
	tt.Key(0, ui.KeyEnter)
	if tab.revealLine != 2 || tab.Draft != "draft\nsecond\nthird" {
		t.Fatal("line navigation changed text or lost target")
	}
}

func TestPreviewToolbarLateMenuCannotMutateReplacementOwner(t *testing.T) {
	for _, change := range []string{"tab", "host", "read-generation"} {
		for _, action := range []string{"Keep draft and preview", "Reload", "Save"} {
			t.Run(change+"/"+action, func(t *testing.T) {
				m, tab := toolbarFixture(t)
				m.SessionID = "" // Readonly-component native editor mode keeps Save in context.
				tab.Editing, tab.Draft = true, "owned draft"
				tt := toolbarTester(&m, 676)
				if err := tt.RightClick("File preview toolbar"); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "tab":
					other := &FileTab{Path: "other.txt", Preview: "other saved", Draft: "other draft", Editing: true}
					tabsFor(m).Open = append(tabsFor(m).Open, other)
					tabsFor(m).Active = other.Path
					*m.SelectedFile = other.Path
					m.Preview, m.EditDraft, m.Editing = &other.Preview, &other.Draft, &other.Editing
				case "host":
					m.BindingID = "replacement-host"
				case "read-generation":
					tab.readID++
				}
				tt.Frame()
				if err := tt.ChooseMenuItem(action); err != nil {
					t.Fatal(err)
				}
				active := selectedTab(m)
				if !*m.Editing || *m.EditDraft == "" || tab.Draft != "owned draft" || active.save != nil || active.readLoading {
					t.Fatal("old menu mutated replacement resource or dispatched save/read")
				}
			})
		}
	}
}
func TestPreviewToolbarBlockedSaveAndViewerInitialization(t *testing.T) {
	m, tab := toolbarFixture(t)
	tab.Draft, tab.Editing, tab.readBlocked = "unsaved", true, true
	writes := 0
	m.OnSaveEdit = func(string, string) error { writes++; return nil }
	tt := toolbarTester(&m, 676)
	if err := tt.Click(i18n.T(i18n.KeyFilesSave)); err != nil {
		t.Fatal(err)
	}
	if writes != 0 || tab.Draft != "unsaved" || tab.Preview != "saved" {
		t.Fatal("blocked save mutated file")
	}
	tab.Editing, tab.readBlocked = false, false
	m.SessionID = ""
	opens := 0
	m.OpenDocumentViewer = func(string) (DocumentSurface, error) { opens++; return &fixtureDocumentSurface{}, nil }
	tt.Frame()
	tt.Frame()
	if opens != 1 || tab.document.surface == nil {
		t.Fatal("compact chooser lost lazy initialization")
	}
}

func TestPreviewTreeLateMenuCannotDispatchAfterTabOrHostChange(t *testing.T) {
	for _, change := range []string{"tab", "host", "read-generation"} {
		m, tab := toolbarFixture(t)
		tab.Editing, tab.Draft = true, "owned draft"
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				rowFile(c, m, tab.Path, tab.Path, 0, c.Theme().Text, c.Theme().TextMuted, c.Theme().Accent)
			})
		}, 451, 200)
		if err := tt.RightClick("File " + tab.Path); err != nil {
			t.Fatal(err)
		}
		switch change {
		case "tab":
			*m.SelectedFile = "other.txt"
		case "host":
			m.BindingID = "replacement-host"
		case "read-generation":
			tab.readID++
		}
		tt.Frame()
		if err := tt.ChooseMenuItem("Keep draft and preview"); err != nil {
			t.Fatal(err)
		}
		if !tab.Editing || tab.Draft != "owned draft" || tab.save != nil || tab.readLoading {
			t.Fatalf("%s: stale tree menu mutated resource", change)
		}
	}
}

func TestPreviewToolbarCompactOpenTarget(t *testing.T) {
	m, _ := toolbarFixture(t)
	m.SessionID = ""
	m.OnOpenDoc = func(string, workspacefs.DocKind, bool) {}
	m.OnReveal = func(string) {}
	tt := toolbarTester(&m, 451)
	primary, mainOK := tt.Find(i18n.T(i18n.KeyFilesOpenExternal))
	more, moreOK := tt.Find("More open options")
	if !mainOK || !moreOK || primary.W != 23 || more.W != 17 || primary.H != 23 || more.H != 23 || more.X != primary.X+primary.W {
		t.Fatalf("compact split geometry: primary=%+v more=%+v", primary, more)
	}
	if err := tt.Click("More open options"); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(tt.Menu(), i18n.T(i18n.KeyFilesReveal)) {
		t.Fatal("compact menu lost reveal")
	}
}
