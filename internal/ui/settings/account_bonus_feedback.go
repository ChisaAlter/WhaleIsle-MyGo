package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func accountBonusFeedback(c *ui.Context, a *AccountAccess, fg, muted ui.Color) {
	textFor := func(zh, en string) string {
		if i18n.Locale() == i18n.LocaleZH {
			return zh
		}
		return en
	}
	settingsNotice(c, "account_bonus_feedback:17", "error", a.BonusesError != "", func() string { return textFor("奖励通知读取失败，请刷新重试：", "Could not load reward notices; refresh to retry: ")+a.BonusesError })
	var message string
	switch a.BonusAckState {
	case "pending":
		message = textFor("正在确认已读…", "Confirming notice…")
	case "acknowledged":
		message = textFor("已确认这条奖励通知。", "Reward notice acknowledged.")
	case "refused":
		message = textFor("当前账户无法确认这条通知，请刷新账户状态。", "This account cannot acknowledge the notice. Refresh the account state.")
	case "failed":
		message = textFor("确认失败，可再次点击重试：", "Could not acknowledge; click again to retry: ") + a.BonusAckError
	}
	settingsNotice(c, "account_bonus_feedback:31", "", message != "", func() string { return message })
}
