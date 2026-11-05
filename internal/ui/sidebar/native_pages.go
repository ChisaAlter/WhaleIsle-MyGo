package sidebar

import (
	"strings"
	"whaleisle/internal/desktop/pluginui"
)

// A view handle is a live capability, while its label is only presentation.
// Blank labels and other slots must not become invented global navigation.
func nativeSidebarPages(m Model) []pluginui.NativeView {
	if m.OnOpenNativePage == nil {
		return nil
	}
	pages := make([]pluginui.NativeView, 0, len(m.NativePages))
	for _, page := range m.NativePages {
		if (page.Slot == "sidebar.page" || page.Slot == "main") && page.Handle != "" && strings.TrimSpace(page.Label) != "" {
			pages = append(pages, page)
		}
	}
	return pages
}
