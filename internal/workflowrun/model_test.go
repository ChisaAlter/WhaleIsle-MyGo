package workflowrun

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestWorkflowScopeIdentityAndNavigation(t *testing.T) {
	raw := []json.RawMessage{
		[]byte(`{"type":"turn/start","seq":1,"data":{"turn":0}}`), []byte(`{"type":"step/start","seq":2,"data":{"turn":0,"step":0}}`),
		[]byte(`{"type":"tool-workflow/run-start","seq":3,"data":{"runId":"r","name":"build"}}`),
		[]byte(`{"type":"tool-workflow/agent-start","seq":4,"data":{"runId":"r","seq":0,"label":"same","childId":"a"}}`),
		[]byte(`{"type":"tool-workflow/agent-start","seq":5,"data":{"runId":"r","seq":1,"label":"same","phase":"","childId":"b"}}`),
		[]byte(`{"type":"tool-workflow/agent-end","seq":6,"data":{"runId":"r","seq":0,"outcome":"completed"}}`),
		[]byte(`{"type":"step/end","seq":7,"data":{"turn":0,"step":0}}`),
		[]byte(`{"type":"tool-workflow/agent-end","seq":8,"data":{"runId":"orphan","seq":1,"outcome":"failed"}}`)}
	before, _ := json.Marshal(raw)
	runs := Fold(raw)
	if len(runs) != 1 || runs[0].Status != "interrupted" || len(runs[0].Phases) != 2 || runs[0].Phases[0].Name != nil || runs[0].Phases[1].Name == nil || *runs[0].Phases[1].Name != "" || runs[0].Phases[0].Members[0].Status != "completed" || runs[0].Phases[1].Members[0].Status != "interrupted" {
		t.Fatalf("fold: %+v", runs)
	}
	if len(runs[0].EventSeqs) != 4 {
		t.Fatal("claimed orphan or boundary")
	}
	after, _ := json.Marshal(raw)
	if string(before) != string(after) {
		t.Fatal("changed journal")
	}
	raw = append(raw, []byte(`{"type":"tool-workflow/run-end","seq":9,"data":{"runId":"r","stopReason":"cancelled"}}`))
	runs = Fold(raw)
	if runs[0].Status != "cancelled" || runs[0].Phases[1].Members[0].Status != "running" {
		t.Fatal("invented member settlement")
	}
	if len(Fold(raw[7:])) != 0 {
		t.Fatal("invented partial-history start")
	}
	addr, ok := OneShotAddress("p", "a", []byte(`{"subagentCatalog":[{"id":"a","mode":"one-shot"}]}`))
	if !ok || !reflect.DeepEqual(addr, map[string]any{"kind": "subagent", "parentSessionId": "p", "childSessionId": "a", "mode": "one-shot"}) {
		t.Fatal("wrong original address")
	}
	if _, ok = OneShotAddress("p", "a", []byte(`{"subagentCatalog":[{"id":"a","mode":"continuable"}]}`)); ok {
		t.Fatal("mode confusion")
	}
}
