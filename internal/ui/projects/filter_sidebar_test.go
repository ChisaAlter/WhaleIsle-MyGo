package projects_test

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/sidebar"
)

// Fixed WorkspaceBrowser.tsx:1684-1705 mounts Projects after Session results and
// passes its shared query. dsh-project client.js:529,533-543 owns the lifecycle
// and history menus; neither a separate title input nor Refresh is in that menu.
func TestProjectSearchHistorySharedSidebarControls(t *testing.T) {
	// The sidebar Project section is commented out in sidebar.View while the
	// feature is reworked; un-skip when the two projects.View calls return.
	t.Skip("sidebar projects.View disabled")
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(locale)
	mutations, refreshes, opens := 0, 0, 0
	query := ""
	m := projects.Model{Expanded: true, Loaded: true, Listing: projectflow.Listing{Available: true, Projects: []projectflow.Project{
		{ID: "ready", Title: "Alpha Project", Lifecycle: "ready", CoordinatorSessionID: "lead-active", CanonicalWorkingDirectory: "/fixture/not-a-title"},
		{ID: "archived", Title: "ALPHA Archive", Lifecycle: "archived", CoordinatorSessionID: "lead-archive"},
	}}, SelectedSession: "lead-active", Workspace: "/fixture/workspace"}
	m.OnHistory = func(v bool) { m.History = v }
	m.OnToggle = func() { m.Expanded = !m.Expanded }
	m.OnCreate = func() { mutations++ }
	m.OnArchiveProject = func(string) { mutations++ }
	m.OnRestoreProject = func(string) { mutations++ }
	m.OnRename = func(string, string) { mutations++ }
	m.OnRefresh = func() { refreshes++ }
	m.OnOpen = func(id string) {
		if id != "ready" {
			t.Fatalf("opened wrong ID: %s", id)
		}
		opens++
	}
	tt := ui.NewTester(func(c *ui.Context) {
		// Same owner projection as frame.navigationView; actual input stays in Sidebar.
		m.SearchQuery = query
		sidebar.View(c, sidebar.Model{Projects: m, SearchQuery: &query, Snapshot: session.ViewState{SelectedID: "lead-active"}, ThemeID: "deepseek.light"})
	}, 1200, 1600)
	if !tt.HasText("Alpha Project") || tt.HasText("ALPHA Archive") || !tt.HasText("Add Project…") {
		t.Fatal("default category/open entry incorrect", tt.Texts())
	}
	if err := tt.RightClick("Alpha Project"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tt.Menu(), []string{"Rename", "Open project directory", "Archive Project"}) {
		t.Fatal("lifecycle menu incorrect", tt.Menu())
	}
	tt.CloseMenu()
	if err := tt.Click("Alpha Project"); err != nil {
		t.Fatal(err)
	}
	if opens != 1 {
		t.Fatal("existing open action lost")
	}
	chooseHistory := func(label string) {
		t.Helper()
		if err := tt.Click("Project options"); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tt.Menu(), []string{label}) {
			t.Fatal("category menu added unsupported actions", tt.Menu())
		}
		if err := tt.ChooseMenuItem(label); err != nil {
			t.Fatal(err)
		}
	}
	chooseHistory("Show archived projects")
	if !m.History || !tt.HasText("ALPHA Archive") || tt.HasText("Alpha Project") {
		t.Fatal("history filter incorrect", tt.Texts())
	}
	if err := tt.RightClick("ALPHA Archive"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Menu(); len(got) != 3 || got[2] != "Restore Project" {
		t.Fatal("restore action missing", got)
	}
	tt.CloseMenu()
	if err := tt.Click(i18n.T(i18n.KeySidebarSearch)); err != nil {
		t.Fatal(err)
	}
	tt.Type("aLpHa")
	if query != "aLpHa" || !tt.HasText("ALPHA Archive") {
		t.Fatal("shared query did not reach Project category", tt.Texts())
	}
	tt.Type(" missing")
	if !tt.HasText("No matching Projects") || tt.HasText("ALPHA Archive") {
		t.Fatal("no-match state incorrect", tt.Texts())
	}
	if err := tt.Click("Clear search"); err != nil {
		t.Fatal(err)
	}
	if query != "" || !tt.HasText("ALPHA Archive") {
		t.Fatal("clear did not restore category")
	}
	if err := tt.Click(i18n.T(i18n.KeySidebarSearch)); err != nil {
		t.Fatal(err)
	}
	tt.Type("ALPHA")
	m.Busy = true
	tt.Frame()
	chooseHistory("Show unarchived projects")
	if m.History || query != "ALPHA" || !tt.HasText("Alpha Project") {
		t.Fatal("local filters blocked/reset during refresh")
	}
	m.Busy = false
	m.Listing.Projects[0].Title = "Renamed after refresh"
	tt.Frame()
	if query != "ALPHA" || !tt.HasText("No matching Projects") {
		t.Fatal("new authoritative listing lost active filter")
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
	if !tt.HasText("Add a project directory to start a continuing conversation.") {
		t.Fatal("empty current category lacks source Add Project card")
	}
	m.Listing.Projects = nil
	tt.Frame()
	if !tt.HasText("Add a project directory to start a continuing conversation.") {
		t.Fatal("empty listing lacks directory authorization entry")
	}
	m.Listing.Available = false
	tt.Frame()
	if !tt.HasText("Add a project directory to start a continuing conversation.") {
		t.Fatal("unavailable Project category lacks the reference guidance")
	}
	if tt.HasText("No matching Projects") {
		t.Fatal("unavailable Project category reported as an empty result")
	}
	if mutations != 0 || refreshes != 0 || opens != 1 || m.SelectedSession != "lead-active" {
		t.Fatal("filters dispatched unrelated actions", mutations, refreshes, opens)
	}
}
