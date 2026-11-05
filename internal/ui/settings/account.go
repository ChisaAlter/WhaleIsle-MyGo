package settings

import (
	"math/big"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"

	"whaleisle/internal/accountdetails"

	"whaleisle/internal/accountlogout"
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
	AttemptErrorCode string
	OnOpenInfo       func()
	OnContact        func()
	OnRefresh        func()
	OnOpenAuthorize  func(url string)
	OnCopyAuthorize  func(url string)
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
	errorSeat := ui.Box(c.Key("account-errors"))
	seen := ui.Local(errorSeat, "last-error", func() string { return "" })
	if a.Err != "" && a.Err != *seen {
		c.AddToast(ui.Toast{ID: "account-read-error", Description: a.Err})
	}
	*seen = a.Err
	if a.Status != "credential-stored" {
		ui.Column(c).FillWidth().Gap(16).Padding(40, 0).Children(func() {
			ui.Text(c, providerCopy("当前未登录 DeepSeek 账号", "You are not signed in to DeepSeek")).FontSize(fonttheme.InterfaceSize(16)).FontWeight(600).TextColor(fg)
			ui.Text(c, providerCopy("登录 DeepSeek 账号获取专属 API Key", "Sign in to DeepSeek to get your dedicated API Key")).FillWidth().FontSize(fonttheme.InterfaceSize(14)).TextColor(muted)
			if ui.Button(c, i18n.T(i18n.KeySettingsAccountSignIn)).AlignSelf(ui.Start).Height(36).Padding(0, 16).Radius(8).Disabled(!a.ClientReady || a.SignInBusy || a.OnSignIn == nil).Clicked() {
				a.OnSignIn()
			}
		})
		return
	}
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Column(c.Key("account-summary")).FillWidth().MinWidth(0).Gap(16).Padding(8, 0).Children(func() {
		bodyCard(c, m, func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() {
				rawAvatar := ""
				if a.Details.Profile.Value.AvatarURL != nil {
					rawAvatar = *a.Details.Profile.Value.AvatarURL
				}
				if bmp := accountAvatar(c, rawAvatar); bmp != nil {
					ui.Image(c, bmp).Size(32, 32).Radius(16).Fit(ui.Cover)
				} else {
					ui.Icon(c, ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="8" r="4"/><path d="M4 22v-3a8 8 0 0 1 16 0v3"/></svg>`))).Size(32, 32).TextColor(muted)
				}
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					name, contact := a.Name, a.Email
					if p := a.Details.Profile.Value.Name; p != nil && *p != "" {
						name = *p
					}
					if p := a.Details.Profile.Value.Contact; p != nil {
						contact = *p
					}
					if name == "" {
						name = providerCopy("已登录", "Signed in")
					}
					if a.Details.Profile.Status == accountdetails.Loading || a.Details.Profile.Status == accountdetails.Unknown {
						contact = i18n.T(i18n.KeySettingsLoading)
					}
					if a.Details.Profile.Status == accountdetails.Failed {
						contact = providerCopy("暂时无法读取账户资料", "Profile unavailable")
					}
					ui.Text(c, name).FillWidth().SingleLine().FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
					ui.Text(c, contact).FillWidth().SingleLine().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				})
				if a.OnOpenInfo != nil && ui.Button(c, providerCopy("更多账号信息 ↗", "More account information ↗")).Shrink(0).Height(32).BorderWidth(0).Background(ui.Transparent).FontSize(fonttheme.InterfaceSize(13)).Clicked() {
					a.OnOpenInfo()
				}
			})
		})
		bodyCard(c, m, func() {
			wallet := func(title string, wallets []accountdetails.Wallet) {
				ui.Row(c).FillWidth().MinWidth(0).Gap(16).Padding(6, 0).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(14)).TextColor(fg)
					ui.Row(c).MinWidth(0).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
						if a.Details.Balance.Status == accountdetails.Loading || a.Details.Balance.Status == accountdetails.Unknown {
							ui.Text(c, i18n.T(i18n.KeySettingsLoading)).TextColor(muted)
							return
						}
						if a.Details.Balance.Status != accountdetails.Ready || len(wallets) == 0 {
							if ui.Button(c, providerCopy("前往开放平台查看 ↗", "View on Platform ↗")).Height(28).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Disabled(a.OnOpenUsage == nil || a.UsageURL == "").Clicked() {
								a.OnOpenUsage(a.UsageURL)
							}
							return
						}
						for i, w := range wallets {
							if i > 0 {
								ui.Text(c, "+").TextColor(muted)
							}
							ui.Text(c, accountMoney(w.Balance, w.Currency)).FontSize(fonttheme.InterfaceSize(18)).FixedLineHeight(fonttheme.InterfaceSize(28)).FontWeight(600).TextColor(fg)
						}
					})
				})
			}
			wallet(providerCopy("充值余额", "Topped-up balance"), a.Details.Balance.Wallets)
			bonuses := positiveBonusWallets(a.Details.Balance.BonusWallets)
			if a.Details.Balance.Status != accountdetails.Ready || len(bonuses) > 0 {
				ui.Box(c).FillWidth().Height(.5).Background(chrome.Color(colors.Border).Alpha(.45))
				wallet(providerCopy("赠金余额", "Granted balance"), bonuses)
			}
			ui.Box(c).FillWidth().Height(.5).Background(chrome.Color(colors.Border).Alpha(.45))
			ui.Row(c).FillWidth().Gap(10).Padding(6, 0).AlignItems(ui.Center).Children(func() {
				ui.Text(c, providerCopy("更多", "More")).Grow(1).FontSize(fonttheme.InterfaceSize(14)).TextColor(muted)
				if ui.Button(c, providerCopy("查询用量", "View usage")).Height(36).Padding(0, 16).Radius(8).Border(.5, c.Theme().Border.Alpha(.45)).FontSize(fonttheme.InterfaceSize(14)).Disabled(a.OnOpenUsage == nil || a.UsageURL == "").Clicked() {
					a.OnOpenUsage(a.UsageURL)
				}
				if ui.Button(c, providerCopy("充值", "Top up")).Height(36).Padding(0, 16).Radius(8).BorderWidth(0).Background(chrome.Color(colors.Accent)).TextColor(ui.RGB(255, 255, 255)).FontSize(fonttheme.InterfaceSize(14)).Disabled(a.OnOpenTopUp == nil || a.TopUpURL == "").Clicked() {
					a.OnOpenTopUp(a.TopUpURL)
				}
			})
		})
	})
}

// Match Platform Web's decimal rules without losing source precision.
func accountMoney(raw, currency string) string {
	symbol := "$"
	if currency == "CNY" {
		symbol = "¥"
	}
	n, ok := new(big.Rat).SetString(strings.TrimSpace(raw))
	if !ok {
		return "—"
	}
	if n.Sign() == 0 {
		return symbol + "0.00"
	}
	negative := n.Sign() < 0
	n.Abs(n)
	cents := new(big.Rat).Mul(n, big.NewRat(100, 1))
	q := new(big.Int).Quo(cents.Num(), cents.Denom())
	prefix := symbol
	if negative {
		prefix = "-" + symbol
		if q.Sign() == 0 {
			q.SetInt64(1)
		} else {
			remainder := new(big.Int).Mod(cents.Num(), cents.Denom())
			if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(cents.Denom()) >= 0 {
				q.Add(q, big.NewInt(1))
			}
		}
	} else if q.Sign() == 0 {
		return "<" + symbol + "0.01"
	}
	value := q.String()
	for len(value) < 3 {
		value = "0" + value
	}
	integer, fraction := value[:len(value)-2], value[len(value)-2:]
	for i := len(integer) - 3; i > 0; i -= 3 {
		integer = integer[:i] + "," + integer[i:]
	}
	return prefix + integer + "." + fraction
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
