package settings

import (
	"whaleisle/internal/accountlogin"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

func AccountSignInDialog(c *ui.Context, a *AccountAccess, visible bool, onClose, onAPIKey func()) {
	if !visible || a == nil || a.Status == "credential-stored" {
		return
	}
	active := a.SignInBusy || accountlogin.Active(a.AttemptPhase)
	busy := a.SignInBusy || a.AttemptPhase == "committing"
	failed := a.Err != "" || a.AttemptPhase == "failed"
	expired := a.AttemptPhase == "expired"
	title := providerCopy("登录账户", "Sign in")
	if active {
		title = providerCopy("在浏览器中完成登录", "Continue in your browser")
	}
	if expired {
		title = providerCopy("登录已超时", "Sign-in timed out")
	}
	if failed {
		title = providerCopy("登录失败", "Sign-in failed")
	}
	dismiss := func() {
		if busy {
			return
		}
		if accountlogin.Active(a.AttemptPhase) && a.SignInAttemptID != "" && a.OnCancelSignIn != nil {
			a.OnCancelSignIn()
		} else if onClose != nil {
			onClose()
		}
	}
	open := true
	accountDialog(c, &open, title, 380, busy, func() {
		description := providerCopy("登录 DeepSeek 账号以开始使用", "Use your DeepSeek account to get started.")
		if active {
			description = providerCopy("请在打开的浏览器页面中完成授权。完成后将自动返回。", "Complete authorization in your browser. You will return here automatically.")
		}
		if expired {
			description = providerCopy("登录链接已过期，请重新登录。", "This sign-in link expired. Try again.")
		}
		if failed {
			description = accountlogin.FailureMessage(a.AttemptPhase, a.AttemptErrorCode)
			if description == "" {
				description = a.Err
			}
		}
		if failed {
			kError(c, description)
		} else {
			kBody(c, description).TextColor(tk(c).MutedForeground)
		}
		if active && a.AuthorizeURL != "" && a.OnCopyAuthorize != nil && coss.Button(c, providerCopy("复制登录链接", "Copy sign-in link"), coss.ButtonOptions{Variant: coss.ButtonLink, Size: coss.ButtonSM}).AlignSelf(ui.Start).Padding(0).Clicked() {
			a.OnCopyAuthorize(a.AuthorizeURL)
		}
	}, func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Wrap().Justify(ui.End).Children(func() {
			secondary := providerCopy("使用 API Key", "Use API key")
			if active {
				secondary = i18n.T(i18n.KeySettingsWhaleCancel)
			}
			if coss.Button(c, secondary, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonLG, Disabled: busy}).Clicked() {
				if active {
					dismiss()
				} else {
					if onClose != nil {
						onClose()
					}
					if onAPIKey != nil {
						onAPIKey()
					}
				}
			}
			label := i18n.T(i18n.KeySettingsAccountSignIn)
			if active && !failed {
				label = providerCopy("等待登录…", "Waiting…")
			} else if failed || expired {
				label = providerCopy("重试", "Retry")
			}
			if coss.Button(c, label, coss.ButtonOptions{Size: coss.ButtonLG, Disabled: busy || active && !failed || !a.ClientReady || a.OnSignIn == nil}).Clicked() {
				a.OnSignIn()
			}
		})
	})
	if !open {
		dismiss()
	}
}
