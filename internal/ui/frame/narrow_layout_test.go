package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestNarrowDrawerOverlaysAndReopens pins the phone band (AppFrame data-phone):
// below PHONE_MAX the sidebar leaves the grid, the conversation column keeps the
// whole width, and a floating menu button reopens the drawer.
func TestNarrowDrawerOverlaysAndReopens(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	sizes := DefaultLayout()
	sizes.Bottom = 0
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: session.New(), Sizes: &sizes, OnOpenSettings: func() {}})
	}, 700, 800)

	if !sizes.drawerMode() {
		t.Fatalf("700 DIP viewport is not the drawer band: %+v", sizes.Responsive)
	}
	if got := sizes.drawerWidth(); got != 320 {
		t.Fatalf("drawer width=%v, want the 320 cap (700-48 is wider)", got)
	}
	// The conversation column owns the full width: the composer card is centred
	// on the window, not on a shrunk track.
	card, ok := tt.Find("Composer card")
	if !ok {
		t.Fatal("composer card missing")
	}
	if centre := card.X + card.W/2; centre < 330 || centre > 370 {
		t.Fatalf("card centre %.1f, want the 700 DIP window's centre", centre)
	}

	opener := i18n.T(i18n.KeySidebarExpand)
	if _, ok := tt.Find(opener); !ok {
		t.Fatalf("closed drawer has no opener; texts=%q", tt.Texts())
	}
	if err := tt.Click(opener); err != nil {
		t.Fatal(err)
	}
	// Open: the navigation paints inside the drawer track and a backdrop covers
	// the conversation.
	if _, ok := tt.Find(i18n.T(i18n.KeySidebarWorkspaceSection)); !ok {
		t.Fatalf("open drawer did not mount navigation; texts=%q", tt.Texts())
	}
	backdrop, ok := tt.Find(i18n.T(i18n.KeySidebarCollapse))
	if !ok || backdrop.W < 300 {
		t.Fatalf("drawer backdrop=%+v", backdrop)
	}
	// Clicking the backdrop collapses the drawer again.
	tt.ClickAt(600, 400)
	tt.Frame()
	if _, ok := tt.Find(i18n.T(i18n.KeySidebarWorkspaceSection)); ok {
		t.Fatal("backdrop click left the drawer open")
	}
	if _, ok := tt.Find(opener); !ok {
		t.Fatal("collapse hid the opener")
	}
}

// TestNarrowHeaderHidesTrailingCluster pins the compact caption band: in the
// phone drawer band (below PHONE_MAX) the titlebar trailing cluster is not built
// at all, so the session title keeps the band; from 768 DIP up a desktop window
// is landscape and keeps the cluster, as it keeps the 280-DIP sidebar track.
func TestNarrowHeaderHidesTrailingCluster(t *testing.T) {
	for _, tc := range []struct {
		width int
		want  bool
	}{
		{700, false},
		{900, true},
		{1200, true},
	} {
		sizes := DefaultLayout()
		sizes.Bottom, sizes.Right = 0, 0
		called := false
		tt := ui.NewTester(func(c *ui.Context) {
			sizes.observeSidebarViewport(float32(tc.width))
			m := Model{Store: session.New(), Sizes: &sizes}
			m.WorkspaceTrailing = func(c *ui.Context, density string) {
				called = true
				ui.Box(c).Size(120, 32)
			}
			ui.Column(c).Fill().Children(func() {
				workspaceSharedCaption(c, m, session.ViewState{}, theme.Resolve("deepseek.light").Light, false)
			})
		}, tc.width, 200)
		if called != tc.want {
			t.Fatalf("width %d: trailing cluster built=%v, want %v", tc.width, called, tc.want)
		}
		if !tc.want {
			// The cluster's reserve disappears with it: the title may use the band.
			if _, ok := tt.Find("Conversation session title"); !ok {
				t.Fatal("narrow header lost the title seat")
			}
		}
	}
}
