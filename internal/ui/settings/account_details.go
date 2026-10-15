package settings

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/accountdetails"
	"whaleisle/internal/ui/i18n"
)

func accountDetailsPane(c *ui.Context, state accountdetails.State, fg, muted ui.Color) {
	textFor := func(zh, en string) string {
		if i18n.Locale() == i18n.LocaleZH {
			return zh
		}
		return en
	}
	row := func(label, value string) {
		ui.Row(c).Gap(12).Children(func() {
			ui.Text(c, label).FontSize(12).TextColor(muted).Width(140)
			ui.Text(c, emptyDash(value)).FontSize(12).TextColor(fg).Grow(1)
			if value != "" && ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(11).Clicked() {
				mygo.Clipboard.WriteText(value)
			}
		})
	}
	status := func(s accountdetails.Status) bool {
		var text string
		switch s {
		case accountdetails.Ready:
			return true
		case accountdetails.Loading:
			text = textFor("正在加载…", "Loading…")
		case accountdetails.Failed:
			text = textFor("读取失败，请刷新重试", "Could not load; refresh to retry")
		default:
			text = textFor("未知（尚未取得当前账户数据）", "Unknown (current account data unavailable)")
		}
		ui.Text(c, text).FontSize(12).TextColor(muted)
		return false
	}
	value := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	ui.Text(c, textFor("账户资料", "Profile")).FontSize(13).Bold().TextColor(fg).Padding(8, 0, 0, 0)
	if status(state.Profile.Status) {
		p := state.Profile.Value
		row(textFor("账户 ID", "Account ID"), value(p.ID))
		row(textFor("名称", "Name"), value(p.Name))
		row(textFor("联系方式", "Contact"), value(p.Contact))
		row(textFor("头像地址", "Avatar URL"), value(p.AvatarURL))
	}
	ui.Text(c, textFor("钱包余额", "Wallets")).FontSize(13).Bold().TextColor(fg).Padding(8, 0, 0, 0)
	if !status(state.Balance.Status) {
		return
	}
	wallets := func(title string, ws []accountdetails.Wallet) {
		ui.Text(c, title).FontSize(12).Bold().TextColor(fg)
		if len(ws) == 0 {
			ui.Text(c, textFor("无钱包", "No wallets")).FontSize(12).TextColor(muted)
			return
		}
		for _, w := range ws {
			row(w.Currency, w.Balance)
		}
	}
	wallets(textFor("充值钱包", "Recharge wallets"), state.Balance.Wallets)
	wallets(textFor("奖励钱包", "Bonus wallets"), state.Balance.BonusWallets)
}
