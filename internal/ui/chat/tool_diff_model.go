package chat

import (
	"encoding/json"

	"whaleisle/internal/store/session"
)

// toolDiffModel contains the installed diff card's display fragments. Paths stay
// verbatim and confer no file-open authority; deriving a model never reads disk.
type toolDiffModel struct {
	Diffs []toolFileDiff
}

// toolDiffModelFor selects running root mutation intent or successful settled
// write/edit metadata. A settled write alone may fall back to its valid intent.
// Malformed calls, nested calls, preparation and failure cues use generic output.
func toolDiffModelFor(call *session.ToolCallView) *toolDiffModel {
	if call == nil || call.ParentCallID != "" || call.RootCallID != "" && call.RootCallID != call.CallID || call.State == "preparing" || toolSummaryHasCue(call.State) {
		return nil
	}
	if call.Result != nil && call.Call == nil {
		return nil
	}
	args, ok := terminalModelObject(json.RawMessage(call.Arguments))
	if !ok {
		return nil
	}
	intended, ok := toolDiffModelIntent(call.Name, args)
	if !ok {
		return nil
	}
	if call.Result == nil {
		return &toolDiffModel{Diffs: []toolFileDiff{intended}}
	}
	if call.Name == "str_replace_editor" {
		return nil
	}
	// The original settled block has a root receipt and retained call head.
	// Validate that receipt before treating missing metadata as write fallback.
	envelope, ok := terminalModelObject(call.Result.Raw)
	if !ok {
		return nil
	}
	typ, ok := terminalModelString(envelope["type"])
	if !ok || typ != "tool/result" || call.Result.EventType != "" && call.Result.EventType != typ {
		return nil
	}
	data, ok := terminalModelObject(envelope["data"])
	if !ok {
		return nil
	}
	if _, ok := terminalModelObject(data["message"]); !ok {
		return nil
	}
	result := toolSummaryReadResult(call.Result)
	if result.nested || result.failed || result.stopped {
		return nil
	}
	meta, _ := terminalModelObject(result.meta)
	applied, valid := toolSummaryAppliedDiffs(meta["diffs"])
	if !valid || len(applied) == 0 {
		if call.Name == "write" {
			return &toolDiffModel{Diffs: []toolFileDiff{intended}}
		}
		return nil
	}
	if recorded := toolSummaryRecordedPath(result.meta); recorded != "" {
		for i := range applied {
			if applied[i].Path == recorded {
				applied[i].Path = intended.Path
			}
		}
	}
	return &toolDiffModel{Diffs: applied}
}

func toolDiffModelIntent(name string, args map[string]json.RawMessage) (toolFileDiff, bool) {
	key := "file_path"
	if name == "str_replace_editor" {
		key = "path"
	}
	path, ok := terminalModelString(args[key])
	if !ok || terminalModelBlank(path) {
		return toolFileDiff{}, false
	}
	diff := toolFileDiff{Path: path}
	switch name {
	case "write":
		diff.NewText, ok = terminalModelString(args["content"])
		return diff, ok
	case "edit":
		old, oldOK := terminalModelString(args["old_string"])
		next, nextOK := terminalModelString(args["new_string"])
		if raw, exists := args["replace_all"]; exists {
			if _, ok := terminalModelBool(raw); !ok {
				return diff, false
			}
		}
		if old != "" {
			diff.OldText = &old
		}
		diff.NewText = next
		return diff, oldOK && nextOK
	case "str_replace_editor":
		command, _ := terminalModelString(args["command"])
		switch command {
		case "create":
			if raw, exists := args["file_text"]; exists {
				diff.NewText, ok = terminalModelString(raw)
				return diff, ok
			}
			return diff, true
		case "str_replace":
			if raw, exists := args["old_str"]; exists {
				old, ok := terminalModelString(raw)
				if !ok {
					return diff, false
				}
				diff.OldText = &old
			}
			if raw, exists := args["new_str"]; exists {
				diff.NewText, ok = terminalModelString(raw)
				return diff, ok
			}
			return diff, true
		}
	}
	return toolFileDiff{}, false
}
