package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
)

// UsagePreferences preserves the existing two-row caller. The embedded page
// paints these settings below its toolbar and above the statistics body.
func UsagePreferences(c *ui.Context, host *BodySettingsAccess, fg, muted ui.Color) {
	UsagePreferencesWithContributions(c, host, pluginui.PublicContributions{}, fg, muted)
}

// UsagePreferencesWithContributions consumes settings.usage.item at that exact
// original settingsRows position, in source order and with the same Host writes.
func UsagePreferencesWithContributions(c *ui.Context, host *BodySettingsAccess, contributions pluginui.PublicContributions, fg, muted ui.Color) {
	finishPreferenceErrors := beginBodyPreferenceErrorNotices(c)
	defer finishPreferenceErrors()
	draw := func(id string) {
		switch id {
		case "session-cost":
			bodyHostPreference(c, host, "ui-conversation", "sessionCost", providerCopy("会话累计费用", "Session cost"), providerCopy("在输入框下方显示峰谷状态条与当前会话累计费用。", "Show the peak/valley strip and this session's accumulated cost below the composer."), fg, muted)
		case "official-peak-valley":
			bodyHostPreference(c, host, "ui-conversation", "officialPeakValley", providerCopy("官方峰谷时", "Official peak and valley hours"), providerCopy("会话累计费用开启时，在会话统计下方显示官方峰谷时段状态条。开启后始终显示；关闭时仅在检测到 DeepSeek API 路线时显示。", "With session cost on, show the official peak/valley strip below the statistics. On always shows it; off shows it only on a detected DeepSeek API route."), fg, muted)
		}
	}
	entries := []pluginui.PublicSlotEntry{}
	if contributions.Entries != nil && contributions.RenderEntry != nil {
		entries = contributions.Entries("settings.usage.item", "")
	}
	if len(entries) == 0 {
		if contributions.NativeWinner("settings.usage.item", "", "session-cost") {
			draw("session-cost")
		}
		if contributions.NativeWinner("settings.usage.item", "", "official-peak-valley") {
			draw("official-peak-valley")
		}
		return
	}
	for _, entry := range entries {
		if entry.Native {
			draw(entry.ID)
		} else {
			contributions.RenderEntry(c, entry.Slot, entry.OwnerKey, entry.Handle)
		}
	}
}
