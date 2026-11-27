package sidebar

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSidebarHoverFocusZeroShiftSweep(t *testing.T) {
	prevLocale := i18n.Locale()
	t.Cleanup(func() {
		i18n.SetLocale(prevLocale)
		design.SetGlass(false, nil)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
	})
	i18n.SetLocale(i18n.LocaleZH)
	design.SetGlass(false, nil)
	chrome.ConfigureBackdrop("", wallpapers.Defaults())

	for _, dark := range []bool{false, true} {
		themeID := theme.IDDeepseekLight
		if dark {
			themeID = theme.IDDeepseekDark
		}
		t.Run(fmt.Sprintf("full/dark%v", dark), func(t *testing.T) {
			query := ""
			archiveExpanded := true
			m := Model{
				ThemeID:            themeID,
				SystemDark:         dark,
				ActivePanel:        "plugins",
				NewSessionShortcut: "Ctrl+N",
				SearchQuery:        &query,
				archiveExpanded:    &archiveExpanded,
				OnToggle:           func() {},
				OnCreate:           func() {},
				OnOpenMarket:       func() {},
				OnOpenSchedule:     func() {},
				OnOpenAssistant:    func() {},
				OnSelect:           func(string) {},
				OnArchive:          func(string) {},
				OnUnarchive:        func(string) {},
				OnDelete:           func(string) {},
				OnPin:              func(string) {},
				OnBatchEdit:        func(string, []session.SessionBatchTarget) {},
				Snapshot: session.ViewState{
					WorkspaceReady: true,
					SelectedID:     "sess-1",
					Workspaces: []session.WorkspaceRow{
						{WorkspaceID: "ws-1", Title: "WhaleIsle", Path: "/ws/whaleisle", SessionIDs: []string{"sess-1", "sess-2"}},
					},
					Items: []session.Item{
						{SessionID: "sess-1", Title: "Active session with long title for zero shift sweep", Cwd: "/ws/whaleisle", UpdatedAt: 1700000000000},
						{SessionID: "sess-2", Title: "Idle session row", Cwd: "/ws/whaleisle", UpdatedAt: 1700000000000},
						{SessionID: "arch-1", Title: "Archived session row", Cwd: "/ws/whaleisle", UpdatedAt: 1700000000000},
					},
					ArchivedSessionIDs: []string{"arch-1"},
				},
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 800)
			if dark {
				tt.SetDark(true)
			}
			tt.Frame()
			tt.Frame()
			ui.AssertHoverFocusZeroShift(t, tt)
		})

		t.Run(fmt.Sprintf("search-rows/dark%v", dark), func(t *testing.T) {
			searchM := Model{
				ThemeID:    themeID,
				SystemDark: dark,
				OnSelect:   func(string) {},
				OnArchive:  func(string) {},
				OnDelete:   func(string) {},
				Snapshot:   session.ViewState{SelectedID: "search-sel"},
			}
			tt := ui.NewTester(func(c *ui.Context) {
				defer withSidebarTheme(c, searchM)()
				tk := tok(c)
				ui.Column(c).Fill().Background(tk.Sidebar).Padding(8).Gap(4).Children(func() {
					renderSearchResultRow(c, searchM, sessionSearchRow{
						Item:      session.Item{SessionID: "search-sel", Title: "Selected Search Result"},
						Workspace: "proj",
					}, tk.Foreground, tk.MutedForeground)
					renderSearchResultRow(c, searchM, sessionSearchRow{
						Item:      session.Item{SessionID: "search-idle", Title: "Idle Search Result"},
						Workspace: "proj",
					}, tk.Foreground, tk.MutedForeground)
				})
			}, 280, 180)
			if dark {
				tt.SetDark(true)
			}
			tt.Frame()
			tt.Frame()
			ui.AssertHoverFocusZeroShift(t, tt)
		})
	}
}
