// Package accountdetails decodes the display-safe account contract without money conversion.
package accountdetails

import (
	"bytes"
	"encoding/json"
	"strings"
)

type Status string

const (
	Unknown Status = "unknown"
	Loading Status = "loading"
	Ready   Status = "ready"
	Failed  Status = "failed"
)

type Profile struct {
	ID        *string `json:"id"`
	Name      *string `json:"name"`
	Contact   *string `json:"contact"`
	AvatarURL *string `json:"avatarUrl,omitempty"`
}
type Wallet struct {
	Currency string `json:"currency"`
	Balance  string `json:"balance"`
}
type ProfileState struct {
	Status Status
	Value  Profile
}
type BalanceState struct {
	Status       Status
	Wallets      []Wallet
	BonusWallets []Wallet
}
type State struct {
	Profile ProfileState
	Balance BalanceState
}

func Empty() State {
	return State{Profile: ProfileState{Status: Unknown}, Balance: BalanceState{Status: Unknown}}
}
func (s State) Clone() State {
	s.Balance.Wallets = append([]Wallet(nil), s.Balance.Wallets...)
	s.Balance.BonusWallets = append([]Wallet(nil), s.Balance.BonusWallets...)
	// Copy nullable strings as well: a snapshot must not mutate store-owned data.
	cp := func(v *string) *string {
		if v == nil {
			return nil
		}
		x := *v
		return &x
	}
	s.Profile.Value = Profile{cp(s.Profile.Value.ID), cp(s.Profile.Value.Name), cp(s.Profile.Value.Contact), cp(s.Profile.Value.AvatarURL)}
	return s
}
func ParseProfile(raw []byte) ProfileState {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return ProfileState{Status: Unknown}
	}
	var wire struct {
		Status Status          `json:"status"`
		Value  json.RawMessage `json:"value"`
	}
	if json.Unmarshal(raw, &wire) != nil || wire.Status != Ready {
		return ProfileState{Status: Failed}
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(wire.Value, &fields) != nil || fields == nil {
		return ProfileState{Status: Failed}
	}
	for _, k := range []string{"id", "name", "contact"} {
		if _, ok := fields[k]; !ok {
			return ProfileState{Status: Failed}
		}
	}
	var p Profile
	if json.Unmarshal(wire.Value, &p) != nil {
		return ProfileState{Status: Failed}
	}
	return ProfileState{Status: Ready, Value: p}
}
func ParseBalance(raw []byte) BalanceState {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return BalanceState{Status: Unknown}
	}
	var wire struct {
		Status       Status   `json:"status"`
		Value        []Wallet `json:"value"`
		BonusWallets []Wallet `json:"bonusWallets"`
	}
	if json.Unmarshal(raw, &wire) != nil || wire.Status != Ready || wire.Value == nil || wire.BonusWallets == nil {
		return BalanceState{Status: Failed}
	}
	for _, ws := range [][]Wallet{wire.Value, wire.BonusWallets} {
		for _, w := range ws {
			if (w.Currency != "CNY" && w.Currency != "USD") || !decimal(w.Balance) {
				return BalanceState{Status: Failed}
			}
		}
	}
	return BalanceState{Status: Ready, Wallets: wire.Value, BonusWallets: wire.BonusWallets}
}
func decimal(v string) bool {
	if strings.HasPrefix(v, "-") {
		v = v[1:]
	}
	parts := strings.Split(v, ".")
	if len(parts) > 2 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// Summary is compatibility text for the usage surface; it never truncates or rounds.
func (s BalanceState) Summary() string {
	if s.Status != Ready {
		return string(s.Status)
	}
	lines := []string{}
	for _, w := range s.Wallets {
		lines = append(lines, "Recharge: "+w.Currency+" "+w.Balance)
	}
	for _, w := range s.BonusWallets {
		lines = append(lines, "Bonus: "+w.Currency+" "+w.Balance)
	}
	if len(lines) == 0 {
		return "No wallets"
	}
	return strings.Join(lines, "\n")
}
