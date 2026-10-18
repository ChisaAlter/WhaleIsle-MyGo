package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func accountBonusFeedback(c *ui.Context, a *AccountAccess, fg, muted ui.Color) {
	textFor := func(zh, en string) string {
		if i18n.Locale() == i18n.LocaleZH {
			return zh
		}
		return en
	}
	if a.BonusesLoading {
		ui.Text(c, textFor("正在读取奖励通知…", "Loading reward notices…")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if a.BonusesError != "" {
		ui.Text(c, textFor("奖励通知读取失败，请刷新重试：", "Could not load reward notices; refresh to retry: ")+a.BonusesError).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
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
	if message != "" {
		ui.Text(c, message).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
}
