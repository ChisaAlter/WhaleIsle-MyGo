package settings

import (
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/accountdetails"
	"whaleisle/internal/accountlogin"
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
	ui.Text(c, i18n.T(i18n.KeySettingsAccountTitle)).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsAccountHint)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Text(c, i18n.T(i18n.KeySettingsAccountCredHint)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)

	if !a.ClientReady {
		ui.Text(c, i18n.T(i18n.KeySettingsAccountStub)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if a.Loading {
		ui.Text(c, i18n.T(i18n.KeySettingsLoading)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	if a.OnContact != nil && ui.Button(c, "联系与反馈 / Contact us").Clicked() {
		a.OnContact()
	}
	accountSignOutPane(c, a, fg, muted)
	if a.HasRunningTasks {
		ui.Text(c, i18n.T(i18n.KeySettingsAccountRunningTasks)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}

	rows := []struct{ label, value string }{
		{i18n.T(i18n.KeySettingsAccountStatus), emptyDash(a.Status)},
		{i18n.T(i18n.KeySettingsAccountUsageURL), emptyDash(a.UsageURL)},
		{i18n.T(i18n.KeySettingsAccountTopUpURL), emptyDash(a.TopUpURL)},
	}
	if a.AttemptPhase != "" || a.AuthorizeURL != "" {
		rows = append(rows,
			struct{ label, value string }{i18n.T(i18n.KeySettingsAccountAttempt), emptyDash(a.AttemptPhase)},
			struct{ label, value string }{i18n.T(i18n.KeySettingsAccountAuthorize), emptyDash(a.AuthorizeURL)},
		)
	}
	for _, row := range rows {
		r := row
		ui.Row(c).Gap(12).Children(func() {
			ui.Text(c, r.label).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Width(140)
			ui.Text(c, r.value).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Grow(1)
			if r.value != "" && r.value != "—" && r.value != "-" {
				if ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
					mygo.Clipboard.WriteText(r.value)
				}
			}
		})
	}

	accountDetailsPane(c, a.Details, fg, muted)

	ui.Text(c, i18n.T(i18n.KeySettingsAccountBonuses)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg).Padding(8, 0, 0, 0)
	accountBonusFeedback(c, a, fg, muted)
	if len(a.Bonuses) == 0 && !a.BonusesLoading && a.BonusesError == "" {
		ui.Text(c, i18n.T(i18n.KeySettingsAccountBonusesEmpty)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	} else {
		for _, b := range a.Bonuses {
			row := b
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, row.Label).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Grow(1)
				if row.Detail != "" {
					ui.Text(c, row.Detail).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				body := strings.TrimSpace(row.Label + " " + row.Detail)
				if body != "" {
					if ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
						mygo.Clipboard.WriteText(body)
					}
				}
				refused := a.BonusAckState == "refused" && a.BonusAckAccount == row.AccountID && a.BonusAckOrder == row.OrderID
				if a.OnAckBonus != nil && row.OrderID != "" && !a.Loading && !a.BonusesLoading && a.BonusAckState != "pending" && !refused {
					if ui.Button(c, i18n.T(i18n.KeySettingsAccountBonusAck)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
						a.OnAckBonus(row.AccountID, row.OrderID)
					}
				}
			})
		}
	}

	ui.Row(c).Gap(8).Children(func() {
		if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
			a.OnRefresh()
		}
		if a.ClientReady && !a.SignInBusy && !accountlogin.Active(a.AttemptPhase) && a.Status != "credential-stored" && a.OnSignIn != nil && ui.Button(c, i18n.T(i18n.KeySettingsAccountSignIn)).Clicked() {
			a.OnSignIn()
		}
		if a.ClientReady && !a.SignInBusy && a.Status == "credential-stored" && a.OnSignOut != nil && ui.Button(c, i18n.T(i18n.KeySettingsAccountSignOut)).Clicked() {
			a.OnSignOut()
		}
		if a.ClientReady && !a.SignInBusy && accountlogin.Active(a.AttemptPhase) && a.AttemptPhase != "committing" && a.OnCancelSignIn != nil && strings.TrimSpace(a.SignInAttemptID) != "" {
			if ui.Button(c, i18n.T(i18n.KeySettingsAccountCancelSignIn)).Clicked() {
				a.OnCancelSignIn()
			}
		}
		if a.OnOpenAuthorize != nil && strings.TrimSpace(a.AuthorizeURL) != "" {
			if ui.Button(c, i18n.T(i18n.KeySettingsAccountAuthorize)).Clicked() {
				a.OnOpenAuthorize(a.AuthorizeURL)
			}
		}
		if a.OnOpenUsage != nil && strings.TrimSpace(a.UsageURL) != "" {
			if ui.Button(c, i18n.T(i18n.KeyUsageOpenWeb)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
				a.OnOpenUsage(a.UsageURL)
			}
		}
		if a.OnOpenTopUp != nil && strings.TrimSpace(a.TopUpURL) != "" {
			if ui.Button(c, i18n.T(i18n.KeyUsageOpenTopUp)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
				a.OnOpenTopUp(a.TopUpURL)
			}
		}
	})
}
