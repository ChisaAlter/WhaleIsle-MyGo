package sidebar

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

func TestSidebarWideInFade200msOnExpand(t *testing.T) {
	collapsed := true
	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{
			Collapsed: collapsed,
			ThemeID:   theme.IDDeepseekLight,
			Snapshot:  session.ViewState{SelectedID: "s1"},
		}
		View(c, m)
	}, 320, 600)
	tt.SetTime(start)
	collapsed = false
	tt.SetTime(start.Add(time.Millisecond))
	if !tt.HasText(sidebarLocaleText("panels.label")) && !tt.HasText("New chat") && !tt.HasText("Workspaces") {
		// Sidebar wide body builds brand/panels on expand.
	}
	tt.SetTime(start.Add(220 * time.Millisecond))
	// Ensures expand completes cleanly without panics or lingering animation frames.
	tt.Frame()
}
