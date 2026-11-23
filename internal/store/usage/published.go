package usage

import (
	"encoding/json"
	"errors"
	"reflect"

	"whaleisle/internal/state"
)

// views is everything the usage store publishes. Each accessor returns its
// part of the current version without locking or copying; writers replace
// slices and maps wholesale or append past every published length, never
// editing an index a published version can see.
type views struct {
	State          ViewState
	Stats          StatsView
	Billing        BillingView
	BillingModels  BillingModelsView
	SessionRanking SessionRanking
	ProjectRanking ProjectRanking
	ProjectCard    ProjectRanking
	Repair         RepairView
	Notices        NoticeState
}

func (s *Store) buildViewsLocked() views {
	v := views{
		State: ViewState{
			Details:          s.details,
			ClientReady:      s.client != nil,
			Loading:          s.loading,
			Err:              s.err,
			AccountStatus:    s.accountStatus,
			UsageURL:         s.usageURL,
			TopUpURL:         s.topUpURL,
			Email:            s.email,
			Name:             s.name,
			Plan:             s.plan,
			AttemptPhase:     s.attemptPhase,
			AttemptID:        s.attemptID,
			AttemptErrorCode: s.attemptErrorCode,
			AuthorizeURL:     s.authorizeURL,
			ExpiryNote:       s.expiryNote,
			BalanceSummary:   s.balanceSummary,
			HasRunningTasks:  s.hasRunningTasks,
			Recent:           s.recent,
			DayBuckets:       s.dayBuckets,
		},
		Stats: StatsView{
			raw: s.stats.Raw, Range: s.stats.Range, RangeStart: s.stats.RangeStart, RangeEnd: s.stats.RangeEnd,
			RangeError: s.stats.RangeError, Loading: s.stats.Loading, Cached: s.stats.Cached,
			Err: s.stats.Err, ErrLead: s.stats.ErrLead, Days: s.stats.Days, Overview: s.stats.parsed.overview,
		},
		Billing:        s.billing.BillingView,
		BillingModels:  s.billing.models,
		SessionRanking: s.ranking.SessionRanking,
		ProjectRanking: s.projectRanking.ProjectRanking,
		ProjectCard:    ProjectRanking{Sort: "tokens"},
		Repair:         s.repair.RepairView,
		Notices:        s.notices,
	}
	if v.State.Err == nil {
		v.State.Err = s.accountStreamErr
	}
	if v.Stats.Days == 0 {
		v.Stats.Days = 7
	}
	if o := v.Stats.Overview; o != nil {
		v.Stats.UpdatedAt = o.UpdatedAt
	}
	if v.SessionRanking.Sort == "" {
		v.SessionRanking.Sort = "tokens"
	}
	if v.ProjectRanking.Sort == "" {
		v.ProjectRanking.Sort = "tokens"
	}
	if s.projectRanking.card != nil {
		v.ProjectCard = s.projectRanking.card.ProjectRanking
	}
	v.Notices.Generation = s.accountEpoch
	return v
}

// unlockPublish publishes the views if they differ from the current version,
// then releases mu. Every critical section that may have changed a published
// field ends with it; a section that changed nothing mints no revision. The
// comparison short-circuits on shared slices, maps and pointers, so an
// unchanged store compares in proportion to its scalar fields. The account
// ViewState has its own publication so Snapshot's revision advances only
// when that view changed.
func (s *Store) unlockPublish() {
	next := s.buildViewsLocked()
	if v := s.pub.Load(); v == nil || !reflect.DeepEqual(v.Value, next) {
		if sv := s.statePub.Load(); sv == nil || !reflect.DeepEqual(sv.Value, next.State) {
			s.statePub.Publish(next.State)
		}
		s.pub.Publish(next)
	}
	s.mu.Unlock()
}

func (s *Store) views() *state.Version[views] {
	if v := s.pub.Load(); v != nil {
		return v
	}
	s.mu.Lock()
	s.unlockPublish()
	return s.pub.Load()
}

// Snapshot returns the current published account view. The same *Version
// is shared by every reader until a change publishes the next revision;
// its Value is immutable.
func (s *Store) Snapshot() *state.Version[ViewState] {
	if v := s.statePub.Load(); v != nil {
		return v
	}
	s.views()
	return s.statePub.Load()
}

// Changed closes when the next version of any usage view is published.
func (s *Store) Changed() <-chan struct{} { return s.pub.Changed() }

// Stats returns the published statistics view; Overview is decoded once
// per payload, outside the state lock, and shared by every reader.
func (s *Store) Stats() StatsView { return s.views().Value.Stats }

func (s *Store) Billing() BillingView             { return s.views().Value.Billing }
func (s *Store) BillingModels() BillingModelsView { return s.views().Value.BillingModels }
func (s *Store) SessionRanking() SessionRanking   { return s.views().Value.SessionRanking }
func (s *Store) ProjectRanking() ProjectRanking   { return s.views().Value.ProjectRanking }
func (s *Store) ProjectCardRanking() ProjectRanking {
	return s.views().Value.ProjectCard
}
func (s *Store) Repair() RepairView   { return s.views().Value.Repair }
func (s *Store) Notices() NoticeState { return s.views().Value.Notices }

// decodedStats is the overview payload decoded once, outside the state
// lock, at the moment the raw payload changes.
type decodedStats struct {
	overview *Overview
	// coverageOK reports a complete failure-coverage block, which repair
	// verification requires.
	coverageOK bool
}

func decodeStats(raw json.RawMessage) decodedStats {
	if len(raw) == 0 {
		return decodedStats{}
	}
	var d decodedStats
	var o Overview
	if json.Unmarshal(raw, &o) == nil {
		d.overview = &o
	}
	var coverage struct {
		Coverage struct {
			FailedIDs *[]string `json:"failedSessionIds"`
			Failed    *int      `json:"sessionsFailed"`
		} `json:"coverage"`
	}
	d.coverageOK = d.overview != nil && json.Unmarshal(raw, &coverage) == nil && coverage.Coverage.FailedIDs != nil && coverage.Coverage.Failed != nil &&
		*coverage.Coverage.Failed >= 0 && (*coverage.Coverage.Failed == 0 || len(*coverage.Coverage.FailedIDs) > 0)
	return d
}

// curatedError carries authored validation copy that is safe to show
// verbatim; faultOf keeps that copy and classifies every other failure.
type curatedError struct{ text string }

func (e *curatedError) Error() string { return e.text }

func curated(text string) error { return &curatedError{text: text} }

func faultOf(err error) *state.Fault {
	var c *curatedError
	if errors.As(err, &c) {
		return state.FaultText(c.text)
	}
	return state.FaultOf(err)
}

// cloneDrafts copies the draft map before an edit: the published billing
// view shares the current one.
func cloneDrafts(d map[string]PriceDraft) map[string]PriceDraft {
	out := make(map[string]PriceDraft, len(d)+1)
	for k, v := range d {
		out[k] = v
	}
	return out
}
