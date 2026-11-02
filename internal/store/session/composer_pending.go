package session

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

const MessageKindSubmission MessageKind = "submission"

type PendingSubmission struct {
	RequestID, SessionID, Text, Placement string
	Attachments                           []ComposerAttachment
	Accepted                              bool
	Order                                 uint64
}
type composerPending struct {
	PendingSubmission
	target  *ComposerTarget
	receipt *composerInboxReceipt
}

type composerInboxReceipt struct {
	target string
	seq    float64
	index  int // -1 means consumed but not yet admitted as a durable user message.
}

type composerInboxMessage struct {
	Source struct {
		Kind  string `json:"kind"`
		RPCID string `json:"rpcId"`
	} `json:"source"`
}
type composerObserved struct {
	target    *ComposerTarget
	seq       float64
	placement string
}

func (t *ComposerTarget) BeginSubmission(text string, attachments []ComposerAttachment) string {
	s := t.store
	s.mu.Lock()
	if !t.currentLocked() || t.SessionID == "" {
		s.mu.Unlock()
		return ""
	}
	id := newRequestID()
	if len(attachments) > 0 {
		if t.retirements == nil {
			t.retirements = make(map[string]*composerRetirement)
		}
		t.retirements[id] = &composerRetirement{done: make(chan struct{})}
	}
	s.composerOrder++
	placement := "transcript"
	running := false
	for _, item := range s.items {
		if item.SessionID == t.SessionID && item.Running {
			running = true
		}
	}
	if s.selectedID == t.SessionID && s.turnStartMs > 0 {
		running = true
	}
	if running {
		placement = "queued"
		if t.mode == PromptModeSteer {
			placement = "steering"
		}
	}
	if s.composerPending == nil {
		s.composerPending = make(map[string]*composerPending)
	}
	s.composerPending[id] = &composerPending{PendingSubmission: PendingSubmission{RequestID: id, SessionID: t.SessionID, Text: text, Placement: placement, Attachments: append([]ComposerAttachment(nil), attachments...), Order: s.composerOrder}, target: t}
	// Slash candidates first go through command adjudication. Recognized
	// commands never enter the prompt sink; unknown spellings mark this edge
	// in Submit immediately before their real session/prompt admission.
	if !strings.HasPrefix(strings.TrimSpace(text), "/") {
		s.markPromptAttemptedLocked(t.SessionID)
	}
	s.mu.Unlock()
	s.fire()
	return id
}
func (t *ComposerTarget) UpdateSubmissionAttachments(id string, attachments []ComposerAttachment) {
	s := t.store
	s.mu.Lock()
	if p := s.composerPending[id]; p != nil && p.target == t && t.currentLocked() {
		p.Attachments = append([]ComposerAttachment(nil), attachments...)
	}
	s.mu.Unlock()
	s.fire()
}
func (t *ComposerTarget) AbandonSubmission(id string, err error) error {
	return t.finishSubmission(id, err)
}

// A slash command is adjudicated after the editor has detached. It has no
// session/prompt request whose durable echo could retire this provisional row.
func (t *ComposerTarget) retireCommandSubmission(id string) {
	s := t.store
	s.mu.Lock()
	if p := s.composerPending[id]; p != nil && p.target == t {
		t.settleRetirementLocked(id, nil)
		delete(s.composerPending, id)
	}
	s.mu.Unlock()
	s.fire()
}

func (t *ComposerTarget) finishSubmission(id string, err error) error {
	s := t.store
	s.mu.Lock()
	if s.composerAdmissions[id] == t {
		delete(s.composerAdmissions, id)
	}
	if !t.currentLocked() {
		err = errors.New("Host changed; draft retained, retry on the current Host")
	}
	observed, seen := s.composerObserved[id]
	if seen && observed.target == t && t.currentLocked() {
		err = nil
	}
	if p := s.composerPending[id]; p != nil && p.target == t {
		if err != nil {
			t.settleRetirementLocked(id, err)
			delete(s.composerPending, id)
		} else {
			p.Accepted = true
		}
	}
	if t.currentLocked() && s.selectedID == t.SessionID && s.followGeneration == t.followGeneration {
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "prompt accepted"
		}
	}
	s.mu.Unlock()
	s.fire()
	return err
}
func (s *Store) composerPromptBusyLocked() bool {
	for _, t := range s.composerAdmissions {
		if t.SessionID == s.selectedID && t.currentLocked() {
			return true
		}
	}
	return false
}
func (s *Store) observeComposerRequestLocked(id string, seq float64, inbox bool) {
	s.invalidateViewLocked()
	p := s.composerPending[id]
	if p == nil || p.SessionID != s.selectedID || !p.target.currentLocked() || inbox && p.Placement != "queued" {
		return
	}
	if s.composerObserved == nil {
		s.composerObserved = make(map[string]composerObserved)
	}
	s.composerObserved[id] = composerObserved{target: p.target, seq: seq, placement: p.Placement}
	p.target.settleRetirementLocked(id, nil)
	delete(s.composerPending, id)
	if !inbox && p.Placement != "queued" {
		kept := s.planBoard.Queue[:0]
		for _, q := range s.planBoard.Queue {
			if q.RequestID != id {
				kept = append(kept, q)
			}
		}
		s.planBoard.Queue = kept
	}
}
func (s *Store) observeComposerEventLocked(raw json.RawMessage) (changed bool) {
	s.invalidateViewLocked()
	var e struct {
		Type string  `json:"type"`
		Seq  float64 `json:"seq"`
		Data struct {
			Source struct {
				Kind  string `json:"kind"`
				RPCID string `json:"rpcId"`
			} `json:"source"`
			Target       string                 `json:"target"`
			Start        int                    `json:"start"`
			RemovedCount int                    `json:"removedCount"`
			Inserted     []composerInboxMessage `json:"inserted"`
			Outcome      string                 `json:"outcome"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &e) != nil {
		return false
	}
	if e.Type == "user/message" && e.Data.Source.Kind == "user" && e.Data.Source.RPCID != "" {
		s.observeComposerRequestLocked(e.Data.Source.RPCID, e.Seq, false)
		return true
	}
	if e.Type == "agent/inbox/spliced" {
		for id, p := range s.composerPending {
			r := p.receipt
			if p.SessionID != s.selectedID || !p.target.currentLocked() || r == nil || r.target != e.Data.Target || r.index < 0 || r.seq >= e.Seq {
				continue
			}
			removed := r.index >= e.Data.Start && r.index < e.Data.Start+e.Data.RemovedCount
			if removed && e.Data.Outcome == "canceled" {
				p.target.settleRetirementLocked(id, errors.New("message canceled before admission; draft and attachments retained"))
				delete(s.composerPending, id)
				changed = true
				continue
			}
			r.seq = e.Seq
			if removed {
				r.index = -1
			} else if r.index >= e.Data.Start {
				r.index += len(e.Data.Inserted) - e.Data.RemovedCount
			}
		}
		s.observeComposerInboxInsertionsLocked(e.Data.Target, e.Data.Inserted, e.Data.Start, e.Seq)
		return
	}
	if e.Type == "request/context" || e.Type == "turn/end" {
		for id, p := range s.composerPending {
			r := p.receipt
			if p.SessionID == s.selectedID && p.target.currentLocked() && r != nil && r.index < 0 && r.seq < e.Seq {
				// Admitted messages already retired through user/message above.
				// Only a consumed, unadmitted occurrence can fail this handoff.
				p.target.settleRetirementLocked(id, errors.New("message was not admitted; draft and attachments retained"))
				delete(s.composerPending, id)
				changed = true
			}
		}
	}
	return changed
}

func (s *Store) observeComposerInboxInsertionsLocked(target string, messages []composerInboxMessage, start int, seq float64) {
	s.invalidateViewLocked()
	for index, message := range messages {
		if message.Source.Kind != "user" || message.Source.RPCID == "" {
			continue
		}
		p := s.composerPending[message.Source.RPCID]
		if p == nil || p.SessionID != s.selectedID || !p.target.currentLocked() || p.Placement == "queued" || p.receipt != nil && p.receipt.seq > seq {
			continue
		}
		p.receipt = &composerInboxReceipt{target: target, seq: seq, index: start + index}
	}
}
func (s *Store) hideComposerQueueLocked(id string) bool {
	if id == "" {
		return false
	}
	if p := s.composerPending[id]; p != nil && p.SessionID == s.selectedID && p.target.currentLocked() && p.Placement != "queued" {
		return true
	}
	if o, ok := s.composerObserved[id]; ok && o.target.SessionID == s.selectedID && o.target.currentLocked() && o.placement != "queued" {
		return true
	}
	return false
}
func samePendingSubmissions(a, b []PendingSubmission) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].RequestID != b[i].RequestID || a[i].SessionID != b[i].SessionID ||
			a[i].Text != b[i].Text || a[i].Placement != b[i].Placement ||
			a[i].Accepted != b[i].Accepted || a[i].Order != b[i].Order ||
			len(a[i].Attachments) != len(b[i].Attachments) {
			return false
		}
		for j := range a[i].Attachments {
			if a[i].Attachments[j] != b[i].Attachments[j] {
				return false
			}
		}
	}
	return true
}

func (s *Store) projectComposerSubmissionsLocked(view *ViewState) bool {
	var pending []PendingSubmission
	for _, p := range s.composerPending {
		if p.SessionID != s.selectedID || !p.target.currentLocked() {
			continue
		}
		copy := p.PendingSubmission
		copy.Attachments = append([]ComposerAttachment(nil), p.Attachments...)
		pending = append(pending, copy)
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].Order < pending[j].Order })
	var inline []PendingSubmission
	for _, p := range pending {
		if p.Placement == "queued" {
			view.PlanBoard.Pending = append(view.PlanBoard.Pending, p)
		} else {
			inline = append(inline, p)
			copy := p
			view.Messages = append(view.Messages, Message{ID: "pending:" + p.RequestID, Kind: MessageKindSubmission, Role: "user", Body: p.Text, Submission: &copy})
		}
	}
	kept := view.PlanBoard.Queue[:0]
	for _, q := range view.PlanBoard.Queue {
		if !s.hideComposerQueueLocked(q.RequestID) {
			kept = append(kept, q)
		}
	}
	view.PlanBoard.Queue = kept
	changed := !samePendingSubmissions(s.historyInlinePending, inline)
	s.historyInlinePending = inline
	return changed
}
func (s *Store) resetComposerHostLocked() {
	s.invalidateViewLocked()
	for id, p := range s.composerPending {
		p.target.settleRetirementLocked(id, errors.New("Host changed; send outcome unconfirmed, check the conversation before retrying"))
	}
	s.promptMode, s.promptModeExplicit = PromptModeQueue, false
	s.composerRosterClient = nil
	s.composerHostGeneration++
	s.composerPending = nil
	s.composerObserved = nil
	s.composerAdmissions = nil
	s.promptAttempted = nil
}

// Original sessions/session.ts marks the engaging edge at beginSubmission and
// prompt, retaining it for that resident Session after its failed echo retires.
func (s *Store) markPromptAttemptedLocked(sessionID string) {
	s.invalidateViewLocked()
	if s.promptAttempted == nil {
		s.promptAttempted = make(map[string]bool)
	}
	s.promptAttempted[sessionID] = true
}

// An accepted-but-unobserved admission is still a barrier to editing the older
// turn, even after its RPC has settled. Normal new messages remain admissible.
func (s *Store) composerPendingForSelectedLocked() bool {
	for _, p := range s.composerPending {
		if p.SessionID == s.selectedID && p.target.currentLocked() {
			return true
		}
	}
	return false
}
