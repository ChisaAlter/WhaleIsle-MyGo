package chat

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
func UsageLabelView(c *ui.Context, text string, _, _, _ ui.Color, onOpen func()) {
	if text == "" {
		return
	}
	t := design.For(c)
	ui.Column(c).Background(t.Muted).Children(func() {
		row := ui.Row(c).Padding(design.Sp(1), design.Sp(3), design.Sp(1), design.Sp(3)).Gap(design.Sp(1.5)).AlignItems(ui.Center)
		if onOpen != nil {
			row.Cursor(ui.CursorPointer)
		}
		row.Children(func() {
			ui.Text(c, text).FontSize(11).TextColor(t.MutedForeground).Grow(1)
		})
		if onOpen != nil && row.Clicked() {
			onOpen()
		}
		coss.Separator(c, coss.Horizontal)
	})
}
