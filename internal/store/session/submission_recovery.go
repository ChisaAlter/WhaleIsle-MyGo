package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"whaleisle/internal/dsh"
)

// ErrSubmissionUnconfirmed means dispatch may have reached the Host. It must
// never be presented as an ordinary failed draft that is safe to resend.
var ErrSubmissionUnconfirmed = errors.New("发送结果待确认，请查看会话；不要重复发送 / Send outcome unconfirmed; check the conversation before resending")

func unconfirmedSubmission(cause error) error {
	if cause == nil {
		return ErrSubmissionUnconfirmed
	}
	return fmt.Errorf("%w: %v", ErrSubmissionUnconfirmed, cause)
}

// SubmissionRecord is the minimal recovery owner for an admitted user action.
// It is kept in the desktop's existing resume store, not in Host history.
type SubmissionRecord struct {
	RequestID   string               `json:"requestId"`
	SessionID   string               `json:"sessionId"`
	Text        string               `json:"text"`
	Placement   string               `json:"placement"`
	Phase       string               `json:"phase"`
	Error       string               `json:"error,omitempty"`
	Attachments []ComposerAttachment `json:"attachments,omitempty"`
	// DraftSnapshot is carried only to the resume writer for the initial atomic
	// checkpoint. It is not duplicated inside the submission journal.
	DraftKey      string          `json:"draftKey,omitempty"`
	DraftSequence uint64          `json:"draftSequence,omitempty"`
	DraftSnapshot json.RawMessage `json:"-"`
}

type SubmissionRecovery struct {
	Version int                         `json:"version"`
	Owner   string                      `json:"owner"`
	Records map[string]SubmissionRecord `json:"records"`
}

type submissionPersistence struct {
	mu       sync.Mutex
	owner    string
	write    func(SubmissionRecord) error
	terminal map[string]bool
}

type submissionWatch struct {
	ctx    context.Context
	cancel context.CancelFunc
	client *dsh.Client
	mux    *dsh.Mux
	epoch  uint64
	wake   chan struct{}
}

// BindSubmissionRecovery reconnects only unresolved requests to this exact
// profile. No loaded record can dispatch a prompt. Persistence is opt-in so
// existing programmatic SubmitComposer callers retain their original contract.
func (s *Store) BindSubmissionRecovery(ctx context.Context, owner string, raw json.RawMessage, write func(SubmissionRecord) error) error {
	var saved SubmissionRecovery
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &saved); err != nil {
			return fmt.Errorf("submission recovery: %w", err)
		}
		if saved.Version != 1 || saved.Owner != owner {
			return errors.New("submission recovery belongs to another profile or version; original retained")
		}
	}
	for id, record := range saved.Records {
		if id == "" || id != record.RequestID || record.SessionID == "" {
			return errors.New("submission recovery contains an invalid request identity")
		}
		if record.Phase == "prepared" {
			// Dispatch is preceded by its own durable transition. A preparation
			// record therefore proves that this request has not been sent.
			record.Phase = "failed"
			record.Error = "发送前准备已中断，草稿已保留 / Preparation interrupted before dispatch; draft retained"
			if write != nil {
				if err := write(record); err != nil {
					return err
				}
			}
			saved.Records[id] = record
		}
	}
	s.mu.Lock()
	writer := &submissionPersistence{owner: owner, write: write, terminal: make(map[string]bool)}
	s.submissionPersistence = writer
	s.submissionContext = ctx
	for id, record := range saved.Records {
		if record.Phase == "confirmed" || record.Phase == "failed" {
			writer.terminal[id] = true
			s.noteRecoveredOutcomeLocked(record)
			continue
		}
		if _, exists := s.composerPending[id]; exists || s.client == nil {
			continue
		}
		t := &ComposerTarget{store: s, client: s.client, SessionID: record.SessionID, epoch: s.composerHostGeneration, mode: PromptModeQueue,
			retirements: map[string]*composerRetirement{id: {done: make(chan struct{})}}}
		if s.composerPending == nil {
			s.composerPending = make(map[string]*composerPending)
		}
		s.composerOrder++
		s.composerPending[id] = &composerPending{PendingSubmission: PendingSubmission{RequestID: id, SessionID: record.SessionID, Text: record.Text,
			Placement: record.Placement, Attachments: append([]ComposerAttachment(nil), record.Attachments...), Accepted: record.Phase == "accepted",
			Unconfirmed: true, Order: s.composerOrder}, target: t, recovered: true, draftKey: record.DraftKey, draftSequence: record.DraftSequence}
		if s.submissionTracked == nil {
			s.submissionTracked = make(map[string]*composerPending)
		}
		s.submissionTracked[id] = s.composerPending[id]
	}
	s.startSubmissionWatchesLocked()
	s.mu.Unlock()
	s.fire()
	return nil
}

func (s *Store) RequireSubmissionRecovery(cause error) {
	s.mu.Lock()
	s.submissionPersistence = &submissionPersistence{write: func(SubmissionRecord) error { return cause }}
	s.mu.Unlock()
}

func (s *Store) noteRecoveredOutcomeLocked(record SubmissionRecord) {
	if s.submissionOutcomes == nil {
		s.submissionOutcomes = make(map[string]error)
	}
	if record.Phase == "failed" {
		s.submissionOutcomes[record.RequestID] = errors.New(record.Error)
	} else {
		s.submissionOutcomes[record.RequestID] = nil
	}
}

// SubmissionOutcome is read by the UI owner to retire the matching detached
// snapshot. A false result preserves the snapshot as awaiting confirmation.
func (s *Store) SubmissionOutcome(requestID string) (error, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err, ok := s.submissionOutcomes[requestID]
	if !ok && s.submissionPersistence == nil {
		_, ok = s.composerObserved[requestID]
	}
	return err, ok
}

// AcknowledgeSubmissionOutcome releases the transient UI delivery result. The
// durable terminal record remains until the existing resume checkpoint retires
// its matching draft snapshot in the same write.
func (s *Store) AcknowledgeSubmissionOutcome(requestID string) {
	s.mu.Lock()
	delete(s.submissionOutcomes, requestID)
	s.mu.Unlock()
}

func (t *ComposerTarget) recordSubmission(id, phase string, cause error) error {
	s := t.store
	s.mu.Lock()
	p, writer := s.submissionTracked[id], s.submissionPersistence
	if p == nil || p.target != t || writer == nil {
		s.mu.Unlock()
		return nil
	}
	record := SubmissionRecord{RequestID: id, SessionID: p.SessionID, Text: p.Text, Placement: p.Placement, Phase: phase,
		Attachments: append([]ComposerAttachment(nil), p.Attachments...)}
	if p.recoveryText != nil {
		record.Text = *p.recoveryText
	}
	record.DraftKey, record.DraftSequence = p.draftKey, p.draftSequence
	if phase == "prepared" {
		record.DraftKey = p.draftKey
		record.DraftSnapshot = append(json.RawMessage(nil), p.draftSnapshot...)
	}
	if cause != nil {
		record.Error = faultText("submission", cause)
	}
	for i := range record.Attachments {
		// A local file remains the recovery owner; never duplicate its potentially
		// large base64 payload in the bounded resume snapshot.
		if record.Attachments[i].Path != "" {
			record.Attachments[i].ImageData = ""
		}
	}
	s.mu.Unlock()
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.terminal[id] {
		return nil
	}
	if writer.write == nil {
		return errors.New("submission recovery storage unavailable")
	}
	return writer.write(record)
}

// StageSubmission persists input ownership before attachment or prompt work.
func (t *ComposerTarget) StageSubmission(id string) error {
	return t.recordSubmission(id, "prepared", nil)
}

func (t *ComposerTarget) StageSubmissionDraft(id, text, key string, snapshot json.RawMessage) error {
	t.store.mu.Lock()
	if p := t.store.submissionTracked[id]; p != nil && p.target == t {
		p.recoveryText = &text
		p.draftKey = key
		p.draftSnapshot = append(json.RawMessage(nil), snapshot...)
		var draft struct{ SendSequence uint64 }
		if json.Unmarshal(snapshot, &draft) == nil {
			p.draftSequence = draft.SendSequence
		}
	}
	t.store.mu.Unlock()
	return t.StageSubmission(id)
}

func (s *Store) stopSubmissionWatchesLocked() {
	for _, watch := range s.submissionWatches {
		watch.cancel()
	}
	s.submissionWatches = nil
}

// A carrier can reject a confirmation stream while its connection owner is
// reconnecting. A fresh Events Ready is the authority to resume those reads,
// including a watch that exited on that temporary handshake failure. Reopening
// only observes the original request; it never repeats admission.
func (s *Store) resumeSubmissionWatchesLocked() {
	s.stopSubmissionWatchesLocked()
	for _, pending := range s.submissionTracked {
		if pending.Accepted || pending.Unconfirmed {
			pending.recovered = true
		}
	}
	s.startSubmissionWatchesLocked()
}

func (s *Store) startSubmissionWatchesLocked() {
	if s.submissionContext == nil || s.submissionContext.Err() != nil || s.mux == nil || s.client == nil {
		return
	}
	if s.submissionWatches == nil {
		s.submissionWatches = make(map[string]*submissionWatch)
	}
	for _, p := range s.submissionTracked {
		if p.Placement == "command" || (!p.Accepted && !p.Unconfirmed) || !p.target.currentLocked() || s.submissionWatches[p.SessionID] != nil {
			continue
		}
		ctx, cancel := context.WithCancel(s.submissionContext)
		watch := &submissionWatch{ctx: ctx, cancel: cancel, client: s.client, mux: s.mux, epoch: s.composerHostGeneration, wake: make(chan struct{}, 1)}
		s.submissionWatches[p.SessionID] = watch
		go s.runSubmissionWatch(p.SessionID, watch)
	}
}
