package session

import "encoding/json"

// InspectorChatTarget retains source coordinates until the current Chat projection
// resolves them. ID identifies a native render occurrence; zero is a valid anchor,
// Turn or Step, so optional numeric coordinates have explicit presence.
type InspectorChatTarget struct {
	ID         string   `json:"nodeKey,omitempty"`
	Seq        float64  `json:"-"`
	AnchorSeq  *float64 `json:"anchorSeq,omitempty"`
	CallID     string   `json:"callId,omitempty"`
	RootCallID string   `json:"rootCallId,omitempty"`
	NodeKind   string   `json:"nodeKind,omitempty"`
	GroupKey   string   `json:"groupKey,omitempty"`
	GroupPart  string   `json:"groupPart,omitempty"`
	Turn       *float64 `json:"turn,omitempty"`
	Step       *float64 `json:"step,omitempty"`
	// Native live block IDs retire at settlement; the owning Assistant Step survives.
	LiveAssistant bool `json:"-"`
}

func inspectorEventChatTarget(raw json.RawMessage) InspectorChatTarget {
	var event struct {
		Type string
		Seq  *float64
		Data struct {
			Turn, Step                    *float64
			CallID, SubCallID, RootCallID string
			Message                       struct{ Source struct{ CallID string } }
		}
	}
	if json.Unmarshal(raw, &event) != nil {
		return InspectorChatTarget{}
	}
	target := InspectorChatTarget{AnchorSeq: event.Seq, Turn: event.Data.Turn, Step: event.Data.Step}
	if event.Seq != nil {
		target.Seq = *event.Seq
	}
	switch event.Type {
	case "assistant/message", "assistant/attempt", "assistant/live-chunk":
		target.NodeKind = "assistant-step"
	case "tool/call":
		target.CallID = event.Data.CallID
	case "tool/result":
		target.CallID = event.Data.Message.Source.CallID
	}
	if target.CallID == "" {
		target.CallID = event.Data.SubCallID
	}
	if event.Data.RootCallID != "" && event.Data.RootCallID != target.CallID {
		target.RootCallID = event.Data.RootCallID
	}
	return target
}

func inspectorChunkChatTarget(target InspectorChatTarget, raw json.RawMessage) InspectorChatTarget {
	var chunk struct {
		Type, BlockType, ID string
		Block               struct{ Type, ID string }
	}
	if json.Unmarshal(raw, &chunk) != nil {
		return target
	}
	part := ""
	switch chunk.Type {
	case "reasoning-delta":
		part = "reasoning"
	case "text-delta":
		part = "text"
	case "block-start":
		part = chunk.BlockType
	case "block-end":
		part = chunk.Block.Type
	}
	switch part {
	case "reasoning":
		target.GroupPart = "reasoning"
	case "text":
		target.GroupPart = "response"
	}
	if target.CallID == "" {
		if chunk.Type == "tool-call-delta" {
			target.CallID = chunk.ID
		} else if chunk.Type == "block-end" && chunk.Block.Type == "tool-call" {
			target.CallID = chunk.Block.ID
		}
	}
	return target
}

// InspectorTarget describes this loaded native occurrence without fetching data.
func (msg Message) InspectorTarget() InspectorChatTarget {
	target := inspectorEventChatTarget(msg.Raw)
	target.ID, target.Seq = msg.ID, msg.Seq
	if target.AnchorSeq == nil && msg.Seq > 0 {
		seq := msg.Seq
		target.AnchorSeq = &seq
	}
	if msg.ConversationTurn != nil {
		target.Turn, target.Step = msg.ConversationTurn, msg.ConversationStep
	}
	if msg.Tool != nil {
		target.NodeKind, target.CallID = "tool-call", msg.Tool.CallID
		if msg.Tool.RootCallID != "" && msg.Tool.RootCallID != msg.Tool.CallID {
			target.RootCallID = msg.Tool.RootCallID
		}
		if msg.Tool.Call != nil {
			owner := msg.Tool.Call.InspectorTarget()
			target.AnchorSeq, target.Turn, target.Step = owner.AnchorSeq, owner.Turn, owner.Step
		}
	} else if msg.Kind == MessageKindThinking {
		target.NodeKind, target.GroupPart = "assistant-step", "reasoning"
	} else if msg.Role == "assistant" {
		target.NodeKind, target.GroupPart = "assistant-step", "response"
	}
	target.LiveAssistant = msg.Streaming && target.NodeKind == "assistant-step"
	return target
}
