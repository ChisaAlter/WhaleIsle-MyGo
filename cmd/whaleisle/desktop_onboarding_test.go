package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/accountdetails"
	"whaleisle/internal/dsh"
	settingsstore "whaleisle/internal/store/settings"
	usagestore "whaleisle/internal/store/usage"
)

func TestOnboardingCreditUsesExactDecimalAndBonusWallets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		balance accountdetails.BalanceState
		funded  bool
	}{
		{"subnormal decimal", accountdetails.BalanceState{Status: accountdetails.Ready, Wallets: []accountdetails.Wallet{{Balance: "0.000000000000000000000000000000000000000000000000001"}}}, true},
		{"bonus", accountdetails.BalanceState{Status: accountdetails.Ready, Wallets: []accountdetails.Wallet{{Balance: "-12"}}, BonusWallets: []accountdetails.Wallet{{Balance: "0.01"}}}, true},
		{"zero and invalid", accountdetails.BalanceState{Status: accountdetails.Ready, Wallets: []accountdetails.Wallet{{Balance: "0.000"}, {Balance: "NaN"}, {Balance: "invalid"}, {Balance: "-0.01"}}}, false},
		{"failed stale positive", accountdetails.BalanceState{Status: accountdetails.Failed, Wallets: []accountdetails.Wallet{{Balance: "100"}}}, false},
		{"unknown", accountdetails.BalanceState{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := onboardingCreditFunded(tc.balance); got != tc.funded {
				t.Fatalf("funded=%v want=%v", got, tc.funded)
			}
		})
	}
}

type onboardingTestTransport func(*http.Request) (*http.Response, error)

func (f onboardingTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOnboardingNativePlatformSuspendsWithoutResettingController(t *testing.T) {
	s := settingsstore.New()
	loaded := make(chan struct{}, 1)
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: onboardingTestTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]any
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request["method"] != "settings/describe" {
			t.Errorf("unexpected RPC %v", request["method"])
		}
		w := httptest.NewRecorder()
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{"writable": true, "namespaces": []any{map[string]any{"ns": settingsstore.DesktopOnboardingNamespace, "revision": 1, "schema": map[string]any{"uid": 1, "refs": map[string]any{"1": map[string]any{"type": "object", "dict": map[string]any{}}}}, "value": map[string]any{"version": 1, "step": "credit"}, "secrets": []any{}}}}}})
		return w.Result(), nil
	})}}, func() {
		if snapshot := s.Snapshot(); !snapshot.Loading && len(snapshot.Namespaces) == 1 {
			select {
			case loaded <- struct{}{}:
			default:
			}
		}
	})
	s.RefreshDescribe(context.Background())
	select {
	case <-loaded:
	case <-time.After(3 * time.Second):
		t.Fatal("onboarding fixture did not finish describing")
	}
	controller := settingsstore.NewDesktopOnboarding(s)
	controller.Sync(settingsstore.OnboardingAccount{Enabled: true, Status: "credential-stored", CreditKnown: true, CreditFunded: true})
	before := controller.Snapshot()
	for _, mode := range []string{"account-platform", "launcher", "hidden"} {
		state := &mainState{settings: s, usage: usagestore.New(), desktopOnboarding: controller}
		switch mode {
		case "account-platform":
			state.accountReturn.platform = &accountPlatformLease{}
			state.webviewOpen = true
		case "launcher":
			state.launcherOpen = true
		case "hidden":
			state.projectWindowHidden = true
		}
		visible := true
		tt := ui.NewTester(func(c *ui.Context) { visible = state.renderDesktopOnboarding(c, "deepseek", false) }, 500, 500)
		if visible || tt.HasText("Add credits") {
			t.Fatalf("%s remained covered by onboarding modal", mode)
		}
		if got := controller.Snapshot(); got != before {
			t.Fatalf("%s reset resumed onboarding: before=%+v got=%+v", mode, before, got)
		}
	}
}
