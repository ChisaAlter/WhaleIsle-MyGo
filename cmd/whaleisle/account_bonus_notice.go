package main

import (
	"time"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/sidebar"

	"github.com/egoist/mygo/ui"
	usagestore "whaleisle/internal/store/usage"
)

// Retained account-notice lifecycle identity; the sidebar anchor is the visual
// owner and the usage store remains the sole acknowledgement queue.
type accountBonusNoticeState struct {
	store      *usagestore.Store
	bonus      usagestore.Bonus
	generation uint64
	lifecycle  uint64
	published  string
	dismissed  bool
}

const accountBonusToastID = "whaleisle-account-bonus"

// The account trigger is the sole visual owner. Keep the store's existing
// shown/ack/expiry queue; the resident pass no longer publishes global toasts.
func (s *mainState) syncAccountBonusNotice(c *ui.Context) {
	s.mu.Lock()
	store := s.usage
	s.mu.Unlock()
	if store == nil {
		return
	}
	v := store.Snapshot()
	for _, b := range v.Bonuses {
		if expiry, err := time.Parse(time.RFC3339, b.ExpiresAt); err == nil && expiry.After(time.Now()) {
			c.After(time.Until(expiry))
		}
	}
}

func (s *mainState) sidebarAccountNotices() []sidebar.AccountNotice {
	s.mu.Lock()
	store := s.usage
	s.mu.Unlock()
	if store == nil {
		return nil
	}
	v := store.Snapshot()
	if !v.ClientReady || v.AccountStatus != "credential-stored" {
		return nil
	}
	notices := []sidebar.AccountNotice{}
	for _, b := range v.Bonuses {
		if expiry, err := time.Parse(time.RFC3339, b.ExpiresAt); err == nil && !expiry.After(time.Now()) {
			continue
		}
		if b.AccountID == "" || b.OrderID == "" {
			continue
		}
		bonus := b
		generation := v.BonusGeneration
		owns := func() bool {
			current := store.Snapshot()
			return current.BonusGeneration == generation && current.ClientReady && current.AccountStatus == "credential-stored"
		}
		notices = append(notices, sidebar.AccountNotice{ID: bonus.AccountID + "/" + bonus.OrderID, Title: accountBonusNoticeTitle(bonus), Message: bonus.Message, OnShown: func() {
			if owns() {
				store.BonusNoticeShown(bonus.AccountID, bonus.OrderID)
			}
		}, OnDismiss: func() {
			if owns() {
				store.DismissBonusNotice(bonus.AccountID, bonus.OrderID)
			}
		}})
	}
	return notices
}

func accountBonusNoticeTitle(bonus usagestore.Bonus) string {
	if bonus.Title != nil {
		return *bonus.Title
	}
	return i18n.T(i18n.KeyCmdRewardNotice)
}
