package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// ErrUnknownEventType is returned when the JSON "type" is not in EventsByType.
var ErrUnknownEventType = errors.New("events: unknown event type")

var goTypeReflect map[string]reflect.Type

func init() {
	samples := allEventStructSamples()
	goTypeReflect = make(map[string]reflect.Type, len(samples))
	for _, s := range samples {
		t := reflect.TypeOf(s)
		goTypeReflect[t.Name()] = t
	}
}

// UnmarshalEvent decodes one session persistence event (JSONL / follow "event" payload)
// into the matching generated struct. The returned value is a concrete struct value
// (not a pointer). Unknown type strings yield ErrUnknownEventType.
func UnmarshalEvent(raw []byte) (any, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("events: decode type: %w", err)
	}
	if head.Type == "" {
		return nil, fmt.Errorf("events: missing type field")
	}
	info, ok := EventsByType[head.Type]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownEventType, head.Type)
	}
	rt, ok := goTypeReflect[info.GoType]
	if !ok {
		return nil, fmt.Errorf("events: no Go type registered for %q", info.GoType)
	}
	ptr := reflect.New(rt)
	if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
		return nil, fmt.Errorf("events: unmarshal %q: %w", head.Type, err)
	}
	return ptr.Elem().Interface(), nil
}

// allEventStructSamples lists every generated event struct so UnmarshalEvent can
// allocate the correct type. When tools/gen-events adds roots, extend this list.
func allEventStructSamples() []any {
	return []any{
		AgentPresetSelected{},
		AgentInboxSpliced{},
		ApprovalAsked{},
		ApprovalDecided{},
		ApprovalPolicy{},
		AssistantAttempt{},
		AssistantMessage{},
		CommandDone{},
		CommandRun{},
		CompactionEnd{},
		CompactionPrune{},
		CompactionStart{},
		CompactionSummary{},
		DeliverablesPresented{},
		DeveloperMessage{},
		FeedbackMessageDelete{},
		FeedbackMessagePut{},
		FeedbackRecord{},
		GoalChange{},
		HookInvoked{},
		HookResult{},
		ImageOffload{},
		LlmRetry{},
		LlmRetryStarted{},
		ModelSelection{},
		PermissionPreset{},
		PlanMode{},
		RequestContext{},
		RequestHeader{},
		SandboxMode{},
		ScheduleChange{},
		SessionLogDeepseekDeliveryAccepted{},
		SessionEndSeed{},
		SessionTitle{},
		SessionTitleLlmRequest{},
		StepEnd{},
		StepStart{},
		SubagentCatalog{},
		SubagentDescriptor{},
		SubagentModelSelectionPolicy{},
		SystemMessage{},
		TeamMember{},
		TeamMessageDelivered{},
		TeamMessageQueued{},
		TeamTask{},
		TodoWrite{},
		ToolWorkflowAgentEnd{},
		ToolWorkflowAgentStart{},
		ToolWorkflowRunEnd{},
		ToolWorkflowRunStart{},
		ToolCall{},
		ToolPtcDispatch{},
		ToolPtcDispatchStart{},
		ToolResult{},
		TurnEnd{},
		TurnStart{},
		UserMessage{},
		WebDeepseekSearchLlmRequest{},
		WorkspaceChanges{},
	}
}
