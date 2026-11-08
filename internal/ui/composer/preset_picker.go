package composer

import (
	"strings"
	"unicode"

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
	PaddingX float32
	Gap      float32
}

type presetNavigation struct {
	Active       int
	HoveredIndex int
	ScrollActive bool
	Focused      bool
	Rows         []*ui.Handle
	Anchor       ui.Handle
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
	owner := ui.Box(c).Absolute().Size(0, 0).PassThrough()
	open := ui.Local(owner, "preset-open", func() bool { return false })
	nav := ui.Local(owner, "preset-navigation", func() presetNavigation { return presetNavigation{} })
	display, selectedID := resolvePresetCaption(caption, options)
	trigger := chip(c, colors, *open, opt.Text)
	if opt.PaddingX > 0 {
		trigger = trigger.Padding(0, opt.PaddingX)
	}
	if opt.Gap > 0 {
		trigger = trigger.Gap(opt.Gap)
	}
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
		nav.HoveredIndex = -1
		nav.ScrollActive = true
	}
	if !*open {
		nav.Focused = false
		nav.HoveredIndex = -1
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
		panel = menuSurface(c, panel.Padding(4).Gap(0).MinWidth(144).MaxWidth(360).
			AttachTo(trigger, ui.AnchorBottomLeft, ui.AnchorTopLeft).Margin(4, 0, 0, 0), colors)
		if panel.Shortcut(0, ui.KeyDown) {
			nav.Active = (nav.Active + 1) % len(options)
			nav.ScrollActive = true
			nav.Rows[nav.Active].RequestFocus(c)
		}
		if panel.Shortcut(0, ui.KeyUp) {
			nav.Active = (nav.Active + len(options) - 1) % len(options)
			nav.ScrollActive = true
			nav.Rows[nav.Active].RequestFocus(c)
		}
		panel.Children(func() {
			ui.Scroll(c).FillWidth().MaxHeight(300).Gap(0).Children(func() {
				hoveredRow := -1
				for i, o := range options {
					i, o := i, o
					row := menuRow(c, colors).Bind(nav.Rows[i])
					if row.Hovered() {
						hoveredRow = i
						if nav.HoveredIndex != i {
							nav.HoveredIndex = i
							nav.Active = i
						}
					}
					if i == nav.Active {
						row = row.Background(chrome.Color(colors.Hover))
						if nav.ScrollActive {
							row.ScrollIntoView()
							nav.ScrollActive = false
						}
					}
					row.Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).MaxWidth(280).Gap(2).Children(func() {
							ui.Text(c, o.Label).MinWidth(0).SingleLine().FontSize(13).FixedLineHeight(20).TextColor(opt.Text)
							if o.Description != "" {
								ui.Text(c, o.Description).FontSize(12).FixedLineHeight(16).MaxLines(2).TextColor(chrome.Color(colors.Caption))
							}
						})
						if o.ID == selectedID {
							ui.Icon(c, checkIcon).Size(14, 14).Shrink(0).TextColor(chrome.Color(colors.FG))
						}
					})
					if row.Clicked() && pick != nil {
						*open = false
						nav.Focused = false
						pick(o.ID)
						nav.Anchor.RequestFocus(c)
					}
				}
				if hoveredRow < 0 {
					nav.HoveredIndex = -1
				}
			})
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
		out = append(out, presetChoice{ID: o.Value, Label: permissionLabel(o.Value, o.Label), Raw: o.Label, Description: permissionDescription(o.Value, o.Description)})
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
	if value == session.PermissionAutoReview {
		return i18n.T(i18n.KeyComposerPermissionAutoReview)
	}
	if product, ok := builtinPermissionLabel(value); ok {
		if hostLabel == "" || hostLabel == value || hostLabel == product.en || hostLabel == product.zh {
			return i18n.T(product.key)
		}
	}
	if hostLabel == "" {
		hostLabel = value
	}
	// The installed presentation helper title-cases conventional preset keys,
	// while intentionally configured display names retain their spelling.
	for _, r := range hostLabel {
		if r != '-' && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return hostLabel
		}
	}
	words := strings.Split(hostLabel, "-")
	for i, word := range words {
		if word == "" {
			return hostLabel
		}
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
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
