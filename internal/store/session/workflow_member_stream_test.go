package session

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// One focused fixture covers the new adapter; old stream suites are not rerun.
func TestWorkflowMemberAssistantStreamAdapter(t *testing.T) {
	address := map[string]any{"kind": "subagent", "parentSessionId": "parent", "childSessionId": "child", "mode": "one-shot"}
	wire, _ := json.Marshal(workflowMemberFollowPayload(address))
	var payload struct {
		Args struct {
			Request struct {
				AssistantStream bool           `json:"assistantStream"`
				Address         map[string]any `json:"address"`
			} `json:"request"`
		} `json:"args"`
	}
	_ = json.Unmarshal(wire, &payload)
	if !payload.Args.Request.AssistantStream || payload.Args.Request.Address["mode"] != "one-shot" || payload.Args.Request.Address["parentSessionId"] != "parent" {
		t.Fatal("changed address or omitted assistantStream")
	}
	p := newWorkflowMemberStream()
	apply := func(raw string) {
		t.Helper()
		restart, err := p.apply(json.RawMessage(raw))
		if err != nil || restart {
			t.Fatalf("unexpected rebaseline: %v %v for %s", restart, err, raw)
		}
	}
	apply(`{"type":"snapshot","cursor":20,"hasMore":true,"records":[],"assistantStream":{"revision":10,"activeAttempt":{"attemptId":"a","turn":2,"step":0,"startedAfterSeq":20,"nextIndex":2,"stream":[{"type":"reasoning-chunks","time0":1,"index":0,"dt":[],"texts":["thinking"]},{"type":"text-chunks","time0":2,"index":1,"dt":[],"texts":["pre"]}]}}}`)
	if rows := p.messages(); len(rows) != 2 || rows[0].Kind != MessageKindThinking || rows[1].Body != "pre" || !rows[1].Streaming || len(p.process.Records) != 0 || p.process.Cursor != 20 {
		t.Fatalf("opening baseline: %+v", rows)
	}
	apply(`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a","revision":11,"index":2,"time":3,"chunk":{"type":"text-delta","index":1,"text":"view"}}}`)
	apply(`{"type":"event","event":{"type":"assistant/message","seq":21,"time":4,"surfaceOp":"append","data":{"turn":2,"step":0,"message":{"id":"m21","content":[{"type":"text","text":"same"}]}}}}`)
	if rows := p.messages(); len(rows) != 2 || rows[1].Body != "preview" || len(p.process.Records) != 1 {
		t.Fatal("pending settlement duplicated/replaced prefix early")
	}
	apply(`{"type":"assistant-stream","frame":{"type":"end","attemptId":"a","revision":12,"index":3,"outcome":{"kind":"committed","eventType":"assistant/message","seq":21}}}`)
	if rows := p.messages(); len(rows) != 1 || rows[0].Body != "same" || rows[0].Streaming || rows[0].ID != "m21" {
		t.Fatalf("canonical settlement: %+v", rows)
	}
	apply(`{"type":"event","event":{"type":"step/end","seq":22,"data":{"turn":2,"step":0}}}`)
	apply(`{"type":"assistant-stream","frame":{"type":"start","attemptId":"b","revision":13,"startedAfterSeq":22,"turn":2,"step":1}}`)
	apply(`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"b","revision":14,"index":0,"time":5,"chunk":{"type":"text-delta","index":0,"text":"same"}}}`)
	if rows := p.messages(); len(rows) != 2 || rows[0].Body != rows[1].Body || rows[0].ID == rows[1].ID {
		t.Fatal("text equality deduplicated different attempts")
	}
	durableBefore, _ := json.Marshal(p.process)
	apply(`{"type":"assistant-stream","frame":{"type":"end","attemptId":"b","revision":15,"index":1,"outcome":{"kind":"abandoned"}}}`)
	durableAfter, _ := json.Marshal(p.process)
	if string(durableBefore) != string(durableAfter) || len(p.messages()) != 1 {
		t.Fatal("abandoned preview leaked into durable history")
	}
	restart, err := p.apply([]byte(`{"type":"assistant-stream","frame":{"type":"start","attemptId":"bad","revision":17,"turn":2,"step":2}}`))
	if err != nil || !restart {
		t.Fatal("revision gap did not request baseline")
	}
	apply(`{"type":"snapshot","cursor":30,"hasMore":false,"records":[],"assistantStream":{"revision":40,"activeAttempt":{"attemptId":"new","turn":3,"step":0,"startedAfterSeq":30,"nextIndex":1,"stream":[{"type":"text-chunks","time0":6,"index":0,"dt":[],"texts":["fresh"]}]}}}`)
	apply(`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"old","revision":41,"index":99,"time":7,"chunk":{"type":"text-delta","index":0,"text":"stale"}}}`)
	if rows := p.messages(); len(rows) != 1 || rows[0].Body != "fresh" || p.process.Cursor != 30 {
		t.Fatal("old attempt affected new baseline")
	}
	view := WorkflowMemberView{Messages: p.messages(), PresentationReady: true, Process: p.process}
	display := WorkflowMemberDisplayText(view)
	if !strings.Contains(display, "live preview") || !strings.Contains(display, "fresh") {
		t.Fatal("preview invisible or unlabelled")
	}
	if strings.Contains(WorkflowProcessText(view), "fresh") {
		t.Fatal("preview exported as durable")
	}
	for _, invalidate := range []string{"close", "parent", "host"} {
		owner := New()
		owner.selectedID = "parent"
		ctx, cancel := context.WithCancel(context.Background())
		owner.workflowMember = workflowMemberState{view: WorkflowMemberView{Token: 8, Open: true, ParentID: "parent"}, cancel: cancel, generation: owner.followGeneration}
		if !owner.workflowMemberCurrentLocked(ctx, 8) {
			t.Fatal("fixture not current")
		}
		switch invalidate {
		case "close":
			owner.CloseWorkflowMember(8)
		case "parent":
			owner.selectSessionLocked("other")
		case "host":
			owner.invalidateFollowLocked()
		}
		if ctx.Err() == nil || owner.workflowMemberCurrentLocked(ctx, 8) {
			t.Fatalf("%s did not cancel/fence observer", invalidate)
		}
	}
}
