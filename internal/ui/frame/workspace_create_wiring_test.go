package frame

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Both the conversation rail and the global-page rail share this routing.
func TestWorkspaceCreateCallbackReachesSharedNavigation(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, shell := range []string{"conversation", "global"} {
		t.Run(shell, func(t *testing.T) {
			store := session.New()
			snap := session.ViewState{
				WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 42},
				Workspaces:      []session.WorkspaceRow{{WorkspaceID: "first", Title: "First workspace"}, {WorkspaceID: "second", Title: "Second workspace"}},
			}
			var got []string
			var generations []uint64
			sizes := DefaultLayout()
			m := Model{Store: store, Sizes: &sizes, ThemeID: theme.IDDeepseekLight, OnCreateWorkspaceSession: func(generation uint64, id string) {
				generations = append(generations, generation)
				got = append(got, id)
			}}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					if shell == "global" {
						sidebarColumns(c, &sizes, func() { navigationView(c, m, snap) }, func() { ui.Text(c, "Global page") })
					} else {
						navigationView(c, m, snap)
					}
				})
			}, 1200, 800)
			for _, row := range []struct{ title, id string }{{"First workspace", "first"}, {"Second workspace", "second"}} {
				r, ok := tt.Find(row.title)
				if !ok {
					t.Fatalf("workspace %q missing: %q", row.title, tt.Texts())
				}
				tt.Move(r.X+r.W/2, r.Y+r.H/2)
				tt.Frame()
				if err := tt.Click(fmt.Sprintf("New session in “%s”", row.title)); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(got, []string{"first", "second"}) || !reflect.DeepEqual(generations, []uint64{42, 42}) {
				t.Fatalf("callbacks ids=%v generations=%v", got, generations)
			}
		})
	}
}
