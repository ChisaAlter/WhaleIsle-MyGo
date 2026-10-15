package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
	generation := s.followGeneration
	fctx, cancel := context.WithCancel(ctx)
	s.followCancel = cancel
	s.mu.Unlock()
	go s.runFollow(fctx, mux, sessionID, generation)
}

func (s *Store) followCurrentLocked(ctx context.Context, sessionID string, generation uint64) bool {
	return ctx.Err() == nil && s.selectedID == sessionID && s.followGeneration == generation
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
	s.mu.Unlock()
	s.fire()
}

func (s *Store) runFollow(ctx context.Context, mux *dsh.Mux, sessionID string, generation uint64) {
	if err := mux.Dial(ctx); err != nil {
		s.reportFollowError(ctx, sessionID, generation, "follow dial: ", err)
		return
	}
	payload := map[string]any{"args": map[string]any{"request": map[string]any{
		"address": sessionAddressJSON(sessionID), "assistantStream": true,
	}}}
	stream, err := mux.Open(ctx, "session/follow", payload)
	if err != nil {
		s.reportFollowError(ctx, sessionID, generation, "follow open: ", err)
		return
	}
	defer stream.Cancel()
	s.mu.Lock()
	if !s.followCurrentLocked(ctx, sessionID, generation) {
		s.mu.Unlock()
		return
	}
	s.messages = nil
	s.turnStartMs = 0
	s.editTurnOpening = false
	s.editTurnSeq = 0
	s.resetPlanBoardLocked()
	s.resetHistoryPagingLocked()
	s.historyThroughSeq = 0
	s.mu.Unlock()
	s.fire()
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if !errors.Is(err, dsh.ErrStreamEnd) {
				s.reportFollowError(ctx, sessionID, generation, "follow: ", err)
			}
			return
		}
		if !s.applyFollowFrameForTarget(ctx, sessionID, generation, raw) {
			return
		}
	}
}

// The generation check and the entire frame fold share one critical section.
// A selection cannot change between validation and any message/plan/history write.
func (s *Store) applyFollowFrameForTarget(ctx context.Context, sessionID string, generation uint64, raw json.RawMessage) bool {
	s.mu.Lock()
	if !s.followCurrentLocked(ctx, sessionID, generation) {
		s.mu.Unlock()
		return false
	}
	changed := s.applyFollowFrameLocked(raw)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
	return true
}

func (s *Store) applyFollowFrame(raw json.RawMessage) {
	s.mu.Lock()
	changed := s.applyFollowFrameLocked(raw)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}

func (s *Store) applyFollowFrameLocked(raw json.RawMessage) bool {
	var head struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &head) != nil {
		return false
	}
	switch head.Type {
	case "snapshot":
		var snap struct {
			Projections *struct {
				AsOfSeq float64
				Values  json.RawMessage
			} `json:"projections"`
			Cursor  *float64 `json:"cursor"`
			HasMore *bool    `json:"hasMore"`
			Records []struct {
				Type  string          `json:"type"`
				Event json.RawMessage `json:"event"`
			} `json:"records"`
		}
		if json.Unmarshal(raw, &snap) != nil || snap.Records == nil {
			return false
		}
		if snap.Projections != nil {
			s.applyCostBaselineLocked(s.selectedID, snap.Projections.AsOfSeq, snap.Projections.Values)
		}
		s.messages = nil
		s.turnStartMs = 0
		s.editTurnOpening = false
		s.editTurnSeq = 0
		s.resetPlanBoardLocked()
		s.resetHistoryPagingLocked()
		s.historyThroughSeq = 0
		for _, rec := range snap.Records {
			if rec.Type == "event" {
				s.applyPersistenceEventLocked(rec.Event)
			}
		}
		if snap.Projections != nil {
			s.applyQueueValuesLocked(snap.Projections.Values, snap.Projections.AsOfSeq)
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
		var wrap struct {
			Event json.RawMessage `json:"event"`
		}
		if json.Unmarshal(raw, &wrap) != nil {
			return false
		}
		return s.applyPersistenceEventLocked(wrap.Event)
	case "assistant-stream":
		return s.applyAssistantStreamLocked(raw)
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

func (s *Store) applyPersistenceEventLocked(raw json.RawMessage) bool {
	var head struct {
		Type string  `json:"type"`
		Seq  float64 `json:"seq"`
		Time float64 `json:"time"`
	}
	if json.Unmarshal(raw, &head) != nil || head.Type == "" {
		return false
	}
	s.bumpHistoryThroughSeq(head.Seq)
	changed := ApplyPlanBoardEvent(&s.planBoard, raw)
	if head.Type == events.TypeAgentPresetSelected {
		var ev events.AgentPresetSelected
		if json.Unmarshal(raw, &ev) == nil && ev.Data.AgentPreset != "" {
			s.selectedAgentPresetLabel = labelForAgentPresetID(ev.Data.AgentPreset, s.agentPresetOptions)
			s.invalidateCommandsLocked()
		}
		return true
	}
	switch head.Type {
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
	msg, ok := classifyPersistenceEvent(raw)
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

func (s *Store) applyAssistantStreamLocked(raw json.RawMessage) bool {
	var wrap struct {
		Frame struct {
			Type  string          `json:"type"`
			Chunk json.RawMessage `json:"chunk"`
		} `json:"frame"`
	}
	if json.Unmarshal(raw, &wrap) != nil {
		return false
	}
	switch wrap.Frame.Type {
	case "start":
		s.messages = append(s.messages, Message{ID: newRequestID(), Kind: MessageKindText, Role: "assistant", TurnSeq: s.editTurnSeq, Streaming: true})
	case "chunk":
		text := extractChunkText(wrap.Frame.Chunk)
		if text == "" {
			return false
		}
		s.appendStreamDeltaLocked("assistant", text)
	case "end":
		if n := len(s.messages); n > 0 && s.messages[n-1].Streaming {
			s.messages[n-1].Streaming = false
		}
	default:
		return false
	}
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
	return extractTextish(chunk)
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
