package session

import "testing"

// The prompt event cache serves the pane read and the system-prompt rows from
// one decode per boundary. It is keyed by the bytes it decoded, so a boundary
// that changes under the same seq must not answer with the earlier event.
func TestPromptEventCacheFollowsTheBytesItDecoded(t *testing.T) {
	s := New()
	s.captureTrajectoryBoundaryLocked([]byte(`{"type":"turn/start","seq":4,"data":{"turn":1}}`))
	first, ok := s.promptEventLocked(4, s.trajectoryBoundaries[4])
	if !ok || first.Type != "turn/start" {
		t.Fatalf("first decode %q ok=%v", first.Type, ok)
	}
	s.trajectoryBoundaries[4] = []byte(`{"type":"step/end","seq":4,"data":{"turn":1}}`)
	second, ok := s.promptEventLocked(4, s.trajectoryBoundaries[4])
	if !ok || second.Type != "step/end" {
		t.Fatalf("second decode %q ok=%v, wanted step/end: the cache answered for bytes it did not decode", second.Type, ok)
	}
	third, _ := s.promptEventLocked(4, []byte(`{"type":"turn/end","seq":4,"data":{"turn":1}}`))
	if third.Type != "turn/end" {
		t.Fatalf("third decode %q, wanted turn/end", third.Type)
	}
}
