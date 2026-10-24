package chat

import (
	"bytes"
	"encoding/json"
	"strconv"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// toolDetailsModel is the installed goal details card's display data. Recorded
// objectives and blocker messages retain their spelling and confer no actions.
type toolDetailsModel struct {
	Items []toolDetailItem
	Empty string
}

type toolDetailItem struct {
	Title  string
	Fields []toolDetailField
}

type toolDetailField struct {
	Label, Value string
}

// toolDetailsModelFor derives only the create_goal/get_goal/update_goal subset
// of the installed details card. Root results borrow their retained raw head;
// nested results embed their own call even in a result-only history window.
func toolDetailsModelFor(call *session.ToolCallView) *toolDetailsModel {
	if call == nil || call.Result == nil || call.State != "ok" {
		return nil
	}
	result, ok := terminalModelReadResult(call.Result)
	if !ok || result.failed {
		return nil
	}
	name, ok := toolDetailsModelCallName(call, result.nested)
	if !ok {
		return nil
	}
	switch name {
	case "create_goal", "get_goal", "update_goal":
	default:
		return nil
	}
	text, ok := terminalModelSingleText(result.content)
	if !ok {
		return nil
	}
	value, ok := terminalModelObject(json.RawMessage(text))
	if !ok {
		return nil
	}
	// This precedes activation validation in the installed goalDetail.
	if bytes.Equal(bytes.TrimSpace(value["goal"]), []byte("null")) {
		return &toolDetailsModel{Items: []toolDetailItem{}, Empty: i18n.T(i18n.KeyToolDetailGoalEmpty)}
	}
	goal, ok := terminalModelObject(value["goal"])
	if !ok {
		return nil
	}
	id, validID := terminalModelString(goal["id"])
	objective, validObjective := terminalModelString(goal["objective"])
	_, validRevision := toolDetailsModelCount(goal["revision"])
	rounds, validRounds := toolDetailsModelCount(goal["roundsStarted"])
	maximum, validMaximum := toolDetailsModelCount(goal["maxGoalRounds"])
	if !validID || terminalModelBlank(id) || !validObjective || terminalModelBlank(objective) || !validRevision || !validRounds || !validMaximum {
		return nil
	}
	phase, _ := terminalModelString(goal["phase"])
	stateKey := ""
	switch phase {
	case "active":
		stateKey = i18n.KeyToolDetailGoalActive
	case "paused":
		stateKey = i18n.KeyToolDetailGoalPaused
	case "blocked":
		stateKey = i18n.KeyToolDetailGoalBlocked
	case "complete":
		stateKey = i18n.KeyToolDetailGoalComplete
	default:
		return nil
	}
	activation, _ := terminalModelString(value["activation"])
	if activation != "armed" && activation != "disarmed" {
		return nil
	}
	if phase == "active" && activation == "disarmed" {
		stateKey = i18n.KeyToolDetailGoalDisarmed
	}
	fields := []toolDetailField{
		{Label: i18n.T(i18n.KeyToolDetailState), Value: i18n.T(stateKey)},
		{Label: i18n.T(i18n.KeyToolDetailGoalRounds), Value: rounds + " / " + maximum},
	}
	if raw, exists := goal["blockedReason"]; exists {
		reason, validReason := terminalModelObject(raw)
		code, validCode := terminalModelString(reason["code"])
		message, validMessage := terminalModelString(reason["message"])
		if !validReason || !validCode || terminalModelBlank(code) || !validMessage || terminalModelBlank(message) {
			return nil
		}
		fields = append(fields, toolDetailField{Label: i18n.T(i18n.KeyToolDetailGoalReason), Value: message})
	}
	return &toolDetailsModel{Items: []toolDetailItem{{Title: objective, Fields: fields}}}
}

// parsedToolCall uses JSON.parse of the actual call's argsRaw, never flattened
// projection fields. childResult supplies JSON.stringify(data.arguments).
func toolDetailsModelCallName(call *session.ToolCallView, nested bool) (string, bool) {
	var data map[string]json.RawMessage
	if nested {
		envelope, _ := terminalModelObject(call.Result.Raw)
		data, _ = terminalModelObject(envelope["data"])
		if _, ok := terminalModelObject(data["arguments"]); !ok {
			return "", false
		}
	} else {
		if call.Call == nil {
			return "", false
		}
		envelope, ok := terminalModelObject(call.Call.Raw)
		if !ok {
			return "", false
		}
		typ, _ := terminalModelString(envelope["type"])
		if typ != "tool/call" || call.Call.EventType != "" && call.Call.EventType != typ {
			return "", false
		}
		data, ok = terminalModelObject(envelope["data"])
		if !ok {
			return "", false
		}
		args, ok := terminalModelString(data["arguments"])
		if !ok {
			return "", false
		}
		if _, ok := terminalModelObject(json.RawMessage(args)); !ok {
			return "", false
		}
	}
	return terminalModelString(data["name"])
}

// count accepts exactly nonnegative Number.isSafeInteger values after JSON's
// IEEE-754 rounding. Zero and rounds above the maximum are valid recorded data.
func toolDetailsModelCount(raw json.RawMessage) (string, bool) {
	number, ok := toolReadModelInteger(raw, 0)
	if !ok || number > 9007199254740991 {
		return "", false
	}
	return strconv.FormatFloat(number, 'f', -1, 64), true
}
