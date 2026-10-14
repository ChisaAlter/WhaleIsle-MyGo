package composer

import "github.com/egoist/mygo/ui"

// composerSelect uses SelectBase's existing selection and keyboard behavior.
// InputBar.select caps the closed trigger at 220px; only the display truncates,
// while the value, accessibility label, tooltip, and menu choices stay complete.
func composerSelect(c *ui.Context, selected *string, options []string) ui.Element {
	selectParts := ui.SelectBase(c, selected)
	trigger := selectParts.Trigger.Label(*selected).Tooltip(*selected).
		MaxWidth(220).MinWidth(0).Padding(0, 4, 0, 8).Gap(4).Radius(8).
		Justify(ui.SpaceBetween).AlignItems(ui.Center)
	trigger.Children(func() {
		ui.Text(c, *selected).Grow(1).MinWidth(0).SingleLine().FixedLineHeight(20)
		ui.Text(c, "▾").FontSize(12).Shrink(0)
	})
	// Keep the native toolkit's existing popup surface and option behavior.
	t := c.Theme()
	selectParts.Popup(func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Radius+2).
			Background(t.Background).Border(1, t.Border).MaxWidthPercent(100).
			Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, .18))
		for _, option := range options {
			item := selectParts.Item(option).Tooltip(option).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius).MinWidth(0)
			switch {
			case item.Highlighted():
				item = item.Background(t.Accent).TextColor(t.AccentText)
			case option == *selected:
				item = item.Background(t.Surface)
			}
			item.Children(func() { ui.Text(c, option).MinWidth(0).MaxWidthPercent(100) })
		}
	})
	return trigger
}
