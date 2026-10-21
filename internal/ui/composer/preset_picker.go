package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// presetChoice is one row of a picker: the opaque value sent back, the
// product-facing caption, and the optional Host detail.
type presetChoice struct {
	// ID is the opaque value the picker sends back.
	ID string
	// Label is the product-facing caption.
	Label string
	// Raw is the Host-supplied label, kept so a caption the Host produced can
	// be resolved back to its choice.
	Raw string
	// Description is the secondary line of the row, when the Host has one.
	Description string
	// Broken marks a preset the Host refused.
	Broken string
}

// presetOptions tunes one picker chip; the zero value is the plain text chip.
type presetOptions struct {
	// Icon is an optional 14px leading glyph (the permission shield).
	Icon *ui.SVG
	// Leading draws replacement content for the caption (the preset
	// introduction view); the chevron is appended by presetSelect.
	Leading func()
	// Text is the caption ink; the zero value keeps the theme's.
	Text ui.Color
	// FontSize and Weight size the caption (13/500 for toolbar chips).
	FontSize float32
	Weight   int
	MaxWidth float32
}

type presetNavigation struct {
	Active  int
	Focused bool
	Rows    []*ui.Handle
	Anchor  ui.Handle
}

// presetSelect renders one closed-chip picker: a 28px capsule showing the
// chosen caption plus a chevron that turns over while the popup is open. The
// popup floats above the chip (the composer sits at the window's bottom edge),
// lists 34px rows with one shared hover/focus highlight, and closes on Escape,
// on an outside press, or on a pick.
//
// The toolkit's SelectBase is not used here because it owns its open flag and
// paints no hover fill, so the chip could show neither its open state nor its
// own hover.
func presetSelect(c *ui.Context, caption string, options []presetChoice, pick func(string), colors theme.Colors, opt presetOptions) ui.Element {
	owner := ui.Box(c)
	open := ui.Local(owner, "preset-open", func() bool { return false })
	nav := ui.Local(owner, "preset-navigation", func() presetNavigation { return presetNavigation{} })
	display, selectedID := resolvePresetCaption(caption, options)
	trigger := chip(c, colors, *open, opt.Text)
	if opt.MaxWidth > 0 {
		trigger = trigger.MaxWidth(opt.MaxWidth)
	}
	trigger = trigger.Label(display).Tooltip(display + "  ·  " + i18n.T(i18n.KeyComposerShortcutEsc))
	trigger.Bind(&nav.Anchor)
	trigger.Children(func() {
		if opt.Icon != nil {
			ui.Icon(c, opt.Icon).Size(14, 14).Shrink(0)
		}
		if opt.Leading != nil {
			opt.Leading()
		} else {
			text := ui.Text(c, display).Grow(1).MinWidth(0).SingleLine()
			if opt.FontSize > 0 {
				text = text.FontSize(opt.FontSize)
			}
			if opt.Weight > 0 {
				text = text.FontWeight(opt.Weight)
			}
		}
		chevron(c, *open)
	})
	if trigger.Clicked() {
		*open = !*open
		nav.Focused = false
	}
	if !*open {
		nav.Focused = false
		return trigger
	}
	if len(nav.Rows) != len(options) {
		nav.Rows = make([]*ui.Handle, len(options))
		for i := range nav.Rows {
			nav.Rows[i] = &ui.Handle{}
		}
		nav.Active = 0
		for i, o := range options {
			if o.ID == selectedID {
				nav.Active = i
				break
			}
		}
	}
	if nav.Active < 0 || nav.Active >= len(options) {
		nav.Active = 0
	}
	ui.PopoverBase(c, trigger, open, func(panel ui.Element) {
		panel = menuSurface(panel.Padding(6).Gap(2).MinWidth(240).MaxWidthPercent(100), colors)
		if panel.Shortcut(0, ui.KeyDown) {
			nav.Active = (nav.Active + 1) % len(options)
			nav.Rows[nav.Active].RequestFocus(c)
		}
		if panel.Shortcut(0, ui.KeyUp) {
			nav.Active = (nav.Active + len(options) - 1) % len(options)
			nav.Rows[nav.Active].RequestFocus(c)
		}
		panel.Children(func() {
			ui.Scroll(c).FillWidth().MaxHeight(300).Gap(0).Children(func() {
				for i, o := range options {
					i, o := i, o
					row := ui.Button(c, "").Bind(nav.Rows[i]).FillWidth().MinHeight(menuRowHeight).Padding(6, 8).Radius(8).BorderWidth(0).
						Background(ui.Transparent).AlignItems(ui.Center)
					if row.Hovered() || i == nav.Active {
						row = row.Background(chrome.Color(colors.Hover))
						nav.Active = i
					}
					row.Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Children(func() {
							ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Text(c, o.Label).Grow(1).MinWidth(0).SingleLine().FontSize(13).TextColor(opt.Text)
								if o.ID == selectedID {
									ui.Icon(c, checkIcon).Size(14, 14).Shrink(0)
								}
							})
							if o.Description != "" {
								ui.Text(c, o.Description).FontSize(11).MaxLines(2).TextColor(chrome.Color(colors.Muted))
							}
						})
					})
					if row.Clicked() && pick != nil {
						*open = false
						nav.Focused = false
						pick(o.ID)
						nav.Anchor.RequestFocus(c)
					}
				}
			})
			ui.Box(c).FillWidth().Height(1).Shrink(0).Margin(4, 6).Background(chrome.Color(colors.Caption).Alpha(.35))
			ui.Text(c, i18n.T(i18n.KeyComposerMenuFooter)).FontSize(10).
				TextColor(chrome.Color(colors.Caption)).Padding(2, 6)
		})
		// The current row takes the focus once as the popup opens, so the arrows
		// and Enter work at once, the way a native select does. Later frames
		// leave the focus wherever the user put it.
		if !nav.Focused && len(nav.Rows) > 0 {
			nav.Focused = true
			nav.Rows[nav.Active].RequestFocus(c)
		}
	})
	return trigger
}

// resolvePresetCaption turns the Host's current caption into the product label
// and the matching choice ID; an unknown caption is shown verbatim.
func resolvePresetCaption(caption string, options []presetChoice) (display, id string) {
	if caption == "" {
		return "", ""
	}
	for _, o := range options {
		if o.Raw == caption || o.Label == caption || o.ID == caption {
			return o.Label, o.ID
		}
	}
	return caption, ""
}

func permissionChoices(options []session.PermissionOption) []presetChoice {
	out := make([]presetChoice, 0, len(options))
	for _, o := range options {
		out = append(out, presetChoice{ID: o.Value, Label: permissionLabel(o.Value, o.Label), Raw: o.Label})
	}
	return out
}

func agentPresetChoices(options []session.AgentPresetOption) []presetChoice {
	out := make([]presetChoice, 0, len(options))
	for _, o := range options {
		detail := o.Description
		if o.Broken != "" {
			if detail != "" {
				detail += " · "
			}
			detail += o.Broken
		}
		label := o.Label
		if localized, ok := builtinPresetName(o.ID); ok {
			label = localized
		}
		out = append(out, presetChoice{ID: o.ID, Label: label, Raw: o.Label, Description: detail, Broken: o.Broken})
	}
	return out
}

// permissionLabel is the one place a permission preset becomes user-facing
// copy: the built-in values map to the product's own names, and a label the
// Host deliberately renamed is kept verbatim.
func permissionLabel(value, hostLabel string) string {
	if product, ok := builtinPermissionLabel(value); ok {
		if hostLabel == "" || hostLabel == value || hostLabel == product.en || hostLabel == product.zh {
			return i18n.T(product.key)
		}
	}
	if hostLabel == "" {
		return value
	}
	return hostLabel
}

type permissionProductLabel struct {
	key string
	en  string
	zh  string
}

// Permission preset values are part of the Host protocol; the two the shared
// session package does not name are spelled here.
const (
	permissionReadOnly       = "read-only"
	permissionWorkspaceWrite = "workspace-write"
)

func builtinPermissionLabel(value string) (permissionProductLabel, bool) {
	switch value {
	case permissionReadOnly:
		return permissionProductLabel{key: i18n.KeyComposerPermissionReadOnly, en: "Read Only", zh: "仅可查看"}, true
	case permissionWorkspaceWrite:
		return permissionProductLabel{key: i18n.KeyComposerPermissionWorkspace, en: "Workspace Write", zh: "工作区内修改"}, true
	case session.PermissionFullAccess:
		return permissionProductLabel{key: i18n.KeyComposerPermissionFullAccess, en: "Full access", zh: "完全权限"}, true
	case session.PermissionAutoReview:
		return permissionProductLabel{key: i18n.KeyComposerPermissionAutoReview, en: "Auto review", zh: "自动审查"}, true
	}
	return permissionProductLabel{}, false
}
