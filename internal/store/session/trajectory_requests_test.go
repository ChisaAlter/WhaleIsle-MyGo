package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTrajectoryRequestsLifecycleRetryUsageAndMetadata(t *testing.T) {
	s := New()
	for _, raw := range []string{
		`{"type":"step/start","seq":1,"time":100,"data":{"turn":1,"step":1}}`,
		`{"type":"request/header","seq":2,"data":{"header":{"config":{"provider":"historical","model":"m"},"tools":[]}}}`,
		`{"type":"assistant/attempt","seq":3,"time":120,"data":{"turn":1,"step":1,"stream":[{"type":"text-delta","text":"tail evidence"}]}}`,
		`{"type":"llm/retry","seq":4,"time":130,"data":{"turn":1,"step":1,"retryId":"r","retry":1,"maxRetries":3,"delayMs":50,"failure":{"message":"transient"}}}`,
		`{"type":"llm/retry-started","seq":5,"time":180,"data":{"turn":1,"step":1,"retryId":"wrong","retry":1}}`,
		`{"type":"llm/retry-started","seq":6,"time":181,"data":{"turn":1,"step":1,"retryId":"r","retry":1}}`,
		`{"type":"assistant/message","seq":7,"time":220,"data":{"turn":1,"step":1,"usage":{"inputTokens":10,"outputTokens":20},"message":{"source":{"provider":"actual","model":"actual-model"}}}}`,
		`{"type":"step/end","seq":8,"time":240,"data":{"turn":1,"step":1}}`,
	} {
		s.captureTrajectoryBoundaryLocked([]byte(raw))
	}
	r := s.trajectoryRequestsLocked()
	if len(r) != 1 {
		t.Fatalf("requests %+v", r)
	}
	if r[0].Status != "complete" || *r[0].StartedAt != 100 || *r[0].CompletedAt != 220 || *r[0].ResultSeq != 7 || r[0].ProviderMetadata.Provider != "actual" || r[0].Retries[0].State != "started" {
		t.Fatalf("request %+v", r[0])
	}
	if len(r[0].Attempts) != 1 || !strings.Contains(string(r[0].Attempts[0].Raw), "tail evidence") || !strings.Contains(string(r[0].Usage), "20") || r[0].Header.Seq != 2 {
		t.Fatal("real request evidence lost")
	}
	if len(r[0].Events) != 7 {
		t.Fatalf("events=%d", len(r[0].Events))
	}
	r[0].Events[0][0] = '!'
	if s.trajectoryRequestsLocked()[0].Events[0][0] != '{' {
		t.Fatal("request copy aliases journal")
	}
}
func TestTrajectoryRequestsPartialInterruptedAndCompaction(t *testing.T) {
	s := New()
	for _, raw := range []string{
		`{"type":"assistant/message","seq":1,"time":100,"data":{"turn":9,"step":1}}`,
		`{"type":"step/start","seq":2,"data":{"turn":1,"step":1}}`,
		`{"type":"step/end","seq":3,"time":150,"data":{"turn":1,"step":1}}`,
		`{"type":"compaction/start","seq":4,"time":200,"data":{"compactionId":"c","turn":null}}`,
		`{"type":"compaction/summary","seq":5,"time":220,"data":{"compactionId":"c","provider":"p","model":"m","summary":"sum","rawOutput":"complete raw","usage":{"inputTokens":12}}}`,
		`{"type":"compaction/end","seq":6,"time":250,"data":{"compactionId":"c","error":"failed"}}`,
		`{"type":"compaction/start","seq":7,"time":300,"data":{"compactionId":"waiting"}}`,
		`{"type":"session/end","seq":8,"time":310,"data":{}}`,
	} {
		s.captureTrajectoryBoundaryLocked([]byte(raw))
	}
	s.messages = []Message{{Raw: json.RawMessage(`{"type":"user/message","seq":5.5,"data":{"source":{"kind":"compact-checkpoint","compactionId":"c"}}}`)}}
	requests := s.trajectoryRequestsLocked()
	if len(requests) != 3 || requests[0].StartedAt != nil || requests[0].Status != "error" {
		t.Fatalf("partial/unknown %+v", requests)
	}
	compact := requests[1]
	if compact.Turn != nil || compact.Status != "error" || compact.ProviderMetadata.Provider != "p" || compact.RawOutput == nil || *compact.ReplacementSeq != 5.5 {
		t.Fatalf("compact %+v", compact)
	}
	if requests[2].Status != "error" || *requests[2].CompletedAt != 310 {
		t.Fatal("session end did not close running compact")
	}
	s.invalidateFollowLocked()
	if len(s.trajectoryRequestsLocked()) != 0 {
		t.Fatal("requests survived binding reset")
	}
}
func TestTrajectoryRequestsRealLiveAttemptIdentity(t *testing.T) {
	f := newAssistantFixture()
	s := f.s
	f.durable(t, `{"type":"step/start","seq":1,"time":10,"data":{"turn":2,"step":3}}`)
	f.start(t, "actual-attempt", 2, 3)
	f.chunk(t, "actual-attempt", 0, map[string]any{"type": "usage", "usage": map[string]float64{"inputTokens": 3}})
	f.chunk(t, "actual-attempt", 1, map[string]any{"type": "usage", "usage": map[string]float64{"outputTokens": 2}})
	r := s.trajectoryRequestsLocked()
	if len(r) != 1 || r[0].LiveAttempt == nil || r[0].LiveAttempt.AttemptID != "actual-attempt" || r[0].LiveAttempt.Usage["inputTokens"] != 3 || r[0].LiveAttempt.Usage["outputTokens"] != 2 || len(r[0].LiveAttempt.Stream) != 2 {
		t.Fatal("live identity lost")
	}
	s.assistantStream.active.Step = 4
	if s.trajectoryRequestsLocked()[0].LiveAttempt != nil {
		t.Fatal("cross-step live attempt borrowed")
	}
}
