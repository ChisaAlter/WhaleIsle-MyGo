package chat

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"whaleisle/internal/store/session"
)

// toolSummaryModel is the generic header's display text and effective file
// navigation data. Summary is authoritative even when empty; callers must not
// substitute Detail or the basic argument summary. FilePath keeps the original
// model/recorded spelling, including legitimate settled-root relative paths.
// Line is a 1-based read offset, or zero when no representable line/link exists.
// No field resolves a path on disk or invokes the host's open-file callback.
// Terminal descriptions/exit badges and localized auto-review copy remain owned
// by their existing callers, rather than this locale-independent model.
type toolSummaryModel struct {
	Summary  string
	FilePath string
	Line     int
}

type toolSummaryResult struct {
	content, meta, failure  json.RawMessage
	nested, failed, stopped bool
}

// toolSummaryModelFor follows the installed tool-call-model, recorded-path and
// ToolRow navigation gate. Complete JSON reproduces the primary argument view;
// ToolCallView has no PartialArguments reader, so incomplete/preparing text uses
// the existing literal first-line fallback instead of pretending to expose a
// decoded streaming description or a complete file_path inside an open object.
func toolSummaryModelFor(call *session.ToolCallView, cwd, home string) toolSummaryModel {
	if call == nil {
		return toolSummaryModel{}
	}
	variant := toolRowVariant(call.Name)
	fileVariant := variant == "read" || variant == "write" || variant == "edit"
	result := toolSummaryReadResult(call.Result)
	settled := call.Result != nil
	argsRaw := call.Arguments
	if settled && call.Call == nil && !result.nested {
		// rootResult has call:null and an EMPTY argument view when the call
		// head is outside the history window. PTC receipts carry their own args.
		argsRaw = ""
	}
	fields, object := contextObjectEntries(json.RawMessage(argsRaw))
	primary, argumentPath := "", ""
	hasArgumentPath := false
	if object {
		if fileVariant {
			argumentPath, hasArgumentPath = toolSummaryPick(fields, "file_path")
			if hasArgumentPath {
				primary = toolSummaryDisplayPath(argumentPath, cwd, home)
			}
		} else if description, ok := contextSourceString(fields, "description"); ok {
			primary = toolRowFirstLine(description)
		}
	}
	base := primary
	if base == "" {
		if argsRaw == "" {
			if call.State != "preparing" {
				base = call.CallID
			}
		} else {
			base = toolSummaryDisplayPath(toolSummaryRawSummary(variant, argsRaw, fields, object), cwd, home)
		}
	}
	m := toolSummaryModel{Summary: base}
	// The original has no legacy shell title alias. The basic native title
	// helper deliberately retains one, but that must not hide the wire name here.
	if toolRowTitleName(call.Name) == "generic" || call.Name == "shell" {
		if m.Summary == "" {
			m.Summary = call.Name
		} else if call.Name != "" {
			m.Summary = call.Name + " · " + m.Summary
		}
	}
	stopped := result.stopped || call.State == "stopped" || call.State == "cancelled" || call.State == "canceled"
	failed := !stopped && (result.failed || call.State == "error" || call.State == "failed")
	if failed && settled {
		if output := toolSummaryResultText(result); output != "" {
			// A nonempty output whose first line is empty intentionally erases
			// the summary; this is distinct from absent/empty result output.
			m.Summary = toolRowFirstLine(output)
		}
	}
	// ToolRow suppresses navigation for both failure and interruption, even
	// when the row model itself derived an absolute recorded/argument path.
	if !fileVariant || failed || stopped {
		return m
	}
	if !hasArgumentPath && object {
		argumentPath, hasArgumentPath = toolSummaryPick(fields, "path", "file_path")
	}
	if !hasArgumentPath {
		return m
	}
	if recorded := toolSummaryRecordedPath(result.meta); settled && recorded != "" {
		m.FilePath = recorded
	} else if terminalModelAbsolutePath(argumentPath) || settled && !result.nested && call.ParentCallID == "" && (call.RootCallID == "" || call.RootCallID == call.CallID) {
		m.FilePath = argumentPath
	}
	if m.FilePath != "" && call.State != "preparing" && call.Name == "read" && object {
		m.Line = toolSummaryReadLine(fields)
	}
	return m
}

// pickString in the original tests the complete value for nonemptiness before
// taking its first line. Whitespace and a value starting with '\n' are strings;
// the latter can therefore establish argument-path presence with an empty line.
func toolSummaryPick(fields []contextSourceField, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := contextSourceString(fields, key); ok && value != "" {
			return toolRowFirstLine(value), true
		}
	}
	return "", false
}

func toolSummaryRawSummary(variant, raw string, fields []contextSourceField, object bool) string {
	if !object {
		// Arrays enumerate their elements; scalar/invalid JSON retains its raw
		// first line. The basic helper already implements those exact cases.
		return toolRowArgumentSummary(variant, raw)
	}
	if variant == "search" {
		if queries, ok := contextRawArray(contextSourceRaw(fields, "queries")); ok {
			var lines []string
			for _, query := range queries {
				if text, ok := contextRawString(query); ok && text != "" {
					lines = append(lines, toolRowFirstLine(text))
				}
			}
			if len(lines) > 0 {
				return strings.Join(lines, ", ")
			}
		}
	}
	var keys []string
	switch variant {
	case "bash":
		keys = []string{"description", "command"}
	case "read":
		keys = []string{"path", "file_path", "url"}
	case "search":
		keys = []string{"query", "pattern", "url"}
	case "write", "edit":
		keys = []string{"path", "file_path"}
	case "code":
		keys = []string{"description"}
	}
	if text, ok := toolSummaryPick(fields, keys...); ok {
		return text
	}
	// contextObjectEntries supplies Object.values order, including numeric
	// index keys and duplicate-key updates, which the basic helper lacks.
	for _, field := range fields {
		if text, ok := contextRawString(field.Raw); ok && text != "" {
			return toolRowFirstLine(text)
		}
	}
	return toolRowFirstLine(raw)
}

// Display helpers are literal prefix operations from dsh-util-workspace-path:
// no normalization, case folding, cwd joining, or Windows home abbreviation.
func toolSummaryDisplayPath(text, cwd, home string) string {
	if cwd != "" {
		root := strings.TrimRight(cwd, `/\`)
		if strings.HasPrefix(text, root+"/") || strings.HasPrefix(text, root+`\`) {
			text = text[len(root)+1:]
		}
	}
	if home == "" || terminalModelWindowsPath(text) || terminalModelWindowsPath(home) {
		return text
	}
	root := strings.TrimRight(home, "/")
	if root == "" {
		return text
	}
	if strings.TrimRight(text, "/") == root {
		return "~"
	}
	if strings.HasPrefix(text, root+"/") {
		return "~" + text[len(root):]
	}
	return text
}

func toolSummaryRecordedPath(raw json.RawMessage) string {
	meta, ok := terminalModelObject(raw)
	if !ok {
		return ""
	}
	value, ok := terminalModelString(meta["path"])
	if ok && terminalModelAbsolutePath(value) {
		return value
	}
	return ""
}

func toolSummaryReadResult(message *session.Message) toolSummaryResult {
	var result toolSummaryResult
	if message == nil {
		return result
	}
	envelope, ok := terminalModelObject(message.Raw)
	if !ok {
		return result
	}
	typ, ok := terminalModelString(envelope["type"])
	if !ok || message.EventType != "" && message.EventType != typ {
		return result
	}
	data, ok := terminalModelObject(envelope["data"])
	if !ok {
		return result
	}
	owner := data
	switch typ {
	case "tool/result":
		owner, ok = terminalModelObject(data["message"])
		if !ok {
			return result
		}
	case "tool/ptc-dispatch":
		result.nested = true
	default:
		return result
	}
	result.content, result.meta, result.failure = owner["content"], data["meta"], data["error"]
	result.failed, _ = terminalModelBool(owner["isError"])
	if failure, ok := terminalModelObject(result.failure); ok {
		code, _ := terminalModelString(failure["code"])
		result.stopped = code == "interrupted"
	}
	return result
}

func toolSummaryResultText(result toolSummaryResult) string {
	content, _ := contextRawArray(result.content)
	parts := make([]string, 0, len(content))
	for _, raw := range content {
		block, _ := terminalModelObject(raw)
		if typ, _ := terminalModelString(block["type"]); typ == "text" {
			text, _ := terminalModelString(block["text"])
			parts = append(parts, text)
		} else {
			parts = append(parts, contextFormattedJSON(raw))
		}
	}
	if len(parts) == 0 {
		if failure, ok := terminalModelObject(result.failure); ok {
			name, named := terminalModelString(failure["name"])
			code, coded := terminalModelString(failure["code"])
			if !named {
				name = "undefined"
			}
			if !coded {
				code = "undefined"
			}
			return name + ": " + code
		}
	}
	return strings.Join(parts, "\n")
}

// readCallLine validates the whole read envelope: file_path must be nonblank,
// offset and limit (when present) must be positive integer JSON numbers. Limit
// has no native-int restriction; an unrepresentable offset yields no Line.
func toolSummaryReadLine(fields []contextSourceField) int {
	filePath, ok := contextSourceString(fields, "file_path")
	if !ok || terminalModelBlank(filePath) {
		return 0
	}
	var offset float64
	for _, key := range []string{"offset", "limit"} {
		raw := contextSourceRaw(fields, key)
		if len(raw) == 0 {
			continue
		}
		if _, ok := contextRawNumber(raw); !ok {
			return 0
		}
		number, _ := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
		if math.IsInf(number, 0) || math.IsNaN(number) || number < 1 || number != math.Trunc(number) {
			return 0
		}
		if key == "offset" {
			offset = number
		}
	}
	if offset >= math.Ldexp(1, strconv.IntSize-1) {
		return 0
	}
	return int(offset)
}
