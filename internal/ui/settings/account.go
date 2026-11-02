package settings

import (
	"math/big"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"

	"whaleisle/internal/accountdetails"
	"whaleisle/internal/accountlogin"
	"whaleisle/internal/accountlogout"
	"whaleisle/internal/product"
	"whaleisle/internal/ui/i18n"
)

// AccountAccess is the dedicated Account / Login settings page
// (account/getState + optional profile/balance from the usage store).
type AccountAccess struct {
	Details          accountdetails.State
	SignInBusy       bool
	ClientReady      bool
	Loading          bool
	Err              string
	Status           string
	Email            string
	Name             string
	Plan             string
	UsageURL         string
	TopUpURL         string
	BalanceSummary   string
	AttemptPhase     string
	AuthorizeURL     string
	OnContact        func()
	OnRefresh        func()
	OnOpenAuthorize  func(url string)
	OnOpenUsage      func(url string)
	OnOpenTopUp      func(url string)
	OnSignIn         func()
	OnSignOut        func()
	SignOut          accountlogout.State
	OnConfirmSignOut func()
	OnCancelSignOut  func()
	OnCheckSignOut   func()
	OnCancelSignIn   func()
	SignInAttemptID  string
	Bonuses          []AccountBonus
	BonusesLoading   bool
	BonusesError     string
	BonusAckState    string
	BonusAckError    string
	BonusAckAccount  string
	BonusAckOrder    string
	OnAckBonus       func(accountID, orderID string)
	HasRunningTasks  bool
}

// AccountBonus is one row from account/getUnnotifiedBonuses.
type AccountBonus struct {
	AccountID string
	OrderID   string
	Label     string
	Detail    string
}

func accountPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Account == nil {
		return
	}
	a := m.Account
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	active := accountlogin.Active(a.AttemptPhase)
	signedIn := a.Status == "credential-stored"
	if !signedIn && !active {
		ui.Column(c.Key("account-signed-out")).FillWidth().Gap(16).Padding(40, 0).Children(func() {
			ui.Text(c, providerCopy("登录 "+product.Name, "Sign in to "+product.Name)).FontSize(fonttheme.InterfaceSize(16)).FontWeight(600).TextColor(fg)
			message := providerCopy("登录后即可使用账户模型并查看余额与用量。", "Sign in to use account models and view your balance and usage.")
			if a.Loading {
				message = i18n.T(i18n.KeySettingsLoading)
			}
			if a.Err != "" {
				message = a.Err
			}
			if !a.ClientReady {
				message = providerCopy("连接 Host 后即可登录。", "Connect to a Host to sign in.")
			}
			ui.Text(c, message).FontSize(fonttheme.InterfaceSize(14)).TextColor(muted)
			if ui.Button(c, i18n.T(i18n.KeySettingsAccountSignIn)).AlignSelf(ui.Start).Height(32).Padding(0, 14).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(!a.ClientReady || a.Loading || a.SignInBusy || a.OnSignIn == nil).Clicked() && a.OnSignIn != nil {
				a.OnSignIn()
			}
		})
		return
	}
	ui.Column(c.Key("account-summary")).FillWidth().MinWidth(0).Gap(16).Padding(8, 0).Children(func() {
		bodyCard(c, m, func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="8" r="4"/><path d="M4 22v-3a8 8 0 0 1 16 0v3"/></svg>`))).Size(32, 32).TextColor(muted)
				ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
					name := providerCopy("已登录", "Signed in")
					contact := a.Email
					if a.Details.Profile.Status == accountdetails.Ready {
						if p := a.Details.Profile.Value.Name; p != nil && *p != "" {
							name = *p
						}
						if p := a.Details.Profile.Value.Contact; p != nil {
							contact = *p
						}
					}
					if !signedIn {
						name = providerCopy("正在登录", "Signing in")
						contact = a.AttemptPhase
					}
					if a.Details.Profile.Status == accountdetails.Loading || a.Details.Profile.Status == accountdetails.Unknown {
						contact = i18n.T(i18n.KeySettingsLoading)
					}
					if a.Details.Profile.Status == accountdetails.Failed {
						contact = providerCopy("暂时无法读取账户资料", "Profile unavailable")
					}
					ui.Text(c, name).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
					ui.Text(c, contact).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				})
			})
		})
		if a.Err != "" {
			ui.Text(c, a.Err).FontSize(fonttheme.InterfaceSize(13)).TextColor(fg)
		}
		if active {
			ui.Row(c).Gap(10).Wrap().Children(func() {
				if a.AuthorizeURL != "" && a.OnOpenAuthorize != nil && ui.Button(c, providerCopy("打开登录页面", "Open sign-in page")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Clicked() {
					a.OnOpenAuthorize(a.AuthorizeURL)
				}
				if a.OnCancelSignIn != nil && ui.Button(c, i18n.T(i18n.KeySettingsAccountCancelSignIn)).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(a.SignInBusy || a.AttemptPhase == "committing" || a.SignInAttemptID == "").Clicked() {
					a.OnCancelSignIn()
				}
			})
		}
		bodyCard(c, m, func() {
			walletRow := func(title string, wallets []accountdetails.Wallet) {
				ui.Row(c).FillWidth().MinWidth(0).Gap(16).Padding(6, 0).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(fonttheme.InterfaceSize(13)).TextColor(fg).Grow(1).MinWidth(0)
					ui.Column(c).MinWidth(0).Gap(3).Children(func() {
						if !signedIn || a.Details.Balance.Status == accountdetails.Loading || a.Details.Balance.Status == accountdetails.Unknown {
							ui.Text(c, i18n.T(i18n.KeySettingsLoading)).TextColor(muted)
							return
						}
						if a.Details.Balance.Status != accountdetails.Ready || len(wallets) == 0 {
							if ui.Button(c, providerCopy("余额暂不可用", "Balance unavailable")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(a.OnOpenUsage == nil || a.UsageURL == "").Clicked() && a.OnOpenUsage != nil {
								a.OnOpenUsage(a.UsageURL)
							}
							return
						}
						for _, w := range wallets {
							symbol := "$"
							if w.Currency == "CNY" {
								symbol = "¥"
							}
							ui.Text(c, symbol+w.Balance).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
						}
					})
				})
			}
			walletRow(providerCopy("余额", "Balance"), a.Details.Balance.Wallets)
			bonuses := positiveBonusWallets(a.Details.Balance.BonusWallets)
			if signedIn && (a.Details.Balance.Status != accountdetails.Ready || len(bonuses) > 0) {
				ui.Box(c).FillWidth().Height(0.5).Background(chrome.Color(colors.Border).Alpha(0.45))
				walletRow(providerCopy("奖励余额", "Bonus balance"), bonuses)
			}
			ui.Box(c).FillWidth().Height(0.5).Background(chrome.Color(colors.Border).Alpha(0.45))
			ui.Row(c).FillWidth().Gap(10).Wrap().Padding(6, 0).AlignItems(ui.Center).Children(func() {
				ui.Text(c, providerCopy("更多", "More")).FontSize(fonttheme.InterfaceSize(13)).TextColor(muted).Grow(1)
				if ui.Button(c, providerCopy("用量", "Usage")).Height(32).Padding(0, 14).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(!signedIn || a.OnOpenUsage == nil || a.UsageURL == "").Clicked() && a.OnOpenUsage != nil {
					a.OnOpenUsage(a.UsageURL)
				}
				if ui.Button(c, providerCopy("充值", "Top up")).Height(32).Padding(0, 14).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(!signedIn || a.OnOpenTopUp == nil || a.TopUpURL == "").Clicked() && a.OnOpenTopUp != nil {
					a.OnOpenTopUp(a.TopUpURL)
				}
			})
		})
		accountSignOutPane(c, a, fg, muted)
		accountBonusFeedback(c, a, fg, muted)
		for _, b := range a.Bonuses {
			bodyCard(c, m, func() {
				ui.Text(c, b.Label).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
				ui.Text(c, b.Detail).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				refused := a.BonusAckState == "refused" && a.BonusAckAccount == b.AccountID && a.BonusAckOrder == b.OrderID
				if a.OnAckBonus != nil && b.OrderID != "" && ui.Button(c, i18n.T(i18n.KeySettingsAccountBonusAck)).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(a.BonusAckState == "pending" || refused || a.BonusesLoading).Clicked() {
					a.OnAckBonus(b.AccountID, b.OrderID)
				}
			})
		}
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(a.Loading || a.SignInBusy).Clicked() {
				a.OnRefresh()
			}
			if signedIn && a.OnSignOut != nil && ui.Button(c, i18n.T(i18n.KeySettingsAccountSignOut)).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(a.Loading || a.SignInBusy).Clicked() {
				a.OnSignOut()
			}
			if a.OnContact != nil && ui.Button(c, providerCopy("联系与反馈", "Contact us")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Clicked() {
				a.OnContact()
			}
		})
	})
}

func positiveBonusWallets(wallets []accountdetails.Wallet) []accountdetails.Wallet {
	result := []accountdetails.Wallet{}
	for _, w := range wallets {
		if n, ok := new(big.Rat).SetString(strings.TrimSpace(w.Balance)); ok && n.Sign() > 0 {
			result = append(result, w)
		}
	}
	return result
}
