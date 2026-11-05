package settings

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
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

func TestAccountSignInRequiresHostAndUsesExistingCallback(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	for _, ready := range []bool{false, true} {
		calls := 0
		a := &AccountAccess{ClientReady: ready, OnSignIn: func() { calls++ }}
		tt := ui.NewTester(func(c *ui.Context) {
			ModalView(c, Model{Desktop: &DesktopAccess{}, AccountActive: true, Account: a})
		}, 1440, 863)
		start := time.Unix(1000, 0)
		tt.SetTime(start)
		tt.SetTime(start.Add(time.Second))
		if !tt.HasText("You are not signed in to DeepSeek") {
			t.Fatal("account navigation did not show the signed-out page")
		}
		if !tt.HasText("Sign in to DeepSeek to get your dedicated API Key") {
			t.Fatal("signed-out account description missing")
		}
		if ready {
			saveBodySnapshot(t, tt, "account-ready.png")
		} else {
			saveBodySnapshot(t, tt, "account-disconnected.png")
		}
		_ = tt.Click(i18n.T(i18n.KeySettingsAccountSignIn))
		want := 0
		if ready {
			want = 1
		}
		if calls != want {
			t.Fatalf("Host ready=%v: sign-in calls=%d, want %d", ready, calls, want)
		}
	}
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
