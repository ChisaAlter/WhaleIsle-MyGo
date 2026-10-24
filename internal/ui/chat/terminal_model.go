package chat

import (
	"bytes"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"

	"whaleisle/internal/store/session"
)

// toolTerminalModel is display-only data derived from the original terminal-card
// model. SessionID identifies locale-owned terminal_send copy; an empty Command
// stays empty until the render site supplies that copy. HasExit includes an
// explicit null status, distinguished from a numeric code by MissingExit.
type toolTerminalModel struct {
	Command, Description, Cwd, Output, Signal, SessionID            string
	Running, Background, HasOutput, HasSignal, HasExit, MissingExit bool
	ExitCode                                                        float64
}

type terminalModelCall struct {
	command, description, workdir, sessionID string
	persistent, background, send             bool
}

type terminalModelResult struct {
	content, meta  json.RawMessage
	nested, failed bool
}

// terminalModelFor reads the call projection and its original result event. It
// neither edits those events nor executes, resolves on disk, or opens commands.
func terminalModelFor(call *session.ToolCallView, sessionCwd string) *toolTerminalModel {
	if call == nil || call.State == "preparing" || call.State == "error" || call.State == "stopped" {
		return nil
	}
	args, ok := terminalModelObject(json.RawMessage(call.Arguments))
	if !ok {
		return nil
	}
	spec, ok := terminalModelReadCall(call.Name, args)
	if !ok || (spec.send && spec.background) {
		return nil
	}
	m := &toolTerminalModel{
		Command: spec.command, Description: spec.description, SessionID: spec.sessionID,
		Background: spec.background,
	}
	if terminalModelAbsolutePath(spec.workdir) {
		m.Cwd = terminalModelNormalizePath(spec.workdir)
	}
	if call.Result == nil {
		m.Running = true
		return m
	}
	result, ok := terminalModelReadResult(call.Result)
	if !ok || result.failed || spec.persistent {
		return nil
	}
	if recorded := terminalModelRecordedCwd(result.meta); recorded != "" {
		m.Cwd = recorded
	} else if !spec.send && !result.nested && call.ParentCallID == "" && (call.RootCallID == "" || call.RootCallID == call.CallID) {
		m.Cwd = terminalModelResolveCwd(spec.workdir, sessionCwd)
	}
	if spec.background {
		// A settled job acknowledgement reports no process exit or terminal output.
		return m
	}
	output, ok := terminalModelSingleText(result.content)
	if !ok || (!spec.send && terminalModelHasSpillNotice(output)) {
		return nil
	}
	m.Output, m.HasOutput = output, true
	if !spec.send {
		terminalModelParseExit(m)
	}
	return m
}

func terminalModelFailed(m *toolTerminalModel) bool {
	return m != nil && !m.Running && (m.MissingExit || (m.HasExit && m.ExitCode != 0) || m.HasSignal)
}

func terminalModelReadCall(name string, args map[string]json.RawMessage) (terminalModelCall, bool) {
	var call terminalModelCall
	switch name {
	case "bash", "pwsh":
		command, ok := terminalModelString(args["command"])
		if !ok || terminalModelBlank(command) {
			return call, false
		}
		if raw, exists := args["timeoutMs"]; exists {
			number := bytes.TrimSpace(raw)
			var timeout float64
			if len(number) == 0 || (number[0] != '-' && (number[0] < '0' || number[0] > '9')) || json.Unmarshal(number, &timeout) != nil || math.IsInf(timeout, 0) || math.IsNaN(timeout) || timeout <= 0 {
				return call, false
			}
		}
		if raw, exists := args["workdir"]; exists {
			var valid bool
			call.workdir, valid = terminalModelString(raw)
			if !valid {
				return call, false
			}
		}
		if raw, exists := args["run_in_background"]; exists {
			var valid bool
			call.background, valid = terminalModelBool(raw)
			if !valid {
				return call, false
			}
		}
		call.command = command
		raw, exists := args["description"]
		if !exists {
			// Persistent providers omit description but still validate the optional
			// standard fields above before discarding workdir and background.
			call.persistent, call.workdir, call.background = true, "", false
			return call, true
		}
		description, ok := terminalModelString(raw)
		if !ok || terminalModelBlank(description) {
			return call, false
		}
		call.description = description
		return call, true
	case "terminal_send":
		sessionID, ok := terminalModelString(args["sessionId"])
		if !ok || sessionID == "" {
			return call, false
		}
		text, ok := terminalModelString(args["text"])
		if !ok {
			return call, false
		}
		if raw, exists := args["submit"]; exists {
			if _, valid := terminalModelBool(raw); !valid {
				return call, false
			}
		}
		if raw, exists := args["run_in_background"]; exists {
			var valid bool
			call.background, valid = terminalModelBool(raw)
			if !valid {
				return call, false
			}
		}
		call.command, call.sessionID, call.send = text, sessionID, true
		return call, true
	default:
		return call, false
	}
}

func terminalModelReadResult(message *session.Message) (terminalModelResult, bool) {
	var result terminalModelResult
	envelope, ok := terminalModelObject(message.Raw)
	if !ok {
		return result, false
	}
	typ, ok := terminalModelString(envelope["type"])
	if !ok || (message.EventType != "" && message.EventType != typ) {
		return result, false
	}
	data, ok := terminalModelObject(envelope["data"])
	if !ok {
		return result, false
	}
	contentOwner := data
	switch typ {
	case "tool/result":
		contentOwner, ok = terminalModelObject(data["message"])
		if !ok {
			return result, false
		}
	case "tool/ptc-dispatch":
		result.nested = true
	default:
		return result, false
	}
	result.content, result.meta = contentOwner["content"], data["meta"]
	dataError, _ := terminalModelBool(data["isError"])
	messageError, _ := terminalModelBool(contentOwner["isError"])
	errorValue := bytes.TrimSpace(data["error"])
	result.failed = dataError || messageError || (len(errorValue) != 0 && string(errorValue) != "null")
	return result, true
}

func terminalModelSingleText(raw json.RawMessage) (string, bool) {
	var blocks []json.RawMessage
	if json.Unmarshal(raw, &blocks) != nil || len(blocks) != 1 {
		return "", false
	}
	block, ok := terminalModelObject(blocks[0])
	if !ok {
		return "", false
	}
	typ, ok := terminalModelString(block["type"])
	if !ok || typ != "text" {
		return "", false
	}
	return terminalModelString(block["text"])
}

var (
	terminalModelSignalMarker  = regexp.MustCompile(`\n\[killed by signal: ([^\]\n]+)\]$`)
	terminalModelExitMarker    = regexp.MustCompile(`\n\[exit code: (-?\d+|null)\]$`)
	terminalModelDotSegment    = regexp.MustCompile(`(?:^|[/\\])\.\.?(?:[/\\]|$)`)
	terminalModelUNCRoot       = regexp.MustCompile(`^[/\\]{2}([^/\\]+)[/\\]+([^/\\]+)`)
	terminalModelImageOmission = regexp.MustCompile(` Omitted ([1-9][0-9]*) images\.$`)
	terminalModelByteOmission  = regexp.MustCompile(`^Omitted (0|[1-9][0-9]*) bytes\.$`)
)

func terminalModelParseExit(m *toolTerminalModel) {
	if match := terminalModelSignalMarker.FindStringSubmatchIndex(m.Output); match != nil {
		m.Signal, m.HasSignal = m.Output[match[2]:match[3]], true
		m.Output = m.Output[:match[0]]
		return
	}
	m.HasExit = true
	if match := terminalModelExitMarker.FindStringSubmatchIndex(m.Output); match != nil {
		value := m.Output[match[2]:match[3]]
		m.Output = m.Output[:match[0]]
		if value == "null" {
			m.MissingExit = true
		} else {
			// Like JavaScript Number, an arbitrarily long marker may become an
			// infinity; the marker grammar has already excluded other parse errors.
			m.ExitCode, _ = strconv.ParseFloat(value, 64)
		}
	}
}

func terminalModelObject(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var object map[string]json.RawMessage
	err := json.Unmarshal(raw, &object)
	return object, err == nil && object != nil
}

func terminalModelString(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	var value string
	if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}

func terminalModelBool(raw json.RawMessage) (bool, bool) {
	switch string(bytes.TrimSpace(raw)) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

// String.trim in the original includes BOM but excludes Unicode NEL.
func terminalModelBlank(value string) bool {
	return strings.TrimFunc(value, func(r rune) bool {
		switch r {
		case '\t', '\n', '\v', '\f', '\r', ' ', '\u00a0', '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000', '\ufeff':
			return true
		default:
			return r >= '\u2000' && r <= '\u200a'
		}
	}) == ""
}

func terminalModelWindowsPath(value string) bool {
	return strings.HasPrefix(value, `\\`) || (len(value) >= 3 && terminalModelDriveLetter(value[0]) && value[1] == ':' && (value[2] == '/' || value[2] == '\\'))
}

func terminalModelDriveLetter(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z')
}

func terminalModelAbsolutePath(value string) bool {
	return strings.HasPrefix(value, "/") || terminalModelWindowsPath(value)
}

func terminalModelRecordedCwd(raw json.RawMessage) string {
	meta, ok := terminalModelObject(raw)
	if !ok {
		return ""
	}
	value, ok := terminalModelString(meta["cwd"])
	if !ok || !terminalModelAbsolutePath(value) {
		return ""
	}
	return value
}

func terminalModelResolveCwd(workdir, sessionCwd string) string {
	if workdir == "" {
		return sessionCwd
	}
	if sessionCwd == "" || terminalModelAbsolutePath(workdir) {
		return terminalModelNormalizePath(workdir)
	}
	separator := "/"
	if terminalModelWindowsPath(sessionCwd) && strings.Contains(sessionCwd, `\`) {
		separator = `\`
	}
	joined := strings.TrimRight(sessionCwd, `/\`) + separator + strings.TrimLeft(workdir, `/\`)
	return terminalModelNormalizePath(joined)
}

func terminalModelNormalizePath(value string) string {
	if !terminalModelDotSegment.MatchString(value) {
		return value
	}
	if unc := terminalModelUNCRoot.FindStringSubmatch(value); unc != nil {
		root := `\\` + unc[1] + `\` + unc[2]
		// The installed model uses collapse's default '/' for the UNC tail.
		rest := terminalModelCollapsePath(value[len(unc[0]):], true, "/")
		if rest == "" {
			return root
		}
		return root + `\` + rest
	}
	separator := "/"
	if strings.Contains(value, `\`) && !strings.Contains(value, "/") {
		separator = `\`
	}
	rooted := strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\`)
	drive := ""
	if len(value) >= 2 && terminalModelDriveLetter(value[0]) && value[1] == ':' {
		drive = value[:2]
	}
	body := terminalModelCollapsePath(value[len(drive):], rooted || drive != "", separator)
	leading := ""
	if rooted {
		leading = separator
	}
	if drive == "" {
		return leading + body
	}
	if rooted {
		return drive + leading + body
	}
	return drive + separator + body
}

func terminalModelCollapsePath(body string, rooted bool, separator string) string {
	kept := make([]string, 0)
	for _, segment := range strings.FieldsFunc(body, func(r rune) bool { return r == '/' || r == '\\' }) {
		switch segment {
		case ".":
			continue
		case "..":
			if len(kept) > 0 && kept[len(kept)-1] != ".." {
				kept = kept[:len(kept)-1]
			} else if !rooted {
				kept = append(kept, segment)
			}
		default:
			kept = append(kept, segment)
		}
	}
	return strings.Join(kept, separator)
}

// terminalModelHasSpillNotice follows dsh-spill-policy/notice rather than matching
// a substring: omission wording, safe integer counts, separators and the final
// closing parenthesis are all part of the recorded convention.
func terminalModelHasSpillNotice(text string) bool {
	const locationText = " Full formatted result stored at: "
	if !strings.HasSuffix(text, ")") {
		return false
	}
	for start := 0; ; {
		next := strings.Index(text[start:], "\n\n(")
		end := len(text) - 1
		if next >= 0 {
			next += start
			end = next
		}
		candidate := text[start:end]
		location := -1
		if len(candidate) > 0 {
			if offset := strings.Index(candidate[1:], locationText); offset >= 0 {
				location = offset + 1
			}
		}
		if strings.HasPrefix(candidate, "(") && location >= 0 && terminalModelSpillOmission(candidate[1:location]) {
			return strings.Contains(text[start+location+len(locationText):], ". ")
		}
		if next < 0 {
			return false
		}
		start = next + 2
	}
}

func terminalModelSpillOmission(text string) bool {
	if image := terminalModelImageOmission.FindStringSubmatchIndex(text); image != nil {
		if !terminalModelSafeCount(text[image[2]:image[3]]) {
			return false
		}
		text = text[:image[0]]
	}
	if text == "" || text == "More bytes were omitted." {
		return true
	}
	bytesOmitted := terminalModelByteOmission.FindStringSubmatch(text)
	return bytesOmitted != nil && terminalModelSafeCount(bytesOmitted[1])
}

func terminalModelSafeCount(text string) bool {
	value, err := strconv.ParseUint(text, 10, 64)
	return err == nil && value <= 9007199254740991
}
