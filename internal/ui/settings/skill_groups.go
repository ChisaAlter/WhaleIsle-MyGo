package settings

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
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
		ui.Row(c).FillWidth().MinWidth(0).Gap(6).Wrap().Children(func() {
			for _, label := range append([]string(nil), (*groups)...) {
				chip := ui.Button(c.Key("selected/"+label), "").Label(providerCopy("移除分组 · ", "Remove group · ")+label).MaxWidthPercent(100).MinWidth(0).Height(28).Gap(4).Padding(0, 10).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).Disabled(disabled)
				chip.Children(func() {
					ui.Text(c, label).MinWidth(0).SingleLine().Tooltip(label).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
					ui.Text(c, "×").Shrink(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				})
				if chip.Clicked() {
					remove(label)
				}
			}
		})
		ui.Row(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
			input = ui.TextInput(c, text).Grow(1).MinWidth(0).Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Label(providerCopy("新分组", "New group")).Disabled(disabled)
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
			if ui.Button(c, "⌄").Label(providerCopy("选择分组", "Choose groups")).Role(ui.RoleDisclosure).Expanded(*open).Size(32, 32).Disabled(disabled).Clicked() {
				*open = !*open
			}
		})
	})
	ui.PopoverBase(c.Key("group-menu"), input, open, func(panel ui.Element) {
		width, height := c.Size()
		panel = chrome.MenuSurface(panel.Width(min(input.Bounds().W, max(0, width-24))).Padding(6).Gap(2).Role(ui.RoleMenu), c.Theme().Background, c.Theme().Border, 8)
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
					option := ui.Button(c.Key("known/"+label), "").Label(label).Role(ui.RoleMenuItemCheckBox).Checked(selected).FillWidth().MinWidth(0).Gap(8).Justify(ui.Start).BorderWidth(0).Background(ui.Transparent)
					option.Children(func() {
						check := ""
						if selected {
							check = "✓"
						}
						ui.Text(c, check).Width(16).Shrink(0).FontSize(fonttheme.InterfaceSize(13)).TextColor(fg)
						ui.Text(c, label).Grow(1).MinWidth(0).SingleLine().Tooltip(label).FontSize(fonttheme.InterfaceSize(13)).TextColor(fg)
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
			if ui.Button(c, providerCopy("清空分组", "Clear groups")).Role(ui.RoleMenuItem).FillWidth().Justify(ui.Start).Disabled(len(*groups) == 0).Clicked() {
				*groups = []string{}
			}
		})
	})
}
