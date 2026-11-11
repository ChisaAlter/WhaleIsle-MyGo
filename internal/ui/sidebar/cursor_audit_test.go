package sidebar

import (
	"sort"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// TestCursorAuditSidebar renders the sidebar headlessly across expanded,
// collapsed rail, and toast states, asserting that every enabled interactive
// element (excluding text inputs) has ui.CursorPointer.
func TestCursorAuditSidebar(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	noop := func() {}
	noopStr := func(string) {}
	searchQuery := "会话"
	searchFocus := false
	wsMenuOpen := false

	baseSnapshot := session.ViewState{
		WorkspaceReady: true,
		SelectedID:     "a",
		Items: []session.Item{
			{SessionID: "a", Title: "第一个会话", Cwd: "/project"},
			{SessionID: "b", Title: "第二个会话", Cwd: "/project"},
			{SessionID: "c", Title: "第三个会话", Cwd: "/project"},
			{SessionID: "d", Title: "第四个会话", Cwd: "/project"},
			{SessionID: "e", Title: "第五个会话", Cwd: "/project"},
			{SessionID: "f", Title: "第六个会话", Cwd: "/project"},
			{SessionID: "g", Title: "第七个会话", Cwd: "/project"},
			{SessionID: "h", Title: "草稿会话", Cwd: "/scratch"},
		},
		Workspaces: []session.WorkspaceRow{
			{WorkspaceID: "project", Title: "项目", Path: "/project", SessionIDs: []string{"a", "b", "c", "d", "e", "f", "g"}},
			{WorkspaceID: "scratch", Title: "草稿", Path: "/scratch", SessionIDs: []string{"h"}},
		},
	}

	assertAllPointer := func(t *testing.T, name string, m Model) {
		t.Helper()
		tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 864)
		tt.Frame()
		boxes := tt.Interactives()
		var lines []string
		for _, b := range boxes {
			if b.Cursor == ui.CursorPointer || b.Cursor == ui.CursorText {
				continue
			}
			lines = append(lines, b.String())
		}
		sort.Strings(lines)
		if len(lines) > 0 {
			t.Fatalf("[%s] %d/%d interactive elements missing CursorPointer:\n%s", name, len(lines), len(boxes), strings.Join(lines, "\n"))
		}
	}

	t.Run("expanded_grouped", func(t *testing.T) {
		m := Model{
			ThemeID:           "deepseek.light",
			ViewportW:         280,
			ViewportH:         864,
			Snapshot:          baseSnapshot,
			SearchQuery:       &searchQuery,
			SearchFocus:       &searchFocus,
			WorkspaceMenuOpen: &wsMenuOpen,
			OnToggle:          noop,
			OnCreate:          noop,
			OnOpenMarket:      noop,
			OnOpenSchedule:    noop,
			OnBrowseWorkspace: noop,
			OnManageWorkspace: noopStr,
			OnSelect:          noopStr,
			OnRevealWorkspace: noop,
			OnOpenSettings:    noop,
			Account:           Account{Name: "Ayase", SignedIn: true, OnOpen: noop},
		}
		assertAllPointer(t, "expanded_grouped", m)
	})

	t.Run("collapsed_rail", func(t *testing.T) {
		q := ""
		m := Model{
			ThemeID:           "deepseek.light",
			Collapsed:         true,
			ViewportW:         280,
			ViewportH:         864,
			Snapshot:          baseSnapshot,
			SearchQuery:       &q,
			SearchFocus:       &searchFocus,
			WorkspaceMenuOpen: &wsMenuOpen,
			OnToggle:          noop,
			OnCreate:          noop,
			OnOpenMarket:      noop,
			OnOpenSchedule:    noop,
			OnBrowseWorkspace: noop,
			OnOpenSettings:    noop,
			Account:           Account{Name: "Ayase", SignedIn: true, OnOpen: noop},
		}
		assertAllPointer(t, "collapsed_rail", m)
	})

	t.Run("action_toast_buttons", func(t *testing.T) {
		q := ""
		m := Model{
			ThemeID:     "deepseek.light",
			ViewportW:   280,
			ViewportH:   864,
			Snapshot:    baseSnapshot,
			SearchQuery: &q,
			SearchFocus: &searchFocus,
			ActionToast: &RowActionToastState{
				Kind:      "archived",
				OnUndo:    noop,
				OnViewAll: noop,
				OnDismiss: noop,
			},
			OnOpenSettings: noop,
		}
		assertAllPointer(t, "action_toast_buttons", m)
	})
}
