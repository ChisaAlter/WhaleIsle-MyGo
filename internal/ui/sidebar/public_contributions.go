package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
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
