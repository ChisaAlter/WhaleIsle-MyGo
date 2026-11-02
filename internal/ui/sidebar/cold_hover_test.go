package sidebar

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

// This fixture uses only the in-memory UI engine and a fixed frame clock. It
// never creates a native window or uses the OS pointer, focus or clipboard.
func coldHoverFixture(t *testing.T, family string) (*ui.Tester, time.Time, string, time.Duration) {
	t.Helper()
	start := time.Date(2026, 10, 28, 2, 0, 0, 0, time.UTC)
	m := Model{Snapshot: session.ViewState{WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 91}}}
	preview, delay := "Session preview: cold", sessionHoverDelay
	if family == "workspace" {
		preview, delay = "Workspace preview: cold", workspaceHoverDelay
	} else if family == "archived" {
		m.Snapshot.ArchivedSessionIDs = []string{"cold"}
		delay = archivedSessionHoverDelay
	}
	tt := ui.NewTester(func(c *ui.Context) {
		root := ui.Column(c).Width(280).FillHeight()
		m = beginSessionHover(root, m)
		root.Children(func() {
			list := ui.Column(c).FillWidth().Height(240)
			list.Children(func() {
				row := ui.ButtonBase(c.Key("cold-hover-row")).Label("Cold hover row").FillWidth().Height(32)
				row.Children(func() { ui.Text(c, "Cold hover title") })
				item := session.Item{SessionID: "cold", Title: "Cold hover title"}
				switch family {
				case "workspace":
					state := workspaceHoverGroup(c, list, m)
					registerWorkspaceHover(c, m, list, state, []workspaceHoverAnchor{{
						row: row, workspace: session.WorkspaceRow{WorkspaceID: "cold", Title: "Cold workspace", Path: "/known/path"}, label: "Cold workspace",
					}})
				case "archived":
					archivedSessionHoverCard(c, m, row, item, item.Title)
				default:
					sessionHoverCard(c, m, row, item, item.Title, "", theme.Colors{})
				}
			})
		})
		renderSessionHover(c, m, root, false)
	}, 760, 400)
	tt.SetTime(start)
	return tt, start, preview, delay
}

func TestColdHoverReentryRequiresFreshDelay(t *testing.T) {
	for _, family := range []string{"ordinary", "workspace", "archived"} {
		for _, leave := range []string{"outside root", "inside blank group"} {
			t.Run(family+"/"+leave, func(t *testing.T) {
				tt, start, preview, delay := coldHoverFixture(t, family)
				row, found := tt.Find("Cold hover row")
				if !found {
					t.Fatal("missing hover anchor")
				}
				x, y := row.X+50, row.Y+row.H/2
				tt.Move(x, y)
				tt.SetTime(start.Add(400 * time.Millisecond))
				if tt.HasText(preview) {
					t.Fatal("cold preview opened before its initial delay")
				}
				if leave == "outside root" {
					tt.Move(700, 350)
				} else {
					tt.Move(100, 180)
				}
				tt.SetTime(start.Add(time.Second))
				tt.Move(x, y)
				if tt.HasText(preview) {
					t.Fatal("cold reentry reused the abandoned candidate's clock")
				}
				tt.SetTime(start.Add(time.Second + delay - time.Millisecond))
				if tt.HasText(preview) {
					t.Fatalf("reentry preview opened before a fresh %v delay", delay)
				}
				tt.SetTime(start.Add(time.Second + delay))
				if !tt.HasText(preview) {
					t.Fatalf("reentry preview missing after a fresh %v delay: %q", delay, tt.Texts())
				}
			})
		}
	}
}
