package chat

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"

	"whaleisle/internal/store/session"
)

// toolReadModel is the installed settled read card's display data. Lines and
// TotalLines retain JavaScript Number semantics within the native int range;
// labels are display text, with no file-open authority or filesystem lookup.
type toolReadModel struct {
	Label      string
	Lines      []toolReadLine
	TotalLines int
	Lang       string
}

// toolReadModelFor derives only a successful, settled root read with a retained
// call head. The model-facing envelope validates the result shape; the card's
// contents come exclusively from metadata. Source calls and events stay intact.
func toolReadModelFor(call *session.ToolCallView, cwd, home string) *toolReadModel {
	if call == nil || call.Name != "read" || call.Call == nil || call.Result == nil || call.State != "ok" || call.ParentCallID != "" || call.RootCallID != "" && call.RootCallID != call.CallID {
		return nil
	}
	args, ok := terminalModelObject(json.RawMessage(call.Arguments))
	if !ok {
		return nil
	}
	path, ok := contextRawString(args["file_path"])
	if !ok || terminalModelBlank(path) {
		return nil
	}
	for _, key := range []string{"offset", "limit"} {
		if raw, exists := args[key]; exists {
			if _, ok := toolReadModelInteger(raw, 1); !ok {
				return nil
			}
		}
	}
	// Require the original root receipt, including agreement with EventType
	// when supplied. A PTC receipt must not acquire a root card through a stale
	// or incomplete projected parent relation.
	envelope, ok := terminalModelObject(call.Result.Raw)
	if !ok {
		return nil
	}
	typ, ok := contextRawString(envelope["type"])
	if !ok || typ != "tool/result" || call.Result.EventType != "" && call.Result.EventType != typ {
		return nil
	}
	result := toolSummaryReadResult(call.Result)
	if result.nested || result.failed || result.stopped {
		return nil
	}
	meta, ok := terminalModelObject(result.meta)
	if !ok {
		return nil
	}
	path, ok = contextRawString(meta["path"])
	if !ok {
		return nil
	}
	offset, ok := toolReadModelInteger(meta["offset"], 1)
	if !ok {
		return nil
	}
	total, ok := toolReadModelInteger(meta["totalLines"], 0)
	if !ok || total >= math.Ldexp(1, strconv.IntSize-1) {
		return nil
	}
	lang := ""
	if raw, exists := meta["lang"]; exists {
		lang, ok = contextRawString(raw)
		if !ok {
			return nil
		}
	}
	rawLines, ok := contextRawArray(meta["lines"])
	if !ok {
		return nil
	}
	lines := make([]toolReadLine, 0, len(rawLines))
	previous := offset - 1
	for _, raw := range rawLines {
		line, ok := terminalModelObject(raw)
		if !ok {
			return nil
		}
		number, ok := toolReadModelInteger(line["number"], 1)
		if !ok || number <= previous || number > total {
			return nil
		}
		text, ok := contextRawString(line["text"])
		if !ok {
			return nil
		}
		// number <= total, whose explicit int bound also protects this cast.
		lines = append(lines, toolReadLine{Number: int(number), Text: text})
		previous = number
	}
	text, ok := terminalModelSingleText(result.content)
	if !ok || !toolReadModelEnvelope.MatchString(text) {
		return nil
	}
	return &toolReadModel{
		Label: toolSummaryDisplayPath(path, cwd, home), Lines: lines,
		TotalLines: int(total), Lang: lang,
	}
}

// Unlike unmarshalling into int, Number.isInteger accepts 1e1 and 10.0, and
// judges the rounded IEEE-754 value, including integers beyond the safe range.
// Arguments and the validation-only offset need no native-int restriction.
func toolReadModelInteger(raw json.RawMessage, minimum float64) (float64, bool) {
	text, ok := contextRawNumber(raw)
	if !ok {
		return 0, false
	}
	number, _ := strconv.ParseFloat(text, 64)
	return number, !math.IsInf(number, 0) && !math.IsNaN(number) && number >= minimum && number == math.Trunc(number)
}

// The installed /^<path>[^\n]*<\/path>\n<type>file<\/type>\n<content>\n
// ([\s\S]*)\n<\/content>$/u allows an empty path and body, but no outer text.
var toolReadModelEnvelope = regexp.MustCompile(`\A<path>[^\n]*</path>\n<type>file</type>\n<content>\n(?s:.*)\n</content>\z`)
