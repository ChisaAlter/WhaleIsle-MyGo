package projects

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

// TestProjectObserverSurfaceSnapshot renders the observer through its real View
// path for evidence/after/surface-project.png. The fixture carries a project, its
// coordinator session, a live queue entry and one delivered request, so the header,
// the section chips and the task list all paint their real contents.
func TestProjectObserverSurfaceSnapshot(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	project := projectflow.Project{
		ID:                        "project-fixture",
		Title:                     "Ship the settings page",
		CanonicalWorkingDirectory: `C:\work\whaleisle`,
		CoordinatorSessionID:      "session-lead",
		Lifecycle:                 "active",
		Queue: &projectflow.RequestQueue{
			Active: &projectflow.RequestCard{ID: "request-live", ProjectID: "project-fixture", Title: "Row layout for settings", Status: "running"},
		},
	}
	project.Activity.Running = 1
	project.Activity.Queued = 1
	base := Model{
		ThemeID:            "deepseek.light",
		Listing:            projectflow.Listing{Available: true, Projects: []projectflow.Project{project}},
		SelectedSession:    "session-lead",
		ObserverHost:       "fixture-host",
		RequestProjectID:   "project-fixture",
		RequestKind:        "current",
		RequestLoaded:      true,
		RequestPageNumber:  1,
		RequestPage:        projectflow.RequestPage{Total: 2},
		ObserverModel:      func(*ObserverState) Model { return Model{} },
		OnObserver:         func(string) {},
		OnRequestActivity:  func() {},
	}
	base.RequestPage.Items = []projectflow.RequestCard{
		{ID: "request-live", ProjectID: "project-fixture", Title: "Row layout for settings", Status: "running", GoalPreview: "Move every setting onto one row."},
		{ID: "request-done", ProjectID: "project-fixture", Title: "Diff gutter", Status: "delivered", GoalPreview: "Pin the line numbers."},
	}
	observer := &ObserverState{
		ProjectID:            "project-fixture",
		CoordinatorSessionID: "session-lead",
		Host:                 "fixture-host",
		Section:              "tasks",
		Initialized:          true,
		RequestModes:         map[string]RequestMode{},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		fixture := base
		ObserverView(c, fixture, observer)
	}, 900, 520)
	dir := os.Getenv("SURFACES_GALLERY_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, "surface-project.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, tt.Image()); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
