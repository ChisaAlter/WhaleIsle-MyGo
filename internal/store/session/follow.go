package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"whaleisle/internal/dsh"
)

// AttachMux binds the shared remote.mux used for session/follow.
func (s *Store) AttachMux(mux *dsh.Mux) {
	s.mu.Lock()
	s.mux = mux
	s.mu.Unlock()
}

// FollowSelected opens session/follow for the current selection until ctx ends
// or the selection changes (call again after Select).
func (s *Store) FollowSelected(ctx context.Context) {
	s.mu.Lock()
	mux := s.mux
	sessionID := s.selectedID
	if mux == nil || sessionID == "" {
		s.mu.Unlock()
		return
	}
	if s.followCancel != nil {
		s.followCancel()
		s.followCancel = nil
	}
	fctx, cancel := context.WithCancel(ctx)
	s.followCancel = cancel
	s.mu.Unlock()

	go s.runFollow(fctx, mux, sessionID)
}

func (s *Store) runFollow(ctx context.Context, mux *dsh.Mux, sessionID string) {
	if err := mux.Dial(ctx); err != nil {
		s.mu.Lock()
		s.lastAction = "follow dial: " + err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	streamTrue := true
	addr, _ := json.Marshal(map[string]any{
		"kind":      "session",
		"sessionId": sessionID,
	})
	payload := map[string]any{
		"args": map[string]any{
			"request": map[string]any{
				"address":         json.RawMessage(addr),
				"assistantStream": streamTrue,
			},
		},
	}
	stream, err := mux.Open(ctx, "session/follow", payload)
	if err != nil {
		if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
			return
		}
		s.mu.Lock()
		s.lastAction = "follow open: " + err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	defer stream.Cancel()

	s.mu.Lock()
	s.messages = s.messages[:0]
	s.resetPlanBoardLocked()
	s.mu.Unlock()
	s.fire()

	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, dsh.ErrStreamEnd) || errors.Is(err, context.Canceled) {
				return
			}
			s.mu.Lock()
			s.lastAction = "follow: " + err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.applyFollowFrame(raw)
	}
}

func (s *Store) applyFollowFrame(raw json.RawMessage) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return
	}
	switch head.Type {
	case "snapshot":
		s.applySnapshot(raw)
	case "event":
		s.applyEventFrame(raw)
	case "assistant-stream":
		s.applyAssistantStream(raw)
	}
}

func (s *Store) applySnapshot(raw json.RawMessage) {
	var snap struct {
		Records []struct {
			Type  string          `json:"type"`
			Event json.RawMessage `json:"event"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		return
	}
	s.mu.Lock()
	s.messages = s.messages[:0]
	s.turnStartMs = 0
	s.resetPlanBoardLocked()
	s.resetHistoryPagingLocked()
	s.mu.Unlock()
	for _, rec := range snap.Records {
		if rec.Type == "event" {
			s.applyPersistenceEvent(rec.Event)
		}
	}
	s.mu.Lock()
	s.syncHistoryHasMoreAfterSnapshotLocked()
	s.mu.Unlock()
	s.fire()
}

func (s *Store) applyEventFrame(raw json.RawMessage) {
	var wrap struct {
		Event json.RawMessage `json:"event"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return
	}
	s.applyPersistenceEvent(wrap.Event)
	s.fire()
}

func (s *Store) applyPersistenceEvent(raw json.RawMessage) {
	var seqHead struct {
		Seq float64 `json:"seq"`
	}
	if json.Unmarshal(raw, &seqHead) == nil {
		s.mu.Lock()
		s.bumpHistoryThroughSeq(seqHead.Seq)
		s.mu.Unlock()
	}
	s.applyPlanBoardEvent(raw)
	if s.applyAgentPresetSelectedEvent(raw) {
		return
	}
	if s.applyTurnTimingEvent(raw) {
		return
	}
	msg, ok := classifyPersistenceEvent(raw)
	if !ok {
		return
	}
	if msg.ID == "" {
		msg.ID = newRequestID()
	}
	s.mu.Lock()
	s.messages = append(s.messages, msg)
	s.mu.Unlock()
}

// applyTurnTimingEvent records turn/start and emits Electron-parity
// 「已完成，用时 …」 status rows on turn/end.
func (s *Store) applyTurnTimingEvent(raw json.RawMessage) bool {
	var head struct {
		Type string  `json:"type"`
		Time float64 `json:"time"`
		Seq  float64 `json:"seq"`
	}
	if json.Unmarshal(raw, &head) != nil {
		return false
	}
	switch head.Type {
	case "turn/start":
		s.mu.Lock()
		s.turnStartMs = head.Time
		s.mu.Unlock()
		return true
	case "turn/end":
		s.mu.Lock()
		start := s.turnStartMs
		s.turnStartMs = 0
		s.mu.Unlock()
		elapsedMs := 0.0
		if start > 0 && head.Time >= start {
			elapsedMs = head.Time - start
		}
		s.mu.Lock()
		s.messages = append(s.messages, Message{
			ID:        newRequestID(),
			Kind:      MessageKindText,
			Role:      "status",
			Seq:       head.Seq,
			EventType: "turn/end",
			Body:      fmt.Sprintf("%.0f", elapsedMs),
			EventTime: head.Time,
		})
		s.mu.Unlock()
		return true
	default:
		return false
	}
}

func (s *Store) applyAssistantStream(raw json.RawMessage) {
	var wrap struct {
		Frame struct {
			Type  string          `json:"type"`
			Chunk json.RawMessage `json:"chunk"`
		} `json:"frame"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return
	}
	switch wrap.Frame.Type {
	case "start":
		s.mu.Lock()
		s.messages = append(s.messages, Message{
			ID:        newRequestID(),
			Kind:      MessageKindText,
			Role:      "assistant",
			Streaming: true,
		})
		s.mu.Unlock()
		s.fire()
	case "chunk":
		text := extractChunkText(wrap.Frame.Chunk)
		if text == "" {
			return
		}
		s.AppendStreamDelta("assistant", text)
		s.fire()
	case "end":
		s.mu.Lock()
		if n := len(s.messages); n > 0 && s.messages[n-1].Streaming {
			s.messages[n-1].Streaming = false
		}
		s.mu.Unlock()
		s.fire()
	}
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
