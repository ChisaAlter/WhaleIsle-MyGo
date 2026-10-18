package accountdetails

import (
	"fmt"
	"strings"
	"testing"
)

// One pure fake fixture exercises the wire shape, precision, complete lists,
// independent outcomes and detached snapshots; no account or network is used.
func TestAccountDetailsFixture(t *testing.T) {
	p := ParseProfile([]byte(`{"status":"ready","value":{"id":"fake-account","name":"Fixture","contact":"f***@example.invalid","avatarUrl":null}}`))
	if p.Status != Ready || p.Value.ID == nil || *p.Value.ID != "fake-account" || p.Value.AvatarURL != nil {
		t.Fatal("profile contract", p)
	}
	rows := make([]string, 32)
	for i := range rows {
		rows[i] = fmt.Sprintf(`{"currency":"CNY","balance":"%d.12345678901234567890"}`, i)
	}
	raw := `{"status":"ready","value":[` + strings.Join(rows, ",") + `],"bonusWallets":[{"currency":"USD","balance":"0.00000000000000000001"}]}`
	b := ParseBalance([]byte(raw))
	if b.Status != Ready || len(b.Wallets) != 32 || b.BonusWallets[0].Balance != "0.00000000000000000001" || !strings.Contains(b.Summary(), "31.12345678901234567890") {
		t.Fatal("lost wallet precision or rows", b)
	}
	s := State{Profile: p, Balance: b}
	copy := s.Clone()
	copy.Balance.Wallets[0].Balance = "mutated"
	*copy.Profile.Value.ID = "mutated"
	if s.Balance.Wallets[0].Balance == "mutated" || *s.Profile.Value.ID != "fake-account" {
		t.Fatal("snapshot aliases state")
	}
	if ParseProfile([]byte(`{"status":"failed"}`)).Status != Failed || b.Status != Ready {
		t.Fatal("independent profile failure")
	}
	for _, raw := range []string{`{"status":"failed"}`, `{"status":"ready","value":[{"currency":"CNY","balance":0.1}],"bonusWallets":[]}`, `{"status":"ready","value":[]}`, `{}`} {
		if ParseBalance([]byte(raw)).Status != Failed {
			t.Fatalf("invalid balance accepted: %s", raw)
		}
	}
	if ParseProfile([]byte("null")).Status != Unknown || ParseBalance([]byte("null")).Status != Unknown {
		t.Fatal("null grant must stay unknown")
	}
	if ParseBalance([]byte(`{"status":"ready","value":[],"bonusWallets":[]}`)).Status != Ready {
		t.Fatal("empty is ready, not unknown")
	}
}
