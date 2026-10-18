package session

import (
	"encoding/json"
	"testing"
)

func TestChatStatusRestoration(t *testing.T) {
	s := New()
	s.Select("retry-session")
	// Establish the same authoritative baseline as session/follow before live events.
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[]}`))
	apply := func(raw string) { s.applyPersistenceEvent(json.RawMessage(raw)) }
	apply(`{"type":"turn/start","seq":1,"data":{"turn":1}}`)
	apply(`{"type":"llm/retry","seq":2,"time":1000,"data":{"turn":1,"step":1,"retryId":"a","retry":1,"mode":"normal","maxRetries":3,"delayMs":5000,"failure":{"code":"RATE_LIMIT","message":"Slow down"}}}`)
	messages := s.Snapshot().Messages
	if len(messages) != 1 || messages[0].Retry == nil || messages[0].Retry.Attempts[0].State != "scheduled" {
		t.Fatalf("scheduled: %+v", messages)
	}
	apply(`{"type":"llm/retry-started","seq":3,"data":{"turn":1,"step":1,"retryId":"other","retry":1}}`)
	if s.Snapshot().Messages[0].Retry.Attempts[0].State != "scheduled" {
		t.Fatal("foreign retry advanced wait")
	}
	apply(`{"type":"llm/retry-started","seq":4,"data":{"turn":1,"step":1,"retryId":"a","retry":1}}`)
	apply(`{"type":"llm/retry","seq":5,"data":{"turn":1,"step":1,"retryId":"a","retry":2,"mode":"normal","maxRetries":3,"delayMs":2000}}`)
	apply(`{"type":"step/end","seq":6,"data":{"turn":1,"step":2}}`)
	attempts := s.Snapshot().Messages[0].Retry.Attempts
	if len(attempts) != 2 || attempts[0].State != "started" || attempts[1].State != "scheduled" {
		t.Fatalf("chain: %+v", attempts)
	}
	apply(`{"type":"turn/end","seq":7,"data":{"turn":1,"reason":{"kind":"error","error":{"message":"Final failure","code":"UPSTREAM"}}}}`)
	messages = s.Snapshot().Messages
	if len(messages) != 2 || messages[0].Retry.Attempts[1].State != "cancelled" || messages[1].EventType != "turn/end" {
		t.Fatalf("terminal: %+v", messages)
	}
	// Page hydration and repeat projection use durable records, without replaying waits.
	history := New()
	for _, raw := range s.trajectoryBoundaries {
		history.captureTrajectoryBoundaryLocked(raw)
	}
	replay := history.projectRetryMessagesLocked(nil)
	if len(replay) != 1 || replay[0].Retry.Attempts[1].State != "cancelled" {
		t.Fatalf("history: %+v", replay)
	}
	if messages[0].Retry.Attempts[0].State != "started" {
		t.Fatal("snapshot mutation")
	}
}
