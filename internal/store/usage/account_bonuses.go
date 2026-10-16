package usage

import (
	"context"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// BonusFeedback describes acknowledgement of a displayed server-authored notice.
// Acknowledgement records that the notice was seen; it never claims an award.
type BonusFeedback struct {
	AccountID string
	OrderID   string
	State     string
	Error     string
}

func readAccountBonuses(ctx context.Context, client *dsh.Client, status string) ([]Bonus, error) {
	if status != "credential-stored" {
		return nil, nil
	}
	_, zoneOff := time.Now().Zone()
	meta := api.AccountClientMetadata{
		Version:               "whaleisle",
		Locale:                "zh-CN",
		TimezoneOffsetSeconds: float64(zoneOff),
	}
	var batch api.GetUnnotifiedBonusesResult
	if err := client.Call(ctx, "account/getUnnotifiedBonuses", map[string]any{"client": meta}, &batch); err != nil {
		return nil, err
	}
	var bonuses []Bonus
	for _, b := range batch.Bonuses {
		if expires, err := time.Parse(time.RFC3339, b.ExpiresAt); err == nil && !expires.After(time.Now()) {
			continue
		}
		bonuses = append(bonuses, Bonus{
			AccountID: batch.AccountId,
			OrderID:   b.OrderId,
			Campaign:  b.Campaign,
			Amount:    b.Amount,
			Currency:  b.Currency,
			Message:   b.Message,
			ExpiresAt: b.ExpiresAt,
		})
	}
	return bonuses, nil
}

// Account stream changes also read notices; signing in need not wait for a
// manual Settings refresh before the account page can present its reward.
func (s *Store) refreshAccountBonuses(ctx context.Context, client *dsh.Client, epoch uint64, status string) {
	s.mu.Lock()
	if s.client != client || s.accountEpoch != epoch {
		s.mu.Unlock()
		return
	}
	s.bonusesLoading = status == "credential-stored"
	s.bonusesError = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		bonuses, err := readAccountBonuses(ctx, client, status)
		s.mu.Lock()
		if s.client != client || s.accountEpoch != epoch {
			s.mu.Unlock()
			return
		}
		s.bonusesLoading = false
		if err != nil {
			s.bonusesError = err.Error()
		} else {
			s.bonuses = bonuses
			s.bonusesError = ""
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// AckBonusNotified acknowledges the selected notice and retains its actual
// result. false means the server cannot accept it for this account/session.
func (s *Store) AckBonusNotified(ctx context.Context, accountID, orderID, sessionID string) {
	accountID = strings.TrimSpace(accountID)
	orderID = strings.TrimSpace(orderID)
	s.mu.Lock()
	client, epoch, status := s.client, s.accountEpoch, s.accountStatus
	found := false
	for _, bonus := range s.bonuses {
		if bonus.AccountID == accountID && bonus.OrderID == orderID {
			found = true
			break
		}
	}
	if client == nil || status != "credential-stored" || accountID == "" || orderID == "" || !found || s.loading || s.bonusesLoading || s.bonusFeedback.State == "pending" {
		s.mu.Unlock()
		return
	}
	s.bonusFeedback = BonusFeedback{AccountID: accountID, OrderID: orderID, State: "pending"}
	s.mu.Unlock()
	s.fire()
	go func() {
		_, zoneOff := time.Now().Zone()
		meta := api.AccountClientMetadata{
			Version:               "whaleisle",
			Locale:                "zh-CN",
			TimezoneOffsetSeconds: float64(zoneOff),
		}
		var acknowledged api.AckBonusNotifiedResult
		err := client.Call(ctx, "account/ackBonusNotified", map[string]any{
			"accountId": accountID,
			"orderId":   orderID,
			"client":    meta,
		}, &acknowledged)
		s.mu.Lock()
		if s.client != client || s.accountEpoch != epoch {
			s.mu.Unlock()
			return
		}
		switch {
		case err != nil:
			s.bonusFeedback.State = "failed"
			s.bonusFeedback.Error = err.Error()
		case !acknowledged:
			s.bonusFeedback.State = "refused"
		default:
			s.bonusFeedback.State = "acknowledged"
			for i, bonus := range s.bonuses {
				if bonus.AccountID == accountID && bonus.OrderID == orderID {
					s.bonuses = append(s.bonuses[:i], s.bonuses[i+1:]...)
					break
				}
			}
		}
		s.mu.Unlock()
		s.fire()
		if err == nil && acknowledged {
			s.refreshAccountDetails(ctx, client, epoch, status)
			s.refreshAccountBonuses(ctx, client, epoch, status)
		}
	}()
}
