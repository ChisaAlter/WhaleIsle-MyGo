package main

import (
	"context"
	"github.com/egoist/mygo/ui"
	"math/big"
	"whaleisle/internal/accountdetails"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/settings"
)

// Called after mainView releases s.mu. Desktop enablement is the native shell's
// actual registration gate; launcher and hidden project windows own no overlay.
func (s *mainState) renderDesktopOnboarding(c *ui.Context, themeID string, dark bool) bool {
	s.mu.Lock()
	active := !s.launcherOpen && !s.projectWindowHidden && !(s.accountReturn.platform != nil && s.webviewOpen)
	store, usage := s.settings, s.usage
	if s.desktopOnboarding == nil && store != nil {
		s.desktopOnboarding = settingsstore.NewDesktopOnboarding(store)
	}
	controller := s.desktopOnboarding
	s.mu.Unlock()
	settings.PreferenceNotices(c, store)
	if !active || store == nil || usage == nil || controller == nil {
		return false
	}
	store.EnsureLoaded(context.Background())
	a := usage.Snapshot()
	balance := a.Details.Balance
	funded := false
	if balance.Status == accountdetails.Ready {
		for _, wallets := range [][]accountdetails.Wallet{balance.Wallets, balance.BonusWallets} {
			for _, wallet := range wallets {
				value, ok := new(big.Rat).SetString(wallet.Balance)
				if ok && value.Sign() > 0 {
					funded = true
				}
			}
		}
	}
	controller.Sync(settingsstore.OnboardingAccount{Enabled: true, Status: a.AccountStatus, Loading: a.Loading, Error: a.Err, CreditKnown: balance.Status == accountdetails.Ready, CreditFunded: funded})
	access := &settings.AccountAccess{Details: a.Details, Status: a.AccountStatus, Loading: a.Loading, Err: a.Err, TopUpURL: a.TopUpURL, OnOpenTopUp: s.openAccountTopUp}
	return settings.DesktopOnboarding(c, controller, access, themeID, dark)
}
