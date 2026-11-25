package workspacepicker

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestWorkspacePickerListReservesScrollbarGutter(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	snap := session.ViewState{WorkspaceReady: true}
	for i := 0; i < 12; i++ {
		snap.Workspaces = append(snap.Workspaces, session.WorkspaceRow{
			WorkspaceID: fmt.Sprintf("ws-%d", i),
			Title:       fmt.Sprintf("Workspace %02d", i),
			Path:        fmt.Sprintf("/repo/ws-%d", i),
		})
	}
	store := session.New()
	tt := ui.NewTester(func(c *ui.Context) {
		anchor := ui.Button(c, "Chip").Height(32)
		pickMenu(c, Model{
			Store:    store,
			Anchor:   anchor,
			OnChoose: func(uint64, string) {},
			OnBrowse: func() {},
		}, snap, &session.WorkspaceNavigationState{Mode: "pick"}, theme.Resolve("system").Light)
	}, 400, 360)

	rowRect, ok := tt.Find("Workspace 00")
	if !ok {
		t.Fatalf("missing workspace row: %v", tt.Texts())
	}
	footerRect, ok := tt.Find("Add workspace…")
	if !ok {
		t.Fatalf("missing footer action: %v", tt.Texts())
	}
	rightGap := (footerRect.X + footerRect.W) - (rowRect.X + rowRect.W)
	if rightGap < design.Sp(3)-0.5 {
		t.Fatalf("workspace list did not reserve right scrollbar gutter: footerRight=%.1f rowRight=%.1f gap=%.1f < %.1f",
			footerRect.X+footerRect.W, rowRect.X+rowRect.W, rightGap, design.Sp(3))
	}
}
