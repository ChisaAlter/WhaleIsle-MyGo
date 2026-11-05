package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParitySystemPromptSourceCoordinates(t *testing.T) {
	for _, tc := range []struct {
		name       string
		events     []string
		turn, step *float64
	}{
		{name: "explicit zero", events: []string{`{"type":"system/message","seq":3,"surfaceOp":"append","data":{"turn":0,"step":0,"message":{"content":[{"type":"text","text":"safe"}]}}}`}, turn: trajectoryTestFloat(0), step: trajectoryTestFloat(0)},
		{name: "known lifecycle", events: []string{`{"type":"turn/start","seq":1,"data":{"turn":7}}`, `{"type":"step/start","seq":2,"data":{"turn":7,"step":3}}`, `{"type":"system/message","seq":3,"surfaceOp":"append","data":{"message":{"content":[{"type":"text","text":"safe"}]}}}`}, turn: trajectoryTestFloat(7), step: trajectoryTestFloat(3)},
		{name: "unknown retained nil", events: []string{`{"type":"system/message","seq":3,"surfaceOp":"append","data":{"message":{"content":[{"type":"text","text":"safe"}]}}}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			s.trajectoryBoundaries = map[float64]json.RawMessage{}
			for i, raw := range tc.events {
				s.trajectoryBoundaries[float64(i+1)] = json.RawMessage(raw)
			}
			rows := s.computeSystemPromptRowsLocked()
			if len(rows) != 1 {
				t.Fatalf("rows %+v", rows)
			}
			r := rows[0]
			same := func(a, b *float64) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
			if !same(r.ConversationTurn, tc.turn) || !same(r.ConversationStep, tc.step) || len(r.Raw) != 0 || strings.Contains(r.Body, "header") {
				t.Fatalf("coordinate/sanitization contract %+v", r)
			}
		})
	}
}
func trajectoryTestFloat(v float64) *float64 { return &v }
