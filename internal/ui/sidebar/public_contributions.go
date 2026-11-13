package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func publicSidebarSlot(c *ui.Context, m Model, slot, ownerKey string, fallback func()) {
	if m.PublicContributions.Slot != nil {
		m.PublicContributions.Slot(c, slot, ownerKey, fallback)
	} else if fallback != nil {
		fallback()
	}
}

func publicSidebarBefore(c *ui.Context, m Model, slot, ownerKey, beforeID string) bool {
	if m.PublicContributions.Before != nil {
		m.PublicContributions.Before(c, slot, ownerKey, beforeID)
	}
	return m.PublicContributions.NativeWinner(slot, ownerKey, beforeID)
}

func publicSidebarRowContext(m Model, id string, menu, hover bool) {
	if m.PublicContributions.RowContext != nil {
		m.PublicContributions.RowContext(id, menu, hover)
	}
}

type publicPanelRow struct {
	id, active, label, handle string
	entry                     *pluginui.PublicSlotEntry
	run                       func()
}

// A row is pinnable when it is driven by a live sidebar.panellist
// registration. The plugin manager stays pinned by contract, and fallback
// rows without a registration behind them have nothing to unpin.
func sidebarPanelPinnable(m Model, row publicPanelRow) bool {
	return row.entry != nil && row.id != "plugins" && m.PanelUnpinned != nil && m.OnSetPanelPinned != nil
}

func sidebarPanelRowUnpinned(m Model, row publicPanelRow) bool {
	return sidebarPanelPinnable(m, row) && m.PanelUnpinned(row.id)
}

// Row identity follows the elected occurrence or live view handle, not its
// reusable registration ID. Different main/sidebar owners may share an ID.
func (row publicPanelRow) rowKey() string {
	if row.entry != nil {
		return "public:" + row.entry.Handle
	}
	if row.handle != "" {
		return "view:" + row.handle
	}
	return "builtin:" + row.id
}

func sidebarPanelRows(m Model) []publicPanelRow {
	builtins := []publicPanelRow{
		{id: "plugins", active: "plugins", label: i18n.T(i18n.KeySidebarMarket), run: m.OnOpenMarket},
		{id: "schedules", active: "schedule", label: i18n.T(i18n.KeySidebarSchedule), run: m.OnOpenSchedule},
	}
	if m.OnOpenAssistant != nil {
		builtins = append(builtins, publicPanelRow{id: "whale", active: "assistant", label: whaleAssistantNavLabel(m), run: m.OnOpenAssistant})
	}
	pages := nativeSidebarPages(m)
	entries := []pluginui.PublicSlotEntry{}
	if m.PublicContributions.Entries != nil {
		entries = m.PublicContributions.Entries("sidebar.panellist", "")
	}
	if len(entries) == 0 {
		for _, page := range pages {
			page := page
			builtins = append(builtins, publicPanelRow{id: page.ID, active: page.Handle, label: page.Label, handle: page.Handle, run: func() { m.OnOpenNativePage(page.Handle) }})
		}
		return builtins
	}
	rows, used := []publicPanelRow{}, map[string]bool{}
	for _, entry := range entries {
		entry := entry
		found := false
		for _, builtin := range builtins {
			if builtin.id == entry.ID {
				builtin.entry = &entry
				if entry.Label != "" && (builtin.id != "whale" || builtin.label == i18n.T(i18n.KeySidebarAssistant)) {
					builtin.label = entry.Label
				}
				rows = append(rows, builtin)
				found = true
				break
			}
		}
		if found {
			continue
		}
		for _, page := range pages {
			if page.ID == entry.ID {
				page := page
				used[page.Handle] = true
				rows = append(rows, publicPanelRow{id: entry.ID, active: page.Handle, label: page.Label, handle: page.Handle, entry: &entry, run: func() { m.OnOpenNativePage(page.Handle) }})
				break
			}
		}
	}
	// Registration IDs may be reused by distinct main/sidebar live owners. Only
	// the exact view handle consumed above is represented by that row.
	for _, page := range pages {
		if !used[page.Handle] {
			page := page
			rows = append(rows, publicPanelRow{id: page.ID, active: page.Handle, label: page.Label, handle: page.Handle, run: func() { m.OnOpenNativePage(page.Handle) }})
		}
	}
	return rows
}

// sidebarPanelIconRows are the launchers shown in the sidebar: the merged
// rows minus the ones the user unpinned into the More menu. The unpinned set
// is region memory owned by the caller; without it nothing is filtered.
func sidebarPanelIconRows(m Model) []publicPanelRow {
	rows := sidebarPanelRows(m)
	if m.PanelUnpinned == nil {
		return rows
	}
	visible := rows[:0]
	for _, row := range rows {
		if !sidebarPanelRowUnpinned(m, row) {
			visible = append(visible, row)
		}
	}
	return visible
}

// sidebarPanelMenuRows lists the unpinned launchers for the panel list's More
// menu, in sidebar order. Opening a row does not pin it.
func sidebarPanelMenuRows(m Model) []publicPanelRow {
	rows := []publicPanelRow{}
	for _, row := range sidebarPanelRows(m) {
		if sidebarPanelRowUnpinned(m, row) {
			rows = append(rows, row)
		}
	}
	return rows
}

// sidebarPanelMoreAvailable reports the contract gate: the region shows More
// only while sidebar.panellist.actions has registrations. Bundled
// registrations are rendered natively; foreign rows stay unrendered rather
// than becoming guessed controls, so the button also requires content.
func sidebarPanelMoreAvailable(m Model) (show, bundled bool) {
	if m.PublicContributions.Registrations == nil {
		return false, false
	}
	regs := m.PublicContributions.Registrations("sidebar.panellist.actions")
	for _, reg := range regs {
		if reg.Native {
			bundled = true
		}
	}
	// The source gates the launcher on the slot having registrations at all
	// (`hasEntryActions && <PanelMoreMenu/>`). Bundled registrations make the
	// native browse/create rows renderable on their own; a foreign owner we
	// cannot render still counts for the gate but only earns a menu while
	// unpinned rows give it content.
	return bundled || (len(regs) > 0 && len(sidebarPanelMenuRows(m)) > 0), bundled
}

func publicPanelGlyph(c *ui.Context, m Model, row publicPanelRow, ink ui.Color) {
	if row.id == "whale" {
		if !m.PublicContributions.NativeWinner("sidebar.panellist", "", row.id) {
			return
		}
		if image := chrome.BrandAvatar(); image != nil {
			ui.Image(c, image).Size(18, 18).Fit(ui.Contain)
		} else {
			ui.Text(c, "◉").TextColor(ink)
		}
		return
	}
	if row.entry != nil && !row.entry.Native && m.PublicContributions.RenderEntry != nil {
		// A plugin's glyph is a web fragment: give it the 16 DIP icon box of the native rows
		// so it can neither claim the row's width nor push the label aside.
		ui.Box(c).Size(16, 16).Shrink(0).Clip().Children(func() {
			m.PublicContributions.RenderEntry(c, row.entry.Slot, row.entry.OwnerKey, row.entry.Handle)
		})
		return
	}
	if !m.PublicContributions.NativeWinner("sidebar.panellist", "", row.id) {
		return
	}
	key := "panel"
	switch row.id {
	case "plugins":
		key = "plugins"
	case "schedules":
		key = "clock"
	case "whale":
		if image := chrome.BrandAvatar(); image != nil {
			ui.Image(c, image).Size(18, 18).Fit(ui.Contain)
		} else {
			ui.Text(c, "◉").TextColor(ink)
		}
		return
	}
	ui.Icon(c, sidebarRailIcons[key]).Size(16, 16).TextColor(ink)
}

// sidebarPanelMoreRow renders the panel list's More row while the contract's
// actions slot offers renderable content. Menu order follows the source:
// unpinned launchers first (opening never pins), then the bundled browse and
// create entries. Foreign action rows never become guessed controls.
func sidebarPanelMoreRow(c *ui.Context, m Model, fg ui.Color, colors theme.Colors) {
	show, bundled := sidebarPanelMoreAvailable(m)
	if !show {
		return
	}
	skinLayout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark)
	row := ui.Button(c.Key("sidebar-panel-more"), "").Label(sidebarLocaleText("panels.more")).FillWidth().Height(skinLayout.NavHeight).Radius(12).
		Background(ui.Transparent).TextColor(fg).BorderWidth(0).
		Padding(7, 8, 7, 8).AlignItems(ui.Center)
	if skinLayout.Skin == theme.SkinCodex {
		row.Padding(6, 8).MinHeight(32)
	}
	if row.Hovered() || row.MenuOpen() {
		row.Background(chrome.Color(colors.Hover))
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, menuEllipsisArtwork).Size(16, 16).TextColor(fg)
			lineHeight := theme.InterfaceSize(22)
			if skinLayout.Active() {
				lineHeight = 20
			}
			ui.Text(c, sidebarLocaleText("panels.more")).FontSize(skinLayout.SidebarFont).
				FixedLineHeight(lineHeight).TextColor(fg).Grow(1)
		})
	})
	row.Menu(func(menu *ui.Menu) { sidebarPanelMoreMenu(m, menu, bundled) })
}

// sidebarPanelMoreMenu fills the panel list's More menu for the expanded
// sidebar row and the rail icon: unpinned launchers in sidebar order (opening
// never pins), a pin-back submenu, then the bundled browse and create rows.
func sidebarPanelMoreMenu(m Model, menu *ui.Menu, bundled bool) {
	menu.CloseOnPointerLeave(true)
	unpinned := sidebarPanelMenuRows(m)
	for _, row := range unpinned {
		row := row
		if menu.Item(row.label).Disabled(row.run == nil).Checked(m.ActivePanel == row.active).Chosen() && row.run != nil {
			row.run()
		}
	}
	if len(unpinned) > 0 && m.OnSetPanelPinned != nil {
		menu.Submenu(sidebarLocaleText("panels.pin"), func(sub *ui.Menu) {
			for _, row := range unpinned {
				row := row
				if sub.Item(row.label).Chosen() {
					m.OnSetPanelPinned(row.id, true)
				}
			}
		})
		menu.Separator()
	}
	if bundled {
		if m.OnOpenMarket != nil && menu.Item(sidebarLocaleText("panels.browse")).Chosen() {
			m.OnOpenMarket()
		}
		if m.OnCreatePluginDraft != nil && menu.Item(sidebarLocaleText("panels.create")).Chosen() {
			m.OnCreatePluginDraft("leftbar")
		}
	}
}
