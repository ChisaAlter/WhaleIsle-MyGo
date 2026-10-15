package usage

import (
	"context"
	"time"
	"whaleisle/internal/accountdetails"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// Every Host binding and account stream frame invalidates pending account reads.
// The Host also returns null when its account grant changes during a request.
func (s *Store) resetAccountDetailsLocked() {
	s.accountEpoch++
	s.accountRead++
	s.details = accountdetails.Empty()
	s.email, s.name, s.plan, s.balanceSummary = "", "", "", ""
}
func (s *Store) refreshAccountDetails(ctx context.Context, client *dsh.Client, epoch uint64, status string) {
	s.mu.Lock()
	if s.client != client || s.accountEpoch != epoch {
		s.mu.Unlock()
		return
	}
	s.accountRead++
	read := s.accountRead
	s.details = accountdetails.Empty()
	s.email, s.name, s.plan, s.balanceSummary = "", "", "", ""
	if status != "credential-stored" {
		s.mu.Unlock()
		s.fire()
		return
	}
	s.details.Profile.Status = accountdetails.Loading
	s.details.Balance.Status = accountdetails.Loading
	s.mu.Unlock()
	s.fire()
	_, off := time.Now().Zone()
	args := map[string]any{"client": api.AccountClientMetadata{Version: "whaleisle", Locale: "zh-CN", TimezoneOffsetSeconds: float64(off)}}
	go func() {
		var raw api.GetProfileResult
		result := accountdetails.ProfileState{Status: accountdetails.Failed}
		if client.Call(ctx, "account/getProfile", args, &raw) == nil {
			result = accountdetails.ParseProfile(raw)
		}
		s.mu.Lock()
		if s.client != client || s.accountEpoch != epoch || s.accountRead != read {
			s.mu.Unlock()
			return
		}
		s.details.Profile = result
		if result.Status == accountdetails.Ready {
			if result.Value.Name != nil {
				s.name = *result.Value.Name
			}
			if result.Value.Contact != nil {
				s.email = *result.Value.Contact
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
	go func() {
		var raw api.GetBalanceResult
		result := accountdetails.BalanceState{Status: accountdetails.Failed}
		if client.Call(ctx, "account/getBalance", args, &raw) == nil {
			result = accountdetails.ParseBalance(raw)
		}
		s.mu.Lock()
		if s.client != client || s.accountEpoch != epoch || s.accountRead != read {
			s.mu.Unlock()
			return
		}
		s.details.Balance = result
		s.balanceSummary = result.Summary()
		s.mu.Unlock()
		s.fire()
	}()
}
