package frame

import "testing"

func TestSidebarResponsivePreferenceAndManualOverride(t *testing.T) {
	s := DefaultLayout()
	s.Sidebar = 336
	if c := s.ResolveColumns(1200, false); c.Sidebar != 336 {
		t.Fatal(c)
	}
	if c := s.ResolveColumns(900, false); c.Sidebar != SidebarCollapsed || s.Sidebar != 336 {
		t.Fatal(c, s.Sidebar)
	}
	s.ToggleSidebar()
	if c := s.ResolveColumns(900, false); c.Sidebar != 336 {
		t.Fatal("manual narrow expand lost", c)
	}
	if c := s.ResolveColumns(850, false); c.Sidebar != 336 {
		t.Fatal("same-band resize reset override", c)
	}
	if c := s.ResolveColumns(900, true); c.Sidebar != SidebarCollapsed {
		t.Fatal("opening resources did not collapse", c)
	}
	s.ToggleSidebar()
	if c := s.ResolveColumns(900, true); c.Sidebar != 336 {
		t.Fatal("open resource reset override each frame", c)
	}
	s.ResolveColumns(1024, false)
	if c := s.ResolveColumns(1023, false); c.Sidebar != SidebarCollapsed {
		t.Fatal("breakpoint did not reset override", c)
	}
	if c := s.ResolveColumns(1200, false); c.Sidebar != 336 || s.Sidebar != 336 {
		t.Fatal("wide preference was overwritten", c)
	}
	s.ToggleSidebar()
	s.ResolveColumns(900, false)
	s.ToggleSidebar()
	if c := s.ResolveColumns(900, false); c.Sidebar != SidebarDefault || s.Sidebar != 0 {
		t.Fatal("narrow reopening wrote the wide preference", c)
	}
	if c := s.ResolveColumns(1200, false); c.Sidebar != SidebarCollapsed {
		t.Fatal("wide collapsed preference lost", c)
	}
}
