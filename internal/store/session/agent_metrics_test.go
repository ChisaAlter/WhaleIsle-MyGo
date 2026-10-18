package session

import (
	"encoding/json"
	"testing"
)

func TestAgentMetricsProjection(t *testing.T) {
	m := decodeAgentMetrics(json.RawMessage(`{"tokenUsage":{"uncachedInputTokens":10,"outputTokens":20,"cacheReadTokens":30,"cacheWriteTokens":40},"subagentTiming":{"settledMs":5000,"active":{"since":1000,"through":3000},"lastTurnCompleted":true}}`), 4)
	if m.Tokens == nil || *m.Tokens != 100 {
		t.Fatalf("missing disjoint total: %+v", m)
	}
	if got, ok := m.Duration(true, 6000); !ok || got != 10000 {
		t.Fatalf("running: %v %v", got, ok)
	}
	if got, ok := m.Duration(false, 6000); !ok || got != 7000 {
		t.Fatalf("inactive: %v %v", got, ok)
	}
	if !m.Completed {
		t.Fatal("completion lost")
	}
	unknown := decodeAgentMetrics(json.RawMessage(`{}`), 1)
	if unknown.Tokens != nil {
		t.Fatal("unknown fabricated zero")
	}
	if _, ok := unknown.Duration(false, 1); ok {
		t.Fatal("unknown duration fabricated")
	}
	bad := decodeAgentMetrics(json.RawMessage(`{"tokenUsage":{"outputTokens":4},"subagentTiming":{"settledMs":-1}}`), 1)
	if bad.Tokens != nil || bad.Timing != nil {
		t.Fatal("partial/invalid metrics accepted")
	}
}

func TestAgentMetricsControlAndCatalog(t *testing.T) {
	s := New()
	s.items = []Item{{SessionID: "child", Running: false}}
	s.applyAgentMetricsControlLocked(json.RawMessage(`{"type":"projection","sessionId":"child","key":"subagentTiming","seq":8,"value":{"settledMs":2500,"lastTurnCompleted":true}}`))
	s.applyAgentMetricsControlLocked(json.RawMessage(`{"type":"projection","sessionId":"child","key":"subagentTiming","seq":7,"value":{"settledMs":1}}`))
	if !s.items[0].AgentMetrics.Completed || s.items[0].AgentMetrics.Timing.SettledMs != 2500 {
		t.Fatal("stale projection replaced metrics")
	}
	rows, err := agentNavigationRows("root", json.RawMessage(`{"subagentCatalog":[{"id":"child","mode":"continuable"},{"id":"missing","mode":"one-shot"},{"id":"outside","mode":"external"}]}`), s.items)
	if err != nil || !rows[0].ActivityKnown || !rows[0].Metrics.Completed || rows[1].ActivityKnown {
		t.Fatal("unknown or completion corrupted", rows, err)
	}
	v := AgentNavigationView{Rows: rows, catalog: agentNavigationCatalog{ready: true}}
	s.agentRootCountsLocked(&v)
	if !v.RootCountsKnown || v.RootChildren != 3 || v.RootRunning != 0 || v.RootActivityUnknown != 1 {
		t.Fatal("root count is not catalog facts", v)
	}
	s.applyAgentMetricsControlLocked(json.RawMessage(`{"type":"baseline","value":{"projections":{"child":{"asOfSeq":9,"values":{}}}}}`))
	if s.items[0].AgentMetrics.Timing != nil || s.items[0].AgentMetrics.Completed {
		t.Fatal("missing key in newer whole baseline was not cleared")
	}
}

func TestAgentMetricsAvailabilityMerge(t *testing.T) {
	old := decodeAgentMetrics(json.RawMessage(`{"subagentTiming":{"settledMs":5000,"lastTurnCompleted":true}}`), 8)
	next := decodeAgentMetrics(json.RawMessage(`{}`), 7)
	if got := mergeAgentMetrics(next, old); !got.Completed || got.TimingSeq != 8 {
		t.Fatal("older availability hint replaced timing")
	}
	if got := mergeAgentMetrics(AgentMetrics{}, old); got.Timing == nil {
		t.Fatal("omitted availability metrics cleared known facts")
	}
	if got := mergeAgentMetrics(decodeAgentMetrics(json.RawMessage(`{}`), 9), old); got.Timing != nil {
		t.Fatal("new authoritative absence did not clear")
	}
}
