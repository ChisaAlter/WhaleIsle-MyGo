package sidebar

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

func TestManyProjectsKeepSettingsFooterInsideSidebar(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	rows := make([]projectflow.Project, 40)
	for i := range rows {
		rows[i] = projectflow.Project{ID: fmt.Sprint(i), Title: fmt.Sprintf("Project %02d", i), Lifecycle: "ready"}
	}
	for _, height := range []int{500, 800} {
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, Model{Projects: projects.Model{Expanded: true, Loaded: true, Workspace: "C:/fixture", Listing: projectflow.Listing{Available: true, Projects: rows}}, OnOpenSettings: func() {}})
		}, 280, height)
		footer, found := tt.Find(i18n.T(i18n.KeySettingsAccountSeat))
		if !found || footer.H <= 0 || footer.Y < 0 || footer.Y+footer.H > float32(height) {
			t.Fatalf("%d: project rows displaced the pinned settings control: %+v", height, footer)
		}
	}
}
