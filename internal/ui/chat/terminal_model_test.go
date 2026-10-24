package chat

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

func terminalModelJSONForTest(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func terminalModelResultForTest(t *testing.T, typ string, data any) *session.Message {
	t.Helper()
	return &session.Message{
		Kind: session.MessageKindTool, EventType: typ,
		Raw: json.RawMessage(terminalModelJSONForTest(t, map[string]any{"type": typ, "data": data})),
		// Flattened presentation fields must never substitute for raw content.
		Body: "flattened output\n[exit code: 99]", Args: `{"command":"wrong"}`,
	}
}

func terminalModelTextCallForTest(t *testing.T, name, args, text string, nested bool) *session.ToolCallView {
	t.Helper()
	content := []any{map[string]any{"type": "text", "text": text}}
	call := &session.ToolCallView{CallID: "call-1", Name: name, Arguments: args, State: "ok"}
	if nested {
		call.ParentCallID = "parent"
		call.RootCallID = "root"
		call.Result = terminalModelResultForTest(t, "tool/ptc-dispatch", map[string]any{"content": content})
	} else {
		call.Result = terminalModelResultForTest(t, "tool/result", map[string]any{"message": map[string]any{"content": content}})
	}
	return call
}

const terminalModelShellArgsForTest = `{"command":"printf hello","description":"Print greeting"}`

func TestTerminalModelExactToolNamesAndObjectArguments(t *testing.T) {
	for _, name := range []string{"shell", "powershell", "sh", "Bash", "functions.bash", "functions.pwsh", "tools.terminal_send", "bash ", ""} {
		t.Run("name/"+name, func(t *testing.T) {
			if got := terminalModelFor(&session.ToolCallView{Name: name, Arguments: terminalModelShellArgsForTest, State: "running"}, "/session"); got != nil {
				t.Fatalf("unsupported tool produced terminal: %#v", got)
			}
		})
	}
	for _, args := range []string{"", "{", "null", "[]", `[{}]`, `"{}"`, "1", "true", "{} {}", "{\"command\":\"x\",}", `{"Command":"x","description":"Run command"}`} {
		t.Run("arguments/"+args, func(t *testing.T) {
			if got := terminalModelFor(&session.ToolCallView{Name: "bash", Arguments: args, State: "running"}, ""); got != nil {
				t.Fatalf("non-object or malformed arguments produced terminal: %#v", got)
			}
		})
	}
	if got := terminalModelFor(nil, ""); got != nil {
		t.Fatal("nil call produced terminal")
	}
}

func TestTerminalModelShellArgumentValidation(t *testing.T) {
	tests := []struct {
		name, args string
		valid      bool
	}{
		{"standard", terminalModelShellArgsForTest, true},
		{"command missing", `{"description":"Run"}`, false},
		{"command null", `{"command":null,"description":"Run"}`, false},
		{"command number", `{"command":42,"description":"Run"}`, false},
		{"command blank", `{"command":" \t\n","description":"Run"}`, false},
		{"command BOM", `{"command":"\ufeff","description":"Run"}`, false},
		{"command NEL is not JS whitespace", `{"command":"\u0085","description":"Run"}`, true},
		{"description absent persistent", `{"command":"pwd"}`, true},
		{"description empty", `{"command":"pwd","description":""}`, false},
		{"description blank", `{"command":"pwd","description":"\u00a0 \n"}`, false},
		{"description null", `{"command":"pwd","description":null}`, false},
		{"description boolean", `{"command":"pwd","description":true}`, false},
		{"timeout positive fraction", `{"command":"pwd","description":"Run","timeoutMs":0.5}`, true},
		{"timeout null", `{"command":"pwd","description":"Run","timeoutMs":null}`, false},
		{"timeout string", `{"command":"pwd","description":"Run","timeoutMs":"1000"}`, false},
		{"timeout zero", `{"command":"pwd","description":"Run","timeoutMs":0}`, false},
		{"timeout negative", `{"command":"pwd","description":"Run","timeoutMs":-1}`, false},
		{"timeout smallest positive float", `{"command":"pwd","description":"Run","timeoutMs":5e-324}`, true},
		{"timeout infinity", `{"command":"pwd","description":"Run","timeoutMs":1e9999}`, false},
		{"timeout underflows to zero", `{"command":"pwd","description":"Run","timeoutMs":1e-9999}`, false},
		{"workdir empty", `{"command":"pwd","description":"Run","workdir":""}`, true},
		{"workdir null", `{"command":"pwd","description":"Run","workdir":null}`, false},
		{"workdir number", `{"command":"pwd","description":"Run","workdir":7}`, false},
		{"background false", `{"command":"pwd","description":"Run","run_in_background":false}`, true},
		{"background true", `{"command":"pwd","description":"Run","run_in_background":true}`, true},
		{"background null", `{"command":"pwd","description":"Run","run_in_background":null}`, false},
		{"background string", `{"command":"pwd","description":"Run","run_in_background":"yes"}`, false},
		{"persistent validates timeout first", `{"command":"pwd","timeoutMs":0}`, false},
		{"persistent validates workdir first", `{"command":"pwd","workdir":null}`, false},
		{"persistent validates background first", `{"command":"pwd","run_in_background":1}`, false},
		{"persistent accepts open roots", `{"command":"pwd","workdir":"/ignored","run_in_background":true,"extension":{"version":1}}`, true},
		{"host owns escalation fields", `{"command":"pwd","description":"Run","sandbox_permissions":7,"justification":null}`, true},
		{"unknown nonfinite numeric field is ignored", `{"command":"pwd","description":"Run","extension":1e9999}`, true},
	}
	for _, name := range []string{"bash", "pwsh"} {
		for _, tc := range tests {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				call := &session.ToolCallView{Name: name, Arguments: tc.args, State: "running"}
				got := terminalModelFor(call, "/session")
				if (got != nil) != tc.valid {
					t.Fatalf("valid=%v, model=%#v", tc.valid, got)
				}
				if got != nil && (!got.Running || got.HasExit || got.HasOutput || got.HasSignal || got.MissingExit) {
					t.Fatalf("running call acquired output or status: %#v", got)
				}
			})
		}
	}
}

func TestTerminalModelPreservesAuthoredCopyAndPersistentRunningSemantics(t *testing.T) {
	call := &session.ToolCallView{
		Name: "pwsh", State: "running",
		Arguments: `{"command":"  Get-Item .\n ","description":"  Read item \n","workdir":"/w/./app/.."}`,
	}
	want := &toolTerminalModel{Command: "  Get-Item .\n ", Description: "  Read item \n", Cwd: "/w", Running: true}
	if got := terminalModelFor(call, "/session"); !reflect.DeepEqual(got, want) {
		t.Fatalf("authored values changed: got %#v, want %#v", got, want)
	}
	for _, name := range []string{"bash", "pwsh"} {
		for _, nested := range []bool{false, true} {
			call := terminalModelTextCallForTest(t, name, `{"command":"pwd","workdir":"/ignored","run_in_background":true}`, "partial\n[exit code: 7]", nested)
			call.Result = nil
			call.State = "running"
			got := terminalModelFor(call, "/session")
			want := &toolTerminalModel{Command: "pwd", Running: true}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("persistent running %s nested=%v: %#v", name, nested, got)
			}
			call = terminalModelTextCallForTest(t, name, `{"command":"pwd","workdir":"/ignored","run_in_background":true}`, "partial", nested)
			if got := terminalModelFor(call, "/session"); got != nil {
				t.Fatalf("persistent settled %s nested=%v produced terminal: %#v", name, nested, got)
			}
		}
	}
}

func TestTerminalModelFinalExitAndSignalMarkers(t *testing.T) {
	tests := []struct {
		name, text, output, signal string
		exit                       float64
		hasExit, missing, failed   bool
	}{
		{name: "implicit success", text: "hello", output: "hello", hasExit: true},
		{name: "empty output exists", hasExit: true},
		{name: "zero", text: "hello\n[exit code: 0]", output: "hello", hasExit: true},
		{name: "nonzero", text: "bad\n[exit code: 7]", output: "bad", exit: 7, hasExit: true, failed: true},
		{name: "negative", text: "bad\n[exit code: -15]", output: "bad", exit: -15, hasExit: true, failed: true},
		{name: "null", text: "unknown\n[exit code: null]", output: "unknown", hasExit: true, missing: true, failed: true},
		{name: "leading zeros accepted", text: "\n[exit code: 0007]", exit: 7, hasExit: true, failed: true},
		{name: "larger than native integer", text: "\n[exit code: 18446744073709551616]", exit: 18446744073709551616, hasExit: true, failed: true},
		{name: "negative zero", text: "\n[exit code: -0]", hasExit: true},
		{name: "signal", text: "gone\n[killed by signal: SIGTERM]", output: "gone", signal: "SIGTERM", failed: true},
		{name: "signal spaces retained", text: "\n[killed by signal:  ]", signal: " ", failed: true},
		{name: "signal CR retained", text: "\n[killed by signal: SIGTERM\r]", signal: "SIGTERM\r", failed: true},
		{name: "only last marker removed", text: "bad\n[exit code: 7]\n[killed by signal: SIGKILL]", output: "bad\n[exit code: 7]", signal: "SIGKILL", failed: true},
		{name: "timeout preserved", text: "partial\n[timed out after 1000ms]\n[exit code: 2]", output: "partial\n[timed out after 1000ms]", exit: 2, hasExit: true, failed: true},
		{name: "marker needs newline", text: "[exit code: 7]", output: "[exit code: 7]", hasExit: true},
		{name: "trailing newline not a marker", text: "bad\n[exit code: 7]\n", output: "bad\n[exit code: 7]\n", hasExit: true},
		{name: "positive plus not a marker", text: "\n[exit code: +7]", output: "\n[exit code: +7]", hasExit: true},
		{name: "decimal not a marker", text: "\n[exit code: 1.5]", output: "\n[exit code: 1.5]", hasExit: true},
		{name: "capital null not a marker", text: "\n[exit code: NULL]", output: "\n[exit code: NULL]", hasExit: true},
		{name: "empty signal not a marker", text: "\n[killed by signal: ]", output: "\n[killed by signal: ]", hasExit: true},
		{name: "multiline signal not a marker", text: "\n[killed by signal: SIG\nTERM]", output: "\n[killed by signal: SIG\nTERM]", hasExit: true},
	}
	for _, name := range []string{"bash", "pwsh"} {
		for _, nested := range []bool{false, true} {
			for _, tc := range tests {
				t.Run(name+"/"+tc.name+map[bool]string{false: "/root", true: "/ptc"}[nested], func(t *testing.T) {
					call := terminalModelTextCallForTest(t, name, terminalModelShellArgsForTest, tc.text, nested)
					want := &toolTerminalModel{Command: "printf hello", Description: "Print greeting", Output: tc.output, HasOutput: true, Signal: tc.signal, HasSignal: tc.signal != "", HasExit: tc.hasExit, MissingExit: tc.missing, ExitCode: tc.exit}
					got := terminalModelFor(call, "")
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("got %#v, want %#v", got, want)
					}
					if terminalModelFailed(got) != tc.failed {
						t.Fatalf("failed=%v, want %v", terminalModelFailed(got), tc.failed)
					}
				})
			}
		}
	}
	huge := terminalModelTextCallForTest(t, "bash", terminalModelShellArgsForTest, "\n[exit code: "+strings.Repeat("9", 400)+"]", false)
	if got := terminalModelFor(huge, ""); got == nil || !got.HasExit || !math.IsInf(got.ExitCode, 1) || !terminalModelFailed(got) {
		t.Fatalf("JavaScript Number overflow status was lost: %#v", got)
	}
}

func TestTerminalModelFailurePresenceAndRunning(t *testing.T) {
	for _, tc := range []struct {
		name   string
		model  *toolTerminalModel
		failed bool
	}{
		{"nil", nil, false},
		{"undefined status", &toolTerminalModel{}, false},
		{"numeric zero", &toolTerminalModel{HasExit: true}, false},
		{"numeric nonzero", &toolTerminalModel{HasExit: true, ExitCode: -1}, true},
		{"explicit null", &toolTerminalModel{HasExit: true, MissingExit: true}, true},
		{"defined signal", &toolTerminalModel{HasSignal: true}, true},
		{"running suppresses failure", &toolTerminalModel{Running: true, HasExit: true, MissingExit: true, HasSignal: true, ExitCode: 7}, false},
		{"background flag alone", &toolTerminalModel{Background: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := terminalModelFailed(tc.model); got != tc.failed {
				t.Fatalf("failed=%v, want %v", got, tc.failed)
			}
		})
	}
}

func TestTerminalModelTerminalSendKeepsNeutralCopyAndRawOutput(t *testing.T) {
	for _, text := range []string{"", "make\n", "  input  "} {
		args := terminalModelJSONForTest(t, map[string]any{"sessionId": "pty-3", "text": text, "submit": false, "extension": true})
		call := &session.ToolCallView{Name: "terminal_send", Arguments: args, State: "running"}
		want := &toolTerminalModel{Command: text, SessionID: "pty-3", Running: true}
		if got := terminalModelFor(call, "/session"); !reflect.DeepEqual(got, want) {
			t.Fatalf("running send copy=%#v, want %#v", got, want)
		}
		for _, nested := range []bool{false, true} {
			output := "raw\n[exit code: 7]\n[killed by signal: SIGTERM]"
			call = terminalModelTextCallForTest(t, "terminal_send", args, output, nested)
			want := &toolTerminalModel{Command: text, SessionID: "pty-3", Output: output, HasOutput: true}
			if got := terminalModelFor(call, "/session"); !reflect.DeepEqual(got, want) || terminalModelFailed(got) {
				t.Fatalf("settled send output was parsed as shell status: %#v", got)
			}
		}
	}
	for _, args := range []string{
		`{"sessionId":"","text":"x"}`, `{"sessionId":null,"text":"x"}`, `{"sessionId":7,"text":"x"}`,
		`{"sessionId":"p"}`, `{"sessionId":"p","text":null}`, `{"sessionId":"p","text":false}`,
		`{"sessionId":"p","text":"","submit":null}`, `{"sessionId":"p","text":"","submit":"yes"}`,
		`{"sessionId":"p","text":"","run_in_background":null}`, `{"sessionId":"p","text":"","run_in_background":"yes"}`,
		`{"sessionId":"p","text":"","run_in_background":true}`,
	} {
		call := terminalModelTextCallForTest(t, "terminal_send", args, "ok", false)
		if got := terminalModelFor(call, ""); got != nil {
			t.Fatalf("invalid/background send %s produced %#v", args, got)
		}
		call.Result = nil
		if got := terminalModelFor(call, ""); got != nil {
			t.Fatalf("running invalid/background send %s produced %#v", args, got)
		}
	}
	// Unlike shell descriptions, session identifiers are nonempty without trim.
	if got := terminalModelFor(&session.ToolCallView{Name: "terminal_send", Arguments: `{"sessionId":" ","text":"","run_in_background":false}`}, ""); got == nil || got.SessionID != " " {
		t.Fatalf("nonempty authored session ID rejected: %#v", got)
	}
}

func TestTerminalModelRequiresExactSingleRawTextBlock(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content any
	}{
		{"missing", nil}, {"empty", []any{}}, {"null block", []any{nil}},
		{"two texts", []any{map[string]any{"type": "text", "text": "a"}, map[string]any{"type": "text", "text": "\n[exit code: 7]"}}},
		{"mixed", []any{map[string]any{"type": "text", "text": "a"}, map[string]any{"type": "image"}}},
		{"image", []any{map[string]any{"type": "image", "text": "a"}}},
		{"missing text", []any{map[string]any{"type": "text"}}},
		{"null text", []any{map[string]any{"type": "text", "text": nil}}},
		{"numeric text", []any{map[string]any{"type": "text", "text": 7}}},
		{"wrong-case type", []any{map[string]any{"type": "Text", "text": "a"}}},
		{"wrong-case field", []any{map[string]any{"type": "text", "Text": "a"}}},
		{"array replaced with object", map[string]any{"type": "text", "text": "a"}},
	} {
		for _, nested := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/root", true: "/ptc"}[nested], func(t *testing.T) {
				validContent := []any{map[string]any{"type": "text", "text": "wrong envelope output"}}
				data := map[string]any{"content": validContent, "message": map[string]any{"content": tc.content}}
				typ := "tool/result"
				if nested {
					typ = "tool/ptc-dispatch"
					data["content"] = tc.content
					data["message"] = map[string]any{"content": validContent}
				}
				call := &session.ToolCallView{Name: "bash", Arguments: terminalModelShellArgsForTest, Result: terminalModelResultForTest(t, typ, data), State: "ok"}
				if got := terminalModelFor(call, ""); got != nil {
					t.Fatalf("non-single-text content produced terminal: %#v", got)
				}
			})
		}
	}
}

func TestTerminalModelResultPresenceErrorsAndMalformedEvents(t *testing.T) {
	call := terminalModelTextCallForTest(t, "bash", terminalModelShellArgsForTest, "ok", false)
	call.State = "running"
	if got := terminalModelFor(call, ""); got == nil || got.Running || !got.HasExit {
		t.Fatalf("result presence did not establish settlement: %#v", got)
	}
	call.Result = nil
	call.State = "ok"
	if got := terminalModelFor(call, ""); got == nil || !got.Running || got.HasExit {
		t.Fatalf("missing result inferred exit from state: %#v", got)
	}
	for _, state := range []string{"preparing", "error", "stopped"} {
		call := terminalModelTextCallForTest(t, "bash", terminalModelShellArgsForTest, "ok", false)
		call.State = state
		if got := terminalModelFor(call, ""); got != nil {
			t.Fatalf("state %s produced terminal: %#v", state, got)
		}
		call.Result = nil
		if got := terminalModelFor(call, ""); got != nil {
			t.Fatalf("state %s without result produced terminal: %#v", state, got)
		}
	}
	for _, nested := range []bool{false, true} {
		for _, field := range []string{"isError", "message.isError", "error"} {
			if nested && field == "message.isError" {
				continue
			}
			call := terminalModelTextCallForTest(t, "bash", terminalModelShellArgsForTest, "ok", nested)
			var ev map[string]any
			if err := json.Unmarshal(call.Result.Raw, &ev); err != nil {
				t.Fatal(err)
			}
			data := ev["data"].(map[string]any)
			switch field {
			case "message.isError":
				data["message"].(map[string]any)["isError"] = true
			case "error":
				data["error"] = map[string]any{"code": "interrupted"}
			default:
				data[field] = true
			}
			call.Result.Raw = json.RawMessage(terminalModelJSONForTest(t, ev))
			if got := terminalModelFor(call, ""); got != nil {
				t.Fatalf("raw %s nested=%v with stale ok state produced terminal: %#v", field, nested, got)
			}
		}
	}
	for _, raw := range []string{
		"", "{", "null", "[]", `{"type":"tool/result","data":null}`,
		`{"type":"tool/result","data":{"message":[]}}`,
		`{"type":"tool/result","data":{"content":[{"type":"text","text":"wrong location"}]}}`,
		`{"type":"tool/ptc-dispatch","data":{"message":{"content":[{"type":"text","text":"wrong location"}]}}}`,
		`{"type":"tool/call","data":{"message":{"content":[{"type":"text","text":"ok"}]}}}`,
	} {
		call := &session.ToolCallView{Name: "bash", Arguments: terminalModelShellArgsForTest, Result: &session.Message{Raw: json.RawMessage(raw)}, State: "ok"}
		if got := terminalModelFor(call, ""); got != nil {
			t.Fatalf("malformed event %s produced %#v", raw, got)
		}
	}
	call = terminalModelTextCallForTest(t, "bash", terminalModelShellArgsForTest, "ok", false)
	call.Result.EventType = "tool/ptc-dispatch"
	if got := terminalModelFor(call, ""); got != nil {
		t.Fatalf("conflicting event types produced terminal: %#v", got)
	}
	call = terminalModelTextCallForTest(t, "bash", terminalModelShellArgsForTest, "ok", false)
	call.Result.EventType = ""
	if got := terminalModelFor(call, ""); got == nil || got.Output != "ok" {
		t.Fatalf("raw event discriminator required redundant Message.EventType: %#v", got)
	}
}

func TestTerminalModelBackgroundShellKeepsOnlyCommandAfterSettlement(t *testing.T) {
	args := `{"command":"run worker","description":"Start worker","run_in_background":true,"workdir":"/worker"}`
	for _, name := range []string{"bash", "pwsh"} {
		for _, nested := range []bool{false, true} {
			for _, output := range []string{"job-123 started", "bad\n[exit code: 7]", "gone\n[killed by signal: SIGTERM]", ""} {
				call := terminalModelTextCallForTest(t, name, args, output, nested)
				want := &toolTerminalModel{Command: "run worker", Description: "Start worker", Cwd: "/worker", Background: true}
				if got := terminalModelFor(call, "/session"); !reflect.DeepEqual(got, want) || terminalModelFailed(got) {
					t.Fatalf("background settlement included acknowledgement/status: %#v", got)
				}
				call.Result = nil
				want.Running = true
				if got := terminalModelFor(call, "/session"); !reflect.DeepEqual(got, want) {
					t.Fatalf("background running model=%#v, want %#v", got, want)
				}
			}
		}
	}
	call := &session.ToolCallView{Name: "bash", Arguments: args, State: "ok", Result: terminalModelResultForTest(t, "tool/result", map[string]any{"message": map[string]any{"content": []any{map[string]any{"type": "image"}}}})}
	if got := terminalModelFor(call, ""); got == nil || !got.Background || got.HasOutput || got.HasExit {
		t.Fatalf("background acknowledgement incorrectly requires terminal output: %#v", got)
	}
}

func TestTerminalModelSpillNoticeMatchesInstalledConvention(t *testing.T) {
	const notice = "(Omitted 50000 bytes. Full formatted result stored at: /spill/output.txt. Read the file.)"
	tests := []struct {
		name, text string
		spilled    bool
	}{
		{"historical notice", notice, true},
		{"after failed preview", "failed\n[exit code: 7]\n\n" + notice, true},
		{"after signal preview", "gone\n[killed by signal: SIGTERM]\n\n" + notice, true},
		{"no omission", "( Full formatted result stored at: /spill/output.txt. Read it.)", true},
		{"unknown omission", "(More bytes were omitted. Full formatted result stored at: /spill/output.txt. Read it.)", true},
		{"zero omitted", strings.ReplaceAll(notice, "50000", "0"), true},
		{"max safe byte count", strings.ReplaceAll(notice, "50000", "9007199254740991"), true},
		{"whole images", strings.ReplaceAll(notice, "bytes.", "bytes. Omitted 2 images."), true},
		{"image only", "( Omitted 2 images. Full formatted result stored at: /spill/output.txt. Read it.)", true},
		{"opaque locator and multiline guidance", "(Omitted 42 bytes. Full formatted result stored at: /spill/报告 (1).txt. Read it.\n\n(additional guidance))", true},
		{"repeated non-notice prefix", strings.Repeat("\n\n(ordinary output", 1000) + "\n\n" + notice, true},
		{"ordinary saved text", "Full formatted result stored at: /file. Read it.)", false},
		{"non-final", notice + "\nordinary output", false},
		{"trailing newline", notice + "\n", false},
		{"missing closing parenthesis", strings.TrimSuffix(notice, ")"), false},
		{"no blank separator", "preview\n" + notice, false},
		{"embedded notice", "prefix" + notice, false},
		{"wrong omission", strings.ReplaceAll(notice, "Omitted", "Ignored"), false},
		{"negative omitted", strings.ReplaceAll(notice, "50000", "-1"), false},
		{"fraction omitted", strings.ReplaceAll(notice, "50000", "1.5"), false},
		{"leading zero omitted", strings.ReplaceAll(notice, "50000", "050000"), false},
		{"unsafe omitted", strings.ReplaceAll(notice, "50000", "9007199254740992"), false},
		{"zero images", strings.ReplaceAll(notice, "bytes.", "bytes. Omitted 0 images."), false},
		{"leading zero images", strings.ReplaceAll(notice, "bytes.", "bytes. Omitted 02 images."), false},
		{"unsafe images", strings.ReplaceAll(notice, "bytes.", "bytes. Omitted 9007199254740992 images."), false},
		{"missing retrieval separator", strings.ReplaceAll(notice, ". Read the file.", ""), false},
		{"ordinary parenthesized output", "\n\n(ordinary output)", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{"bash", "pwsh"} {
				for _, nested := range []bool{false, true} {
					call := terminalModelTextCallForTest(t, name, terminalModelShellArgsForTest, tc.text, nested)
					got := terminalModelFor(call, "")
					if (got == nil) != tc.spilled {
						t.Fatalf("%s nested=%v spilled=%v, model=%#v", name, nested, tc.spilled, got)
					}
				}
			}
		})
	}
	call := terminalModelTextCallForTest(t, "terminal_send", `{"sessionId":"p","text":""}`, notice, false)
	if got := terminalModelFor(call, ""); got == nil || got.Output != notice || got.HasExit {
		t.Fatalf("shell spill guard changed terminal_send semantics: %#v", got)
	}
	call = terminalModelTextCallForTest(t, "bash", `{"command":"start","description":"Start","run_in_background":true}`, notice, false)
	if got := terminalModelFor(call, ""); got == nil || !got.Background || got.HasOutput || got.HasExit {
		t.Fatalf("spill guard changed background command acknowledgement: %#v", got)
	}
}

func TestTerminalModelCwdUsesRecordedLocationAndPlatformIndependentFallbacks(t *testing.T) {
	tests := []struct {
		name, workdir, sessionCwd, meta, want string
		running, nested                       bool
	}{
		{name: "settled session cwd", sessionCwd: "/w/app", want: "/w/app"},
		{name: "omitted workdir keeps session spelling", sessionCwd: "/w/./app/", want: "/w/./app/"},
		{name: "relative request", workdir: "packages/ui", sessionCwd: "/w/app", want: "/w/app/packages/ui"},
		{name: "relative parent", workdir: "../sibling", sessionCwd: "/w/app", want: "/w/sibling"},
		{name: "dot request", workdir: ".", sessionCwd: "/w/app", want: "/w/app"},
		{name: "cannot climb POSIX root", workdir: "../../..", sessionCwd: "/w", want: "/"},
		{name: "absolute POSIX", workdir: "/srv/./app/../other", sessionCwd: `C:\host`, want: "/srv/other"},
		{name: "POSIX spelling preserved without dots", workdir: "/srv//app/", want: "/srv//app/"},
		{name: "relative without session", workdir: "../elsewhere", want: "../elsewhere"},
		{name: "relative collapse without session", workdir: "a/../../b", want: "../b"},
		{name: "no cwd", want: ""},
		{name: "Windows absolute on POSIX host", workdir: `C:\ws\app\..`, sessionCwd: "/w", want: `C:\ws`},
		{name: "Windows relative", workdir: `..\sibling`, sessionCwd: `C:\ws\app`, want: `C:\ws\sibling`},
		{name: "Windows drive root", workdir: `..\..\..`, sessionCwd: `D:\ws`, want: `D:\`},
		{name: "Windows slash drive", workdir: "../other", sessionCwd: "D:/ws/app", want: "D:/ws/other"},
		{name: "Windows mixed separators normalize to slash", workdir: "./child/../other", sessionCwd: `D:\ws`, want: "D:/ws/other"},
		{name: "drive relative is not absolute", workdir: "C:child", sessionCwd: "/w", want: "/w/C:child"},
		{name: "single backslash is not absolute", workdir: `\child`, sessionCwd: "/w", want: "/w/child"},
		{name: "UNC share root protected", workdir: "..", sessionCwd: `\\server\share`, want: `\\server\share`},
		{name: "UNC nested parent", workdir: "..", sessionCwd: `\\server\share\app`, want: `\\server\share`},
		{name: "UNC repeated parents", workdir: "../../..", sessionCwd: `\\server\share\app`, want: `\\server\share`},
		{name: "installed UNC tail separators", workdir: "./other/child", sessionCwd: `\\server\share`, want: `\\server\share\other/child`},
		{name: "UNC absolute request", workdir: `\\server\share\..\a`, sessionCwd: "/w", want: `\\server\share\a`},
		{name: "slash UNC normalized like installed source", workdir: "//server/share/./a", want: `\\server\share\a`},
		{name: "running session fallback omitted", sessionCwd: "/w/app", running: true},
		{name: "running relative omitted", workdir: "../other", sessionCwd: "/w/app", running: true},
		{name: "running absolute request", workdir: `C:\ws\app\..`, sessionCwd: "/w", running: true, want: `C:\ws`},
		{name: "running root-relative Windows omitted", workdir: `\root`, running: true},
		{name: "nested session fallback omitted", sessionCwd: "/w", nested: true},
		{name: "nested relative omitted", workdir: "child", sessionCwd: "/w", nested: true},
		{name: "nested absolute", workdir: "/w/app/..", sessionCwd: "/session", nested: true, want: "/w"},
		{name: "recorded absolute wins verbatim", workdir: "/request", sessionCwd: "/session", meta: `{"cwd":"/recorded/./app/../"}`, want: "/recorded/./app/../"},
		{name: "nested recorded absolute wins", workdir: "child", sessionCwd: "/session", nested: true, meta: `{"cwd":"C:\\recorded\\app\\.."}`, want: `C:\recorded\app\..`},
		{name: "recorded UNC verbatim", meta: `{"cwd":"\\\\server\\share\\.."}`, want: `\\server\share\..`},
		{name: "relative recorded fallback", workdir: "child", sessionCwd: "/session", meta: `{"cwd":"relative"}`, want: "/session/child"},
		{name: "drive relative recorded fallback", sessionCwd: "/session", meta: `{"cwd":"C:relative"}`, want: "/session"},
		{name: "single backslash recorded fallback", sessionCwd: "/session", meta: `{"cwd":"\\root"}`, want: "/session"},
		{name: "array metadata fallback", sessionCwd: "/session", meta: `[{"cwd":"/ignored"}]`, want: "/session"},
		{name: "null metadata fallback", sessionCwd: "/session", meta: `null`, want: "/session"},
		{name: "numeric cwd fallback", sessionCwd: "/session", meta: `{"cwd":7}`, want: "/session"},
		{name: "case-sensitive metadata", sessionCwd: "/session", meta: `{"Cwd":"/ignored"}`, want: "/session"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := terminalModelJSONForTest(t, map[string]any{"command": "pwd", "description": "Show directory", "workdir": tc.workdir})
			call := terminalModelTextCallForTest(t, "bash", args, "ok", tc.nested)
			if tc.meta != "" {
				var ev map[string]json.RawMessage
				if err := json.Unmarshal(call.Result.Raw, &ev); err != nil {
					t.Fatal(err)
				}
				var data map[string]json.RawMessage
				if err := json.Unmarshal(ev["data"], &data); err != nil {
					t.Fatal(err)
				}
				data["meta"] = json.RawMessage(tc.meta)
				ev["data"] = json.RawMessage(terminalModelJSONForTest(t, data))
				call.Result.Raw = json.RawMessage(terminalModelJSONForTest(t, ev))
			}
			if tc.running {
				call.Result, call.State = nil, "running"
			}
			got := terminalModelFor(call, tc.sessionCwd)
			if got == nil || got.Cwd != tc.want {
				t.Fatalf("cwd model=%#v, want %q", got, tc.want)
			}
		})
	}
	call := terminalModelTextCallForTest(t, "bash", terminalModelShellArgsForTest, "ok", false)
	call.RootCallID = "another-root"
	if got := terminalModelFor(call, "/session"); got == nil || got.Cwd != "" {
		t.Fatalf("PTC root relation acquired top-level session fallback: %#v", got)
	}
	call = terminalModelTextCallForTest(t, "terminal_send", `{"sessionId":"p","text":""}`, "ok", false)
	var ev map[string]any
	if err := json.Unmarshal(call.Result.Raw, &ev); err != nil {
		t.Fatal(err)
	}
	ev["data"].(map[string]any)["meta"] = map[string]any{"cwd": "/recorded"}
	call.Result.Raw = json.RawMessage(terminalModelJSONForTest(t, ev))
	if got := terminalModelFor(call, "/session"); got == nil || got.Cwd != "/recorded" {
		t.Fatalf("terminal_send lost recorded cwd: %#v", got)
	}
}

func TestTerminalModelNeverMutatesSourceEventsOrUsesFlattenedArguments(t *testing.T) {
	call := terminalModelTextCallForTest(t, "pwsh", terminalModelShellArgsForTest, "failed\n[exit code: null]", false)
	call.Call = &session.Message{Args: `{"command":"wrong","description":"Wrong source"}`, Raw: json.RawMessage(`{"type":"tool/call","data":{"name":"bash","arguments":"wrong"}}`)}
	call.Events = []session.Message{*call.Call, *call.Result}
	before := terminalModelJSONForTest(t, call)
	got := terminalModelFor(call, "/session")
	if got == nil || got.Command != "printf hello" || got.Output != "failed" || !got.MissingExit {
		t.Fatalf("model did not derive exact call/result data: %#v", got)
	}
	got.Command, got.Output, got.Cwd = "edited", "edited", "edited"
	if after := terminalModelJSONForTest(t, call); after != before {
		t.Fatalf("source event projection changed:\nbefore: %s\nafter: %s", before, after)
	}
	fresh := terminalModelFor(call, "/session")
	if fresh == nil || fresh.Command != "printf hello" || fresh.Output != "failed" {
		t.Fatalf("derived model mutation leaked into source/cache: %#v", fresh)
	}
}
