package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

type presetChoice struct{ ID, Label, Description, Broken string }

// The toolkit's selection value is the opaque ID. Captions are display-only,
// including when multiple catalog rows share exactly the same name.
func presetSelect(c *ui.Context, caption string, options []presetChoice, pick func(string)) ui.Element {
	owner := ui.Box(c)
	selected := ui.Local(owner, "preset-selection", func() string { return "" })
	parts := ui.SelectBase(c, selected)
	trigger := parts.Trigger.Label(caption).Tooltip(caption).MaxWidth(220).MinWidth(0).Padding(0, 4, 0, 8).Gap(4).Radius(8).AlignItems(ui.Center)
	trigger.Children(func() { ui.Text(c, caption).Grow(1).MinWidth(0).SingleLine(); ui.Text(c, "▾") })
	t := c.Theme()
	parts.Popup(func(panel ui.Element) {
		panel.Padding(t.Space(1)).Radius(t.Radius+2).Background(t.Background).Border(1, t.Border)
		for _, o := range options {
			row := parts.Item(o.ID).Label(o.Label).Tooltip(o.Label+" · "+o.ID+"\n"+o.Description+"\n"+o.Broken).Padding(t.Space(1.5), t.Space(2.5))
			if row.Highlighted() {
				row = row.Background(t.Accent).TextColor(t.AccentText)
			}
			row.Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					ui.Text(c, o.Label)
					if o.Description != "" {
						ui.Text(c, o.Description).FontSize(11).MaxWidth(280)
					}
				})
			})
		}
	})
	if trigger.Changed() {
		id := *selected
		*selected = ""
		if pick != nil {
			pick(id)
		}
	}
	return trigger
}
func permissionChoices(options []session.PermissionOption) []presetChoice {
	out := make([]presetChoice, 0, len(options))
	for _, o := range options {
		out = append(out, presetChoice{ID: o.Value, Label: o.Label})
	}
	return out
}
func agentPresetChoices(options []session.AgentPresetOption) []presetChoice {
	out := make([]presetChoice, 0, len(options))
	for _, o := range options {
		out = append(out, presetChoice{ID: o.ID, Label: o.Label, Description: o.Description, Broken: o.Broken})
	}
	return out
}
