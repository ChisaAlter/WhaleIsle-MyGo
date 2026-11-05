package sidebar

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/workspacepicker"
)

func TestWorkspaceDirectoryMenuUsesPathLabelAndHidesOnlyRefusedAction(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(locale) })
	for _, path := range []string{"/active", "/new", "/refused"} {
		t.Run(path, func(t *testing.T) {
			selected := ""
			m := Model{DirectoryActions: []workspacepicker.DirectoryAction{
				{ID: "project.create", Run: func(value string) { selected = value }, Visible: func(value string) bool { return value != "/refused" }, LabelForPath: func(value string) string {
					if value == "/active" {
						return "Open Project"
					}
					return "Create Project in this directory"
				}},
				{ID: "legacy", Label: "Inspect directory", Run: func(value string) { selected = value }},
			}}
			group := session.WorkspaceGroup{Key: "row", Kind: "workspace", Workspace: session.WorkspaceRow{Path: path}}
			tt := ui.NewTester(func(c *ui.Context) {
				header := ui.Row(c).Width(280).Height(36)
				header.Children(func() { workspaceRowActions(c, m, header, group, c.Theme().TextMuted) })
			}, 320, 200)
			tt.Move(30, 18)
			if err := tt.Click(fmt.Sprintf(workspaceText("工作区“%s”的操作", "Actions for workspace “%s”"), path)); err != nil {
				t.Fatal(err)
			}
			want := []string{"Inspect directory", "Reveal workspace in Finder"}
			if path == "/active" {
				want = []string{"Open Project", "Inspect directory", "Reveal workspace in Finder"}
			}
			if path == "/new" {
				want = []string{"Create Project in this directory", "Inspect directory", "Reveal workspace in Finder"}
			}
			if !reflect.DeepEqual(tt.Menu(), want) {
				t.Fatalf("menu=%v want=%v", tt.Menu(), want)
			}
			if err := tt.ChooseMenuItem(want[0]); err != nil {
				t.Fatal(err)
			}
			if selected != path {
				t.Fatalf("clicked another directory: %q", selected)
			}
		})
	}
}
