package session

import (
	"encoding/json"
	"strings"
	"unicode"
)

// MessageKindSubmittedPlan is a view-only, immutable document card. It never
// becomes a persistence event or an answer to a pending plan review.
const MessageKindSubmittedPlan MessageKind = "submitted-plan"

// SubmittedPlan is the complete document from an exit_plan_mode invocation.
type SubmittedPlan struct {
	CallID   string
	Title    string
	Markdown string
}

// SubmittedPlanFromEvent mirrors the Host's native/PTC document contract. Native
// arguments are JSON text; PTC arguments are objects. Settlement cannot rewrite
// the plan from an earlier invocation because WithSubmittedPlanCards keeps the first.
func SubmittedPlanFromEvent(raw json.RawMessage) (SubmittedPlan, bool) {
	var event struct {
		Type string `json:"type"`
		Data struct {
			Name      string          `json:"name"`
			CallID    string          `json:"callId"`
			SubCallID string          `json:"subCallId"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &event) != nil || event.Data.Name != "exit_plan_mode" {
		return SubmittedPlan{}, false
	}
	callID, arguments := event.Data.CallID, event.Data.Arguments
	switch event.Type {
	case "tool/call":
		var source string
		if json.Unmarshal(arguments, &source) != nil {
			return SubmittedPlan{}, false
		}
		arguments = json.RawMessage(source)
	case "tool/ptc-dispatch-start", "tool/ptc-dispatch":
		callID = event.Data.SubCallID
	default:
		return SubmittedPlan{}, false
	}
	if callID == "" {
		return SubmittedPlan{}, false
	}
	var args struct {
		Plan *string `json:"plan"`
	}
	if json.Unmarshal(arguments, &args) != nil || args.Plan == nil {
		return SubmittedPlan{}, false
	}
	markdown := *args.Plan
	trimmed := strings.TrimSpace(markdown)
	// Host submittedPlan requires a first-level heading at the start.
	if len(trimmed) < 3 || trimmed[0] != '#' {
		return SubmittedPlan{}, false
	}
	rest := trimmed[1:]
	runes := []rune(rest)
	if len(runes) == 0 || !unicode.IsSpace(runes[0]) {
		return SubmittedPlan{}, false
	}
	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
	if rest == "" {
		return SubmittedPlan{}, false
	}
	title := rest
	if end := strings.IndexAny(title, "\r\n"); end >= 0 {
		title = title[:end]
	}
	return SubmittedPlan{CallID: callID, Title: title, Markdown: markdown}, true
}

// WithSubmittedPlanCards derives persistent document cards from loaded history.
// Snapshot replay, pagination and reconnect already restore Message.Raw; this
// function needs no extra RPC, mutable cache, or cross-session document registry.
// Apply it before ProjectToolMessages so nested PTC cards survive tool folding.
func WithSubmittedPlanCards(messages []Message) []Message {
	out := make([]Message, 0, len(messages))
	seen := make(map[string]bool)
	for _, message := range messages {
		if message.Kind == MessageKindSubmittedPlan {
			continue
		}
		out = append(out, message)
		plan, ok := SubmittedPlanFromEvent(message.Raw)
		if !ok || seen[plan.CallID] {
			continue
		}
		seen[plan.CallID] = true
		out = append(out, Message{
			ID: "submitted-plan:" + plan.CallID, Kind: MessageKindSubmittedPlan,
			Role: "plan", Body: plan.Markdown, Seq: message.Seq, TurnSeq: message.TurnSeq,
			EventType: message.EventType, Raw: append(json.RawMessage(nil), message.Raw...),
			EventTime: message.EventTime,
		})
	}
	return out
}
