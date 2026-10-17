package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestContextOccupancyFollowBaseline(t *testing.T) {
	s := New()
	s.selectedID = "s"
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[],"projections":{"asOfSeq":5,"values":{"contextPressure":{"pressureTokens":10,"projectedTokens":250,"contextWindow":1000},"contextBreakdown":{"systemTokens":20,"toolsTokens":30,"messageTokens":50}}}}`))
	raw, _ := json.Marshal(s.Snapshot())
	if !strings.Contains(string(raw), `"UsedTokens":250`) || !strings.Contains(string(raw), `"Percent":25`) {
		t.Fatal("follow context projection missing from snapshot")
	}
}

func TestContextOccupancyLiveIsolationAndUnknowns(t *testing.T) {
	s := New()
	s.selectedID = "s"
	apply := func(key, value string, seq int) {
		raw := json.RawMessage(fmt.Sprintf(`{"type":"projection","sessionId":"s","key":%q,"seq":%d,"value":%s}`, key, seq, value))
		s.applyContextControlLocked(raw)
	}
	apply("contextPressure", `{"pressureTokens":20,"contextWindow":100}`, 10)
	apply("contextBreakdown", `{"systemTokens":200,"toolsTokens":300,"messageTokens":500}`, 4)
	if v := s.contextOccupancyLocked(); v == nil || v.Percent != 20 || v.Breakdown.SystemTokens != 200 {
		t.Fatal("breakdown replaced total or separate seq rejected")
	}
	apply("contextPressure", `{"projectedTokens":150,"contextWindow":100}`, 11)
	if v := s.contextOccupancyLocked(); v.Percent != 100 || v.UsedTokens != 150 {
		t.Fatal("occupancy clamping lost numerator")
	}
	apply("contextPressure", `{"pressureTokens":1,"contextWindow":100}`, 9)
	if s.contextOccupancyLocked().Percent != 100 {
		t.Fatal("stale projection won")
	}
	s.applyContextControlLocked(json.RawMessage(`{"type":"projection","sessionId":"other","key":"contextPressure","seq":50,"value":null}`))
	if s.contextOccupancyLocked() == nil {
		t.Fatal("cross-session event cleared reading")
	}
	for _, invalid := range []string{`{}`, `null`, `{"pressureTokens":5,"contextWindow":0}`, `{"pressureTokens":-1,"contextWindow":100}`} {
		apply("contextPressure", invalid, 12)
		if s.contextOccupancyLocked() != nil {
			t.Fatal("unknown capacity became a reading")
		}
	}
	apply("contextPressure", `{"projectedTokens":0,"pressureTokens":50,"contextWindow":100}`, 13)
	if s.contextOccupancyLocked().Percent != 0 {
		t.Fatal("zero projected tokens fell back")
	}
	s.followGeneration++
	if s.contextOccupancyLocked() != nil {
		t.Fatal("reading crossed follow generation")
	}
	s.selectedID = "other"
	if s.contextOccupancyLocked() != nil {
		t.Fatal("reading crossed session")
	}
}
