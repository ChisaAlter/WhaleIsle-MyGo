package settings

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestAccountI18nKeys(t *testing.T) {
	keys := []string{
		i18n.KeySettingsAccountNav,
		i18n.KeySettingsAccountTitle,
		i18n.KeySettingsAccountHint,
		i18n.KeySettingsAccountStub,
		i18n.KeySettingsAccountCredHint,
		i18n.KeySettingsAccountStatus,
		i18n.KeySettingsAccountEmail,
		i18n.KeySettingsAccountName,
		i18n.KeySettingsAccountPlan,
		i18n.KeySettingsAccountUsageURL,
		i18n.KeySettingsAccountTopUpURL,
		i18n.KeySettingsAccountBalance,
		i18n.KeySettingsAccountAttempt,
		i18n.KeySettingsAccountAuthorize,
	}
	i18n.SetLocale(i18n.LocaleZH)
	for _, k := range keys {
		if got := i18n.T(k); got == "" || got == k {
			t.Fatalf("zh missing %q got %q", k, got)
		}
	}
	i18n.SetLocale(i18n.LocaleEN)
	for _, k := range keys {
		if got := i18n.T(k); got == "" || got == k {
			t.Fatalf("en missing %q got %q", k, got)
		}
	}
	i18n.SetLocale(i18n.DefaultLocale)
}

func TestAccountAccessFieldsPresent(t *testing.T) {
	a := AccountAccess{
		ClientReady:    true,
		Status:         "signedIn",
		Email:          "a@b.c",
		Name:           "Ada",
		Plan:           "pro",
		UsageURL:       "https://usage.example",
		TopUpURL:       "https://topup.example",
		BalanceSummary: "{}",
		AttemptPhase:   "pending",
		AuthorizeURL:   "https://auth.example",
	}
	if !a.ClientReady || a.Status == "" || a.Email == "" {
		t.Fatal("expected summary fields to round-trip")
	}
}
