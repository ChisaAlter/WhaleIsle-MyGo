// Package interaction holds pending Host waterfall requests (approval, questions).
package interaction

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/state"
)

const (
	EventApproval        = "approval/request"
	EventUserQuestions   = "user-questions/request"
	outcomeAllowedOnce   = "allowed-once"
	outcomeRejected      = "rejected"
	maxCancelledEventIDs = 1024
)

// Pending is one waterfall waiting for a UI or policy answer.
type Pending struct {
	Continued bool // durable late-answer card; submits only through userQuestions/answer
	EventID   string
	// SourceRequestID is the Host approval identity, not the transport EventID.
	SourceRequestID string
	Event           string
	AgentID         string
	Title           string
	Detail          string
	// Keep approval-owned text intact; UI locale changes must not discard the
	// original reason or the localized displayReason alternatives.
	ApprovalOriginalReason string
	ApprovalDisplayReason  string
	// Args preserves the full original tool input for inspection and copying.
	Args              string
	ApprovalContext   string
	approvalCallID    string
	approvalRequester bool
	approvalResolved  bool
	questionSeen      bool // this call appeared in an authoritative question projection
	questionProjected bool // timed tool protocol, including its indefinite foreground wait
	// CallID is the harness tool call id when present.
	CallID  string
	Sending bool
	Error   *state.Fault
}

// ViewState is UI-safe interaction state.
type ViewState struct {
	QuestionDraftError *state.Fault // render after QuestionDraftSaveFailed
	QuestionHistory    []QuestionHistoryView
	QuestionStatus     string
	AutoAllow          bool
	Banner             string
	Pending            []Pending
	QuestionCards      map[string]QuestionCardView
}

// Store serializes waterfall handlers with the main-thread UI.
type Store struct {
	eventsDeliveryOwner *EventsDeliveryOwner
	eventsDeliveries    map[string]*eventsDelivery
	questionWriter      questionDraftWriter
	questionStorage     questionDraftStorage
	questionFocusFrame  bool
	questionFocusScopes map[string]string
	questionHistory     map[string]*questionHistoryState
	questionRecovery    questionRecovery
	mu                  sync.Mutex
	notify              func()
	client              *dsh.Client
	mux                 *dsh.Mux
	autoAllow           bool
	banner              string
	pending             []Pending
	waiters             map[string]chan dsh.EventResultOutcome
	waitDones           map[string]chan struct{}
	dupDones            map[string]chan struct{}
	activeWait          map[string]uint64
	waitSeq             uint64
	responses           map[string]dsh.EventResultOutcome
	cancelled           map[string]bool
	cancelledOrder      []string
	questionDrafts      map[string]*draftSession
	planReviewOpened    map[planReviewOpenKey]bool
	waitCancels         map[string]context.CancelFunc
	pub                 state.Published[ViewState]
}

// New returns an interaction store with manual tool approval (Electron default).
func New() *Store {
	return &Store{
		autoAllow:      false,
		banner:         "",
		waiters:        make(map[string]chan dsh.EventResultOutcome),
		waitDones:      make(map[string]chan struct{}),
		dupDones:       make(map[string]chan struct{}),
		activeWait:     make(map[string]uint64),
		responses:      make(map[string]dsh.EventResultOutcome),
		cancelled:      make(map[string]bool),
		questionDrafts: make(map[string]*draftSession),
		waitCancels:    make(map[string]context.CancelFunc),
	}
}

// Attach sets the UI refresh hook.
func (s *Store) Attach(notify func()) {
	s.mu.Lock()
	s.notify = notify
	s.unlockPublish()
}

// Detach clears pending state when harness stops.
func (s *Store) Detach() {
	s.mu.Lock()
	s.retireEventsDeliveryLocked(s.eventsDeliveryOwner)
	s.resetQuestionRecoveryLocked()
	s.client = nil
	s.mux = nil
	for id, cancel := range s.waitCancels {
		cancel()
		delete(s.waitCancels, id)
	}
	for id, ch := range s.waiters {
		select {
		case ch <- dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "Error", Message: "client stopped"}}:
		default:
		}
		delete(s.waiters, id)
	}
	for id, done := range s.waitDones {
		close(done)
		delete(s.waitDones, id)
	}
	for id, done := range s.dupDones {
		close(done)
		delete(s.dupDones, id)
	}
	s.activeWait = make(map[string]uint64)
	s.pending = nil
	s.responses = make(map[string]dsh.EventResultOutcome)
	s.cancelled = make(map[string]bool)
	s.cancelledOrder = nil
	s.banner = ""
	s.questionDrafts = make(map[string]*draftSession)
	s.planReviewOpened = nil
	s.unlockPublish()
	s.fire()
}

func (s *Store) markCancelledLocked(id string) {
	if id == "" {
		return
	}
	if s.cancelled == nil {
		s.cancelled = make(map[string]bool)
	}
	if s.cancelled[id] {
		return
	}
	s.cancelled[id] = true
	s.cancelledOrder = append(s.cancelledOrder, id)
	for len(s.cancelledOrder) > maxCancelledEventIDs {
		evict := s.cancelledOrder[0]
		s.cancelledOrder = s.cancelledOrder[1:]
		delete(s.cancelled, evict)
	}
}

// Snapshot returns the current published version: pending requests in
// display order, banner, auto-allow and the draft-save failure. The same
// *Version is shared until a change publishes the next revision; its Value
// is immutable. Question cards are not part of it: they read the live
// answer drafts the UI binds through the Question*Ptr accessors and the
// running wait clock, so they come from QuestionCards and SnapshotForAgent.
func (s *Store) Snapshot() *state.Version[ViewState] {
	if v := s.pub.Load(); v != nil {
		return v
	}
	s.mu.Lock()
	s.unlockPublish()
	return s.pub.Load()
}

// Changed closes when the next version is published.
func (s *Store) Changed() <-chan struct{} { return s.pub.Changed() }

// QuestionCards builds the question cards from the live drafts. This is the
// mutable draft channel; it is computed per call, as before.
func (s *Store) QuestionCards() map[string]QuestionCardView {
	s.mu.Lock()
	defer s.mu.Unlock()
	cards := make(map[string]QuestionCardView, len(s.questionDrafts))
	for id, ds := range s.questionDrafts {
		cards[id] = ds.cardView(id)
	}
	return cards
}

// planReviewPendingLocked reports whether eventID's draft is a plan review,
// which the original domain ranks above other questions and approvals.
func (s *Store) planReviewPendingLocked(eventID string) bool {
	ds := s.questionDrafts[eventID]
	return ds != nil && planReviewOf(ds.req.Questions) != nil
}

// pendingViewLocked is the pending list in display order. Original
// PendingInteractionDomain uses publication order and lets the latest
// value win within a domain's equal precedence.
func (s *Store) pendingViewLocked() []Pending {
	if len(s.pending) == 0 {
		return nil
	}
	out := make([]Pending, 0, len(s.pending))
	for index := len(s.pending) - 1; index >= 0; index-- {
		out = append(out, s.pending[index])
	}
	rank := func(p Pending) int {
		if s.planReviewPendingLocked(p.EventID) {
			return 2
		}
		if p.Event == EventUserQuestions {
			return 1
		}
		return 0
	}
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i]) > rank(out[j]) })
	return out
}

// unlockPublish publishes the view if it no longer matches the working
// state, then releases mu. Every critical section ends with it; one that
// changed nothing mints no revision. Pending entries are edited in place
// through findPending, so the published list is always a fresh copy.
func (s *Store) unlockPublish() {
	next := ViewState{
		QuestionDraftError: s.questionStorage.err,
		AutoAllow:          s.autoAllow,
		Banner:             s.banner,
		Pending:            s.pendingViewLocked(),
	}
	if v := s.pub.Load(); v == nil || !viewEqual(&v.Value, &next) {
		s.pub.Publish(next)
	}
	s.mu.Unlock()
}

func viewEqual(a, b *ViewState) bool {
	return a.QuestionDraftError == b.QuestionDraftError && a.AutoAllow == b.AutoAllow &&
		a.Banner == b.Banner && slices.Equal(a.Pending, b.Pending)
}

// SetAutoAllow toggles immediate allow for tool approvals.
func (s *Store) SetAutoAllow(on bool) {
	s.mu.Lock()
	s.autoAllow = on
	if on {
		s.banner = ""
	}
	s.unlockPublish()
	s.fire()
}

// ResolveAllow answers the oldest or matching pending approval with allowed-once.
func (s *Store) ResolveAllow(eventID string) {
	s.resolve(eventID, approvalOutcome(outcomeAllowedOnce))
}

// ResolveAllowAlways is retained for callers compiled against the old shell.
// The Host protocol grants only one action; it never enables automatic approval.
func (s *Store) ResolveAllowAlways(eventID string) { s.ResolveAllow(eventID) }

// ResolveDeny rejects a pending approval or question.
func (s *Store) ResolveDeny(eventID string) {
	s.mu.Lock()
	p := s.findPending(eventID)
	question := p != nil && p.Event == EventUserQuestions
	if p != nil && p.Continued {
		s.unlockPublish()
		return
	}
	s.unlockPublish()
	if question {
		s.resolve(eventID, dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "UserQuestionError", Code: "ASK_CANCELLED", Message: "User declined to answer"}})
		return
	}
	s.resolve(eventID, approvalOutcome(outcomeRejected))
}

// ResolveQuestions submits the user's option selections for a question waterfall.
func (s *Store) ResolveQuestions(eventID string) {
	s.mu.Lock()
	ds := s.questionDrafts[eventID]
	if ds == nil || planReviewOf(ds.req.Questions) != nil || !ds.answerValid() {
		s.unlockPublish()
		return
	}
	answer := ds.buildAnswer()
	p := s.findPending(eventID)
	continued := p != nil && p.Continued
	s.unlockPublish()
	if continued {
		s.resolveContinuedQuestions(eventID, answer)
		return
	}

	raw, err := json.Marshal(answer)
	if err != nil {
		s.resolve(eventID, dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "Error", Message: err.Error()}})
		return
	}
	s.resolve(eventID, dsh.EventResultOutcome{Kind: "result", Value: raw})
}

// SetQuestionSingleSelect picks one option for a single-select question.
func (s *Store) SetQuestionSingleSelect(eventID, questionID, label string) {
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	defer s.unlockPublish()
	if p := s.findPending(eventID); p != nil && p.Sending {
		return
	}
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		return
	}
	s.engageQuestionLocked(eventID)
	ad.skipped = false
	ad.custom = ""
	ad.selected = []string{label}
	ad.selectWire = label
	for l, ptr := range ad.checks {
		if ptr != nil {
			*ptr = l == label
		}
	}
}

// SetQuestionOptionChecked toggles one option (checkbox multi-select or radio-style single).
func (s *Store) SetQuestionOptionChecked(eventID, questionID, label string, on bool) {
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	defer s.unlockPublish()
	if p := s.findPending(eventID); p != nil && p.Sending {
		return
	}
	ds := s.questionDrafts[eventID]
	if ds == nil {
		return
	}
	var multi bool
	for _, q := range ds.req.Questions {
		if q.ID == questionID {
			multi = q.MultiSelect
			break
		}
	}
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		return
	}
	s.engageQuestionLocked(eventID)
	ad.skipped = false
	if multi {
		if ptr := ad.checks[label]; ptr != nil {
			*ptr = on
		}
		if on {
			ad.selected = appendUnique(ad.selected, label)
		} else {
			ad.selected = removeLabel(ad.selected, label)
		}
		return
	}
	if on {
		ad.custom = ""
		ad.selected = []string{label}
		ad.selectWire = label
		for l, ptr := range ad.checks {
			if ptr != nil {
				*ptr = l == label
			}
		}
	}
}

// SetQuestionCustom sets free-text when a question has no options.
func (s *Store) SetQuestionCustom(eventID, questionID, text string) {
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	defer s.unlockPublish()
	if p := s.findPending(eventID); p != nil && p.Sending {
		return
	}
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		return
	}
	s.engageQuestionLocked(eventID)
	ad.skipped = false
	ad.custom = text
	if strings.TrimSpace(text) != "" {
		ds := s.questionDrafts[eventID]
		for _, q := range ds.req.Questions {
			if q.ID == questionID && !q.MultiSelect {
				ad.selected = nil
				ad.selectWire = ""
				for _, ptr := range ad.checks {
					if ptr != nil {
						*ptr = false
					}
				}
			}
		}
	}
}

// QuestionSingleSelectPtr returns a stable select binding for one question.
func (s *Store) QuestionSingleSelectPtr(eventID, questionID string) *string {
	s.mu.Lock()
	defer s.unlockPublish()
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		empty := ""
		return &empty
	}
	return &ad.selectWire
}

// QuestionCustomPtr returns a stable text binding for a free-text question.
func (s *Store) QuestionCustomPtr(eventID, questionID string) *string {
	s.mu.Lock()
	defer s.unlockPublish()
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		empty := ""
		return &empty
	}
	return &ad.custom
}

// QuestionOptionCheckedPtr returns a stable checkbox binding for one option label.
func (s *Store) QuestionOptionCheckedPtr(eventID, questionID, label string) *bool {
	s.mu.Lock()
	defer s.unlockPublish()
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		f := false
		return &f
	}
	if ad.checks == nil {
		ad.checks = make(map[string]*bool)
	}
	if ptr, ok := ad.checks[label]; ok && ptr != nil {
		return ptr
	}
	v := containsLabel(ad.selected, label)
	ad.checks[label] = &v
	return ad.checks[label]
}

// QuestionsCanSubmit reports whether all answers are filled for a pending card.
func (s *Store) QuestionsCanSubmit(eventID string) bool {
	s.mu.Lock()
	defer s.unlockPublish()
	ds := s.questionDrafts[eventID]
	if ds == nil || planReviewOf(ds.req.Questions) != nil {
		return false
	}
	return ds.answerValid()
}

func (s *Store) answerDraftLocked(eventID, questionID string) *answerDraft {
	ds := s.questionDrafts[eventID]
	if ds == nil {
		return nil
	}
	ad := ds.answers[questionID]
	if ad == nil {
		ad = &answerDraft{checks: make(map[string]*bool)}
		ds.answers[questionID] = ad
	}
	return ad
}

func (s *Store) resolve(eventID string, outcome dsh.EventResultOutcome) {
	s.mu.Lock()
	p := s.findPending(eventID)
	if p == nil || p.Sending {
		s.unlockPublish()
		return
	}
	eventID = p.EventID
	ch := s.waiters[eventID]
	if ch == nil {
		s.unlockPublish()
		return
	}
	p.Sending, p.Error = true, nil
	s.responses[eventID] = outcome
	s.closeDupDoneLocked(eventID)
	select {
	case ch <- outcome:
	default:
	}
	s.unlockPublish()
	s.fire()
}

// OnResult retains the card and its exact response on delivery failure.
// Returning true lets the event stream wait for an explicit user retry.
func (s *Store) OnResult(result dsh.EventResult, err error) bool {
	return s.onResultForDelivery(nil, result, err)
}

func (s *Store) onResultForDelivery(owner *EventsDeliveryOwner, result dsh.EventResult, err error) bool {
	if strings.TrimSpace(result.EventID) == "" {
		return false
	}
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	if owner != nil && !s.eventsResultCurrentLocked(owner, result) {
		s.unlockPublish()
		return false
	}
	p := s.findPending(result.EventID)
	if result.Outcome.Delivery != nil && s.responses[result.EventID].Delivery != result.Outcome.Delivery {
		s.unlockPublish()
		return false
	}
	retry := false
	if p != nil {
		if err == nil {
			if p.Event == EventUserQuestions && p.CallID != "" && p.questionProjected {
				// HTTP delivery is not Host acceptance: an expired waterfall
				// outcome may be silently ignored. Keep this draft and timed
				// claim until the Host closes or continues the actual request.
				s.unlockPublish()
				s.fire()
				return false
			}
			agentID := p.AgentID
			delete(s.waiters, result.EventID)
			s.dropPending(result.EventID)
			s.reconcileProjectedQuestionsLocked(agentID)
		} else {
			p.Sending = false
			p.Error = state.FaultOf(err)
			retry = true
		}
	}
	if owner != nil && p == nil {
		s.forgetEventsDeliveryLocked(result.EventID)
		delete(s.responses, result.EventID)
	}
	s.unlockPublish()
	s.fire()
	return retry
}

func (s *Store) Retry(eventID string) {
	s.mu.Lock()
	p := s.findPending(eventID)
	if p != nil && p.Event == EventUserQuestions && p.Error != nil {
		draft := s.questionDrafts[eventID]
		if draft != nil && planReviewOf(draft.req.Questions) == nil {
			s.unlockPublish()
			s.ResolveQuestions(eventID)
			return
		}
	}
	if p != nil && p.Continued {
		draft := s.questionDrafts[eventID]
		plan := draft != nil && planReviewOf(draft.req.Questions) != nil
		s.unlockPublish()
		if plan {
			s.ApprovePlanReview(eventID)
			return
		}
		s.ResolveQuestions(eventID)
		return
	}
	outcome, ok := s.responses[eventID]
	ok = ok && p != nil && p.Error != nil
	s.unlockPublish()
	if ok {
		s.resolve(eventID, outcome)
	}
}

// SnapshotForAgent prevents a request from appearing in another conversation.
// Pending comes from the published version; the agent's question cards,
// recovery status and history are read live, as before.
func (s *Store) SnapshotForAgent(agentID string) ViewState {
	snap := s.Snapshot().Value
	var pending []Pending
	for _, p := range snap.Pending {
		if p.AgentID == agentID && agentID != "" {
			pending = append(pending, p)
		}
	}
	snap.Pending = pending
	cards := make(map[string]QuestionCardView)
	s.mu.Lock()
	for _, p := range pending {
		if ds := s.questionDrafts[p.EventID]; ds != nil {
			cards[p.EventID] = ds.cardView(p.EventID)
		}
	}
	snap.QuestionCards = cards
	if p := s.questionRecovery.agents[agentID]; p != nil {
		snap.QuestionStatus = p.status
	}
	snap.QuestionHistory = s.questionHistorySnapshotLocked(agentID)
	s.unlockPublish()
	return snap
}

func (s *Store) dropPending(eventID string) {
	if eventID == "" {
		return
	}
	s.rememberQuestionDraftLocked(eventID)
	s.forgetEventsDeliveryLocked(eventID)
	s.closeWaitDoneLocked(eventID)
	s.closeDupDoneLocked(eventID)
	delete(s.activeWait, eventID)
	if cancel := s.waitCancels[eventID]; cancel != nil {
		cancel()
		delete(s.waitCancels, eventID)
	}
	delete(s.responses, eventID)
	delete(s.questionDrafts, eventID)
	out := s.pending[:0]
	for _, p := range s.pending {
		if p.EventID != eventID {
			out = append(out, p)
		}
	}
	s.pending = out
}

func (s *Store) ensureWaitDoneLocked(eventID string) chan struct{} {
	if s.waitDones == nil {
		s.waitDones = make(map[string]chan struct{})
	}
	ch := s.waitDones[eventID]
	if ch == nil {
		ch = make(chan struct{})
		s.waitDones[eventID] = ch
	}
	return ch
}

func (s *Store) nextWaitTokenLocked(eventID string) uint64 {
	if s.activeWait == nil {
		s.activeWait = make(map[string]uint64)
	}
	s.waitSeq++
	s.activeWait[eventID] = s.waitSeq
	return s.waitSeq
}

func (s *Store) clearActiveWait(eventID string, tok uint64) {
	if tok == 0 {
		return
	}
	s.mu.Lock()
	if s.activeWait[eventID] == tok {
		delete(s.activeWait, eventID)
	}
	s.unlockPublish()
}

func (s *Store) ensureDupDoneLocked(eventID string) chan struct{} {
	if s.dupDones == nil {
		s.dupDones = make(map[string]chan struct{})
	}
	ch := s.dupDones[eventID]
	if ch == nil {
		ch = make(chan struct{})
		s.dupDones[eventID] = ch
	}
	return ch
}

func (s *Store) closeWaitDoneLocked(eventID string) {
	if ch := s.waitDones[eventID]; ch != nil {
		close(ch)
		delete(s.waitDones, eventID)
	}
}

func (s *Store) closeDupDoneLocked(eventID string) {
	if ch := s.dupDones[eventID]; ch != nil {
		close(ch)
		delete(s.dupDones, eventID)
	}
}

func (s *Store) registerWaiterLocked(owner *EventsDeliveryOwner, eventID string, ch chan dsh.EventResultOutcome) (*eventsDelivery, chan struct{}, uint64) {
	if s.waitDones == nil {
		s.waitDones = make(map[string]chan struct{})
	}
	if prev := s.waitDones[eventID]; prev != nil {
		close(prev)
	}
	waitDone := make(chan struct{})
	s.waitDones[eventID] = waitDone
	if s.dupDones == nil {
		s.dupDones = make(map[string]chan struct{})
	}
	if prev := s.dupDones[eventID]; prev != nil {
		close(prev)
	}
	s.dupDones[eventID] = make(chan struct{})
	s.waiters[eventID] = ch
	tok := s.nextWaitTokenLocked(eventID)
	return s.registerEventsDeliveryLocked(owner, eventID, ch), waitDone, tok
}

func (s *Store) waitDuplicateWaterfall(owner *EventsDeliveryOwner, eventID string, dupDone <-chan struct{}) *dsh.EventResultOutcome {
	var ownerDone <-chan struct{}
	if owner != nil {
		ownerDone = owner.ctx.Done()
	}
	select {
	case <-dupDone:
	case <-ownerDone:
	}
	if owner != nil {
		s.mu.Lock()
		current := s.eventsOwnerCurrentLocked(owner)
		cancelled := s.cancelled[eventID]
		s.unlockPublish()
		if !current {
			return expiredEventsOutcome(owner, eventID)
		}
		if cancelled {
			out := approvalOutcome("cancelled")
			return s.finishEventsOutcome(owner, eventID, nil, out)
		}
	}
	return &dsh.EventResultOutcome{Kind: "next"}
}

func (s *Store) waitPrimaryWaterfall(owner *EventsDeliveryOwner, d *eventsDelivery, waitDone <-chan struct{}, eventID string, ch chan dsh.EventResultOutcome, tok uint64) (*dsh.EventResultOutcome, error) {
	defer s.clearActiveWait(eventID, tok)
	var deliveryDone, ownerDone <-chan struct{}
	if d != nil {
		deliveryDone = d.ctx.Done()
	}
	if owner != nil {
		ownerDone = owner.ctx.Done()
	}
	select {
	case out := <-ch:
		return s.finishEventsOutcome(owner, eventID, ch, out), nil
	case <-waitDone:
		if owner != nil {
			return expiredEventsOutcome(owner, eventID), nil
		}
		return &dsh.EventResultOutcome{Kind: "next"}, nil
	case <-deliveryDone:
		return expiredEventsOutcome(owner, eventID), nil
	case <-ownerDone:
		return expiredEventsOutcome(owner, eventID), nil
	}
}

func (s *Store) findPending(eventID string) *Pending {
	for i := range s.pending {
		if s.pending[i].EventID == eventID {
			return &s.pending[i]
		}
	}
	if eventID == "" && len(s.pending) > 0 {
		return &s.pending[0]
	}
	return nil
}

// HandleWaterfall implements dsh.EventsHandler waterfall dispatch for the shell.
func (s *Store) HandleWaterfall(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	return s.handleWaterfallForDelivery(nil, wf)
}

func (s *Store) handleWaterfallForDelivery(owner *EventsDeliveryOwner, wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	if strings.TrimSpace(wf.EventID) == "" {
		if owner != nil {
			return expiredEventsOutcome(owner, wf.EventID), nil
		}
		return &dsh.EventResultOutcome{Kind: "next"}, nil
	}
	s.mu.Lock()
	if owner != nil && !s.eventsOwnerCurrentLocked(owner) {
		s.unlockPublish()
		return expiredEventsOutcome(owner, wf.EventID), nil
	}
	existingCh := s.waiters[wf.EventID]
	var (
		retryTok uint64
		waitDone chan struct{}
		delivery *eventsDelivery
		dupDone  chan struct{}
	)
	if existingCh != nil {
		if s.activeWait[wf.EventID] == 0 {
			retryTok = s.nextWaitTokenLocked(wf.EventID)
			waitDone = s.ensureWaitDoneLocked(wf.EventID)
			delivery = s.eventsDeliveries[wf.EventID]
		} else {
			dupDone = s.ensureDupDoneLocked(wf.EventID)
		}
	}
	cancelled := s.cancelled[wf.EventID]
	s.unlockPublish()
	if cancelled {
		out := approvalOutcome("cancelled")
		return s.finishEventsOutcome(owner, wf.EventID, nil, out), nil
	}
	if retryTok != 0 {
		return s.waitPrimaryWaterfall(owner, delivery, waitDone, wf.EventID, existingCh, retryTok)
	}
	if dupDone != nil {
		return s.waitDuplicateWaterfall(owner, wf.EventID, dupDone), nil
	}

	switch wf.Event {
	case EventApproval:
		return s.handleApprovalForDelivery(owner, wf)
	case EventUserQuestions:
		return s.handleUserQuestionsForDelivery(owner, wf)
	default:
		return nil, nil
	}
}

func (s *Store) handleApproval(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	return s.handleApprovalForDelivery(nil, wf)
}

func (s *Store) handleApprovalForDelivery(owner *EventsDeliveryOwner, wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	var req approvalRequest
	_ = json.Unmarshal(wf.Request, &req)
	title := req.ToolName
	if title == "" {
		title = wf.Event
	}
	detail := req.Reason
	if len(req.DisplayReason) > 0 {
		var localized map[string]string
		var plain string
		if json.Unmarshal(req.DisplayReason, &plain) == nil && plain != "" {
			detail = plain
		} else if json.Unmarshal(req.DisplayReason, &localized) == nil && localized["en"] != "" {
			detail = localized["en"]
		}
	}
	args := approvalArgsPreview(req)
	callID := req.CallID
	if callID == "" {
		callID = req.RequestID
	}

	s.mu.Lock()
	auto := s.autoAllow
	s.unlockPublish()
	if auto {
		s.mu.Lock()
		if owner != nil && !s.eventsOwnerCurrentLocked(owner) {
			s.unlockPublish()
			return expiredEventsOutcome(owner, wf.EventID), nil
		}
		s.banner = title
		s.unlockPublish()
		s.fire()
		out := approvalOutcome(outcomeAllowedOnce)
		return s.finishEventsOutcome(owner, wf.EventID, nil, out), nil
	}

	ch := make(chan dsh.EventResultOutcome, 1)
	s.mu.Lock()
	if owner != nil && !s.eventsOwnerCurrentLocked(owner) {
		s.unlockPublish()
		return expiredEventsOutcome(owner, wf.EventID), nil
	}
	if s.cancelled[wf.EventID] {
		s.unlockPublish()
		out := approvalOutcome("cancelled")
		return s.finishEventsOutcome(owner, wf.EventID, nil, out), nil
	}
	if existingCh := s.waiters[wf.EventID]; existingCh != nil {
		if s.activeWait[wf.EventID] == 0 {
			tok := s.nextWaitTokenLocked(wf.EventID)
			waitDone := s.ensureWaitDoneLocked(wf.EventID)
			d := s.eventsDeliveries[wf.EventID]
			s.unlockPublish()
			return s.waitPrimaryWaterfall(owner, d, waitDone, wf.EventID, existingCh, tok)
		}
		dupDone := s.ensureDupDoneLocked(wf.EventID)
		s.unlockPublish()
		return s.waitDuplicateWaterfall(owner, wf.EventID, dupDone), nil
	}
	d, waitDone, tok := s.registerWaiterLocked(owner, wf.EventID, ch)
	s.pending = append(s.pending, Pending{
		EventID:                wf.EventID,
		SourceRequestID:        req.RequestID,
		Event:                  wf.Event,
		AgentID:                wf.AgentID,
		Title:                  title,
		Detail:                 detail,
		ApprovalOriginalReason: req.Reason,
		ApprovalDisplayReason:  string(req.DisplayReason),
		Args:                   args,
		ApprovalContext:        approvalRequesterContext(req),
		approvalCallID:         req.CallID,
		approvalRequester:      req.Requester != nil,
		CallID:                 callID,
	})
	s.unlockPublish()
	s.fire()

	return s.waitPrimaryWaterfall(owner, d, waitDone, wf.EventID, ch, tok)
}

func (s *Store) handleUserQuestions(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	return s.handleUserQuestionsForDelivery(nil, wf)
}

func (s *Store) handleUserQuestionsForDelivery(owner *EventsDeliveryOwner, wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	var req userQuestionsRequest
	_ = json.Unmarshal(wf.Request, &req)
	title := questionCardTitle(req)
	detail := string(wf.Request)
	callID := strings.TrimSpace(req.CallID)
	if callID == "" && req.Wait != nil {
		callID = req.Wait.CallID
	}

	ch := make(chan dsh.EventResultOutcome, 1)
	s.mu.Lock()
	if owner != nil && !s.eventsOwnerCurrentLocked(owner) {
		s.unlockPublish()
		return expiredEventsOutcome(owner, wf.EventID), nil
	}
	mux := s.mux
	if s.cancelled[wf.EventID] {
		s.unlockPublish()
		out := approvalOutcome("cancelled")
		return s.finishEventsOutcome(owner, wf.EventID, nil, out), nil
	}
	if existingCh := s.waiters[wf.EventID]; existingCh != nil {
		if s.activeWait[wf.EventID] == 0 {
			tok := s.nextWaitTokenLocked(wf.EventID)
			waitDone := s.ensureWaitDoneLocked(wf.EventID)
			d := s.eventsDeliveries[wf.EventID]
			s.unlockPublish()
			return s.waitPrimaryWaterfall(owner, d, waitDone, wf.EventID, existingCh, tok)
		}
		dupDone := s.ensureDupDoneLocked(wf.EventID)
		s.unlockPublish()
		return s.waitDuplicateWaterfall(owner, wf.EventID, dupDone), nil
	}
	d, waitDone, tok := s.registerWaiterLocked(owner, wf.EventID, ch)
	s.questionDrafts[wf.EventID] = s.liveQuestionDraftLocked(wf.AgentID, callID, req)
	waitCtx, cancelWait := context.WithCancel(context.Background())
	s.waitCancels[wf.EventID] = cancelWait
	s.pending = append(s.pending, Pending{
		EventID:           wf.EventID,
		SourceRequestID:   unnamedPlanSourceRequestID(wf.Request, callID, req),
		Event:             wf.Event,
		AgentID:           wf.AgentID,
		Title:             title,
		Detail:            detail,
		CallID:            callID,
		questionProjected: req.Wait != nil,
	})
	s.unlockPublish()
	s.fire()
	if callID != "" && req.Wait != nil && req.Wait.Timed {
		s.attachQuestionWait(waitCtx, mux, wf.AgentID, callID, wf.EventID)
	}

	return s.waitPrimaryWaterfall(owner, d, waitDone, wf.EventID, ch, tok)
}

// OnCancel drops a pending waterfall when the host cancels it.
func (s *Store) OnCancel(c dsh.EventsCancel) {
	s.onCancelForDelivery(nil, c)
}

func (s *Store) onCancelForDelivery(owner *EventsDeliveryOwner, c dsh.EventsCancel) {
	if strings.TrimSpace(c.EventID) == "" {
		return
	}
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	if owner != nil && !s.eventsOwnerCurrentLocked(owner) {
		s.unlockPublish()
		return
	}
	s.markCancelledLocked(c.EventID)
	ch, ok := s.waiters[c.EventID]
	if ok {
		delete(s.waiters, c.EventID)
	}
	agentID := ""
	if p := s.findPending(c.EventID); p != nil {
		agentID = p.AgentID
	}
	s.dropPending(c.EventID)
	s.reconcileProjectedQuestionsLocked(agentID)
	s.unlockPublish()
	if ok {
		select {
		case ch <- dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "Error", Message: "cancelled"}}:
		default:
		}
	}
	s.fire()
}

func approvalOutcome(decision string) dsh.EventResultOutcome {
	raw, _ := json.Marshal(decision)
	return dsh.EventResultOutcome{Kind: "result", Value: raw}
}

func defaultQuestionAnswer(req userQuestionsRequest) userQuestionsAnswer {
	var answers []userQuestionsAnswerItem
	for _, q := range req.Questions {
		item := userQuestionsAnswerItem{ID: q.ID}
		if len(q.Options) > 0 {
			item.Selected = []string{q.Options[0].Label}
		} else {
			item.Custom = strPtr("Continue")
		}
		answers = append(answers, item)
	}
	return userQuestionsAnswer{Answers: answers}
}

func strPtr(s string) *string { return &s }

type approvalRequest struct {
	Requester     *approvalRequester `json:"requester"`
	ToolName      string             `json:"toolName"`
	Reason        string             `json:"reason"`
	DisplayReason json.RawMessage    `json:"displayReason"`
	RequestID     string             `json:"requestId"`
	CallID        string             `json:"callId"`
	Input         json.RawMessage    `json:"input"`
	Args          json.RawMessage    `json:"args"`
	Parameters    json.RawMessage    `json:"parameters"`
	Command       string             `json:"command"`
	Description   string             `json:"description"`
}

type userQuestionsRequest struct {
	CallID string `json:"callId"`
	Wait   *struct {
		CallID string `json:"callId"`
		Timed  bool   `json:"timed"`
	} `json:"wait"`
	Questions []userQuestionItem `json:"questions"`
}

type userQuestionItem struct {
	Intent      *planReviewIntent    `json:"intent"`
	ID          string               `json:"id"`
	Question    string               `json:"question"`
	Detail      string               `json:"detail"`
	Header      string               `json:"header"`
	MultiSelect bool                 `json:"multiSelect"`
	Options     []userQuestionOption `json:"options"`
}

func questionCardTitle(req userQuestionsRequest) string {
	if len(req.Questions) == 0 {
		return "Question"
	}
	if len(req.Questions) == 1 {
		if req.Questions[0].Header != "" {
			return req.Questions[0].Header
		}
		if req.Questions[0].Question != "" {
			return req.Questions[0].Question
		}
		return "Question"
	}
	if req.Questions[0].Header != "" {
		return req.Questions[0].Header
	}
	return "Questions"
}

type userQuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type userQuestionsAnswer struct {
	Answers []userQuestionsAnswerItem `json:"answers"`
}

type userQuestionsAnswerItem struct {
	ID       string   `json:"id"`
	Selected []string `json:"selected"`
	Custom   *string  `json:"custom,omitempty"`
}

func (s *Store) fire() {
	s.mu.Lock()
	n := s.notify
	s.mu.Unlock()
	if n != nil {
		n()
	}
}
