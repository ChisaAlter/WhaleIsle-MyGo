package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/events"
)

// AttachMux binds the shared remote.mux used for session/follow.
func (s *Store) AttachMux(mux *dsh.Mux) {
	s.mu.Lock()
	if s.mux != mux {
		s.invalidateFollowLocked()
	}
	s.mux = mux
	s.mu.Unlock()
}

// FollowSelected opens session/follow for the captured selection/generation.
func (s *Store) FollowSelected(ctx context.Context) {
	s.mu.Lock()
	mux, sessionID := s.mux, s.selectedID
	if mux == nil || sessionID == "" || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	s.invalidateFollowLocked()
	s.historyLoading = true
	generation := s.followGeneration
	fctx, cancel := context.WithCancel(ctx)
	s.followCancel = cancel
	control, _ := s.beginControlWatchLocked(ctx)
	s.mu.Unlock()
	s.fire()
	go s.runFollow(fctx, mux, sessionID, generation)
	if control != nil {
		go control()
	}
}

func (s *Store) followCurrentLocked(ctx context.Context, sessionID string, generation uint64) bool {
	return ctx.Err() == nil && s.selectedID == sessionID && s.followGeneration == generation
}

func (s *Store) clearHistoryErrorLocked() {
	s.invalidateViewLocked()
	s.setHistoryErrLocked("", nil)
}

func (s *Store) reportFollowError(ctx context.Context, sessionID string, generation uint64, prefix string, err error) {
	if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
		return
	}
	s.mu.Lock()
	if !s.followCurrentLocked(ctx, sessionID, generation) {
		s.mu.Unlock()
		return
	}
	s.historyLoading = false
	s.historyReconnecting = false
	s.setHistoryErrLocked(prefix, err)
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
}

func (s *Store) markFollowReconnecting(ctx context.Context, sessionID string, generation uint64) bool {
	s.mu.Lock()
	if !s.followCurrentLocked(ctx, sessionID, generation) {
		s.mu.Unlock()
		return false
	}
	changed := !s.historyReconnecting || s.historyFault != nil || s.historyLoading
	s.historyReconnecting = true
	s.historyLoading = false
	s.clearHistoryErrorLocked()
	s.invalidateViewLocked()
	s.mu.Unlock()
	if changed {
		s.fire()
	}
	return true
}

func (s *Store) runFollow(ctx context.Context, mux *dsh.Mux, sessionID string, generation uint64) {
	backoff := 100 * time.Millisecond
	for {
		restart, progressed := s.runFollowAttempt(ctx, mux, sessionID, generation)
		if !restart {
			return
		}
		if progressed {
			backoff = 100 * time.Millisecond
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if !progressed && backoff < 2*time.Second {
			backoff *= 2
			if backoff > 2*time.Second {
				backoff = 2 * time.Second
			}
		}
	}
}

func (s *Store) runFollowConnection(ctx context.Context, mux *dsh.Mux, sessionID string, generation uint64) bool {
	restart, _ := s.runFollowAttempt(ctx, mux, sessionID, generation)
	return restart
}

func (s *Store) runFollowAttempt(ctx context.Context, mux *dsh.Mux, sessionID string, generation uint64) (bool, bool) {
	s.mu.Lock()
	current := s.followCurrentLocked(ctx, sessionID, generation)
	s.mu.Unlock()
	if !current {
		return false, false
	}
	if err := mux.Dial(ctx); err != nil {
		if errors.Is(err, dsh.ErrCarrierLost) {
			return s.markFollowReconnecting(ctx, sessionID, generation), false
		}
		s.reportFollowError(ctx, sessionID, generation, "follow dial: ", err)
		return false, false
	}
	payload := map[string]any{"args": map[string]any{"request": map[string]any{
		"address": sessionAddressJSON(sessionID), "assistantStream": true,
		"maxMessages": historyPageMaxMessages,
		"turnWindow":  map[string]any{"minMessages": 50, "minTurns": 2},
	}}}
	stream, err := mux.Open(ctx, "session/follow", payload)
	if err != nil {
		if errors.Is(err, dsh.ErrCarrierLost) {
			return s.markFollowReconnecting(ctx, sessionID, generation), false
		}
		s.reportFollowError(ctx, sessionID, generation, "follow open: ", err)
		return false, false
	}
	defer stream.Cancel()
	s.mu.Lock()
	if !s.followCurrentLocked(ctx, sessionID, generation) {
		s.mu.Unlock()
		return false, false
	}
	s.mu.Unlock()
	opened, cursorKnown := false, false
	cursor := float64(-1)
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			// Clean logical stream EOF restarts silently for the active selection.
			if errors.Is(err, dsh.ErrStreamEnd) && opened {
				s.mu.Lock()
				current := s.followCurrentLocked(ctx, sessionID, generation)
				s.mu.Unlock()
				return current, true
			}
			// Physical mux loss preserves loaded transcript and enters reconnecting
			// wait instead of flashing a terminal error or giving up.
			if errors.Is(err, dsh.ErrCarrierLost) {
				return s.markFollowReconnecting(ctx, sessionID, generation), opened
			}
			// A lazily activated follow service, or a stream this client dropped on
			// overflow, re-opens with the loop's backoff and a fresh snapshot.
			if errors.Is(err, dsh.ErrEndpointNotReady) || errors.Is(err, dsh.ErrStreamOverflow) {
				s.mu.Lock()
				current := s.followCurrentLocked(ctx, sessionID, generation)
				s.mu.Unlock()
				return current, opened
			}
			if !errors.Is(err, dsh.ErrStreamEnd) {
				s.reportFollowError(ctx, sessionID, generation, "follow: ", err)
			} else {
				s.mu.Lock()
				awaitingSnapshot := s.followCurrentLocked(ctx, sessionID, generation) && s.historyLoading
				s.mu.Unlock()
				if awaitingSnapshot {
					s.reportFollowError(ctx, sessionID, generation, "follow: ", errors.New("history stream ended before snapshot"))
				}
			}
			return false, opened
		}
		var frame struct {
			Type   string   `json:"type"`
			Cursor *float64 `json:"cursor"`
			Event  struct {
				Seq *float64 `json:"seq"`
			} `json:"event"`
		}
		if json.Unmarshal(raw, &frame) == nil && frame.Type == "event" && frame.Event.Seq != nil && cursorKnown {
			if *frame.Event.Seq <= cursor {
				continue
			}
			if *frame.Event.Seq != cursor+1 {
				// Never fold an event over a hole. Reopening obtains the atomic
				// durable window and assistant baseline from the Host.
				return true, opened
			}
		}
		if !s.applyFollowFrameForTarget(ctx, sessionID, generation, raw) {
			return false, opened
		}
		if frame.Type == "snapshot" {
			opened = true
			if frame.Cursor != nil {
				cursor, cursorKnown = *frame.Cursor, true
			}
		} else if frame.Type == "event" && frame.Event.Seq != nil {
			cursor, cursorKnown = *frame.Event.Seq, true
		}
		s.mu.Lock()
		restart := s.followCurrentLocked(ctx, sessionID, generation) && s.assistantStream.rebaseline
		s.mu.Unlock()
		if restart {
			return true, opened
		}
	}
}

type parsedEventHead struct {
	Type string  `json:"type"`
	Seq  float64 `json:"seq"`
	Time float64 `json:"time"`
}

type followSnapshotRecord struct {
	Type          string          `json:"type"`
	Event         json.RawMessage `json:"event"`
	seq           float64
	seqOK         bool
	head          parsedEventHead
	headOK        bool
	preclassified bool
	classifiedMsg Message
	classifiedOK  bool
}

type followSnapshotFrame struct {
	Projections *struct {
		AsOfSeq float64
		Values  json.RawMessage
	} `json:"projections"`
	AssistantStream *assistantStreamBaseline `json:"assistantStream"`
	Cursor          *float64                 `json:"cursor"`
	HasMore         *bool                    `json:"hasMore"`
	Records         []followSnapshotRecord   `json:"records"`
}

type parsedFollowFrame struct {
	raw      json.RawMessage
	kind     string
	kindOK   bool
	snap     followSnapshotFrame
	snapOK   bool
	eventRaw json.RawMessage
	eventOK  bool
}

func parseFollowFrame(raw json.RawMessage) parsedFollowFrame {
	out := parsedFollowFrame{raw: raw}
	var head struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &head) != nil {
		return out
	}
	out.kind = head.Type
	out.kindOK = true
	switch head.Type {
	case "snapshot":
		if json.Unmarshal(raw, &out.snap) == nil && out.snap.Records != nil {
			for i := range out.snap.Records {
				if out.snap.Records[i].Type == "event" {
					evRaw := out.snap.Records[i].Event
					var evHead struct {
						Type string   `json:"type"`
						Seq  *float64 `json:"seq"`
						Time float64  `json:"time"`
					}
					if json.Unmarshal(evRaw, &evHead) == nil && evHead.Seq != nil {
						seq := *evHead.Seq
						out.snap.Records[i].seq = seq
						out.snap.Records[i].seqOK = true
						if evHead.Type != "" {
							out.snap.Records[i].head = parsedEventHead{
								Type: evHead.Type,
								Seq:  seq,
								Time: evHead.Time,
							}
							out.snap.Records[i].headOK = true
							if isToolTimingEventType(evHead.Type) && seq > 0 {
								readToolLifecycleMemo(seq, evRaw)
							}
							switch evHead.Type {
							case events.TypeAgentPresetSelected, "session/presentation", events.TypeModelSelection, "turn/start", "turn/end":
							default:
								msg, ok := classifyPersistenceEvent(evRaw)
								out.snap.Records[i].preclassified = true
								out.snap.Records[i].classifiedMsg = msg
								out.snap.Records[i].classifiedOK = ok
							}
						}
					}
				}
			}
			out.snapOK = true
		}
	case "event":
		var wrap struct {
			Event json.RawMessage `json:"event"`
		}
		if json.Unmarshal(raw, &wrap) == nil {
			out.eventRaw = wrap.Event
			out.eventOK = true
		}
	}
	return out
}

// The generation check and the entire frame fold share one critical section.
// A selection cannot change between validation and any message/plan/history write.
func (s *Store) applyFollowFrameForTarget(ctx context.Context, sessionID string, generation uint64, raw json.RawMessage) bool {
	wantGoalRefresh := goalRefreshFollowFrame(raw)
	parsed := parseFollowFrame(raw)
	s.mu.Lock()
	if !s.followCurrentLocked(ctx, sessionID, generation) {
		s.mu.Unlock()
		return false
	}
	changed := s.applyFollowFrameParsedLocked(parsed)
	client := s.client
	refreshGoal := changed && client != nil && wantGoalRefresh
	s.mu.Unlock()
	if changed {
		s.fireFollowFrame(parsed)
	}
	if refreshGoal {
		go s.runRefreshGoal(ctx, client, sessionID, generation)
	}
	return true
}

func (s *Store) applyFollowFrame(raw json.RawMessage) {
	parsed := parseFollowFrame(raw)
	s.mu.Lock()
	changed := s.applyFollowFrameParsedLocked(parsed)
	s.mu.Unlock()
	if changed {
		s.fireFollowFrame(parsed)
	}
}

// fireFollowFrame publishes the revision a followed frame produced. An
// assistant-stream frame only moves the streaming message state; anything it
// does beyond that invalidates the full view itself while it holds the lock.
func (s *Store) fireFollowFrame(parsed parsedFollowFrame) {
	if parsed.kindOK && parsed.kind == "assistant-stream" {
		s.fireStream()
		return
	}
	s.fire()
}

func (s *Store) applyFollowFrameLocked(raw json.RawMessage) bool {
	return s.applyFollowFrameParsedLocked(parseFollowFrame(raw))
}

func (s *Store) applyFollowFrameParsedLocked(parsed parsedFollowFrame) bool {
	s.invalidateStreamViewLocked()
	if !parsed.kindOK {
		s.invalidateViewLocked()
		return false
	}
	if parsed.kind != "assistant-stream" {
		s.invalidateViewLocked()
	}
	switch parsed.kind {
	case "snapshot":
		if !parsed.snapOK {
			return false
		}
		snap := parsed.snap
		s.historyLoading = false
		s.historyReconnecting = false
		s.historyReady = true
		s.clearHistoryErrorLocked()
		expectedBinding := fmt.Sprintf("%p\x00%s", s.client, s.selectedID)
		sameSession := s.selectedID != "" && len(s.messages) > 0 && s.planHistoryBinding == expectedBinding
		snapMinSeq, snapHasSeq := snapshotMinPersistenceSeq(snap.Records)
		prevMinSeq := oldestPersistenceSeq(s.messages)
		prevMaxSeq := maxPersistenceSeq(s.messages)
		keepPrefix := sameSession && snapHasSeq && prevMinSeq > 0 && prevMinSeq < snapMinSeq && snapMinSeq <= prevMaxSeq+1
		hasGap := sameSession && snapHasSeq && prevMinSeq > 0 && prevMaxSeq > 0 && snapMinSeq > prevMaxSeq+1

		var (
			prefixMsgs     []Message
			prefixWorkflow map[float64]json.RawMessage
			prefixTraj     map[float64]json.RawMessage
			prefixAnchors  map[float64]float64
			prefixInsp     map[float64]json.RawMessage
			prefixSettled  map[float64]string
			prefixHasMore  bool
		)
		if keepPrefix {
			prefixHasMore = s.historyHasMore
			for _, m := range s.messages {
				if m.Seq > 0 && m.Seq < snapMinSeq {
					prefixMsgs = append(prefixMsgs, m)
				}
			}
			for seq, raw := range s.workflowBoundaries {
				if seq < snapMinSeq {
					if prefixWorkflow == nil {
						prefixWorkflow = map[float64]json.RawMessage{}
					}
					prefixWorkflow[seq] = raw
				}
			}
			for seq, raw := range s.trajectoryBoundaries {
				if seq < snapMinSeq {
					if prefixTraj == nil {
						prefixTraj = map[float64]json.RawMessage{}
					}
					prefixTraj[seq] = raw
				}
			}
			for seq, anchor := range s.systemPromptAnchors {
				if seq < snapMinSeq {
					if prefixAnchors == nil {
						prefixAnchors = map[float64]float64{}
					}
					prefixAnchors[seq] = anchor
				}
			}
			if s.inspector.client == s.client && s.inspector.session == s.selectedID && s.inspector.generation == s.followGeneration {
				for seq, raw := range s.inspector.events {
					if seq < snapMinSeq {
						if prefixInsp == nil {
							prefixInsp = map[float64]json.RawMessage{}
						}
						prefixInsp[seq] = raw
					}
				}
				for seq, root := range s.inspector.settledRoots {
					if seq < snapMinSeq {
						if prefixSettled == nil {
							prefixSettled = map[float64]string{}
						}
						prefixSettled[seq] = root
					}
				}
			}
		} else {
			s.clearPerSessionDecodeCachesLocked()
		}

		if snap.Projections != nil {
			s.applySessionPresentationValuesLocked(s.selectedID, "sequenced", snap.Projections.AsOfSeq, snap.Projections.Values)
			s.applySessionSubagentIdentityValuesLocked(s.selectedID, "sequenced", snap.Projections.AsOfSeq, snap.Projections.Values)
			s.applyCostBaselineLocked(s.selectedID, snap.Projections.AsOfSeq, snap.Projections.Values)
		}
		s.resetInspectorLocked()
		s.planHistoryBinding = expectedBinding
		s.turnFilesHistoryBinding = turnFileHistoryBinding(s.client, s.selectedID)
		s.messages = nil
		s.workflowBoundaries = nil
		s.trajectoryBoundaries = nil
		s.systemPromptAnchors = nil
		s.systemPromptRowsCache = nil
		s.promptEventCache = nil
		s.retryRowsCache = nil
		s.durableTimedCache = nil
		s.resetAssistantStreamLocked()
		s.turnStartMs = 0
		s.editTurnOpening = false
		s.editTurnSeq = 0
		s.resetPlanBoardLocked()
		s.resetHistoryPagingLocked()
		s.historyThroughSeq = 0
		for _, rec := range snap.Records {
			if rec.Type == "event" && rec.seqOK {
				s.publishAssistantDurableRecordLocked(rec)
			}
		}
		if keepPrefix && len(prefixMsgs) > 0 {
			for seq, raw := range prefixWorkflow {
				if s.workflowBoundaries == nil {
					s.workflowBoundaries = map[float64]json.RawMessage{}
				}
				if _, exists := s.workflowBoundaries[seq]; !exists {
					s.workflowBoundaries[seq] = raw
				}
			}
			for seq, raw := range prefixTraj {
				if s.trajectoryBoundaries == nil {
					s.trajectoryBoundaries = map[float64]json.RawMessage{}
				}
				if _, exists := s.trajectoryBoundaries[seq]; !exists {
					s.trajectoryBoundaries[seq] = raw
				}
			}
			for seq, anchor := range prefixAnchors {
				if s.systemPromptAnchors == nil {
					s.systemPromptAnchors = map[float64]float64{}
				}
				if _, exists := s.systemPromptAnchors[seq]; !exists {
					s.systemPromptAnchors[seq] = anchor
				}
			}
			for seq, raw := range prefixInsp {
				if _, exists := s.inspector.events[seq]; !exists {
					s.noteInspectorEventLocked(seq)
					s.inspector.events[seq] = raw
					s.inspector.rawBytes += len(raw)
				}
			}
			for seq, root := range prefixSettled {
				if s.inspector.settledRoots == nil {
					s.inspector.settledRoots = map[float64]string{}
				}
				if _, exists := s.inspector.settledRoots[seq]; !exists {
					s.inspector.settledRoots[seq] = root
				}
			}
			s.pruneInspectorEventsLocked()
			for _, m := range prefixMsgs {
				if m.Seq > 0 {
					s.noteHistoryBaseLocked(m.Seq)
				}
			}
			s.messages = append(prefixMsgs, s.messages...)
		}
		s.pruneInspectorSettlementsLocked()
		s.restoreAssistantStreamLocked(snap.AssistantStream)
		if snap.Projections != nil {
			s.applyQueueValuesLocked(snap.Projections.Values, snap.Projections.AsOfSeq)
			s.applyQuestionValuesLocked(snap.Projections.Values, snap.Projections.AsOfSeq)
			s.applyPermissionValuesLocked(snap.Projections.Values, snap.Projections.AsOfSeq)
			s.applyModelValuesLocked(snap.Projections.Values, snap.Projections.AsOfSeq)
			s.applyContextValuesLocked(snap.Projections.Values, snap.Projections.AsOfSeq)
			s.applyPlanModeValuesLocked(snap.Projections.Values, snap.Projections.AsOfSeq)
			s.applyImageLimitValuesLocked(snap.Projections.Values, snap.Projections.AsOfSeq)
			s.applyHistoryOutlineValuesLocked(snap.Projections.Values, snap.Projections.AsOfSeq)
		}
		if snap.Cursor != nil {
			s.bumpHistoryThroughSeq(*snap.Cursor)
		}
		if keepPrefix && len(prefixMsgs) > 0 {
			s.historyHasMore = prefixHasMore
		} else if snap.HasMore != nil {
			s.historyHasMore = *snap.HasMore
			if hasGap {
				s.historyHasMore = true
			}
		} else {
			s.syncHistoryHasMoreAfterSnapshotLocked()
			if hasGap {
				s.historyHasMore = true
			}
		}
		return true
	case "event":
		if !parsed.eventOK {
			return false
		}
		return s.acceptAssistantDurableLocked(parsed.eventRaw)
	case "assistant-stream":
		return s.applyAssistantStreamLocked(parsed.raw)
	default:
		return false
	}
}

func snapshotMinPersistenceSeq(records []followSnapshotRecord) (float64, bool) {
	var minSeq float64
	found := false
	for _, rec := range records {
		if rec.Type != "event" || !rec.seqOK || rec.seq <= 0 {
			continue
		}
		if !found || rec.seq < minSeq {
			minSeq = rec.seq
			found = true
		}
	}
	return minSeq, found
}

func (s *Store) applyPersistenceEvent(raw json.RawMessage) {
	s.mu.Lock()
	changed := s.applyPersistenceEventLocked(raw)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}

func (s *Store) publishAssistantDurableRecordLocked(rec followSnapshotRecord) bool {
	if s.assistantStream.published == nil {
		s.assistantStream.published = make(map[float64]bool)
	}
	s.assistantStream.published[rec.seq] = true
	return s.applyPersistenceRecordLocked(rec.Event, rec.head, rec.headOK, rec.preclassified, rec.classifiedMsg, rec.classifiedOK)
}

func (s *Store) applyPersistenceEventLocked(raw json.RawMessage) bool {
	return s.applyPersistenceRecordLocked(raw, parsedEventHead{}, false, false, Message{}, false)
}

func (s *Store) applyPersistenceRecordLocked(raw json.RawMessage, head parsedEventHead, headOK bool, preclassified bool, preMsg Message, preOK bool) bool {
	s.invalidateViewLocked()
	if !headOK {
		if json.Unmarshal(raw, &head) != nil || head.Type == "" {
			return false
		}
	} else if head.Type == "" {
		return false
	}
	if isToolTimingEventType(head.Type) && head.Seq > 0 {
		readToolLifecycleMemo(head.Seq, raw)
	}
	var inspectorChanged bool
	if headOK {
		inspectorChanged = s.captureInspectorEventWithHeadLocked(raw, head.Type, head.Seq)
	} else {
		inspectorChanged = s.captureInspectorEventLocked(raw)
	}
	s.captureWorkflowBoundaryLocked(raw)
	timingChanged := s.captureTrajectoryBoundaryWithSeqLocked(raw, head.Seq)
	var submissionChanged bool
	if head.Type == "user/message" || head.Type == "agent/inbox/spliced" || head.Type == "request/context" || head.Type == "turn/end" {
		submissionChanged = s.observeComposerEventLocked(raw)
	}
	s.bumpHistoryThroughSeq(head.Seq)
	s.noteHistoryBaseLocked(head.Seq)
	var changed bool
	if head.Type != "assistant/message" && head.Type != "assistant/attempt" && head.Type != "tool/result" && head.Type != "user/message" && head.Type != "step/start" && head.Type != "step/end" && head.Type != "turn/end" {
		changed = ApplyPlanBoardEvent(&s.planBoard, raw)
	}
	changed = changed || timingChanged || inspectorChanged || submissionChanged
	if head.Type == events.TypeAgentPresetSelected {
		var ev events.AgentPresetSelected
		if json.Unmarshal(raw, &ev) == nil && ev.Data.AgentPreset != "" {
			s.selectedAgentPresetLabel = labelForAgentPresetID(ev.Data.AgentPreset, s.agentPresetOptions)
			s.invalidateCommandsLocked()
			s.invalidateSkillOptionsLocked()
			go s.RefreshSkills(context.Background())
		}
		return true
	}
	switch head.Type {
	case "session/presentation":
		var event struct {
			Data *SessionPresentation `json:"data"`
		}
		if json.Unmarshal(raw, &event) == nil {
			s.applySessionPresentationLocked(s.selectedID, "sequenced", head.Seq, event.Data)
			return true
		}
		return changed
	case events.TypeModelSelection:
		return s.applyModelSelectionEventLocked(raw)
	case "turn/start":
		s.editTurnSeq, s.editTurnOpening = head.Seq, true
		s.turnStartMs = head.Time
		return true
	case "turn/end":
		turnSeq := s.editTurnSeq
		elapsed := 0.0
		if s.turnStartMs > 0 && head.Time >= s.turnStartMs {
			elapsed = head.Time - s.turnStartMs
		}
		s.turnStartMs = 0
		s.editTurnOpening = false
		s.editTurnSeq = 0
		s.messages = append(s.messages, Message{ID: newRequestID(), Kind: MessageKindText, Role: "status", Seq: head.Seq, EventType: "turn/end", TurnSeq: turnSeq, Raw: append(json.RawMessage(nil), raw...), Body: fmt.Sprintf("%.0f", elapsed), EventTime: head.Time})
		return true
	}
	var msg Message
	var ok bool
	if preclassified {
		msg, ok = preMsg, preOK
	} else {
		msg, ok = classifyPersistenceEvent(raw)
	}
	if !ok {
		return changed
	}
	if msg.ID == "" {
		msg.ID = newRequestID()
	}
	annotateEditMessage(&msg, s.editTurnSeq, &s.editTurnOpening)
	s.messages = append(s.messages, msg)
	return true
}

func summarizeEvent(typ string, data json.RawMessage) (role, body string) {
	switch typ {
	case "user/message", "user/prompt":
		return "user", extractTextish(data)
	case "assistant/message":
		return "assistant", extractTextish(data)
	case "assistant/thinking":
		return "thinking", extractTextish(data)
	default:
		if text := extractTextish(data); text != "" {
			return typ, text
		}
		return "", ""
	}
}

func extractChunkText(chunk json.RawMessage) string {
	if len(chunk) == 0 || string(chunk) == "null" {
		return ""
	}
	var asString string
	if err := json.Unmarshal(chunk, &asString); err == nil {
		return asString
	}
	var typed struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(chunk, &typed) == nil && (typed.Type == "text-delta" || typed.Type == "text") {
		return typed.Text
	}
	return ""
}

func extractTextish(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if img := stringAsImagePlaceholder(s); img != "" {
			return img
		}
		return s
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	for _, key := range []string{"text", "content", "message", "delta"} {
		if v, ok := obj[key]; ok {
			if t := extractTextish(v); t != "" {
				return t
			}
		}
	}
	if blocks, ok := obj["blocks"]; ok {
		var arr []json.RawMessage
		if err := json.Unmarshal(blocks, &arr); err == nil {
			var out string
			for _, b := range arr {
				out = appendBlockText(out, b)
			}
			return out
		}
	}
	if content, ok := obj["content"]; ok {
		var arr []json.RawMessage
		if err := json.Unmarshal(content, &arr); err == nil {
			var out string
			for _, b := range arr {
				out = appendBlockText(out, b)
			}
			return out
		}
	}
	return ""
}
