package frame

import "testing"

// TestSidebarResponsivePreferenceAndManualOverride pins the two bands the shell
// keeps: a landscape desktop keeps its saved track at any width (280 DIP by
// default, 336 here), and only below PHONE_MAX does the phone drawer take the
// sidebar out of the grid, with a transient manual expand.
func TestSidebarResponsivePreferenceAndManualOverride(t *testing.T) {
	s := DefaultLayout()
	s.Sidebar = 336
	if c := s.ResolveColumns(1200, false); c.Sidebar != 336 {
		t.Fatal(c)
	}
	// A 900-DIP desktop window keeps the track: the auto-collapse band is the
	// portrait phone band, not a window width.
	if c := s.ResolveColumns(900, false); c.Sidebar != 336 || s.Sidebar != 336 {
		t.Fatal("900 DIP collapsed the desktop track", c, s.Sidebar)
	}
	// Below PHONE_MAX the drawer band starts and the sidebar leaves the grid.
	if c := s.ResolveColumns(700, false); c.Sidebar != SidebarCollapsedWidth() || !s.drawerMode() {
		t.Fatal("drawer band kept a sidebar track", c)
	}
	// A manual expand in the drawer band re-opens over the centre, at the saved
	// preference.
	s.ToggleSidebar()
	if c := s.ResolveColumns(700, false); c.Sidebar != 336 {
		t.Fatal("manual drawer expand lost the preference", c)
	}
	// Crossing the band resets that transient override.
	s.ResolveColumns(1200, false)
	if c := s.ResolveColumns(700, false); c.Sidebar != SidebarCollapsedWidth() {
		t.Fatal("band crossing did not reset the drawer override", c)
	}
	if c := s.ResolveColumns(1200, false); c.Sidebar != 336 || s.Sidebar != 336 {
		t.Fatal("wide preference was overwritten", c)
	}
	// Collapsing at a desktop width writes the rail; expanding restores 336.
	s.ToggleSidebar()
	if c := s.ResolveColumns(1200, false); c.Sidebar != SidebarCollapsedWidth() || s.Sidebar != 0 {
		t.Fatal("desktop collapse kept a track", c, s.Sidebar)
	}
	s.ToggleSidebar()
	if c := s.ResolveColumns(1200, false); c.Sidebar != 336 || s.SidebarExpanded != 336 {
		t.Fatal("desktop expand lost the width", c, s.SidebarExpanded)
	}
}
