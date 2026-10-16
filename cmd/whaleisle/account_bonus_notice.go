package main

import (
	"context"
	"time"

	"github.com/egoist/mygo/ui"
	usagestore "whaleisle/internal/store/usage"
)

// The native toast viewport is already rendered by MyGo above every main panel.
// This is only the presentation of the usage store's account notice; it owns no
// account reader, polling timer, or second acknowledgement queue.
type accountBonusNoticeState struct {
	store      *usagestore.Store
	bonus      usagestore.Bonus
	generation uint64
	published  string
	phase      string
	dismissed  bool
}

const accountBonusToastID = "whaleisle-account-bonus"

func (s *mainState) syncAccountBonusNotice(c *ui.Context) {
	s.mu.Lock()
	store := s.usage
	s.mu.Unlock()
	var snapshot usagestore.ViewState
	if store != nil {
		snapshot = store.Snapshot()
	}
	var candidate usagestore.Bonus
	for _, bonus := range snapshot.Bonuses {
		if expires, err := time.Parse(time.RFC3339, bonus.ExpiresAt); err == nil && !expires.After(time.Now()) {
			continue
		}
		if snapshot.BonusFeedback.AccountID == bonus.AccountID && snapshot.BonusFeedback.OrderID == bonus.OrderID &&
			snapshot.BonusFeedback.State == "acknowledged" {
			continue
		}
		candidate = bonus
		break
	}
	s.mu.Lock()
	current := &s.accountNotice
	active := store != nil && snapshot.ClientReady && snapshot.AccountStatus == "credential-stored"
	if active && candidate.OrderID == "" && (snapshot.Loading || snapshot.BonusesLoading) && current.store == store {
		candidate = current.bonus
	}
	if !active || candidate.AccountID == "" || candidate.OrderID == "" {
		// Retire ownership before closing: a programmatic close after sign-out
		// or account replacement is not the user seeing the old account's notice.
		generation := current.generation + 1
		hadToast := current.published != ""
		*current = accountBonusNoticeState{generation: generation}
		s.mu.Unlock()
		if hadToast {
			c.CloseToast(accountBonusToastID)
		}
		return
	}
	if current.store != store || current.bonus.AccountID != candidate.AccountID || current.bonus.OrderID != candidate.OrderID {
		*current = accountBonusNoticeState{store: store, bonus: candidate, generation: current.generation + 1}
	} else {
		current.bonus = candidate
	}
	phase, detail := "offered", candidate.Message
	feedback := snapshot.BonusFeedback
	if feedback.AccountID == candidate.AccountID && feedback.OrderID == candidate.OrderID {
		switch feedback.State {
		case "pending":
			phase = "pending"
			detail += "\n" + resumeText("正在确认已读…", "Confirming notice…")
		case "failed":
			phase = "failed"
			detail += "\n" + resumeText("确认失败，可重试：", "Could not acknowledge; retry: ") + feedback.Error
		case "refused":
			phase = "refused"
			detail += "\n" + resumeText("当前账户无法确认这条通知，请刷新账户状态。", "This account cannot acknowledge the notice. Refresh the account state.")
		}
	}
	if current.phase != phase {
		current.dismissed = false
	}
	current.phase = phase
	key := candidate.AccountID + "\n" + candidate.OrderID + "\n" + candidate.ExpiresAt + "\n" + phase + "\n" + detail
	if current.dismissed || current.published == key {
		s.mu.Unlock()
		return
	}
	current.published = key
	generation := current.generation
	s.mu.Unlock()

	action := resumeText("知道了", "Got it")
	if phase == "failed" {
		action = resumeText("重试", "Retry")
	} else if phase == "pending" {
		action = ""
	} else if phase == "refused" {
		action = resumeText("关闭", "Close")
	}
	actionTaken := false
	acknowledge := func() {
		s.mu.Lock()
		current := &s.accountNotice
		if current.store != store || current.generation != generation {
			s.mu.Unlock()
			return
		}
		if phase == "pending" || phase == "refused" {
			current.dismissed = true
			s.mu.Unlock()
			return
		}
		current.published = ""
		current.dismissed = false
		s.mu.Unlock()
		// Clicking Got it, Retry, or dismissing the offered card proves it was
		// seen. The existing store owns busy/error/false-result handling.
		store.AckBonusNotified(context.Background(), candidate.AccountID, candidate.OrderID, "")
	}
	c.AddToast(ui.Toast{
		ID: accountBonusToastID, Title: resumeText("奖励通知", "Reward notice"), Description: detail,
		Timeout: -1, Action: action,
		OnAction: func() {
			actionTaken = true
			acknowledge()
		},
		OnClose: func() {
			if !actionTaken {
				acknowledge()
			}
		},
	})
	if expires, err := time.Parse(time.RFC3339, candidate.ExpiresAt); err == nil {
		c.After(time.Until(expires))
	}
}
