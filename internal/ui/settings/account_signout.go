package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/accountlogout"
	"whaleisle/internal/ui/i18n"
)

func signOutText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}
func signOutError(code string) string {
	switch code {
	case accountlogout.ErrorIdentityUnavailable:
		return signOutText("无法核验当前账户身份，未发送退出请求。请刷新账户后重新操作。", "Could not verify the current account identity. No sign-out request was sent. Refresh the account and try again.")
	case accountlogout.ErrorAccountChanged:
		return signOutText("账户身份已改变，旧确认已失效。请重新打开退出确认。", "The account changed, so the previous confirmation is no longer valid. Open a new sign-out review.")
	case accountlogout.ErrorImpactChanged:
		return signOutText("任务状态已改变，请查看最新影响后再次确认。", "Task activity changed. Review the updated impact and confirm again.")
	case accountlogout.ErrorFailed:
		return signOutText("退出未完成，已确认该账户仍处于登录状态。请重新确认后重试。", "Sign-out did not complete. This account is still signed in. Confirm again to retry.")
	case accountlogout.ErrorOutcomeUnknown:
		return signOutText("退出结果尚未确认。请先检查账户状态，不要重复提交。关闭提示不会撤销已发出的请求。", "The sign-out outcome is unknown. Check the account state before trying again. Closing this notice does not undo the request.")
	}
	return ""
}
func accountSignOutPane(c *ui.Context, a *AccountAccess, fg, muted ui.Color) {
	s := a.SignOut
	if s.Error != "" {
		ui.Text(c, signOutError(s.Error)).FontSize(12).TextColor(fg)
	}
	if !s.Open {
		return
	}
	ui.Column(c).Gap(8).Padding(12).Children(func() {
		ui.Text(c, signOutText("确认退出账户", "Confirm account sign-out")).FontSize(14).Bold().TextColor(fg)
		if s.AccountID != "" {
			ui.Text(c, signOutText("账户：", "Account: ")+s.AccountID).FontSize(12).TextColor(muted)
		}
		message := signOutText("退出登录不会丢失任何数据，你仍可以登录此账号。", "Signing out will not delete any data. You can sign in to this account again.")
		if s.Impact == accountlogout.ImpactRunning {
			message = signOutText("当前有使用此账户的任务正在运行。退出登录可能中断这些任务，是否继续？", "Tasks using this account are running. Signing out may interrupt them. Continue?")
		}
		if s.Impact == accountlogout.ImpactUnknown {
			message = signOutText("暂时无法确认任务状态。退出登录可能会中断使用此账号的任务，是否继续？", "Could not check running tasks. Signing out may interrupt tasks using this account. Continue?")
		}
		ui.Text(c, message).FontSize(12).TextColor(fg)
		if s.Busy {
			ui.Text(c, signOutText("正在处理，请稍候…", "Working, please wait…")).FontSize(12).TextColor(muted)
		}
		ui.Row(c).Gap(8).Children(func() {
			cancel := signOutText("取消", "Cancel")
			if s.Unknown {
				cancel = signOutText("关闭提示", "Close notice")
			}
			if a.OnCancelSignOut != nil && ui.Button(c, cancel).Disabled(s.Busy).Clicked() {
				a.OnCancelSignOut()
			}
			if s.Unknown {
				if a.OnCheckSignOut != nil && ui.Button(c, signOutText("检查退出结果", "Check sign-out result")).Disabled(s.Busy).Clicked() {
					a.OnCheckSignOut()
				}
			} else if a.OnConfirmSignOut != nil && ui.Button(c, signOutText("确认退出账户", "Confirm sign out")).Disabled(s.Busy || s.AccountID == "").Clicked() {
				a.OnConfirmSignOut()
			}
		})
	})
}
