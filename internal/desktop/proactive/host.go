package proactive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/dsh/events"
)

type Host struct{ Client *dsh.Client }
type RunResult struct{ Phase, State, Summary, Error string }

// Run first reads the durable Session. Only prepared executions may send;
// accepted or uncertain submissions are inspected without resending.
func (h Host) Run(ctx context.Context, task Task, progress func(string) error) (RunResult, error) {
	if h.Client == nil || task.Execution == nil {
		return RunResult{}, errors.New("task Host or execution is unavailable")
	}
	e := task.Execution
	var projections api.SessionProjectionsValue
	err := h.Client.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": e.SessionID}}, &projections)
	missing := err == nil && len(projections.Values) == 0
	if err != nil {
		var remote *dsh.Error
		if !errors.As(err, &remote) || remote.Code != "session/not-found" {
			return RunResult{}, err
		}
		missing = true
	}
	if missing {
		if e.Phase != "prepared" {
			return RunResult{Phase: "interrupted", State: "waiting_input", Error: "The saved execution Session is missing. No prompt was resent."}, nil
		}
		request := api.SessionCreateRequest{SessionId: &e.SessionID}
		if e.Spec.WorkspaceID != "" {
			request.WorkspaceId = &e.Spec.WorkspaceID
		} else {
			request.Cwd = &e.Spec.Cwd
		}
		var created api.SessionCreateValue
		if err := h.Client.Call(ctx, "session/create", map[string]any{"request": request}, &created); err != nil {
			return RunResult{}, err
		}
		if created.SessionId != e.SessionID {
			return RunResult{}, errors.New("Host did not preserve task Session identity")
		}
		var renamed api.SessionRenameValue
		if err := h.Client.Call(ctx, "session/rename", map[string]any{"request": map[string]string{"sessionId": e.SessionID, "title": task.Title}}, &renamed); err != nil {
			return RunResult{}, err
		}
	} else if err := projections.RequireSequenced(); err != nil {
		return RunResult{}, err
	}
	mux := &dsh.Mux{Endpoint: h.Client.Endpoint, HTTP: h.Client.HTTP}
	defer mux.Close()
	if err := mux.Dial(ctx); err != nil {
		return RunResult{}, err
	}
	address, _ := json.Marshal(map[string]string{"kind": "session", "sessionId": e.SessionID})
	maxMessages := float64(500)
	stream, err := mux.Open(ctx, "session/follow", map[string]any{"args": map[string]any{"request": api.SessionFollowRequest{Address: address, MaxMessages: &maxMessages, TurnWindow: &api.SessionFollowRequestTurnWindow{MinMessages: 50, MinTurns: 2}}}})
	if err != nil {
		return RunResult{}, err
	}
	defer stream.Cancel()
	raw, err := stream.Recv(ctx)
	if err != nil {
		return RunResult{}, err
	}
	var snapshot struct {
		Type    string  `json:"type"`
		Cursor  float64 `json:"cursor"`
		HasMore bool    `json:"hasMore"`
		Records []struct {
			Type  string          `json:"type"`
			Event json.RawMessage `json:"event"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil || snapshot.Type != "snapshot" {
		return RunResult{}, fmt.Errorf("task history did not return an opening snapshot")
	}
	records := []json.RawMessage{}
	for _, r := range snapshot.Records {
		if r.Type == "event" {
			records = append(records, r.Event)
		}
	}
	observed := taskObservation{requestID: e.RequestID}
	for _, r := range records {
		observed.apply(r)
	}
	// A long historical Session may place this request before the opening window.
	for snapshot.HasMore && !observed.found {
		if len(records) == 0 {
			return RunResult{}, errors.New("task history pagination has no cursor")
		}
		var first struct {
			Seq float64 `json:"seq"`
		}
		if err := json.Unmarshal(records[0], &first); err != nil {
			return RunResult{}, err
		}
		max := float64(500)
		var page api.SessionPage
		if err := h.Client.Call(ctx, "session/page", map[string]any{"request": api.SessionPageRequest{Address: address, ThroughSeq: snapshot.Cursor, BeforeSeq: &first.Seq, MaxMessages: &max}}, &page); err != nil {
			return RunResult{}, err
		}
		older := []json.RawMessage{}
		for _, r := range page.Records {
			if r.Type == "event" {
				raw, _ := json.Marshal(r.Event)
				older = append(older, raw)
			}
		}
		if len(older) == 0 && page.HasMore {
			return RunResult{}, errors.New("task history pagination did not advance")
		}
		records = append(older, records...)
		snapshot.HasMore = page.HasMore
		observed = taskObservation{requestID: e.RequestID}
		for _, r := range records {
			observed.apply(r)
		}
	}
	if observed.done {
		return observed.result(), nil
	}
	if observed.found || e.Phase != "prepared" {
		var list api.SessionListValue
		if err := h.Client.Call(ctx, "session/list", map[string]any{"request": api.SessionListRequest{}}, &list); err != nil {
			return RunResult{}, err
		}
		running := false
		for _, item := range list.Items {
			if item.SessionId == e.SessionID {
				running = item.Running
			}
		}
		if !running {
			return RunResult{Phase: "interrupted", State: "waiting_input", Summary: strings.Join(observed.summary, "\n\n"), Error: "The Session is idle without a completed result for this request. Continue explicitly to send a new turn."}, nil
		}
	}
	if !observed.found && e.Phase == "prepared" {
		bindingClient := pluginrpc.Client{BaseURL: h.Client.Endpoint.Origin, Cookie: h.Client.Endpoint.Cookie, HTTPClient: h.Client.HTTP}
		var binding struct {
			TaskID    string `json:"taskId"`
			RequestID string `json:"requestId"`
			SessionID string `json:"sessionId"`
		}
		if err := bindingClient.Call(ctx, "native-proactive", "bind", map[string]string{"taskId": task.ID, "requestId": e.RequestID, "sessionId": e.SessionID}, &binding); err != nil {
			return RunResult{}, err
		}
		if binding.TaskID != task.ID || binding.RequestID != e.RequestID || binding.SessionID != e.SessionID {
			return RunResult{}, errors.New("Host did not confirm task tool policy binding")
		}
		if task.Autonomy == 1 {
			client := pluginrpc.Client{BaseURL: h.Client.Endpoint.Origin, Cookie: h.Client.Endpoint.Cookie, HTTPClient: h.Client.HTTP}
			var result struct{ Sandbox, Approval string }
			if err := client.Call(ctx, "native-proactive", "read-only", map[string]string{"sessionId": e.SessionID}, &result); err != nil {
				return RunResult{}, err
			}
			if result.Sandbox != "read-only" || result.Approval != "ask" {
				return RunResult{}, errors.New("Host did not confirm read-only task permissions")
			}
		} else {
			var result api.ExecuteResult
			if err := h.Client.Call(ctx, "commands/execute", map[string]any{"agentId": e.SessionID, "line": "/permission workspace-write", "submittedAttachments": []any{}}, &result); err != nil {
				return RunResult{}, err
			}
			var outcome struct{ Kind, Text string }
			if json.Unmarshal(result.Result, &outcome) != nil || outcome.Kind != "success" {
				return RunResult{}, fmt.Errorf("task permissions: %s", outcome.Text)
			}
		}
		if err := progress("sending"); err != nil {
			return RunResult{}, err
		}
		part, _ := json.Marshal(map[string]string{"type": "text", "text": e.Prompt})
		var accepted api.SessionPromptValue
		request := api.SessionPromptRequest{SessionId: e.SessionID, RequestId: e.RequestID, Mode: "queue", Content: []json.RawMessage{part}}
		if err := h.Client.Call(ctx, "session/prompt", map[string]any{"request": request}, &accepted); err != nil {
			return RunResult{}, err
		}
		if !accepted.Accepted {
			return RunResult{}, errors.New("task prompt was not accepted")
		}
	}
	if err := progress("accepted"); err != nil {
		return RunResult{}, err
	}
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			return RunResult{}, err
		}
		var frame struct {
			Type  string          `json:"type"`
			Event json.RawMessage `json:"event"`
		}
		if err := json.Unmarshal(raw, &frame); err != nil {
			return RunResult{}, err
		}
		if frame.Type == "event" {
			observed.apply(frame.Event)
		}
		if observed.done {
			return observed.result(), nil
		}
	}
}

func (h Host) Stop(ctx context.Context, sessionID string) error {
	var result api.SessionCancelValue
	if err := h.Client.Call(ctx, "session/cancel", map[string]any{"request": api.SessionCancelRequest{SessionId: sessionID}}, &result); err != nil {
		return err
	}
	if !result.Accepted {
		return errors.New("Host did not accept task cancellation")
	}
	return nil
}

type taskObservation struct {
	requestID         string
	currentTurn, turn float64
	found, done       bool
	reason, detail    string
	summary           []string
}

func (o *taskObservation) apply(raw json.RawMessage) {
	var head struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &head) != nil {
		return
	}
	switch head.Type {
	case "turn/start":
		var e events.TurnStart
		if json.Unmarshal(raw, &e) == nil {
			o.currentTurn = e.Data.Turn
		}
	case "user/message":
		var e events.UserMessage
		if json.Unmarshal(raw, &e) != nil {
			return
		}
		var source struct {
			RPCID string `json:"rpcId"`
		}
		if json.Unmarshal(e.Data.Source, &source) == nil && source.RPCID == o.requestID {
			o.found = true
			o.turn = o.currentTurn
		}
	case "assistant/message":
		var e events.AssistantMessage
		if json.Unmarshal(raw, &e) != nil || !o.found || e.Data.Turn != o.turn {
			return
		}
		for _, part := range e.Data.Message.Content {
			var text struct{ Type, Text string }
			if json.Unmarshal(part, &text) == nil && text.Type == "text" && text.Text != "" {
				o.summary = append(o.summary, text.Text)
			}
		}
	case "turn/end":
		var e events.TurnEnd
		if json.Unmarshal(raw, &e) != nil || !o.found || e.Data.Turn != o.turn {
			return
		}
		var reason struct{ Kind, Message string }
		_ = json.Unmarshal(e.Data.Reason, &reason)
		o.done, o.reason, o.detail = true, reason.Kind, reason.Message
		if o.detail == "" && reason.Kind != "completed" {
			o.detail = string(e.Data.Reason)
		}
	}
}

func (o taskObservation) result() RunResult {
	if o.reason == "completed" {
		return RunResult{Phase: "completed", State: "verifying", Summary: strings.Join(o.summary, "\n\n")}
	}
	return RunResult{Phase: "interrupted", State: "waiting_input", Summary: strings.Join(o.summary, "\n\n"), Error: o.detail}
}
