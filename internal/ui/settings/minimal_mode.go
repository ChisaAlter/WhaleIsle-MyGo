package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/theme"
)

// MinimalModeAccess is the appearance page's 极简模式 block: the switch that
// pauses the effects the page below edits, and the startup offer that can turn
// it back on.
type MinimalModeAccess struct {
	Enabled             *bool
	Suggestions         *bool
	OnToggle            func(bool) error
	OnToggleSuggestions func(bool) error
}

// minimalModeDescription says what the switch pauses and, just as important,
// what it keeps: the preferences below and the move that restores them.
func minimalModeDescription() string {
	return providerCopy(
		"暂停壁纸、动态背景、透明模糊、指针与输入装饰、输入框流光和界面过渡；主题、字体和下面的偏好原样保留，关闭后恢复。",
		"Pauses wallpapers, dynamic backgrounds, transparent and blurred materials, pointer and typing decorations, the composer beam and interface transitions. Theme, fonts and the preferences below are kept, and come back when this is off.")
}

// minimalModeSwitchLabel names what the switch does next, as the composer's
// permission switch does: assistive technology then announces the action rather
// than repeating the heading above it.
func minimalModeSwitchLabel(enabled bool) string {
	if enabled {
		return providerCopy("关闭极简模式", "Turn off minimal mode")
	}
	return providerCopy("开启极简模式", "Turn on minimal mode")
}

func minimalSuggestionSwitchLabel(enabled bool) string {
	if enabled {
		return providerCopy("关闭启动卡顿提示", "Turn off startup suggestions")
	}
	return providerCopy("开启启动卡顿提示", "Turn on startup suggestions")
}

func minimalSuggestionDescription() string {
	return providerCopy(
		"启动明显卡顿时提示开启极简模式；每次启动最多提示一次。",
		"Offer minimal mode when startup noticeably stutters; at most once per launch.")
}

// minimalOverrideHint appends the override note to a row's hint, or a notice
// under a block that has no hint of its own. A paused row stays editable: the
// user is tuning what comes back when minimal mode is off, so the note explains
// why nothing moved rather than taking the control away.
func minimalOverrideHint(hint string) string {
	if !renderpolicy.Minimal() {
		return hint
	}
	note := providerCopy("被极简模式覆盖", "Overridden by minimal mode")
	if hint == "" {
		return note
	}
	return hint + " · " + note
}

func minimalModePane(c *ui.Context, m *MinimalModeAccess, fg, muted ui.Color) {
	if m == nil || m.Enabled == nil {
		return
	}
	block := ui.Column(c.Key("appearance-minimal-mode")).FillWidth().MaxWidth(720).MinWidth(0).Gap(12)
	settingsSearchTarget(c, "appearance-minimal-mode", providerCopy("极简模式", "Minimal mode"), block)
	failure := ui.Local(block, "error", func() string { return "" })
	block.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, providerCopy("极简模式", "Minimal mode")).FillWidth().FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
				ui.Text(c, minimalModeDescription()).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
			})
			enabled := *m.Enabled
			if settingsPreferenceSwitch(c, &enabled, settingsSwitchInput).Label(minimalModeSwitchLabel(enabled)).Disabled(m.OnToggle == nil).Changed() && m.OnToggle != nil {
				if err := m.OnToggle(enabled); err != nil {
					*failure = errcopy.Sentence(err)
				} else {
					*m.Enabled, *failure = enabled, ""
				}
			}
		})
		settingsNotice(c, "appearance-minimal-mode:error", "error", *failure != "", func() string { return *failure })
	})
	if m.Suggestions != nil {
		suggestions := *m.Suggestions
		bodyPreferenceRow(c, "appearance-minimal-suggestions", providerCopy("启动卡顿时提示", "Suggest on slow startup"), minimalSuggestionDescription(), fg, muted, func() {
			if settingsPreferenceSwitch(c, &suggestions, settingsSwitchInput).Label(minimalSuggestionSwitchLabel(suggestions)).Disabled(m.OnToggleSuggestions == nil).Changed() && m.OnToggleSuggestions != nil {
				if err := m.OnToggleSuggestions(suggestions); err != nil {
					*failure = errcopy.Sentence(err)
				} else {
					*m.Suggestions, *failure = suggestions, ""
				}
			}
		})
	}
}
