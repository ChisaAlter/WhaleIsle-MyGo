package main

import (
	"math/big"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/accountdetails"
)

// The first-run onboarding page ("开始设置" / credits) is disabled: users asked
// to open straight into the workspace. The controller, store and account access
// helpers stay in place so the page can be re-enabled deliberately.
func (s *mainState) renderDesktopOnboarding(c *ui.Context, themeID string, dark bool) bool {
	return false
}

// onboardingCreditFunded reports whether any wallet holds a positive balance.
// Kept for the re-enabled onboarding path and its unit test.
func onboardingCreditFunded(balance accountdetails.BalanceState) bool {
	if balance.Status == accountdetails.Ready {
		for _, wallets := range [][]accountdetails.Wallet{balance.Wallets, balance.BonusWallets} {
			for _, wallet := range wallets {
				value, ok := new(big.Rat).SetString(wallet.Balance)
				if ok && value.Sign() > 0 {
					return true
				}
			}
		}
	}
	return false
}
