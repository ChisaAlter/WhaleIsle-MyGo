package settings

import (
	"fmt"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// The color well retains the native anchored picker, keyboard focus and Escape
// handling. Empty effect colors deliberately inherit the current theme.
func appearanceColorWell(c *ui.Context, value *string, label string, disabled, inherit bool) bool {
	changed := false
	ui.Row(c.Key("color-well-" + label)).Gap(8).AlignItems(ui.Center).Children(func() {
		color := c.Theme().Accent
		if *value != "" {
			color = chrome.Color(*value)
		}
		if ui.ColorWell(c, &color).Label(label).Size(36, 32).Disabled(disabled).Changed() {
			*value = fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B)
			changed = true
		}
		text := *value
		if text == "" {
			text = providerCopy("跟随主题", "Theme")
		}
		ui.Text(c, text).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
		if inherit && *value != "" && coss.Button(c, "↺", coss.ButtonOptions{
			Variant:  coss.ButtonOutline,
			Size:     coss.ButtonIconSM,
			Label:    label + " " + providerCopy("跟随主题", "Use theme"),
			Disabled: disabled,
			Content: func() {
				kText(c, "↺", design.TextSM, design.WeightMedium, tk(c).Foreground).PassThrough()
			},
		}).Clicked() {
			*value = ""
			changed = true
		}
	})
	return changed
}

// Visual palettes span the preference width, below their heading; forcing
// their wrapping grid into a trailing pill column would overflow the dialog.
func appearanceChoiceRow(key string) bool {
	switch key {
	case "background-preset", "cursor-effect", "cursor-preset", "effect-choice-palette.id", "effect-choice-colors.id":
		return true
	}
	return false
}

func appearanceChoiceCards(c *ui.Context, key string, value *string, choices [][2]string, swatches func(string) []string, disabled bool) bool {
	changed := false
	ui.Row(c.Key(key)).FillWidth().Wrap().Gap(8).Children(func() {
		for _, choice := range choices {
			id, label := choice[0], choice[1]
			button := chrome.FrostSurface(ui.ButtonBase(c.Key(id)).Label(label).Width(104).MinHeight(68).Padding(10), theme.Default().Active(c.Theme().Dark), c.Theme().Dark, c.Theme().Surface, 12).Border(.5, c.Theme().Border).Disabled(disabled)
			if *value == id {
				button.Border(2, c.Theme().Accent)
			}
			if button.Hovered() {
				button.Background(c.Theme().SurfaceHover)
			}
			button.Children(func() {
				ui.Column(c).FillWidth().Gap(8).Children(func() {
					if swatches != nil {
						ui.Row(c).FillWidth().Height(18).Gap(2).Children(func() {
							for _, hex := range swatches(id) {
								color := c.Theme().Accent
								if hex != "" {
									color = chrome.Color(hex)
								}
								ui.Box(c).Basis(0).Grow(1).Height(18).Radius(4).Background(color)
							}
						})
					}
					ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Text).SingleLine()
				})
			})
			if button.Clicked() {
				*value = id
				changed = true
			}
		}
	})
	return changed
}

func settingsHelpPopover(c *ui.Context, key, label string, body func()) {
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonIconXS,
		Label:   label,
		Key:     key,
		Content: func() {
			ui.Icon(c, bodyIcon("help")).Size(16, 16).TextColor(tk(c).MutedForeground).PassThrough()
		},
	}).Shrink(0)
	open := ui.Local(button, "open", func() bool { return false })
	pinned := ui.Local(button, "pinned", func() bool { return false })
	wasFocused := ui.Local(button, "focused", func() bool { return false })
	wasHovered := ui.Local(button, "hovered", func() bool { return false })
	hovered := button.Hovered()
	panelHovered := false
	if button.Clicked() {
		*pinned = !*pinned
		*open = *pinned
	} else if button.Focused() && !*wasFocused || hovered && !*wasHovered {
		*open = true
	}
	*wasFocused = button.Focused()
	*wasHovered = hovered
	ui.PopoverBase(c, button, open, func(panel ui.Element) {
		panelHovered = panel.Hovered() || panel.FocusWithin()
		vw, vh := c.Size()
		panel = chrome.PopoverReveal(c, chrome.FrostSurface(panel.AttachTo(button, ui.AnchorRight, ui.AnchorLeft).Margin(0, 0, 0, 6).Width(min(float32(360), max(float32(0), vw-32))).MaxWidthPercent(90).MaxHeight(max(float32(0), vh-32)).Padding(16), theme.Default().Active(c.Theme().Dark), c.Theme().Dark, c.Theme().Surface, 12).Border(.5, c.Theme().Border).Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, .16)))
		panel.Children(func() {
			ui.Scroll(c).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(float32(0), vh-64)).Gap(10).Children(body)
		})
	})
	if !*pinned && !button.Focused() && !hovered && !panelHovered {
		*open = false
	}
	if !*open {
		*pinned = false
	}
}

type rowHelp struct {
	key, label string
	body       func()
}

var pendingRowHelp *rowHelp

func withRowHelp(help rowHelp, render func()) {
	pendingRowHelp = &help
	defer func() { pendingRowHelp = nil }()
	render()
}
func takeRowHelp() *rowHelp { help := pendingRowHelp; pendingRowHelp = nil; return help }
func preferenceRowTitle(c *ui.Context, title string, fg ui.Color, help *rowHelp) {
	ui.Row(c).FillWidth().MinWidth(0).Gap(4).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).MinWidth(0).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
		if help != nil {
			settingsHelpPopover(c, help.key, help.label, help.body)
		}
	})
}

func appearancePresetChoices(cursor bool) [][2]string {
	choices := [][2]string{{"default", providerCopy("跟随主题", "Follow theme")}}
	if cursor {
		choices = append(choices, [2]string{"rainbow", providerCopy("彩虹", "Rainbow")})
	}
	return append(choices, [][2]string{{"aurora", providerCopy("极光", "Aurora")}, {"sunset", providerCopy("晚霞", "Sunset")}, {"ocean", providerCopy("海洋", "Ocean")}, {"sakura", providerCopy("樱花", "Sakura")}, {"custom", providerCopy("自定义", "Custom")}}...)
}
func effectChoiceOptionLabel(typing bool, path, value string) string {
	if !typing && path == "mode" && value == "aurora" {
		return "Aurora"
	}
	if typing && (path == "colors.id" || path == "palette.id") && value == "sunset" {
		return providerCopy("落日", "Sunset")
	}
	return effectOptionLabel(value)
}
func effectOptionLabel(value string) string {
	labels := map[string][2]string{
		"legacy": {"Legacy", "Legacy"}, "lounge": {"Lounge", "Lounge"}, "aurora": {"极光", "Aurora"}, "reactive": {"Reactive", "Reactive"}, "custom": {"自定义", "Custom"},
		"clockwise": {"顺时针", "Clockwise"}, "counterclockwise": {"逆时针", "Counterclockwise"}, "pingPong": {"往返", "Ping-pong"}, "pingpong": {"往返", "Ping-pong"}, "linear": {"线性", "Linear"}, "ease": {"缓入缓出", "Ease"}, "ease-in-out": {"平滑缓入缓出", "Ease in/out"}, "cubic-bezier": {"平滑曲线", "Smooth curve"}, "smoothstep": {"平滑曲线", "Smooth curve"},
		"preset": {"预设", "Preset"}, "theme": {"跟随主题", "Follow theme"}, "drop": {"落锤", "Drop"}, "rise": {"上浮", "Rise"}, "flash": {"闪现", "Flash"}, "native": {"跟随系统", "System caret"}, "system": {"跟随系统", "System caret"}, "block": {"方块", "Block"}, "underline": {"下划线", "Underline"},
		"warm": {"暖色", "Warm"}, "rose": {"玫瑰", "Rose"}, "glacier": {"冰川", "Glacier"}, "cyber": {"赛博", "Cyber"}, "sunset": {"日落", "Sunset"}, "cyan": {"青色", "Cyan"}, "ocean": {"海洋", "Ocean"}, "candy": {"糖果", "Candy"}, "forest": {"森林", "Forest"}, "violet": {"紫藤", "Violet"}, "ember": {"熔金", "Ember"},
	}
	if pair, ok := labels[value]; ok {
		return providerCopy(pair[0], pair[1])
	}
	return value
}
