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
	if s.historyError != "" && s.lastAction == s.historyError {
		s.lastAction = ""
	}
	s.historyError = ""
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
	s.lastAction = prefix + err.Error()
	s.historyLoading = false
	s.historyError = s.lastAction
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
}

func (s *Store) runFollow(ctx context.Context, mux *dsh.Mux, sessionID string, generation uint64) {
	for s.runFollowConnection(ctx, mux, sessionID, generation) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (s *Store) runFollowConnection(ctx context.Context, mux *dsh.Mux, sessionID string, generation uint64) bool {
	s.mu.Lock()
	current := s.followCurrentLocked(ctx, sessionID, generation)
	s.mu.Unlock()
	if !current {
		return false
	}
	if err := mux.Dial(ctx); err != nil {
		s.reportFollowError(ctx, sessionID, generation, "follow dial: ", err)
		return false
	}
	payload := map[string]any{"args": map[string]any{"request": map[string]any{
		"address": sessionAddressJSON(sessionID), "assistantStream": true,
		"maxMessages": historyPageMaxMessages,
		"turnWindow":  map[string]any{"minMessages": 50, "minTurns": 2},
	}}}
	stream, err := mux.Open(ctx, "session/follow", payload)
	if err != nil {
		s.reportFollowError(ctx, sessionID, generation, "follow open: ", err)
		return false
	}
	defer stream.Cancel()
	s.mu.Lock()
	if !s.followCurrentLocked(ctx, sessionID, generation) {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()
	opened, cursorKnown := false, false
	cursor := float64(-1)
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			// The original journal treats an accepted follow's unexplained end
			// as carrier loss. Business error frames remain terminal; the main
			// mux owner separately restores physical socket loss.
			if errors.Is(err, dsh.ErrStreamEnd) && opened {
				s.mu.Lock()
				current := s.followCurrentLocked(ctx, sessionID, generation)
				s.mu.Unlock()
				return current
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
			return false
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
				return true
			}
		}
		if !s.applyFollowFrameForTarget(ctx, sessionID, generation, raw) {
			return false
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
			return true
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
		s.historyReady = true
		s.clearHistoryErrorLocked()
		if snap.Projections != nil {
			s.applySessionPresentationValuesLocked(s.selectedID, "sequenced", snap.Projections.AsOfSeq, snap.Projections.Values)
			s.applySessionSubagentIdentityValuesLocked(s.selectedID, "sequenced", snap.Projections.AsOfSeq, snap.Projections.Values)
			s.applyCostBaselineLocked(s.selectedID, snap.Projections.AsOfSeq, snap.Projections.Values)
		}
		s.resetInspectorLocked()
		s.planHistoryBinding = fmt.Sprintf("%p\x00%s", s.client, s.selectedID)
		s.turnFilesHistoryBinding = turnFileHistoryBinding(s.client, s.selectedID)
		s.messages = nil
		s.workflowBoundaries = nil
		s.trajectoryBoundaries = nil
		s.systemPromptAnchors = nil
		s.systemPromptRowsCache = nil
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
		if snap.HasMore != nil {
			s.historyHasMore = *snap.HasMore
		} else {
			s.syncHistoryHasMoreAfterSnapshotLocked()
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
