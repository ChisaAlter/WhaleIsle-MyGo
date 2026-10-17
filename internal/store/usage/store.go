// Package usage loads account and session activity stubs for the M9 usage panel.
//
// Generated dsh APIs expose account/getState (links.usageUrl), account/getBalance,
// and job/list — not the dsh-usage-panel chart RPCs. Full charts stay in the
// harness plugin until native porting.
package usage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// Entry is one row in the recent-activity list.
type Entry struct {
	Label  string
	Detail string
	// JobID/Status set for job/list rows so the UI can call job/kill.
	JobID  string
	Status string
}

// ViewState is a UI-safe snapshot of the store.
type ViewState struct {
	ClientReady    bool
	Loading        bool
	Err            string
	AccountStatus  string
	UsageURL       string
	TopUpURL       string
	Email          string
	Name           string
	Plan           string
	AttemptPhase   string
	AttemptID      string
	AuthorizeURL    string
	ExpiryNote      string
	BalanceSummary  string
	HasRunningTasks bool
	Bonuses         []Bonus
	Recent          []Entry
	DayBuckets      []DayBucket
}

// Bonus is one unnotified account bonus (account/getUnnotifiedBonuses).
type Bonus struct {
	AccountID string
	OrderID   string
	Campaign  string
	Amount    string
	Currency  string
	Message   string
}

// Store fetches account stats and optional session job list from dsh RPC.
type Store struct {
	mu     sync.Mutex
	client *dsh.Client
	mux    *dsh.Mux
	notify func()

	jobFollowCancel    context.CancelFunc
	accountWatchCancel context.CancelFunc
	expiryWatchCancel  context.CancelFunc

	loading        bool
	err            error
	accountStatus  string
	usageURL       string
	topUpURL       string
	email          string
	name           string
	plan           string
	attemptPhase   string
	attemptID      string
	authorizeURL   string
	expiryNote     string
	balanceSummary  string
	hasRunningTasks bool
	bonuses         []Bonus
	recent          []Entry
	dayBuckets      []DayBucket
}

// New returns an empty usage store.
func New() *Store {
	return &Store{}
}

// Attach binds a dsh client and an optional UI refresh hook.
func (s *Store) Attach(client *dsh.Client, notify func()) {
	s.mu.Lock()
	s.client = client
	s.notify = notify
	s.err = nil
	s.mu.Unlock()
}

// Detach clears the client when harness stops.
func (s *Store) Detach() {
	s.mu.Lock()
	if s.jobFollowCancel != nil {
		s.jobFollowCancel()
		s.jobFollowCancel = nil
	}
	if s.accountWatchCancel != nil {
		s.accountWatchCancel()
		s.accountWatchCancel = nil
	}
	if s.expiryWatchCancel != nil {
		s.expiryWatchCancel()
		s.expiryWatchCancel = nil
	}
	s.client = nil
	s.mux = nil
	s.loading = false
	s.accountStatus = ""
	s.usageURL = ""
	s.topUpURL = ""
	s.email = ""
	s.name = ""
	s.plan = ""
	s.attemptPhase = ""
	s.attemptID = ""
	s.authorizeURL = ""
	s.expiryNote = ""
	s.balanceSummary = ""
	s.hasRunningTasks = false
	s.bonuses = nil
	s.recent = nil
	s.dayBuckets = nil
	s.mu.Unlock()
	s.fire()
}

// Snapshot copies observable state for one frame.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{
		ClientReady:    s.client != nil,
		Loading:        s.loading,
		AccountStatus:  s.accountStatus,
		UsageURL:       s.usageURL,
		TopUpURL:       s.topUpURL,
		Email:          s.email,
		Name:           s.name,
		Plan:           s.plan,
		AttemptPhase:   s.attemptPhase,
		AttemptID:      s.attemptID,
		AuthorizeURL:   s.authorizeURL,
		ExpiryNote:     s.expiryNote,
		BalanceSummary:  s.balanceSummary,
		HasRunningTasks: s.hasRunningTasks,
	}
	if s.err != nil {
		st.Err = s.err.Error()
	}
	st.Bonuses = append([]Bonus(nil), s.bonuses...)
	st.Recent = append([]Entry(nil), s.recent...)
	st.DayBuckets = append([]DayBucket(nil), s.dayBuckets...)
	return st
}

// Refresh loads account/getState, account/getBalance, and job/list for sessionID.
func (s *Store) Refresh(ctx context.Context, sessionID string) {
	s.mu.Lock()
	client := s.client
	if client == nil {
		s.mu.Unlock()
		return
	}
	s.loading = true
	s.err = nil
	s.mu.Unlock()
	s.fire()

	go func() {
		var state api.AccountView
		if err := client.Call(ctx, "account/getState", nil, &state); err != nil {
			s.finishRefresh(accountRefresh{
				err: err,
			})
			return
		}
		status := state.Status
		usageURL := state.Links.UsageUrl
		topUpURL := state.Links.TopUpUrl
		attemptPhase := state.Attempt.Phase
		attemptID := state.Attempt.Id
		authorizeURL := ""
		if state.Attempt.AuthorizeUrl != nil {
			authorizeURL = *state.Attempt.AuthorizeUrl
		}

		_, zoneOff := time.Now().Zone()
		clientMeta := api.AccountClientMetadata{
			Version:               "whaleisle",
			Locale:                "zh-CN",
			TimezoneOffsetSeconds: float64(zoneOff),
		}

		var profileRaw api.GetProfileResult
		email, name, plan := "", "", ""
		if err := client.Call(ctx, "account/getProfile", map[string]any{"client": clientMeta}, &profileRaw); err == nil {
			email, name, plan = ParseProfileSummary(profileRaw)
		}

		var balanceRaw api.GetBalanceResult
		balanceSummary := ""
		if err := client.Call(ctx, "account/getBalance", map[string]any{"client": clientMeta}, &balanceRaw); err != nil {
			balanceSummary = fmt.Sprintf("(account/getBalance: %v)", err)
		} else if len(balanceRaw) > 0 {
			balanceSummary = truncateJSON(string(balanceRaw), 480)
		} else {
			balanceSummary = "(empty balance response)"
		}

		// account/watch + account/watchExpiry are ModeStream — see WatchAccount / WatchExpiry.
		var hasTasks api.HasRunningAccountTasksResult
		_ = client.Call(ctx, "account/hasRunningAccountTasks", map[string]any{}, &hasTasks)

		var bonuses []Bonus
		var bonusRaw api.GetUnnotifiedBonusesResult
		if err := client.Call(ctx, "account/getUnnotifiedBonuses", map[string]any{"client": clientMeta}, &bonusRaw); err == nil {
			for _, b := range bonusRaw.Bonuses {
				msg := strings.TrimSpace(b.Message)
				if msg == "" {
					msg = strings.TrimSpace(b.Campaign)
				}
				bonuses = append(bonuses, Bonus{
					AccountID: bonusRaw.AccountId,
					OrderID:   b.OrderId,
					Campaign:  b.Campaign,
					Amount:    b.Amount,
					Currency:  b.Currency,
					Message:   msg,
				})
			}
		}

		var recent []Entry
		var buckets []DayBucket
		recent = append(recent, Entry{
			Label:  "account",
			Detail: fmt.Sprintf("status=%s", status),
		})
		for _, b := range bonuses {
			recent = append(recent, Entry{
				Label:  "bonus",
				Detail: strings.TrimSpace(b.Amount + " " + b.Currency + " · " + b.Message),
			})
		}
		if usageURL != "" {
			recent = append(recent, Entry{
				Label:  "usageUrl",
				Detail: usageURL,
			})
		}

		if sessionID != "" {
			var jobs api.JobListFrame
			req := api.JobListRequest{SessionId: sessionID}
			if err := client.Call(ctx, "job/list", map[string]any{"request": req}, &jobs); err != nil {
				recent = append(recent, Entry{
					Label:  "job/list",
					Detail: err.Error(),
				})
			} else {
				buckets = BuildDayBuckets(jobs.Jobs, 28, time.Now())
				for _, j := range jobs.Jobs {
					label := j.Kind
					if j.Label != "" {
						label = j.Label
					}
					recent = append(recent, Entry{
						Label:  label,
						Detail: fmt.Sprintf("%s · %s · started %.0f", j.Kind, j.Status, j.StartedAt),
						JobID:  j.Id,
						Status: j.Status,
					})
				}
				if len(jobs.Jobs) == 0 {
					recent = append(recent, Entry{
						Label:  "job/list",
						Detail: "no jobs in selected session",
					})
				}
			}
		}

		s.finishRefresh(accountRefresh{
			status:          status,
			usageURL:        usageURL,
			topUpURL:        topUpURL,
			email:           email,
			name:            name,
			plan:            plan,
			attemptPhase:    attemptPhase,
			attemptID:       attemptID,
			authorizeURL:    authorizeURL,
			balanceSummary:  balanceSummary,
			hasRunningTasks: bool(hasTasks),
			bonuses:         bonuses,
			recent:          recent,
			buckets:         buckets,
		})
	}()
}

type accountRefresh struct {
	err             error
	status          string
	usageURL        string
	topUpURL        string
	email           string
	name            string
	plan            string
	attemptPhase    string
	attemptID       string
	authorizeURL    string
	balanceSummary  string
	hasRunningTasks bool
	bonuses         []Bonus
	recent          []Entry
	buckets         []DayBucket
}

func (s *Store) finishRefresh(r accountRefresh) {
	s.mu.Lock()
	s.loading = false
	s.err = r.err
	s.accountStatus = r.status
	s.usageURL = r.usageURL
	s.topUpURL = r.topUpURL
	s.email = r.email
	s.name = r.name
	s.plan = r.plan
	s.attemptPhase = r.attemptPhase
	s.attemptID = r.attemptID
	s.authorizeURL = r.authorizeURL
	s.balanceSummary = r.balanceSummary
	s.hasRunningTasks = r.hasRunningTasks
	s.bonuses = r.bonuses
	s.recent = r.recent
	s.dayBuckets = r.buckets
	s.mu.Unlock()
	s.fire()
}

// AckBonusNotified calls account/ackBonusNotified then refreshes.
func (s *Store) AckBonusNotified(ctx context.Context, accountID, orderID, sessionID string) {
	accountID = strings.TrimSpace(accountID)
	orderID = strings.TrimSpace(orderID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || accountID == "" || orderID == "" {
		return
	}
	go func() {
		_, zoneOff := time.Now().Zone()
		clientMeta := api.AccountClientMetadata{
			Version:               "whaleisle",
			Locale:                "zh-CN",
			TimezoneOffsetSeconds: float64(zoneOff),
		}
		var out api.AckBonusNotifiedResult
		err := client.Call(ctx, "account/ackBonusNotified", map[string]any{
			"accountId": accountID,
			"orderId":   orderID,
			"client":    clientMeta,
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.err = err
			s.mu.Unlock()
			s.fire()
			return
		}
		s.err = nil
		s.mu.Unlock()
		s.Refresh(ctx, sessionID)
	}()
}

// KillJob calls job/kill for a running job in sessionID, then refreshes.
func (s *Store) KillJob(ctx context.Context, sessionID, jobID string) {
	sessionID = strings.TrimSpace(sessionID)
	jobID = strings.TrimSpace(jobID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || sessionID == "" || jobID == "" {
		return
	}
	go func() {
		var out api.JobKillValue
		err := client.Call(ctx, "job/kill", map[string]any{
			"request": api.JobKillRequest{SessionId: sessionID, JobId: jobID},
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.err = err
			s.mu.Unlock()
			s.fire()
			return
		}
		s.err = nil
		s.mu.Unlock()
		s.Refresh(ctx, sessionID)
	}()
}

func truncateJSON(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func (s *Store) fire() {
	s.mu.Lock()
	fn := s.notify
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}
