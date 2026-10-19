package settings

import "github.com/egoist/mygo/ui"

var bodyIcons = map[string]*ui.SVG{}

func bodyIcon(name string) *ui.SVG {
	if icon := bodyIcons[name]; icon != nil {
		return icon
	}
	paths := map[string]string{
		"edit":     `<path d="m10 3 3 3M3 10l7-7 3 3-7 7-4 1z"/>`,
		"trash":    `<path d="M2 4h12M6 2h4M4 4l1 10h6l1-10M7 6v6M9 6v6"/>`,
		"folder":   `<path d="M2 13V3h4l2 2h6v3M2 13l2-5h11l-2 5z"/>`,
		"plus":     `<path d="M8 3v10M3 8h10"/>`,
		"refresh":  `<path d="M13 6a5 5 0 1 0 0 5M13 2v4H9"/>`,
		"search":   `<circle cx="7" cy="7" r="4"/><path d="m10 10 4 4"/>`,
		"document": `<path d="M4 2h6l3 3v9H4zM10 2v4h3M6 8h5M6 11h5"/>`,
		"down":     `<path d="m4 6 4 4 4-4"/>`,
		"right":    `<path d="m6 4 4 4-4 4"/>`,
	}
	icon := ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" stroke-linejoin="round">` + paths[name] + `</svg>`))
	bodyIcons[name] = icon
	return icon
}
func bodyIconAction(c *ui.Context, label, icon string, disabled bool) bool {
	button := ui.Button(c, "").Label(label).Size(28, 28).Padding(6).Radius(6).BorderWidth(0).Background(ui.Transparent).Disabled(disabled).Tooltip(label)
	button.Children(func() { ui.Icon(c, bodyIcon(icon)).Size(16, 16) })
	return button.Clicked()
}
