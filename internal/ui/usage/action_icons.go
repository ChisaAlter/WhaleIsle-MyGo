package usage

import "github.com/egoist/mygo/ui"

var usageActionIcons = map[string]*ui.SVG{}

// usageActionIcon renders the leading glyph of the page-header buttons,
// matching the original's 16 DIP stroke icons (⤓ 导出, ⚙ 设置, ⟳ 刷新).
func usageActionIcon(name string) *ui.SVG {
	if icon := usageActionIcons[name]; icon != nil {
		return icon
	}
	paths := map[string]string{
		"export":   `<path d="M8 2v7M5 6.5 8 9.5l3-3"/><path d="M3 11v2.5h10V11"/>`,
		"settings": `<circle cx="8" cy="8" r="4.2"/><circle cx="8" cy="8" r="1.8"/><path d="M8 1.8v1.5M8 12.7v1.5M1.8 8h1.5M12.7 8h1.5M3.6 3.6l1.1 1.1M11.3 11.3l1.1 1.1M12.4 3.6l-1.1 1.1M4.7 11.3l-1.1 1.1"/>`,
		"refresh":  `<path d="M13 6a5 5 0 1 0 0 5M13 2v4H9"/>`,
	}
	icon := ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" stroke-linejoin="round">` + paths[name] + `</svg>`))
	usageActionIcons[name] = icon
	return icon
}

func usageLeadingIcon(name string) func(*ui.Context) {
	return func(c *ui.Context) { ui.Icon(c, usageActionIcon(name)).Size(16, 16) }
}
