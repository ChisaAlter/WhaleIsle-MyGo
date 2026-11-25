package settings

import (
	"slices"
	"strings"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// The options menu stays anchored to the input. Existing labels remain exact
// array members; Enter or a typed comma commits a new label.
func skillGroupPicker(c *ui.Context, key string, groups *[]string, skills []SkillRow, disabled bool, fg, muted ui.Color) {
	box := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(6)
	text := ui.Local(box, "new-group", func() string { return "" })
	open := ui.Local(box, "group-options", func() bool { return false })
	composing := ui.Local(box, "group-composing", func() bool { return false })
	if disabled {
		*open = false
	}
	remove := func(label string) {
		next := make([]string, 0, len(*groups))
		for _, current := range *groups {
			if current != label {
				next = append(next, current)
			}
		}
		*groups = next
	}
	add := func() {
		label := strings.TrimSpace(*text)
		if label != "" && !slices.Contains(*groups, label) {
			*groups = append(*groups, label)
		}
		*text = ""
	}
	var input ui.Element
	box.Children(func() {
		ui.Text(c, providerCopy("分组", "Groups")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Wrap().Children(func() {
			for _, label := range append([]string(nil), (*groups)...) {
				chip := coss.Button(c, "", coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonSM,
					Label:    providerCopy("移除分组 · ", "Remove group · ") + label,
					Disabled: disabled,
					Key:      "selected/" + label,
					Content: func() {
						ui.Text(c, label).MinWidth(0).SingleLine().Tooltip(label).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).PassThrough()
						ui.Text(c, "×").Shrink(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).PassThrough()
					},
				}).MaxWidthPercent(100).MinWidth(0)
				if chip.Clicked() {
					remove(label)
				}
			}
		})
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
			input = kInputRow(c, text, coss.InputOptions{
				Label:    providerCopy("新分组", "New group"),
				Disabled: disabled,
			})
			input.HandleInput(func(event ui.InputEvent) bool {
				if event.Kind == ui.InputCompose {
					*composing = event.Text != ""
					return false
				}
				if disabled || *composing {
					return false
				}
				if event.Kind == ui.InputKeyDown && event.Mods == 0 && (event.Key == ui.KeyEnter || event.Key == ui.KeyComma) || event.Kind == ui.InputText && event.Text == "," {
					add()
					return true
				}
				return false
			})
			if coss.Button(c, "", coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonIcon,
				Icon:     coss.IconChevronDown,
				Label:    providerCopy("选择分组", "Choose groups"),
				Disabled: disabled,
			}).Role(ui.RoleDisclosure).Expanded(*open).Clicked() {
				*open = !*open
			}
		})
	})
	ui.PopoverBase(c.Key("group-menu"), input, open, func(panel ui.Element) {
		width, height := c.Size()
		panel = chrome.PopoverReveal(c, chrome.MenuSurface(panel.Width(min(input.Bounds().W, max(0, width-24))).Padding(4).Gap(2).Role(ui.RoleMenu), c.Theme().Surface, c.Theme().Border, 16))
		panel.Children(func() {
			ui.Scroll(c).FillWidth().MaxHeight(min(280, max(0, height-120))).MinHeight(0).Children(func() {
				known := []string{}
				for _, section := range skillSections(skills) {
					if section.Name != "" {
						known = append(known, section.Name)
					}
				}
				for _, label := range *groups {
					if !slices.Contains(known, label) {
						known = append(known, label)
					}
				}
				if typed := strings.TrimSpace(*text); typed != "" && !slices.Contains(known, typed) {
					known = append(known, typed)
				}
				for _, label := range known {
					selected := slices.Contains(*groups, label)
					option := ui.ButtonBase(c.Key("known/"+label)).Label(label).Role(ui.RoleMenuItemCheckBox).Checked(selected).FillWidth().MinWidth(0).Gap(design.Sp(2)).Padding(design.Sp(1.5), design.Sp(2)).Radius(design.RadiusMD).Justify(ui.Start).AlignItems(ui.Center).Cursor(ui.CursorPointer)
					if option.Hovered() {
						option.Background(tk(c).Accent)
					}
					option.Children(func() {
						check := ""
						if selected {
							check = "✓"
						}
						ui.Text(c, check).Width(16).Shrink(0).FontSize(fonttheme.InterfaceSize(13)).TextColor(fg).PassThrough()
						ui.Text(c, label).Grow(1).MinWidth(0).SingleLine().Tooltip(label).FontSize(fonttheme.InterfaceSize(13)).TextColor(fg).PassThrough()
					})
					if option.Clicked() {
						if selected {
							remove(label)
						} else {
							*groups = append(*groups, label)
						}
					}
				}
			})
			if kGhost(c, providerCopy("清空分组", "Clear groups"), len(*groups) == 0).Role(ui.RoleMenuItem).FillWidth().Justify(ui.Start).Clicked() {
				*groups = []string{}
			}
		})
	})
}
