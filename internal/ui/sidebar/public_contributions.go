package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

type nativePanelRow struct {
	id, active, label string
	run               func()
}

func sidebarPanelRows(m Model) []nativePanelRow {
	rows := []nativePanelRow{
		{id: "plugins", active: "plugins", label: i18n.T(i18n.KeySidebarMarket), run: m.OnOpenMarket},
		{id: "schedules", active: "schedule", label: i18n.T(i18n.KeySidebarSchedule), run: m.OnOpenSchedule},
	}
	if m.OnOpenAssistant != nil {
		rows = append(rows, nativePanelRow{id: "whale", active: "assistant", label: whaleAssistantNavLabel(m), run: m.OnOpenAssistant})
	}
	return rows
}

func sidebarPanelIcon(c *ui.Context, id string, ink ui.Color) {
	switch id {
	case "whale":
		if image := chrome.BrandAvatar(); image != nil {
			ui.Image(c, image).Size(18, 18).Fit(ui.Contain)
		} else {
			ui.Text(c, "◉").TextColor(ink)
		}
	case "plugins":
		ui.Icon(c, sidebarRailIcons["plugins"]).Size(16, 16).TextColor(ink)
	case "schedules":
		ui.Icon(c, sidebarRailIcons["clock"]).Size(16, 16).TextColor(ink)
	default:
		ui.Icon(c, sidebarRailIcons["panel"]).Size(16, 16).TextColor(ink)
	}
}
