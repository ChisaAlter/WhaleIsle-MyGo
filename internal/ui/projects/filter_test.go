package projects

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

func TestProjectSearchHistory(t *testing.T) {
	items := []projectflow.Project{
		{ID: "ready", Title: "Alpha Project", Lifecycle: "ready", CoordinatorSessionID: "lead-active", CanonicalWorkingDirectory: "/fixture/not-a-title"},
		{ID: "archived", Title: "ALPHA Archive", Lifecycle: "archived", CoordinatorSessionID: "lead-archive"},
		{ID: "pending", Title: "Éclair 项目", Lifecycle: "provisioning", Archiving: true},
		{ID: "case", Title: "Gamma", Lifecycle: "Archived"},
		{ID: "empty", Title: "", Lifecycle: ""},
	}
	before := append([]projectflow.Project(nil), items...)
	for _, tc := range []struct {
		name, query string
		history     bool
		want        []string
	}{
		{"default excludes archived only", "", false, []string{"ready", "pending", "case", "empty"}},
		{"history is exact lifecycle", "", true, []string{"archived"}},
		{"title case insensitive", "aLpHa", false, []string{"ready"}},
		{"history and title intersect", "aLpHa", true, []string{"archived"}},
		{"query is not trimmed", " Alpha ", false, nil},
		{"spaces are literal", " ", false, []string{"ready", "pending"}},
		{"unicode title", "éCLAIR", false, []string{"pending"}},
		{"chinese title", "项目", false, []string{"pending"}},
		{"no unicode normalization", "E\u0301clair", false, nil},
		{"not directory", "not-a-title", false, nil},
		{"not identifier", "ready", false, nil},
		{"no result", "missing", true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, p := range filteredProjects(items, tc.query, tc.history) {
				got = append(got, p.ID)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	if !reflect.DeepEqual(items, before) {
		t.Fatal("filter changed the authoritative listing")
	}
	if got := filteredProjects(nil, "", false); len(got) != 0 {
		t.Fatal("empty input produced rows")
	}
	filtered := filteredProjects(items, "alpha", false)
	filtered[0].Lifecycle = "archived"
	if !reflect.DeepEqual(items, before) {
		t.Fatal("filtered rows alias the authoritative listing")
	}

	t.Run("native controls preserve selection and actions", func(t *testing.T) {
		locale := i18n.Locale()
		i18n.SetLocale(i18n.LocaleEN)
		defer i18n.SetLocale(locale)
		mutations, refreshes, opens := 0, 0, 0
		m := Model{Expanded: true, Loaded: true, Listing: projectflow.Listing{Available: true, Projects: append([]projectflow.Project(nil), items[:2]...)}, SelectedSession: "lead-active", Workspace: "/fixture/workspace"}
		m.OnSearchQuery = func(q string) { m.SearchQuery = q }
		m.OnHistory = func(v bool) { m.History = v }
		m.OnToggle = func() { m.Expanded = !m.Expanded }
		m.OnCreate = func() { mutations++ }
		m.OnArchiveProject = func(string) { mutations++ }
		m.OnRestoreProject = func(string) { mutations++ }
		m.OnRename = func(string, string) { mutations++ }
		m.OnOpen = func(id string) {
			if id != "ready" {
				t.Fatalf("opened wrong ID: %s", id)
			}
			opens++
		}
		m.OnRefresh = func() { refreshes++ }
		tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 1200, 1600)
		if !tt.HasText("Alpha Project") || tt.HasText("ALPHA Archive") {
			t.Fatal("default filter incorrect", tt.Texts())
		}
		if !tt.HasText("Add Project…") {
			t.Fatal("directory authorization entry missing")
		}
		if err := tt.RightClick("Alpha Project"); err != nil {
			t.Fatal(err)
		}
		if len(tt.Menu()) < 3 || !reflect.DeepEqual(tt.Menu()[:3], []string{"Rename", "Open project directory", "Archive Project"}) {
			t.Fatal("project lifecycle actions missing", tt.Menu())
		}
		tt.CloseMenu()
		if err := tt.Click("Alpha Project"); err != nil {
			t.Fatal(err)
		}
		if opens != 1 {
			t.Fatal("existing open action lost")
		}
		chooseOption := func(label string) {
			t.Helper()
			if err := tt.Click("Project options"); err != nil {
				t.Fatal(err)
			}
			if err := tt.ChooseMenuItem(label); err != nil {
				t.Fatal(err)
			}
		}
		chooseOption("Show archived projects")
		if !m.History || !tt.HasText("ALPHA Archive") || tt.HasText("Alpha Project") {
			t.Fatal("history filter incorrect", tt.Texts())
		}
		if err := tt.RightClick("ALPHA Archive"); err != nil {
			t.Fatal(err)
		}
		if got := tt.Menu(); len(got) < 3 || got[2] != "Restore Project" {
			t.Fatal("restore action missing", got)
		}
		tt.CloseMenu()
		if err := tt.Click("Search Project titles"); err != nil {
			t.Fatal(err)
		}
		tt.Type("aLpHa")
		if m.SearchQuery != "aLpHa" || !tt.HasText("ALPHA Archive") {
			t.Fatal("query callback or title matching failed")
		}
		tt.Type(" missing")
		if !tt.HasText("No matching Projects") || tt.HasText("ALPHA Archive") {
			t.Fatal("no-match state incorrect", tt.Texts())
		}
		if err := tt.Click("Clear search"); err != nil {
			t.Fatal(err)
		}
		if m.SearchQuery != "" || !tt.HasText("ALPHA Archive") {
			t.Fatal("clear did not restore rows")
		}
		m.SearchQuery = "ALPHA"
		m.Busy = true
		tt.Frame()
		chooseOption("Show unarchived projects")
		if m.History || m.SearchQuery != "ALPHA" || !tt.HasText("Alpha Project") {
			t.Fatal("local filters blocked or reset during refresh")
		}
		m.Busy = false
		tt.Frame()
		chooseOption("Refresh")
		if refreshes != 1 || m.History || m.SearchQuery != "ALPHA" {
			t.Fatal("refresh changed filters")
		}
		m.Listing.Projects[0].Title = "Renamed after refresh"
		tt.Frame()
		if !tt.HasText("No matching Projects") {
			t.Fatal("new listing was not filtered")
		}
		if err := tt.Click("Clear search"); err != nil {
			t.Fatal(err)
		}
		if !tt.HasText("Renamed after refresh") || m.SelectedSession != "lead-active" {
			t.Fatal("filter changed selection")
		}
		m.History = true
		m.Listing.Projects = m.Listing.Projects[:1]
		tt.Frame()
		if !tt.HasText("No archived Projects") {
			t.Fatal("empty history mislabeled")
		}
		m.Listing.Projects[0].Lifecycle = "archived"
		m.History = false
		tt.Frame()
		if !tt.HasText("No unarchived Projects") {
			t.Fatal("empty active category mislabeled")
		}
		m.Listing.Projects = nil
		tt.Frame()
		if !tt.HasText("No Projects yet") {
			t.Fatal("empty listing mislabeled")
		}
		m.Listing.Available = false
		tt.Frame()
		if tt.HasText("No Projects yet") || tt.HasText("No matching Projects") {
			t.Fatal("unavailable listing reported as empty")
		}
		if mutations != 0 || refreshes != 1 || opens != 1 || m.SelectedSession != "lead-active" {
			t.Fatal("filters dispatched unrelated actions", mutations, refreshes, opens)
		}
	})
}
