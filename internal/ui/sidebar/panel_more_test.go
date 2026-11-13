package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
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

// --- widget-layer coverage: real rows, real menus, real hit-testing ---

func panelMoreWidgetModel(unpinned map[string]bool, calls map[string]int, pinnedCalls *[]string, opened *string, created *string) Model {
	m := Model{
		ThemeID: "deepseek.dark", SystemDark: true,
		NativePages: []pluginui.NativeView{
			{Handle: "h-schedules", Slot: "main", ID: "schedules", Label: "Schedules"},
			{Handle: "h-notes", Slot: "main", ID: "notes", Label: "Notes"},
		},
		OnOpenNativePage: func(h string) { *opened = h },
		PanelUnpinned:    func(id string) bool { return unpinned[id] },
		OnSetPanelPinned: func(id string, pinned bool) {
			*pinnedCalls = append(*pinnedCalls, id)
			if pinned {
				delete(unpinned, id)
			} else {
				unpinned[id] = true
			}
		},
		OnOpenMarket:        func() { calls["market"]++ },
		OnCreatePluginDraft: func(region string) { *created = region },
	}
	m.PublicContributions.Entries = func(slot, ownerKey string) []pluginui.PublicSlotEntry {
		return []pluginui.PublicSlotEntry{
			{Slot: "sidebar.panellist", Handle: "e:plugins", ID: "plugins", Label: "插件", Registrant: "bundle", Native: true},
			{Slot: "sidebar.panellist", Handle: "e:schedules", ID: "schedules", Label: "自动化任务", Registrant: "bundle", Native: true},
			{Slot: "sidebar.panellist", Handle: "e:notes", ID: "notes", Label: "Notes", Registrant: "plugin/x"},
		}
	}
	m.PublicContributions.Registrations = func(slot string) []pluginui.PublicSlotEntry {
		return []pluginui.PublicSlotEntry{{Slot: "sidebar.panellist.actions", Handle: "a:1", ID: "entry-customization", Native: true}}
	}
	return m
}

func TestSidebarPanelMoreMenuWidgetFlow(t *testing.T) {
	unpinned := map[string]bool{"notes": true}
	calls := map[string]int{}
	var pinnedCalls []string
	var opened, created string
	m := panelMoreWidgetModel(unpinned, calls, &pinnedCalls, &opened, &created)
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 600)

	if err := tt.Click("更多"); err != nil {
		t.Fatalf("more row missing: %v texts=%q", err, tt.Texts())
	}
	want := []string{"Notes", "固定到左侧栏", "-", "浏览更多插件", "创建你的插件"}
	got := tt.Menu()
	if len(got) != len(want) {
		t.Fatalf("menu = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("menu[%d] = %q, want %q (all=%v)", i, got[i], want[i], got)
		}
	}
	if err := tt.ChooseMenuItem("Notes"); err != nil {
		t.Fatalf("choose unpinned row: %v", err)
	}
	if opened != "h-notes" {
		t.Fatalf("unpinned open = %q, want h-notes", opened)
	}
	if unpinned["notes"] != true {
		t.Fatal("opening an unpinned row must not pin it")
	}

	if err := tt.Click("更多"); err != nil {
		t.Fatalf("more row missing on reopen: %v", err)
	}
	if err := tt.ChooseMenuItem("浏览更多插件"); err != nil {
		t.Fatalf("choose browse: %v", err)
	}
	if calls["market"] != 1 {
		t.Fatalf("market opens = %d", calls["market"])
	}

	if err := tt.Click("更多"); err != nil {
		t.Fatalf("more row missing on third open: %v", err)
	}
	if err := tt.ChooseMenuItem("创建你的插件"); err != nil {
		t.Fatalf("choose create: %v", err)
	}
	if created != "leftbar" {
		t.Fatalf("create region = %q, want leftbar", created)
	}

	if err := tt.Click("更多"); err != nil {
		t.Fatalf("more row missing on fourth open: %v", err)
	}
	if err := tt.ChooseMenuItem("固定到左侧栏", "Notes"); err != nil {
		t.Fatalf("choose pin-back submenu: %v", err)
	}
	if len(pinnedCalls) != 1 || pinnedCalls[0] != "notes" || unpinned["notes"] {
		t.Fatalf("pin-back calls=%v unpinned=%v", pinnedCalls, unpinned)
	}
}

func TestSidebarPanelRowUnpinContextMenuWidget(t *testing.T) {
	unpinned := map[string]bool{}
	calls := map[string]int{}
	var pinnedCalls []string
	var opened, created string
	m := panelMoreWidgetModel(unpinned, calls, &pinnedCalls, &opened, &created)
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 600)

	if err := tt.RightClick("Notes"); err != nil {
		t.Fatalf("panel row missing for context menu: %v texts=%q", err, tt.Texts())
	}
	got := tt.Menu()
	if len(got) != 1 || got[0] != "从左侧栏取消固定" {
		t.Fatalf("row context menu = %v", got)
	}
	if err := tt.ChooseMenuItem("从左侧栏取消固定"); err != nil {
		t.Fatalf("choose unpin: %v", err)
	}
	if len(pinnedCalls) != 1 || pinnedCalls[0] != "notes" || !unpinned["notes"] {
		t.Fatalf("unpin calls=%v unpinned=%v", pinnedCalls, unpinned)
	}
}

func TestSidebarPanelMoreRowHiddenWithoutActionsRegistration(t *testing.T) {
	unpinned := map[string]bool{"notes": true}
	calls := map[string]int{}
	var pinnedCalls []string
	var opened, created string
	m := panelMoreWidgetModel(unpinned, calls, &pinnedCalls, &opened, &created)
	m.PublicContributions.Registrations = func(slot string) []pluginui.PublicSlotEntry { return nil }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 600)
	if _, ok := tt.Find("更多"); ok {
		t.Fatalf("more row rendered without actions registrations: %q", tt.Texts())
	}
	// The unpinned row still hides from the icon list (pin memory applies
	// regardless of whether the menu exists).
	if _, ok := tt.Find("Notes"); ok {
		t.Fatalf("unpinned row still rendered as icon: %q", tt.Texts())
	}
}

func TestSidebarRailPanelMoreMenuWidget(t *testing.T) {
	unpinned := map[string]bool{"notes": true}
	calls := map[string]int{}
	var pinnedCalls []string
	var opened, created string
	m := panelMoreWidgetModel(unpinned, calls, &pinnedCalls, &opened, &created)
	m.Collapsed = true
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 64, 600)

	if err := tt.Click("更多"); err != nil {
		t.Fatalf("rail more button missing: %v texts=%q", err, tt.Texts())
	}
	got := tt.Menu()
	if len(got) == 0 || got[0] != "Notes" {
		t.Fatalf("rail more menu = %v", got)
	}
	if err := tt.ChooseMenuItem("Notes"); err != nil {
		t.Fatalf("rail choose unpinned: %v", err)
	}
	if opened != "h-notes" {
		t.Fatalf("rail unpinned open = %q", opened)
	}
}
