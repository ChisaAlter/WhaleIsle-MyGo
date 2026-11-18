package usage

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"whaleisle/internal/dsh"
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
	_, bonuses, err := readAccountBonusBatch(ctx, client, status)
	return bonuses, err
}

// errAccountBonusesUnavailable records a contract the pinned runtime no longer
// exposes. The rc.4 Host answered account/getUnnotifiedBonuses and
// account/ackBonusNotified; the pinned 0.2.2 Host replaced both with the
// account notification endpoints, and moving this view onto them is a separate
// change. Reporting the gap keeps the bonus list empty instead of inventing one.
var errAccountBonusesUnavailable = errors.New("account bonuses are unavailable: the pinned Host runtime replaced the bonus endpoints with account notifications")

func readAccountBonusBatch(ctx context.Context, client *dsh.Client, status string) (string, []Bonus, error) {
	if status != "credential-stored" {
		return "", nil, nil
	}
	return "", nil, errAccountBonusesUnavailable
}

func bonusMetadataText(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}

func cloneBonuses(bonuses []Bonus) []Bonus {
	out := append([]Bonus(nil), bonuses...)
	for i := range out {
		if out[i].Title != nil {
			value := *out[i].Title
			out[i].Title = &value
		}
		meta := &out[i].Metadata
		meta.Campaign = append(json.RawMessage(nil), meta.Campaign...)
		meta.Amount = append(json.RawMessage(nil), meta.Amount...)
		meta.Currency = append(json.RawMessage(nil), meta.Currency...)
		meta.GrantedAt = append(json.RawMessage(nil), meta.GrantedAt...)
	}
	return out
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
		accountID, bonuses, err := readAccountBonusBatch(ctx, client, status)
		s.mu.Lock()
		if s.client != client || s.accountEpoch != epoch {
			s.mu.Unlock()
			return
		}
		s.bonusesLoading = false
		if err != nil {
			s.bonusesError = err.Error()
		} else {
			s.applyAccountBonusesLocked(accountID, bonuses, status)
			s.bonusesError = ""
		}
		s.mu.Unlock()
		s.fire()
	}()
}

type bonusOrder struct{ accountID, orderID string }
type bonusNoticeLifecycle struct {
	generation                uint64
	accountID                 string
	shown, dismissed, settled map[string]bool
	pending                   []bonusOrder
	running                   bool
	cancel                    context.CancelFunc
}

// ConfigureBonusAcknowledgements consumes the existing account namespace's
// deployment policy. Absent fields use the original client's 1s / 60s defaults.
func (s *Store) ConfigureBonusAcknowledgements(raw json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bonusRetryConfig == string(raw) && s.bonusRetryDelay > 0 {
		return
	}
	s.bonusRetryConfig = string(raw)
	var policy struct {
		Delay   float64 `json:"bonusAckRetryDelayMs"`
		Maximum float64 `json:"bonusAckRetryMaxDelayMs"`
	}
	_ = json.Unmarshal(raw, &policy)
	period := func(ms float64, fallback time.Duration) time.Duration {
		if ms < 1 || ms > float64((1<<63-1)/int64(time.Millisecond)) {
			return fallback
		}
		return time.Duration(ms * float64(time.Millisecond))
	}
	s.bonusRetryDelay, s.bonusRetryMaximum = period(policy.Delay, time.Second), period(policy.Maximum, time.Minute)
}

func (s *Store) resetBonusNoticesLocked() {
	if s.bonusNotices.cancel != nil {
		s.bonusNotices.cancel()
	}
	s.bonusNotices = bonusNoticeLifecycle{generation: s.bonusNotices.generation + 1}
	s.bonusFeedback = BonusFeedback{}
}

func (s *Store) applyAccountBonusesLocked(accountID string, bonuses []Bonus, status string) {
	if status != "credential-stored" {
		s.resetBonusNoticesLocked()
		s.bonuses = nil
		return
	}
	if accountID != s.bonusNotices.accountID {
		s.resetBonusNoticesLocked()
		s.bonusNotices.accountID = accountID
	}
	lifecycle := &s.bonusNotices
	if lifecycle.shown == nil {
		lifecycle.shown, lifecycle.dismissed, lifecycle.settled = map[string]bool{}, map[string]bool{}, map[string]bool{}
	}
	visible := []Bonus{}
	for _, b := range bonuses {
		if !lifecycle.dismissed[b.OrderID] {
			visible = append(visible, b)
		}
	}
	// Keep the presented foreground card until dismissal, even after its ack
	// removes it from the next server batch. New orders wait behind that card.
	if len(s.bonuses) > 0 {
		front := s.bonuses[0]
		if front.AccountID == accountID && lifecycle.shown[front.OrderID] && !lifecycle.dismissed[front.OrderID] {
			for i, bonus := range visible {
				if bonus.OrderID == front.OrderID {
					front = bonus
					visible = append(visible[:i], visible[i+1:]...)
					break
				}
			}
			visible = append([]Bonus{front}, visible...)
		}
	}
	s.bonuses = visible
}

// BonusNoticeShown is called only after the native notice was presented.
func (s *Store) BonusNoticeShown(accountID, orderID string) {
	s.queueBonusAcknowledgement(accountID, orderID)
}

// AckBonusNotified also supports the explicit acknowledgement/retry action.
func (s *Store) AckBonusNotified(ctx context.Context, accountID, orderID, sessionID string) {
	s.queueBonusAcknowledgement(accountID, orderID)
}

func (s *Store) DismissBonusNotice(accountID, orderID string) {
	s.queueBonusAcknowledgement(accountID, orderID)
	s.mu.Lock()
	if accountID != "" && orderID != "" && s.bonusNotices.accountID == accountID && s.bonusNotices.dismissed != nil {
		s.bonusNotices.dismissed[orderID] = true
		for i, bonus := range s.bonuses {
			if bonus.AccountID == accountID && bonus.OrderID == orderID {
				s.bonuses = append(s.bonuses[:i], s.bonuses[i+1:]...)
				break
			}
		}
	}
	s.mu.Unlock()
	s.fire()
}

func (s *Store) queueBonusAcknowledgement(accountID, orderID string) {
	accountID = strings.TrimSpace(accountID)
	orderID = strings.TrimSpace(orderID)
	s.mu.Lock()
	client, status := s.client, s.accountStatus
	lifecycle := &s.bonusNotices
	found := false
	for _, bonus := range s.bonuses {
		if bonus.AccountID == accountID && bonus.OrderID == orderID {
			found = true
			break
		}
	}
	if client == nil || status != "credential-stored" || accountID == "" || orderID == "" || !found || lifecycle.accountID != accountID {
		s.mu.Unlock()
		return
	}
	lifecycle.shown[orderID] = true
	if lifecycle.settled[orderID] {
		s.mu.Unlock()
		return
	}
	queued := false
	for _, pending := range lifecycle.pending {
		if pending.orderID == orderID {
			queued = true
			break
		}
	}
	if !queued {
		lifecycle.pending = append(lifecycle.pending, bonusOrder{accountID, orderID})
	}
	if lifecycle.running {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	lifecycle.running, lifecycle.cancel = true, cancel
	generation := lifecycle.generation
	s.mu.Unlock()
	go s.runBonusAcknowledgements(ctx, client, generation)
}

// One queue belongs to one account lifecycle. Transport failures back off like
// One queue belongs to one account lifecycle. The pinned runtime no longer
// carries the acknowledgement endpoint, so a queued notice settles with the
// reported gap rather than a transport retry: retrying a removed contract would
// only repeat the same answer.
func (s *Store) runBonusAcknowledgements(ctx context.Context, client *dsh.Client, generation uint64) {
	for {
		s.mu.Lock()
		lifecycle := &s.bonusNotices
		if ctx.Err() != nil || s.client != client || lifecycle.generation != generation {
			s.mu.Unlock()
			return
		}
		if len(lifecycle.pending) == 0 {
			lifecycle.running, lifecycle.cancel = false, nil
			s.mu.Unlock()
			return
		}
		order := lifecycle.pending[0]
		s.bonusFeedback = BonusFeedback{AccountID: order.accountID, OrderID: order.orderID, State: "failed", Error: errAccountBonusesUnavailable.Error()}
		lifecycle.settled[order.orderID] = true
		lifecycle.pending = lifecycle.pending[1:]
		s.mu.Unlock()
		s.fire()
	}
}
