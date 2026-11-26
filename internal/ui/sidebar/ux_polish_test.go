package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSessionRowActionHitTargetsAndToggleShortcutTooltip(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(oldLocale)

	snap := session.ViewState{
		ClientReady:    true,
		WorkspaceReady: true,
		SelectedID:     "s1",
		Items: []session.Item{
			{SessionID: "s1", Title: "Alpha Session", UpdatedAt: 1700000000000},
		},
		Workspaces: []session.WorkspaceRow{
			{WorkspaceID: "ws1", Title: "Workspace", SessionIDs: []string{"s1"}},
		},
	}
	m := Model{
		ConnectionReady:    true,
		Snapshot:           snap,
		ThemeID:            "system",
		MenuShortcutLabels: map[string]string{"sidebar.toggle": "Cmd+Shift+B"},
		OnToggle:           func() {},
		OnSelect:           func(string) {},
		OnArchive:          func(string) {},
		OnPin:              func(string) {},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 320, 600)

	// Hover the session row so action buttons become visible and laid out in flow.
	rowBounds, ok := tt.Find("Alpha Session")
	if !ok {
		t.Fatalf("session row not found: %v", tt.Texts())
	}
	tt.Move(rowBounds.X+10, rowBounds.Y+rowBounds.H/2)

	for _, label := range []string{
		sessionActionsLabel("s1", "Alpha Session"),
		i18n.T(i18n.KeySidebarArchive),
		i18n.T(i18n.KeySidebarPin),
	} {
		b, found := tt.Find(label)
		if !found {
			t.Fatalf("action button %q not found", label)
		}
		if b.W < 24 || b.H < 24 {
			t.Fatalf("action button %q hit target too small: %.1fx%.1f (want >= 24x24)", label, b.W, b.H)
		}
	}

	// Verify tooltip helper respects custom MenuShortcutLabels["sidebar.toggle"] rather than hardcoded "(Ctrl+B)".
	got := sidebarToggleTooltip(m, i18n.T(i18n.KeySidebarCollapse))
	want := i18n.T(i18n.KeySidebarCollapse) + " (Cmd+Shift+B)"
	if got != want {
		t.Fatalf("sidebarToggleTooltip = %q, want %q", got, want)
	}
	mEmpty := Model{}
	if gotEmpty := sidebarToggleTooltip(mEmpty, "Collapse"); gotEmpty != "Collapse" {
		t.Fatalf("sidebarToggleTooltip without binding = %q, want %q", gotEmpty, "Collapse")
	}
}
