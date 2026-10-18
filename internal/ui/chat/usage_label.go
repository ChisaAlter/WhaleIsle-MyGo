package chat

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

const usageBalanceMaxRunes = 28

// FormatUsageLabel builds a compact one-line strip for jobs today and/or balance.
// Returns "" when there is nothing useful to show (hides the label).
// jobsToday nil means the jobs segment is omitted; non-nil shows even when zero.
func FormatUsageLabel(jobsToday *int, balance string) string {
	var parts []string
	if jobsToday != nil {
		parts = append(parts, fmt.Sprintf("%s %d", i18n.T(i18n.KeyChatUsageJobsToday), *jobsToday))
	}
	if bal := CompactUsageBalance(balance); bal != "" {
		parts = append(parts, fmt.Sprintf("%s %s", i18n.T(i18n.KeyChatUsageBalance), bal))
	}
	return strings.Join(parts, " · ")
}

// CompactUsageBalance shortens a balance stub for the chat strip.
// Empty, error, and placeholder responses hide the balance segment.
func CompactUsageBalance(balance string) string {
	s := strings.TrimSpace(balance)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "(account/getBalance:") || s == "(empty balance response)" {
		return ""
	}
	if utf8.RuneCountInString(s) <= usageBalanceMaxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:usageBalanceMaxRunes]) + "…"
}

// UsageLabelView paints a one-line usage strip above the message list.
// Empty text hides it. When onOpen is set, a click opens the usage panel.
func UsageLabelView(c *ui.Context, text string, muted, field, border ui.Color, onOpen func()) {
	if text == "" {
		return
	}
	ui.Column(c).Background(field).Children(func() {
		row := ui.Row(c).Padding(4, 12, 4, 12).Gap(6).AlignItems(ui.Center)
		row.Children(func() {
			ui.Text(c, text).FontSize(11).TextColor(muted).Grow(1)
		})
		if onOpen != nil && row.Clicked() {
			onOpen()
		}
		ui.Box(c).Fill().Height(1).Background(border)
	})
}
