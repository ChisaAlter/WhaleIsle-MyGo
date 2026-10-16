// Package usage loads account state and real token statistics from the built-in usage panel.
package usage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/accountdetails"
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
	Details         accountdetails.State
	ClientReady     bool
	Loading         bool
	Err             string
	AccountStatus   string
	UsageURL        string
	TopUpURL        string
	Email           string
	Name            string
	Plan            string
	AttemptPhase    string
	AttemptID       string
	AuthorizeURL    string
	ExpiryNote      string
	BalanceSummary  string
	HasRunningTasks bool
	Bonuses         []Bonus
	BonusesLoading  bool
	BonusesError    string
	BonusFeedback   BonusFeedback
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
	ExpiresAt string
}

// Store fetches account stats and optional session job list from dsh RPC.
type Store struct {
	details        accountdetails.State
	accountEpoch   uint64
	accountRead    uint64
	ranking        rankingState
	projectRanking projectRankingState
	repair         repairState
	billing        billingState
	stats          statsState
	statsCachePath string
	statsEpoch     uint64
	statsCancel    context.CancelFunc
	mu             sync.Mutex
	client         *dsh.Client
	mux            *dsh.Mux
	notify         func()

	jobFollowCancel    context.CancelFunc
	accountWatchCancel context.CancelFunc
	expiryWatchCancel  context.CancelFunc

	loading         bool
	err             error
	accountStatus   string
	usageURL        string
	topUpURL        string
	email           string
	name            string
	plan            string
	attemptPhase    string
	attemptID       string
	authorizeURL    string
	expiryNote      string
	balanceSummary  string
	hasRunningTasks bool
	bonuses         []Bonus
	bonusesLoading  bool
	bonusesError    string
	bonusFeedback   BonusFeedback
	recent          []Entry
	dayBuckets      []DayBucket
}

// New returns an empty usage store.
func New() *Store {
	return &Store{details: accountdetails.Empty()}
}

// Attach binds a dsh client and an optional UI refresh hook.
func (s *Store) Attach(client *dsh.Client, notify func()) {
	s.mu.Lock()
	s.client = client
	s.resetAccountDetailsLocked()
	s.accountStatus, s.usageURL, s.topUpURL = "", "", ""
	s.attemptPhase, s.attemptID, s.authorizeURL, s.expiryNote = "", "", "", ""
	s.bonuses = nil
	s.mux = nil
	if s.accountWatchCancel != nil {
		s.accountWatchCancel()
		s.accountWatchCancel = nil
	}
	if s.expiryWatchCancel != nil {
		s.expiryWatchCancel()
		s.expiryWatchCancel = nil
	}
	s.statsEpoch++
	if s.statsCancel != nil {
		s.statsCancel()
		s.statsCancel = nil
	}
	s.stats = statsState{}
	s.restoreStatsCacheLocked()
	s.resetRankingLocked()
	s.resetBillingLocked()
	s.resetRepairLocked()
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
	s.resetAccountDetailsLocked()
	s.statsEpoch++
	if s.statsCancel != nil {
		s.statsCancel()
		s.statsCancel = nil
	}
	s.stats = statsState{}
	s.resetRankingLocked()
	s.resetBillingLocked()
	s.resetRepairLocked()
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
		Details:         s.details.Clone(),
		ClientReady:     s.client != nil,
		Loading:         s.loading,
		AccountStatus:   s.accountStatus,
		UsageURL:        s.usageURL,
		TopUpURL:        s.topUpURL,
		Email:           s.email,
		Name:            s.name,
		Plan:            s.plan,
		AttemptPhase:    s.attemptPhase,
		AttemptID:       s.attemptID,
		AuthorizeURL:    s.authorizeURL,
		ExpiryNote:      s.expiryNote,
		BalanceSummary:  s.balanceSummary,
		HasRunningTasks: s.hasRunningTasks,
		BonusesLoading:  s.bonusesLoading,
		BonusesError:    s.bonusesError,
		BonusFeedback:   s.bonusFeedback,
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
	s.RefreshStats(ctx, false)
	s.mu.Lock()
	client := s.client
	if client == nil {
		s.mu.Unlock()
		return
	}
	s.resetAccountDetailsLocked()
	epoch := s.accountEpoch
	s.loading = true
	s.bonusesLoading = true
	s.err = nil
	s.mu.Unlock()
	s.fire()

	go func() {
		var state api.AccountView
		if err := client.Call(ctx, "account/getState", nil, &state); err != nil {
			s.finishRefresh(accountRefresh{
				err:    err,
				client: client, epoch: epoch,
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

		s.refreshAccountDetails(ctx, client, epoch, status)

		// account/watch + account/watchExpiry are ModeStream — see WatchAccount / WatchExpiry.
		var hasTasks api.HasRunningAccountTasksResult
		_ = client.Call(ctx, "account/hasRunningAccountTasks", map[string]any{}, &hasTasks)

		bonuses, bonusErr := readAccountBonuses(ctx, client, status)

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
			status:       status,
			usageURL:     usageURL,
			topUpURL:     topUpURL,
			attemptPhase: attemptPhase,
			attemptID:    attemptID,
			authorizeURL: authorizeURL,
			client:       client, epoch: epoch,
			hasRunningTasks: bool(hasTasks),
			bonuses:         bonuses,
			bonusErr:        bonusErr,
			recent:          recent,
			buckets:         buckets,
		})
	}()
}

type accountRefresh struct {
	client          *dsh.Client
	epoch           uint64
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
	bonusErr        error
	recent          []Entry
	buckets         []DayBucket
}

func (s *Store) finishRefresh(r accountRefresh) {
	s.mu.Lock()
	if s.client != r.client || s.accountEpoch != r.epoch {
		s.mu.Unlock()
		return
	}
	s.loading = false
	s.err = r.err
	s.accountStatus = r.status
	s.usageURL = r.usageURL
	s.topUpURL = r.topUpURL
	s.attemptPhase = r.attemptPhase
	s.attemptID = r.attemptID
	s.authorizeURL = r.authorizeURL
	s.hasRunningTasks = r.hasRunningTasks
	s.bonuses = r.bonuses
	s.bonusesLoading = false
	s.bonusesError = ""
	if r.bonusErr != nil {
		s.bonusesError = r.bonusErr.Error()
	}
	s.recent = r.recent
	s.dayBuckets = r.buckets
	s.mu.Unlock()
	s.fire()
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
