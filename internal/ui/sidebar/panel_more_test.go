package sidebar

import (
	"testing"

	"whaleisle/internal/desktop/pluginui"
)

func panelRowModel(unpinned map[string]bool, registrations []pluginui.PublicSlotEntry, entries []pluginui.PublicSlotEntry, pages []pluginui.NativeView) Model {
	m := Model{
		NativePages:         pages,
		OnOpenNativePage:    func(string) {},
		PanelUnpinned:       func(id string) bool { return unpinned[id] },
		OnSetPanelPinned:    func(id string, pinned bool) {},
		OnOpenMarket:        func() {},
		OnCreatePluginDraft: func(string) {},
	}
	m.PublicContributions.Entries = func(slot, ownerKey string) []pluginui.PublicSlotEntry {
		return entries
	}
	m.PublicContributions.Registrations = func(slot string) []pluginui.PublicSlotEntry {
		return registrations
	}
	return m
}

func panellistEntry(id, label string) pluginui.PublicSlotEntry {
	return pluginui.PublicSlotEntry{Slot: "sidebar.panellist", Handle: "e:" + id, ID: id, Label: label, Registrant: "owner"}
}

func TestSidebarPanelIconRowsFilterUnpinned(t *testing.T) {
	entries := []pluginui.PublicSlotEntry{panellistEntry("plugins", "Plugins"), panellistEntry("schedules", "Schedules"), panellistEntry("notes", "Notes")}
	pages := []pluginui.NativeView{
		{Handle: "h-schedules", Slot: "main", ID: "schedules", Label: "Schedules"},
		{Handle: "h-notes", Slot: "main", ID: "notes", Label: "Notes"},
	}
	m := panelRowModel(map[string]bool{"notes": true}, nil, entries, pages)

	var iconIDs []string
	for _, row := range sidebarPanelIconRows(m) {
		iconIDs = append(iconIDs, row.id)
	}
	for _, id := range iconIDs {
		if id == "notes" {
			t.Fatalf("unpinned panel still in sidebar icons: %v", iconIDs)
		}
	}

	var menuIDs []string
	for _, row := range sidebarPanelMenuRows(m) {
		menuIDs = append(menuIDs, row.id)
	}
	if len(menuIDs) != 1 || menuIDs[0] != "notes" {
		t.Fatalf("menu rows = %v, want [notes]", menuIDs)
	}
}

func TestSidebarPanelPinNeverAppliesToPluginsRow(t *testing.T) {
	entries := []pluginui.PublicSlotEntry{panellistEntry("plugins", "Plugins")}
	m := panelRowModel(map[string]bool{"plugins": true}, nil, entries, nil)
	for _, row := range sidebarPanelRows(m) {
		if sidebarPanelRowUnpinned(m, row) {
			t.Fatalf("plugins row treated as unpinnable")
		}
	}
	if rows := sidebarPanelMenuRows(m); len(rows) != 0 {
		t.Fatalf("plugins row entered More menu: %v", rows)
	}
}

func TestSidebarPanelMoreGateFollowsRegistrations(t *testing.T) {
	bundled := []pluginui.PublicSlotEntry{
		{Slot: "sidebar.panellist.actions", Handle: "c1", ID: "entry-customization", Registrant: "owner", Native: true, Catalog: true},
	}
	foreign := []pluginui.PublicSlotEntry{
		{Slot: "sidebar.panellist.actions", Handle: "c2", ID: "foreign-action", Registrant: "plugin-x", Catalog: true},
	}

	if show, _ := sidebarPanelMoreAvailable(panelRowModel(nil, nil, nil, nil)); show {
		t.Fatal("More shown with no registrations")
	}
	if show, isBundled := sidebarPanelMoreAvailable(panelRowModel(nil, bundled, nil, nil)); !show || !isBundled {
		t.Fatalf("bundled registration should show More natively: show=%v bundled=%v", show, isBundled)
	}
	// A foreign-only registration cannot render rows natively; without
	// unpinned launchers there is nothing honest to show, so the row stays
	// hidden instead of advertising an empty menu.
	if show, _ := sidebarPanelMoreAvailable(panelRowModel(nil, foreign, nil, nil)); show {
		t.Fatal("More shown for unrenderable foreign-only actions")
	}
	// Foreign registrations plus renderable unpinned launchers still open the
	// menu: the launchers are owner data, not foreign components.
	entries := []pluginui.PublicSlotEntry{panellistEntry("notes", "Notes")}
	pages := []pluginui.NativeView{{Handle: "h-notes", Slot: "main", ID: "notes", Label: "Notes"}}
	m := panelRowModel(map[string]bool{"notes": true}, foreign, entries, pages)
	if show, isBundled := sidebarPanelMoreAvailable(m); !show || isBundled {
		t.Fatalf("unpinned launchers should keep More visible: show=%v bundled=%v", show, isBundled)
	}
}

func TestSidebarPanelUnpinnedRowsWithoutPinSourceStayVisible(t *testing.T) {
	// No pin callbacks wired (old contract, no owner): nothing filters.
	entries := []pluginui.PublicSlotEntry{panellistEntry("notes", "Notes")}
	pages := []pluginui.NativeView{{Handle: "h-notes", Slot: "main", ID: "notes", Label: "Notes"}}
	m := Model{NativePages: pages, OnOpenNativePage: func(string) {}}
	m.PublicContributions.Entries = func(slot, ownerKey string) []pluginui.PublicSlotEntry { return entries }
	if rows := sidebarPanelIconRows(m); len(rows) != 1 {
		t.Fatalf("rows filtered without a pin source: %v", rows)
	}
}
