package chat

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

const toolDetailsModelGoalForTest = `{"id":"g","objective":" objective ","revision":1,"roundsStarted":2,"maxGoalRounds":10,"phase":"active"}`

func toolDetailsModelJSONForTest(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func toolDetailsModelValueForTest(goal string) string {
	return `{"goal":` + goal + `,"activation":"armed"}`
}

func toolDetailsModelGoalFieldForTest(field, raw string) string {
	return strings.TrimSuffix(toolDetailsModelGoalForTest, "}") + `,"` + field + `":` + raw + `}`
}

func toolDetailsModelCallForTest(t *testing.T, nested bool, name, args, value string) *session.ToolCallView {
	t.Helper()
	content := `[{"type":"text","text":` + toolDetailsModelJSONForTest(t, value) + `}]`
	return toolDetailsModelContentCallForTest(t, nested, name, args, content)
}

// Keep the actual root/nested wire shape, with unrelated flattened fields so a
// successful model cannot silently depend on a synthetic projection fixture.
func toolDetailsModelContentCallForTest(t *testing.T, nested bool, name, args, content string) *session.ToolCallView {
	t.Helper()
	headType, resultType := "tool/call", "tool/result"
	head := `{"callId":"goal-123","name":` + toolDetailsModelJSONForTest(t, name) + `,"arguments":` + toolDetailsModelJSONForTest(t, args) + `}`
	result := `{"message":{"toolCallId":"goal-123","source":{"kind":"tool","callId":"goal-123"},"content":` + content + `}}`
	if nested {
		headType, resultType = "tool/ptc-dispatch-start", "tool/ptc-dispatch"
		head = `{"rootCallId":"root","parentCallId":"parent","subCallId":"goal-123","name":` + toolDetailsModelJSONForTest(t, name) + `,"arguments":` + args + `}`
		result = strings.TrimSuffix(head, "}") + `,"content":` + content + `}`
	}
	call := &session.ToolCallView{
		CallID: "goal-123", Name: "unrelated projected name", Arguments: "unrelated projected arguments", State: "ok",
		Call: &session.Message{Kind: session.MessageKindTool, EventType: headType, Seq: 1,
			Raw: json.RawMessage(`{"type":` + toolDetailsModelJSONForTest(t, headType) + `,"data":` + head + `}`), Body: "flattened head", Args: "flattened arguments"},
		Result: &session.Message{Kind: session.MessageKindTool, EventType: resultType, Seq: 2,
			Raw: json.RawMessage(`{"type":` + toolDetailsModelJSONForTest(t, resultType) + `,"data":` + result + `}`), Body: "flattened output", Args: "flattened result arguments"},
	}
	if nested {
		call.ParentCallID, call.RootCallID = "parent", "root"
	}
	return call
}

func toolDetailsModelLocaleForTest(t *testing.T, locale string) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(locale)
	t.Cleanup(func() { i18n.SetLocale(previous) })
}

// Recorded with CURRENT installed files in isolated node:vm SourceTextModules,
// including actual toolDefinition start/update/buildViewNode and dictionaries:
// details-card-model.js SHA256 61babee6dd3ea6351bcceffb9540641744853638e6dc8640aea3323884cd2da7
// raw-tool-call.js SHA256 c97f8fa5e1b85e63b2bb4c65c1b7bbb265793dd0631ec0a3f32c38b3c15b44a8
// detail-model-shared.js SHA256 bdc969d96403522b340da929252720a2080d42ddef861d5be61e7b99c7f13b76
// ui-chat/conversation-nodes/tool.js SHA256 5a260720b7d1d12b01e8414c56ceb383385886da09d5be3f7799d87506dcb26b
// ui-conversation/locales.js SHA256 9a7d75db4ce85fbf163e77fd99eea6ed30e67d54116ff7a8b6228f09e9720066
// The retained tests require neither Node nor installed sources or host access.
func TestToolDetailsModelInstalledGoldens(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	var fixtures []struct {
		Name, Tool, ArgsRaw, ValueRaw, ContentRaw, HeadTool, HeadArgsRaw string
		Nested, ResultOnly, IsError                                      bool
		Want                                                             *toolDetailsModel
	}
	if err := json.Unmarshal([]byte(toolDetailsModelInstalledGoldensForTest), &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			name, args, value := fixture.Tool, fixture.ArgsRaw, fixture.ValueRaw
			if name == "" {
				name = "create_goal"
			}
			if args == "" {
				args = `{}`
			}
			if value == "" {
				value = toolDetailsModelValueForTest(toolDetailsModelGoalForTest)
			}
			call := toolDetailsModelCallForTest(t, fixture.Nested, name, args, value)
			if fixture.ContentRaw != "" {
				call = toolDetailsModelContentCallForTest(t, fixture.Nested, name, args, fixture.ContentRaw)
			}
			if fixture.HeadTool != "" || fixture.HeadArgsRaw != "" {
				head := toolDetailsModelCallForTest(t, fixture.Nested, fixture.HeadTool, fixture.HeadArgsRaw, value)
				call.Call = head.Call
			}
			if fixture.ResultOnly {
				call.Call = nil
			}
			if fixture.IsError {
				call.State = "error"
				call.Result.Raw = json.RawMessage(strings.Replace(string(call.Result.Raw), `"content":`, `"isError":true,"content":`, 1))
			}
			before := toolDetailsModelJSONForTest(t, call)
			if got := toolDetailsModelFor(call); !reflect.DeepEqual(got, fixture.Want) {
				t.Fatalf("model = %#v, want %#v", got, fixture.Want)
			}
			if toolDetailsModelJSONForTest(t, call) != before {
				t.Fatal("derivation mutated retained input")
			}
		})
	}
}

func TestToolDetailsModelInstalledLocalesAndPhases(t *testing.T) {
	for _, locale := range []struct {
		name, state, empty, rounds, reason, active, paused, blocked, complete, disarmed string
	}{
		{"en", "Status", "No goal", "Rounds", "Blocker", "Active", "Paused", "Blocked", "Completed", "Awaiting continuation"},
		{"zh-CN", "状态", "没有目标", "执行轮次", "受阻原因", "进行中", "已暂停", "受阻", "已完成", "等待继续"},
	} {
		t.Run(locale.name, func(t *testing.T) {
			toolDetailsModelLocaleForTest(t, locale.name)
			for _, activation := range []string{"armed", "disarmed"} {
				for _, phase := range []struct{ name, label string }{{"active", locale.active}, {"paused", locale.paused}, {"blocked", locale.blocked}, {"complete", locale.complete}} {
					goal := toolDetailsModelGoalFieldForTest("phase", toolDetailsModelJSONForTest(t, phase.name))
					goal = strings.TrimSuffix(goal, "}") + `,"blockedReason":{"code":" code ","message":" reason "}}`
					value := `{"goal":` + goal + `,"activation":` + toolDetailsModelJSONForTest(t, activation) + `}`
					label := phase.label
					if phase.name == "active" && activation == "disarmed" {
						label = locale.disarmed
					}
					want := &toolDetailsModel{Items: []toolDetailItem{{Title: " objective ", Fields: []toolDetailField{
						{Label: locale.state, Value: label}, {Label: locale.rounds, Value: "2 / 10"}, {Label: locale.reason, Value: " reason "},
					}}}}
					if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, value)); !reflect.DeepEqual(got, want) {
						t.Fatalf("%s/%s = %#v, want %#v", phase.name, activation, got, want)
					}
				}
			}
			for _, suffix := range []string{"", `,"activation":null`, `,"activation":false`, `,"activation":{}`, `,"activation":"unknown"`} {
				call := toolDetailsModelCallForTest(t, false, "get_goal", `{}`, `{"goal":null`+suffix+`}`)
				want := &toolDetailsModel{Items: []toolDetailItem{}, Empty: locale.empty}
				if got := toolDetailsModelFor(call); !reflect.DeepEqual(got, want) {
					t.Fatalf("empty%s = %#v, want %#v", suffix, got, want)
				}
			}
		})
	}
}

func TestToolDetailsModelStrictGoalRecord(t *testing.T) {
	for _, value := range []string{"", "{", "null", "[]", "true", "1", `"value"`, `{}`, `{"Goal":null}`, `{"goal":null} {}`, `{"goal":null,}`, "\ufeff{\"goal\":null}"} {
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, value)); got != nil {
			t.Fatalf("invalid whole JSON result %q produced %#v", value, got)
		}
	}
	for _, field := range []string{"id", "objective", "revision", "roundsStarted", "maxGoalRounds", "phase"} {
		goal := map[string]json.RawMessage{}
		if err := json.Unmarshal([]byte(toolDetailsModelGoalForTest), &goal); err != nil {
			t.Fatal(err)
		}
		delete(goal, field)
		value := toolDetailsModelValueForTest(toolDetailsModelJSONForTest(t, goal))
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "create_goal", `{}`, value)); got != nil {
			t.Fatalf("missing %s produced %#v", field, got)
		}
	}
	for _, field := range []string{"id", "objective"} {
		for _, raw := range []string{`null`, `true`, `1`, `[]`, `{}`, `""`, `" \t\r\n\u00a0\ufeff"`} {
			value := toolDetailsModelValueForTest(toolDetailsModelGoalFieldForTest(field, raw))
			if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "update_goal", `{}`, value)); got != nil {
				t.Fatalf("%s=%s produced %#v", field, raw, got)
			}
		}
	}
	for _, raw := range []string{`null`, `true`, `1`, `[]`, `{}`, `""`, `"ACTIVE"`, `"completed"`, `"active "`} {
		value := toolDetailsModelValueForTest(toolDetailsModelGoalFieldForTest("phase", raw))
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, value)); got != nil {
			t.Fatalf("phase=%s produced %#v", raw, got)
		}
	}
	for _, activation := range []string{"", `,"activation":null`, `,"activation":true`, `,"activation":1`, `,"activation":[]`, `,"activation":{}`, `,"activation":""`, `,"activation":"Armed"`, `,"activation":"disarmed "`} {
		value := `{"goal":` + toolDetailsModelGoalForTest + activation + `}`
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, value)); got != nil {
			t.Fatalf("invalid activation %s produced %#v", activation, got)
		}
	}
	for _, raw := range []string{`null`, `true`, `1`, `[]`, `{}`, `"goal"`} {
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, toolDetailsModelValueForTest(raw))); (got != nil) != (raw == `null`) {
			t.Fatalf("goal=%s wrong model: %#v", raw, got)
		}
	}
}

func TestToolDetailsModelSafeIntegerCounts(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	for _, tc := range []struct{ raw, want string }{
		{"0", "0"}, {"-0", "0"}, {"1e1", "10"}, {"10.0", "10"}, {"1.99999999999999999", "2"},
		{"9007199254740991", "9007199254740991"}, {"9007199254740991.1", "9007199254740991"},
		{"1e-9999", "0"}, {"-1e-9999", "0"},
	} {
		for _, field := range []string{"revision", "roundsStarted", "maxGoalRounds"} {
			value := toolDetailsModelValueForTest(toolDetailsModelGoalFieldForTest(field, tc.raw))
			got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, value))
			rounds := "2 / 10"
			if field == "roundsStarted" {
				rounds = tc.want + " / 10"
			} else if field == "maxGoalRounds" {
				rounds = "2 / " + tc.want
			}
			if got == nil || got.Items[0].Fields[1].Value != rounds {
				t.Fatalf("%s=%s produced %#v, want rounds %q", field, tc.raw, got, rounds)
			}
		}
	}
	for _, raw := range []string{`null`, `true`, `false`, `"0"`, `[]`, `{}`, "-1", "-0.5", "0.5", "1e-300", "-1e-300", "9007199254740991.5", "9007199254740992", "9007199254740993", "1e300", "1e9999", "-1e9999"} {
		for _, field := range []string{"revision", "roundsStarted", "maxGoalRounds"} {
			value := toolDetailsModelValueForTest(toolDetailsModelGoalFieldForTest(field, raw))
			if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, value)); got != nil {
				t.Fatalf("invalid safe count %s=%s produced %#v", field, raw, got)
			}
		}
	}
}

func TestToolDetailsModelBlockerAndJavaScriptTrim(t *testing.T) {
	for _, reason := range []string{`null`, `true`, `1`, `[]`, `"reason"`, `{}`, `{"code":"x"}`, `{"message":"x"}`} {
		value := toolDetailsModelValueForTest(toolDetailsModelGoalFieldForTest("blockedReason", reason))
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, value)); got != nil {
			t.Fatalf("invalid blockedReason=%s produced %#v", reason, got)
		}
	}
	for _, field := range []string{"code", "message"} {
		for _, raw := range []string{`null`, `true`, `1`, `[]`, `{}`, `""`, `" \u00a0\ufeff"`} {
			reason := `{"code":"x","message":" x ","` + field + `":` + raw + `}`
			value := toolDetailsModelValueForTest(toolDetailsModelGoalFieldForTest("blockedReason", reason))
			if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, value)); got != nil {
				t.Fatalf("blockedReason.%s=%s produced %#v", field, raw, got)
			}
		}
	}
	whitespace := "\t\n\v\f\r \u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000\ufeff"
	for _, text := range []string{whitespace, whitespace + "x" + whitespace, "\u0085", "\u180e", "\u200b", "\x00"} {
		valid := text != whitespace
		for _, field := range []string{"id", "objective", "code", "message"} {
			goal := toolDetailsModelGoalFieldForTest(field, toolDetailsModelJSONForTest(t, text))
			if field == "code" || field == "message" {
				reason := `{"code":"x","message":" x ","` + field + `":` + toolDetailsModelJSONForTest(t, text) + `}`
				goal = toolDetailsModelGoalFieldForTest("blockedReason", reason)
			}
			got := toolDetailsModelFor(toolDetailsModelCallForTest(t, true, "get_goal", `{}`, toolDetailsModelValueForTest(goal)))
			if (got != nil) != valid {
				t.Fatalf("JS trim field=%s value=%q valid=%v produced %#v", field, text, valid, got)
			}
			if valid && field == "objective" && got.Items[0].Title != text {
				t.Fatal("objective was trimmed")
			}
			if valid && field == "message" && got.Items[0].Fields[2].Value != text {
				t.Fatal("reason message was trimmed")
			}
		}
	}
}

func TestToolDetailsModelCallAndContentGates(t *testing.T) {
	valid := toolDetailsModelValueForTest(toolDetailsModelGoalForTest)
	for _, nested := range []bool{false, true} {
		for _, name := range []string{"", "Create_goal", "get_goal ", "functions.get_goal", "plugin:get_goal", "schedule_create", "schedule_list", "job_list", "cordis_inspect_self ", "todo_write", "unknown"} {
			if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, nested, name, `{}`, valid)); got != nil {
				t.Fatalf("unsupported exact name %q produced %#v", name, got)
			}
		}
		for _, args := range []string{"null", "[]", `"args"`, "true", "1"} {
			if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, nested, "get_goal", args, valid)); got != nil {
				t.Fatalf("nonrecord arguments %s produced %#v", args, got)
			}
		}
		for _, content := range []string{"null", "[]", "{}", `"content"`, "[null]", "[[]]", "[1]", "[{}]", `[{"type":"image"}]`, `[{"type":"text"}]`, `[{"type":"text","text":null}]`, `[{"type":"text","text":true}]`, `[{"type":"text","text":{"goal":null}}]`, `[{"type":"text","text":"{\"goal\":null}"},{"type":"text","text":""}]`} {
			if got := toolDetailsModelFor(toolDetailsModelContentCallForTest(t, nested, "get_goal", `{}`, content)); got != nil {
				t.Fatalf("invalid single text content %s produced %#v", content, got)
			}
		}
		for _, state := range []string{"", "preparing", "running", "error", "stopped", "cancelled"} {
			call := toolDetailsModelCallForTest(t, nested, "get_goal", `{}`, valid)
			call.State = state
			if got := toolDetailsModelFor(call); got != nil {
				t.Fatalf("non-ok projected state %q produced %#v", state, got)
			}
		}
		for _, marker := range []string{`"isError":true,`, `"error":{},`, `"error":{"code":"interrupted"},`, `"error":false,`} {
			call := toolDetailsModelCallForTest(t, nested, "get_goal", `{}`, valid)
			call.Result.Raw = json.RawMessage(strings.Replace(string(call.Result.Raw), `"data":{`, `"data":{`+marker, 1))
			if got := toolDetailsModelFor(call); got != nil {
				t.Fatalf("raw error marker %s produced %#v", marker, got)
			}
		}
	}
	for _, args := range []string{"", "{", "{} {}", "{},", "\ufeff{}"} {
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "get_goal", args, valid)); got != nil {
			t.Fatalf("invalid whole root arguments %q produced %#v", args, got)
		}
	}
	if toolDetailsModelFor(nil) != nil {
		t.Fatal("nil call produced model")
	}
	call := toolDetailsModelCallForTest(t, false, "get_goal", `{}`, valid)
	call.Result = nil
	if toolDetailsModelFor(call) != nil {
		t.Fatal("unsettled head produced model")
	}
}

func TestToolDetailsModelRawEnvelopeAndOwnership(t *testing.T) {
	valid := toolDetailsModelValueForTest(toolDetailsModelGoalForTest)
	for _, nested := range []bool{false, true} {
		for _, raw := range []string{"", "{", "null", "[]", `{}`, `{"type":"tool/result","data":null}`, `{"type":"tool/result","data":{}}`, `{"type":"tool/result","data":{"message":[]}}`, `{"type":"tool/ptc-dispatch-start","data":{"content":[]}}`} {
			call := toolDetailsModelCallForTest(t, nested, "get_goal", `{}`, valid)
			call.Result.Raw = json.RawMessage(raw)
			if got := toolDetailsModelFor(call); got != nil {
				t.Fatalf("invalid receipt %q produced %#v", raw, got)
			}
		}
		call := toolDetailsModelCallForTest(t, nested, "get_goal", `{}`, valid)
		call.Result.EventType = "tool/call"
		if toolDetailsModelFor(call) != nil {
			t.Fatal("raw/EventType mismatch produced model")
		}
		call.Result.EventType = ""
		if toolDetailsModelFor(call) == nil {
			t.Fatal("absent EventType did not use raw receipt type")
		}
		call.Name, call.Arguments = "get_goal", `{}`
		call.ParentCallID, call.RootCallID = "unrelated", "projection"
		if nested {
			call.Call = nil
		} else {
			call.Call.EventType = ""
		}
		if toolDetailsModelFor(call) == nil {
			t.Fatal("valid raw call was replaced by projected presentation")
		}
		owner := call.Call
		if nested {
			owner = call.Result
		}
		owner.Raw = json.RawMessage(strings.Replace(string(owner.Raw), `"name":"get_goal"`, `"name":"other"`, 1))
		if toolDetailsModelFor(call) != nil {
			t.Fatal("projected name granted a model to another raw tool")
		}
	}
	for _, raw := range []string{"", "{", "null", "[]", `{}`, `{"type":"tool/ptc-dispatch-start","data":{}}`, `{"type":"tool/call","data":null}`, `{"type":"tool/call","data":{"name":"get_goal","arguments":{}}}`} {
		call := toolDetailsModelCallForTest(t, false, "get_goal", `{}`, valid)
		call.Call.Raw = json.RawMessage(raw)
		if got := toolDetailsModelFor(call); got != nil {
			t.Fatalf("invalid raw retained head %q produced %#v", raw, got)
		}
	}
	call := toolDetailsModelCallForTest(t, false, "get_goal", `{}`, valid)
	call.Call.EventType = "tool/ptc-dispatch-start"
	if toolDetailsModelFor(call) != nil {
		t.Fatal("head EventType/raw mismatch produced model")
	}
	call = toolDetailsModelCallForTest(t, false, "get_goal", `{}`, valid)
	call.Result.Raw = json.RawMessage(strings.Replace(string(call.Result.Raw), `"message":{`, `"message":{"isError":true,`, 1))
	if toolDetailsModelFor(call) != nil {
		t.Fatal("raw root message error produced model")
	}
}

func TestToolDetailsModelActualProjectionAndIndependentModels(t *testing.T) {
	valid := toolDetailsModelValueForTest(toolDetailsModelGoalFieldForTest("blockedReason", `{"code":"x","message":" reason "}`))
	for _, nested := range []bool{false, true} {
		call := toolDetailsModelCallForTest(t, nested, "get_goal", `{}`, valid)
		if nested {
			// A previous child start can disagree; installed childResult owns its
			// embedded settled name/arguments, while the Go projection keeps start.
			head := toolDetailsModelCallForTest(t, true, "other", `[]`, valid)
			call.Call = head.Call
		}
		messages := []session.Message{*call.Call, *call.Result}
		before := toolDetailsModelJSONForTest(t, messages)
		for _, window := range []struct {
			name string
			raw  []session.Message
			want bool
		}{{"paired", messages, true}, {"result only", messages[1:], nested}, {"head only", messages[:1], false}} {
			t.Run(toolDetailsModelJSONForTest(t, []any{nested, window.name}), func(t *testing.T) {
				projected := session.ProjectToolMessages(window.raw)
				if len(projected) != 1 || projected[0].Tool == nil {
					t.Fatalf("wire records did not project: %#v", projected)
				}
				view := projected[0].Tool
				got := toolDetailsModelFor(view)
				if (got != nil) != window.want {
					t.Fatalf("retained window want=%v, model=%#v", window.want, got)
				}
				if got == nil {
					return
				}
				sourceBefore := toolDetailsModelJSONForTest(t, view)
				second := toolDetailsModelFor(view)
				secondBefore := toolDetailsModelJSONForTest(t, second)
				got.Empty, got.Items[0].Title = "changed", "changed"
				got.Items[0].Fields[0].Label, got.Items[0].Fields[2].Value = "changed", "changed"
				got.Items[0].Fields = append(got.Items[0].Fields, toolDetailField{Value: "appended"})
				got.Items = append(got.Items, toolDetailItem{Title: "appended"})
				if secondBefore != toolDetailsModelJSONForTest(t, second) || !reflect.DeepEqual(second, toolDetailsModelFor(view)) {
					t.Fatal("models share mutable item or field slices")
				}
				if sourceBefore != toolDetailsModelJSONForTest(t, view) {
					t.Fatal("derivation or model mutation changed retained call/events")
				}
			})
		}
		if before != toolDetailsModelJSONForTest(t, messages) {
			t.Fatal("projection or model derivation mutated source messages")
		}
	}
}

const toolDetailsModelInstalledGoldensForTest = `[{"name":"create root","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"get root","tool":"get_goal","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"update root","tool":"update_goal","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"empty ignores activation","valueRaw":"{\"goal\":null,\"activation\":{},\"unrelated\":1e9999}","want":{"items":[],"empty":"No goal"}},{"name":"empty without activation","valueRaw":"{\"goal\":null}","want":{"items":[],"empty":"No goal"}},{"name":"empty nested result only","nested":true,"resultOnly":true,"valueRaw":"{\"goal\":null}","want":{"items":[],"empty":"No goal"}},{"name":"active disarmed","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\"},\"activation\":\"disarmed\"}","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Awaiting continuation"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"paused disarmed","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"phase\":\"paused\"},\"activation\":\"disarmed\"}","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Paused"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"blocked disarmed","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"phase\":\"blocked\"},\"activation\":\"disarmed\"}","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Blocked"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"complete disarmed","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"phase\":\"complete\"},\"activation\":\"disarmed\"}","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Completed"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"reason exact text","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"blockedReason\":{\"code\":\" code \",\"message\":\" \\tblocked\\r\\n \",\"ignored\":1e9999}},\"activation\":\"armed\"}","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"2 / 10"},{"label":"Blocker","value":" \tblocked\r\n "}]}]}},{"name":"zero maximum with greater rounds","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"maxGoalRounds\":0},\"activation\":\"armed\"}","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"2 / 0"}]}]}},{"name":"safe counts and rounded fraction","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"revision\":9007199254740991,\"roundsStarted\":9007199254740991.1,\"maxGoalRounds\":1.99999999999999999},\"activation\":\"armed\"}","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"9007199254740991 / 2"}]}]}},{"name":"negative zero underflow","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"revision\":-0,\"roundsStarted\":-1e-9999,\"maxGoalRounds\":1e-9999},\"activation\":\"armed\"}","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"0 / 0"}]}]}},{"name":"duplicate last and ignored overflow","argsRaw":"{\"x\":null,\"x\":1e9999}","valueRaw":"{\"goal\":false,\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"objective\":null,\"objective\":\" last \",\"ignored\":1e9999},\"activation\":null,\"activation\":\"armed\",\"ignored\":1e9999}","want":{"items":[{"title":" last ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"NEL visible","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"objective\":\"\\u0085\"},\"activation\":\"armed\"}","want":{"items":[{"title":"\u0085","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"nested result only","nested":true,"resultOnly":true,"want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"nested result owns call","nested":true,"headTool":"wrong","headArgsRaw":"[]","want":{"items":[{"title":" objective ","fields":[{"label":"Status","value":"Active"},{"label":"Rounds","value":"2 / 10"}]}]}},{"name":"root result without head","resultOnly":true,"want":null},{"name":"nested embedded name is exact","nested":true,"tool":"functions.get_goal","want":null},{"name":"BOM blank objective","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"objective\":\" \\u00a0\\ufeff\"},\"activation\":\"armed\"}","want":null},{"name":"null reason declined","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"blockedReason\":null},\"activation\":\"armed\"}","want":null},{"name":"unsafe rounded count declined","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"roundsStarted\":9007199254740993},\"activation\":\"armed\"}","want":null},{"name":"boolean count declined","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"revision\":true},\"activation\":\"armed\"}","want":null},{"name":"null count declined","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\",\"maxGoalRounds\":null},\"activation\":\"armed\"}","want":null},{"name":"root args must be record","argsRaw":"[]","want":null},{"name":"root entire args required","argsRaw":"{} {}","want":null},{"name":"nested args must be record","nested":true,"argsRaw":"[]","want":null},{"name":"single text only","contentRaw":"[{\"type\":\"text\",\"text\":\"{\\\"goal\\\":null}\"},{\"type\":\"text\",\"text\":\"\"}]","want":null},{"name":"text type exact","contentRaw":"[{\"type\":\"Text\",\"text\":\"{\\\"goal\\\":null}\"}]","want":null},{"name":"whole result JSON only","valueRaw":"{\"goal\":{\"id\":\"g\",\"objective\":\" objective \",\"revision\":1,\"roundsStarted\":2,\"maxGoalRounds\":10,\"phase\":\"active\"},\"activation\":\"armed\"} {}","want":null},{"name":"error result","isError":true,"want":null},{"name":"unknown tool","tool":"get_goal ","want":null},{"name":"no goal","valueRaw":"{}","want":null},{"name":"nonrecord result","valueRaw":"[]","want":null}]`

// Node 24 goldens for the installed Intl.DateTimeFormat shapes. The 2025-2026
// instants were recorded with TZ=Asia/Shanghai; the year-0 and 0099 instants use
// TZ=Etc/GMT-8 because Asia/Shanghai's pre-1901 LMT (+08:05:43) would shift them,
// while these tests install a fixed +08:00 zone. Modern instants are identical
// under both zones.
func toolDetailsModelScheduleZoneForTest(t *testing.T) {
	t.Helper()
	previous := time.Local
	time.Local = time.FixedZone("GMT+8", 8*60*60)
	t.Cleanup(func() { time.Local = previous })
}

func toolDetailsModelScheduleRecordForTest(extra string) string {
	return `{"id":" s1 ","prompt":" prompt ","deliveryMode":"host","state":"scheduled","scheduledAt":"2026-10-24T07:59:30.000Z"` + extra + `}`
}

func TestToolDetailsModelScheduleKinds(t *testing.T) {
	toolDetailsModelScheduleZoneForTest(t)
	cases := []struct{ name, extra, en, zh, enState, zhState string }{
		{"at", `,"kind":"at"`, "Once", "单次", "Scheduled", "等待触发"},
		{"after", `,"kind":"after","afterSeconds":90`, "Once", "单次", "Scheduled", "等待触发"},
		{"after overdue", `,"kind":"after","afterSeconds":1,"state":"overdue"`, "Once", "单次", "Overdue, awaiting session resume", "已到期，等待会话恢复"},
		{"every days", `,"kind":"every","everySeconds":172800`, "Every 2 d", "每 2 天", "Scheduled", "等待触发"},
		{"every hours", `,"kind":"every","everySeconds":7200`, "Every 2 h", "每 2 小时", "Scheduled", "等待触发"},
		{"every minutes", `,"kind":"every","everySeconds":420`, "Every 7 min", "每 7 分钟", "Scheduled", "等待触发"},
		{"every seconds", `,"kind":"every","everySeconds":59`, "Every 59 s", "每 59 秒", "Scheduled", "等待触发"},
		{"daily", `,"kind":"daily","time":"07:05:00.000","timeZone":"Asia/Shanghai"`, "Every day at 07:05 (Asia/Shanghai)", "每天 07:05（Asia/Shanghai）", "Scheduled", "等待触发"},
		{"daily keeps plain seconds", `,"kind":"daily","time":"07:05:00","timeZone":"UTC"`, "Every day at 07:05:00 (UTC)", "每天 07:05:00（UTC）", "Scheduled", "等待触发"},
		{"daily strips bare millis", `,"kind":"daily","time":"07:05.000","timeZone":"UTC"`, "Every day at 07:05 (UTC)", "每天 07:05（UTC）", "Scheduled", "等待触发"},
		{"weekly", `,"kind":"weekly","time":"09:05:00.000","timeZone":"Asia/Shanghai","weekdays":[1,3,5]`, "Weekly on Mon, Wed, Fri at 09:05 (Asia/Shanghai)", "每周 周一、周三、周五 09:05（Asia/Shanghai）", "Scheduled", "等待触发"},
		{"weekly duplicate day", `,"kind":"weekly","time":"09:05:00.000","timeZone":"UTC","weekdays":[7,7]`, "Weekly on Sun, Sun at 09:05 (UTC)", "每周 周日、周日 09:05（UTC）", "Scheduled", "等待触发"},
		{"cron", `,"kind":"cron","expression":"*/5 * * * *","timeZone":"UTC"`, "Cron */5 * * * * (UTC)", "Cron */5 * * * *（UTC）", "Scheduled", "等待触发"},
	}
	for _, locale := range []struct{ name, when, frequency, state, date string }{
		{"en", "Scheduled for", "Repeat", "Status", "Oct 24, 2026, 03:59:30 PM GMT+8"},
		{"zh-CN", "计划时间", "重复", "状态", "2026年10月24日 GMT+8 15:59:30"},
	} {
		toolDetailsModelLocaleForTest(t, locale.name)
		for _, tc := range cases {
			frequency, state := tc.en, tc.enState
			if locale.name != i18n.LocaleEN {
				frequency, state = tc.zh, tc.zhState
			}
			want := &toolDetailsModel{Items: []toolDetailItem{{Title: " prompt ", Fields: []toolDetailField{
				{locale.when, locale.date}, {locale.frequency, frequency}, {locale.state, state},
			}}}}
			got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_create", `{}`, toolDetailsModelScheduleRecordForTest(tc.extra)))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s/%s = %#v, want %#v", locale.name, tc.name, got, want)
			}
		}
	}
	value := toolDetailsModelScheduleRecordForTest(`,"kind":"at"`)
	if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_update", `{}`, value)); got == nil || got.Summary != "" || got.Empty != "" || len(got.Items) != 1 {
		t.Fatalf("schedule_update = %#v", got)
	}
}

func TestToolDetailsModelScheduleDateText(t *testing.T) {
	toolDetailsModelScheduleZoneForTest(t)
	goldens := []struct{ instant, en, zh string }{
		{"2026-10-24T07:59:30.000Z", "Oct 24, 2026, 03:59:30 PM GMT+8", "2026年10月24日 GMT+8 15:59:30"},
		{"2026-10-23T16:00:00.000Z", "Oct 24, 2026, 12:00:00 AM GMT+8", "2026年10月24日 GMT+8 00:00:00"},
		{"2026-10-24T04:00:00.000Z", "Oct 24, 2026, 12:00:00 PM GMT+8", "2026年10月24日 GMT+8 12:00:00"},
		{"2026-06-15T12:00:00.000Z", "Jun 15, 2026, 08:00:00 PM GMT+8", "2026年6月15日 GMT+8 20:00:00"},
		{"2026-01-01T00:30:05.000Z", "Jan 1, 2026, 08:30:05 AM GMT+8", "2026年1月1日 GMT+8 08:30:05"},
		{"0000-02-03T04:05:06.000Z", "Feb 3, 1, 12:05:06 PM GMT+8", "1年2月3日 GMT+8 12:05:06"},
		{"0099-12-31T23:59:59.999Z", "Jan 1, 100, 07:59:59 AM GMT+8", "100年1月1日 GMT+8 07:59:59"},
	}
	for _, locale := range []struct {
		name string
		en   bool
	}{{"en", true}, {"zh-CN", false}} {
		toolDetailsModelLocaleForTest(t, locale.name)
		for _, golden := range goldens {
			value := `{"id":"s","prompt":"p","deliveryMode":"host","state":"scheduled","scheduledAt":` + toolDetailsModelJSONForTest(t, golden.instant) + `,"kind":"at"}`
			got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_create", `{}`, value))
			want := golden.zh
			if locale.en {
				want = golden.en
			}
			if got == nil || len(got.Items) != 1 || got.Items[0].Fields[0].Value != want {
				t.Fatalf("%s %s = %#v, want %q", locale.name, golden.instant, got, want)
			}
		}
	}
}

func TestToolDetailsModelScheduleRejects(t *testing.T) {
	toolDetailsModelScheduleZoneForTest(t)
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	reject := func(values ...string) {
		t.Helper()
		for _, value := range values {
			if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_create", `{}`, value)); got != nil {
				t.Fatalf("schedule_create %s produced %#v", value, got)
			}
		}
	}
	for _, at := range []string{
		`"2026-10-24"`, `"2026-10-24T07:59:30Z"`, `"2026-10-24T07:59:30.000+00:00"`,
		`"2026-10-24T07:59:30.000+08:00"`, `"2026-10-24T07:59:30.0000Z"`, `"2026-10-24T07:59:30.00Z"`,
		`"2026-10-24T07:59:30.000z"`, `"2026-10-24 07:59:30.000Z"`, `"2026-02-30T00:00:00.000Z"`,
		`"2026-10-24T24:00:00.000Z"`, `"2026-10-24T07:59:60.000Z"`, `"not a date"`, `""`,
		`null`, `1`, `true`, `[]`, `{}`,
	} {
		reject(`{"id":"s","prompt":"p","deliveryMode":"host","state":"scheduled","scheduledAt":` + at + `,"kind":"at"}`)
	}
	base := toolDetailsModelScheduleRecordForTest(`,"kind":"at"`)
	for _, field := range []string{`"deliveryMode":"local"`, `"deliveryMode":"HOST"`, `"deliveryMode":"host "`, `"deliveryMode":null`, `"deliveryMode":1`} {
		reject(strings.Replace(base, `"deliveryMode":"host"`, field, 1))
	}
	for _, field := range []string{`"state":"fired"`, `"state":"Scheduled"`, `"state":"scheduled "`, `"state":null`, `"state":1`} {
		reject(strings.Replace(base, `"state":"scheduled"`, field, 1))
	}
	reject(`{"prompt":"p","deliveryMode":"host","state":"scheduled","scheduledAt":"2026-10-24T07:59:30.000Z","kind":"at"}`,
		`{"id":"s","deliveryMode":"host","state":"scheduled","scheduledAt":"2026-10-24T07:59:30.000Z","kind":"at"}`,
		toolDetailsModelScheduleRecordForTest(`,"kind":"AT"`),
		toolDetailsModelScheduleRecordForTest(`,"kind":"later"`),
		toolDetailsModelScheduleRecordForTest(`,"kind":null`),
		toolDetailsModelScheduleRecordForTest(`,"kind":1`),
		toolDetailsModelScheduleRecordForTest(""),
		`{"id":"s","prompt":"p","deliveryMode":"host","state":"scheduled","scheduledAt":"2026-10-24T07:59:30.000Z","kind":"at"} {}`,
		`{}`, `[]`, `null`, `1`, `"x"`, `{`)
	for _, extra := range []string{
		`,"id":""`, `,"id":" "`, `,"id":null`, `,"id":1`,
		`,"prompt":""`, `,"prompt":" \t"`, `,"prompt":null`, `,"prompt":1`,
		`,"kind":"after","afterSeconds":0`, `,"kind":"after","afterSeconds":-0`, `,"kind":"after","afterSeconds":-1`,
		`,"kind":"after","afterSeconds":1.5`, `,"kind":"after","afterSeconds":"60"`, `,"kind":"after"`,
		`,"kind":"after","afterSeconds":null`, `,"kind":"after","afterSeconds":true`, `,"kind":"after","afterSeconds":[]`,
		`,"kind":"after","afterSeconds":9007199254740992`, `,"kind":"after","afterSeconds":9007199254740991.5`,
		`,"kind":"after","afterSeconds":1e9999`,
		`,"kind":"every","everySeconds":0`, `,"kind":"every","everySeconds":-60`, `,"kind":"every","everySeconds":2.5`,
		`,"kind":"every","everySeconds":"60"`, `,"kind":"every"`, `,"kind":"every","everySeconds":9007199254740992`,
	} {
		reject(toolDetailsModelScheduleRecordForTest(extra))
	}
	for _, days := range []string{`[]`, `[0]`, `[8]`, `[1.5]`, `["1"]`, `[true]`, `[null]`, `[[1]]`, `[1,8]`, `{}`, `null`, `1`, `"1"`} {
		reject(toolDetailsModelScheduleRecordForTest(`,"kind":"weekly","time":"09:05","timeZone":"UTC","weekdays":` + days))
	}
	reject(toolDetailsModelScheduleRecordForTest(`,"kind":"weekly","time":"09:05","timeZone":"UTC"`),
		toolDetailsModelScheduleRecordForTest(`,"kind":"weekly","time":" ","timeZone":"UTC","weekdays":[1]`),
		toolDetailsModelScheduleRecordForTest(`,"kind":"weekly","time":"09:05","timeZone":" ","weekdays":[1]`),
		toolDetailsModelScheduleRecordForTest(`,"kind":"daily","time":"  ","timeZone":"UTC"`),
		toolDetailsModelScheduleRecordForTest(`,"kind":"daily","time":"09:05","timeZone":" "`),
		toolDetailsModelScheduleRecordForTest(`,"kind":"daily"`),
		toolDetailsModelScheduleRecordForTest(`,"kind":"cron","expression":" ","timeZone":"UTC"`),
		toolDetailsModelScheduleRecordForTest(`,"kind":"cron","expression":"* * * * *","timeZone":" "`),
		toolDetailsModelScheduleRecordForTest(`,"kind":"cron","timeZone":"UTC"`))
}

func TestToolDetailsModelScheduleRecordSpelling(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	for _, tc := range []struct{ name, extra, want string }{
		{"title wins", `,"title":" raw title "`, " raw title "},
		{"blank title falls back", `,"title":" \t\n "`, " prompt "},
		{"null title falls back", `,"title":null`, " prompt "},
		{"numeric title falls back", `,"title":1`, " prompt "},
		{"duplicate title keeps last", `,"title":"a","title":"b"`, "b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := toolDetailsModelScheduleRecordForTest(`,"kind":"at"` + tc.extra)
			got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_create", `{}`, value))
			if got == nil || got.Items[0].Title != tc.want {
				t.Fatalf("title = %#v, want %q", got, tc.want)
			}
		})
	}
	value := `{"id":" id ","prompt":" prompt ","deliveryMode":"host","state":"scheduled","scheduledAt":"2026-10-24T07:59:30.000Z","kind":"at"}`
	if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_create", `{}`, value)); got == nil || got.Items[0].Title != " prompt " {
		t.Fatalf("retained prompt = %#v", got)
	}
}

func TestToolDetailsModelScheduleListAndDelete(t *testing.T) {
	toolDetailsModelScheduleZoneForTest(t)
	const entryA = `{"id":"a","prompt":" A ","deliveryMode":"host","state":"scheduled","scheduledAt":"2026-10-24T07:59:30.000Z","kind":"at"}`
	const entryB = `{"id":"b","title":" B ","prompt":" ignored ","deliveryMode":"session-local","state":"overdue","scheduledAt":"2026-10-23T16:00:00.000Z","kind":"after","afterSeconds":30}`
	for _, locale := range []struct{ name, count, countZero, empty, when, frequency, state, scheduled, overdue, deleted, once, dateA, dateB string }{
		{"en", "2 reminders", "0 reminders", "No reminders", "Scheduled for", "Repeat", "Status", "Scheduled", "Overdue, awaiting session resume", "Deleted", "Once", "Oct 24, 2026, 03:59:30 PM GMT+8", "Oct 24, 2026, 12:00:00 AM GMT+8"},
		{"zh-CN", "2 个定时任务", "0 个定时任务", "没有定时任务", "计划时间", "重复", "状态", "等待触发", "已到期，等待会话恢复", "已删除", "单次", "2026年10月24日 GMT+8 15:59:30", "2026年10月24日 GMT+8 00:00:00"},
	} {
		toolDetailsModelLocaleForTest(t, locale.name)
		want := &toolDetailsModel{Items: []toolDetailItem{
			{Title: " A ", Fields: []toolDetailField{{locale.when, locale.dateA}, {locale.frequency, locale.once}, {locale.state, locale.scheduled}}},
			{Title: " B ", Fields: []toolDetailField{{locale.when, locale.dateB}, {locale.frequency, locale.once}, {locale.state, locale.overdue}}},
		}, Summary: locale.count, Empty: locale.empty}
		got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_list", `{}`, `[`+entryA+`,`+entryB+`]`))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s list = %#v, want %#v", locale.name, got, want)
		}
		empty := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_list", `{}`, `[]`))
		if empty == nil || empty.Items == nil || len(empty.Items) != 0 || empty.Summary != locale.countZero || empty.Empty != locale.empty {
			t.Fatalf("%s empty list = %#v", locale.name, empty)
		}
		for _, value := range []string{`{}`, `null`, `1`, `"x"`, `true`, `[`, `[] {}`, `[1]`, `[{}]`, `[` + entryA + `,{}]`, `[null]`} {
			if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_list", `{}`, value)); got != nil {
				t.Fatalf("%s invalid list %s produced %#v", locale.name, value, got)
			}
		}
		deleted := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_delete", `{}`, `{"id":" s1 ","deleted":true}`))
		wantDeleted := &toolDetailsModel{Items: []toolDetailItem{{Title: " s1 ", Fields: []toolDetailField{{locale.state, locale.deleted}}}}}
		if !reflect.DeepEqual(deleted, wantDeleted) {
			t.Fatalf("%s delete = %#v, want %#v", locale.name, deleted, wantDeleted)
		}
		for _, value := range []string{
			`{"id":"s1","deleted":false}`, `{"id":"s1","deleted":"true"}`, `{"id":"s1","deleted":1}`,
			`{"id":"s1"}`, `{"deleted":true}`, `{"id":" ","deleted":true}`, `{"id":1,"deleted":true}`,
			`{"id":"s1","deleted":true} {}`, `[]`, `null`,
		} {
			if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "schedule_delete", `{}`, value)); got != nil {
				t.Fatalf("%s invalid delete %s produced %#v", locale.name, value, got)
			}
		}
	}
}

func TestToolDetailsModelScheduleProjectionAndIndependence(t *testing.T) {
	toolDetailsModelScheduleZoneForTest(t)
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	value := `[{"id":"a","prompt":" A ","deliveryMode":"host","state":"scheduled","scheduledAt":"2026-10-24T07:59:30.000Z","kind":"at"},{"id":"b","prompt":" B ","deliveryMode":"session-local","state":"overdue","scheduledAt":"2026-10-23T16:00:00.000Z","kind":"cron","expression":"0 9 * * *","timeZone":"UTC"}]`
	for _, nested := range []bool{false, true} {
		call := toolDetailsModelCallForTest(t, nested, "schedule_list", `{}`, value)
		if nested {
			head := toolDetailsModelCallForTest(t, true, "other", `[]`, value)
			call.Call = head.Call
		}
		messages := []session.Message{*call.Call, *call.Result}
		before := toolDetailsModelJSONForTest(t, messages)
		for _, window := range []struct {
			name string
			raw  []session.Message
			want bool
		}{{"paired", messages, true}, {"result only", messages[1:], nested}, {"head only", messages[:1], false}} {
			projected := session.ProjectToolMessages(window.raw)
			if len(projected) != 1 || projected[0].Tool == nil {
				t.Fatalf("nested=%v %s: wire records did not project", nested, window.name)
			}
			view := projected[0].Tool
			got := toolDetailsModelFor(view)
			if (got != nil) != window.want {
				t.Fatalf("nested=%v %s: model=%#v", nested, window.name, got)
			}
			if got == nil {
				continue
			}
			if got.Summary != "2 reminders" || len(got.Items) != 2 || got.Items[1].Title != " B " {
				t.Fatalf("nested=%v %s: projected model=%#v", nested, window.name, got)
			}
			sourceBefore := toolDetailsModelJSONForTest(t, view)
			second := toolDetailsModelFor(view)
			secondBefore := toolDetailsModelJSONForTest(t, second)
			got.Summary, got.Empty, got.Items[0].Title = "changed", "changed", "changed"
			got.Items[0].Fields[0].Label = "changed"
			got.Items[0].Fields = append(got.Items[0].Fields, toolDetailField{Value: "appended"})
			got.Items = append(got.Items, toolDetailItem{Title: "appended"})
			if secondBefore != toolDetailsModelJSONForTest(t, second) || !reflect.DeepEqual(second, toolDetailsModelFor(view)) {
				t.Fatal("models share mutable item or field slices")
			}
			if sourceBefore != toolDetailsModelJSONForTest(t, view) {
				t.Fatal("derivation or model mutation changed retained call/events")
			}
		}
		if before != toolDetailsModelJSONForTest(t, messages) {
			t.Fatal("projection or model derivation mutated source messages")
		}
	}
}

// The background-job families were read from the installed client bundles:
// control-details-model.js SHA256 ff46094f8712c459cadb12d0b1fb7544e63fd054a62f35d2ba008e4e641d595a
// detail-model-shared.js SHA256 bdc969d96403522b340da929252720a2080d42ddef861d5be61e7b99c7f13b76
func toolDetailsModelJobPtrForTest(value string) *string { return &value }

const toolDetailsModelJobLinesForTest = "job-1 [bash] running — npm test\njob-2 [agent] completed — review\njob-3 [bash] blocked — deploy"

func TestToolDetailsModelJobList(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	want := &toolDetailsModel{
		Items: []toolDetailItem{
			{Title: "npm test", Subtitle: toolDetailsModelJobPtrForTest("job-1"), Badge: &toolDetailBadge{Label: "Running", Tone: "info"}, Fields: []toolDetailField{{Label: "Type", Value: "bash"}}},
			{Title: "review", Subtitle: toolDetailsModelJobPtrForTest("job-2"), Badge: &toolDetailBadge{Label: "Completed", Tone: "success"}, Fields: []toolDetailField{{Label: "Type", Value: "agent"}}},
			{Title: "deploy", Subtitle: toolDetailsModelJobPtrForTest("job-3"), Badge: &toolDetailBadge{Label: "Blocked", Tone: "warning"}, Fields: []toolDetailField{{Label: "Type", Value: "bash"}}},
		},
		Summary: "3 background jobs",
		Caption: "Recorded result",
		Empty:   "No results",
	}
	call := toolDetailsModelCallForTest(t, false, "job_list", `{}`, toolDetailsModelJobLinesForTest)
	if got := toolDetailsModelFor(call); !reflect.DeepEqual(got, want) {
		t.Fatalf("job_list = %#v, want %#v", got, want)
	}
	// A single malformed line, an empty text and a JSON result all reject the
	// whole card so the generic evidence body remains.
	for _, text := range []string{
		"job-1 [bash] running — npm test\nbroken",
		"job-1 [bash] running — npm test\n",
		"",
		"job-1 running — npm test",
		"job-1 [bash] — npm test",
		`{"goal":null}`,
	} {
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_list", `{}`, text)); got != nil {
			t.Fatalf("job_list %q produced %#v", text, got)
		}
	}
	for _, locale := range []struct{ name, caption, empty, count, running, blocked, zero string }{
		{"en", "Recorded result", "No results", "3 background jobs", "Running", "Blocked", "0 background jobs"},
		{"zh-CN", "调用结果", "暂无结果", "3 个后台任务", "运行中", "受阻", "0 个后台任务"},
	} {
		toolDetailsModelLocaleForTest(t, locale.name)
		got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_list", `{}`, toolDetailsModelJobLinesForTest))
		if got == nil || got.Caption != locale.caption || got.Empty != locale.empty || got.Summary != locale.count ||
			got.Items[0].Badge.Label != locale.running || got.Items[2].Badge.Label != locale.blocked {
			t.Fatalf("%s job_list = %#v", locale.name, got)
		}
		empty := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_list", `{}`, "(no background jobs)"))
		if empty == nil || empty.Items == nil || len(empty.Items) != 0 || empty.Summary != locale.zero || empty.Caption != locale.caption {
			t.Fatalf("%s empty job_list = %#v", locale.name, empty)
		}
	}
}

func TestToolDetailsModelJobOutput(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	args := `{"target":"job-9"}`
	want := &toolDetailsModel{
		Items: []toolDetailItem{{
			Title: "job-9", Badge: &toolDetailBadge{Label: "Completed", Tone: "success"},
			Description: toolDetailsModelJobPtrForTest("exit 0"),
			Code:        &toolDetailCode{Text: "line one\nline two"},
		}},
		Summary: "job-9 · Completed", Caption: "Recorded result", Empty: "No results", ExpandedSummary: "job-9",
	}
	call := toolDetailsModelCallForTest(t, false, "job_output", args, "line one\nline two\n[status: completed, exit 0]")
	if got := toolDetailsModelFor(call); !reflect.DeepEqual(got, want) {
		t.Fatalf("job_output = %#v, want %#v", got, want)
	}
	// A status without its second part records no description at all.
	got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_output", args, "out\n[status: running]"))
	if got == nil || got.Items[0].Description != nil || got.Items[0].Badge.Tone != "info" || got.Summary != "job-9 · Running" || got.Items[0].Code.Text != "out" {
		t.Fatalf("job_output bare status = %#v", got)
	}
	// The truncation suffix leaves the code and adds its own description part.
	got = toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_output", args, "out\n[output truncated]\n[status: failed, boom]"))
	if got == nil || got.Items[0].Code.Text != "out" || got.Items[0].Description == nil ||
		*got.Items[0].Description != "boom · Output truncated" || got.Items[0].Badge.Tone != "error" {
		t.Fatalf("job_output truncated = %#v", got)
	}
	got = toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_output", args, "out\n[output truncated]\n[status: failed]"))
	if got == nil || got.Items[0].Description == nil || *got.Items[0].Description != "Output truncated" {
		t.Fatalf("job_output truncation only = %#v", got)
	}
	for _, text := range []string{"no status here", "out\n[status: completed] tail", "", "[status: ]", "out\n[status: completed, extra\n]"} {
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_output", args, text)); got != nil {
			t.Fatalf("job_output %q produced %#v", text, got)
		}
	}
	// The target falls back through the installed alias order and stays empty
	// when every alias is absent.
	for _, target := range []struct{ args, want string }{
		{`{"target":"t","agent_id":"a","sessionId":"s","job_id":"j"}`, "t"},
		{`{"target":"","agent_id":"a","sessionId":"s","job_id":"j"}`, "a"},
		{`{"target":1,"agent_id":"a","sessionId":"s","job_id":"j"}`, "a"},
		{`{"sessionId":"s","job_id":"j"}`, "s"},
		{`{"job_id":"j"}`, "j"},
		{`{}`, ""},
	} {
		got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_output", target.args, "out\n[status: completed]"))
		if got == nil || got.Items[0].Title != target.want || got.ExpandedSummary != target.want || got.Summary != target.want+" · Completed" {
			t.Fatalf("job_output target %s = %#v", target.args, got)
		}
	}
}

func TestToolDetailsModelJobKill(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	other := &toolDetailBadge{Label: "Already finished", Tone: "neutral"}
	for _, tc := range []struct {
		name, args, text string
		want             *toolDetailsModel
	}{
		{"cancel", `{"target":"job-5","reason":"user asked"}`, "requested cancellation of job job-5", &toolDetailsModel{
			Items: []toolDetailItem{{Title: "job-5", Badge: &toolDetailBadge{Label: "Cancellation requested", Tone: "warning"}, Description: toolDetailsModelJobPtrForTest("user asked")}},
			Summary: "job-5 · Cancellation requested", Caption: "Recorded result", Empty: "No results", ExpandedSummary: "job-5",
		}},
		{"cancel without reason", `{"target":"job-5"}`, "requested cancellation of job job-5", &toolDetailsModel{
			Items: []toolDetailItem{{Title: "job-5", Badge: &toolDetailBadge{Label: "Cancellation requested", Tone: "warning"}, Description: toolDetailsModelJobPtrForTest("")}},
			Summary: "job-5 · Cancellation requested", Caption: "Recorded result", Empty: "No results", ExpandedSummary: "job-5",
		}},
		{"already finished", `{"target":"job-5"}`, "job job-5 had already finished with exit code 0", &toolDetailsModel{
			Items:   []toolDetailItem{{Title: "job-5", Badge: other}},
			Summary: "job-5 · Already finished", Caption: "Recorded result", Empty: "No results", ExpandedSummary: "job-5",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_kill", tc.args, tc.text))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("job_kill = %#v, want %#v", got, tc.want)
			}
		})
	}
	for _, tc := range []struct{ args, text string }{
		{`{"target":"job-5"}`, "requested cancellation of job job-6"},
		{`{"target":"job-5"}`, "job job-5 was cancelled"},
		{`{"target":"job-6"}`, "requested cancellation of job job-5"},
		{`{"target":""}`, "requested cancellation of job"},
	} {
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_kill", tc.args, tc.text)); got != nil {
			t.Fatalf("job_kill %q/%s produced %#v", tc.text, tc.args, got)
		}
	}
}

// job_list/job_output/job_kill are control details: they derive from the settled
// receipt before the goal/reminder JSON fallback, and a nested receipt owns its
// embedded name and arguments. Arguments that are not a JSON record reject the
// card, as the installed parsedToolCall does.
func TestToolDetailsModelJobDerivation(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	for _, nested := range []bool{false, true} {
		call := toolDetailsModelCallForTest(t, nested, "job_output", `{"target":"job-7"}`, "out\n[status: completed]")
		model := toolDetailsModelFor(call)
		if model == nil || model.Items[0].Title != "job-7" {
			t.Fatalf("nested=%v model = %#v", nested, model)
		}
		messages := []session.Message{*call.Call, *call.Result}
		window := messages
		if nested {
			window = messages[1:]
		}
		projected := session.ProjectToolMessages(window)
		if len(projected) != 1 || projected[0].Tool == nil {
			t.Fatalf("nested=%v wire records did not project", nested)
		}
		current := toolDetailsModelFor(projected[0].Tool)
		if current == nil || !reflect.DeepEqual(current, model) {
			t.Fatalf("nested=%v projected model = %#v", nested, current)
		}
		// Mutating a derived model must not leak into a later derivation.
		second := toolDetailsModelFor(projected[0].Tool)
		second.Items[0].Badge.Label, second.Items[0].Code.Text = "changed", "changed"
		second.Items[0].Title, second.Items[0].Fields = "changed", append(second.Items[0].Fields, toolDetailField{Value: "appended"})
		second.Summary = "changed"
		if !reflect.DeepEqual(toolDetailsModelFor(projected[0].Tool), model) {
			t.Fatalf("nested=%v models share mutable state", nested)
		}
	}
	for _, args := range []string{"null", "[]", `"args"`, "{} {}", ""} {
		if got := toolDetailsModelFor(toolDetailsModelCallForTest(t, false, "job_list", args, toolDetailsModelJobLinesForTest)); got != nil {
			t.Fatalf("non-record arguments %q produced %#v", args, got)
		}
	}
}

// The generic ToolDetails vocabulary was read from the installed ui-tool bundle:
// ToolDetails.js renders lines, code and groups in that order and paints the
// unchanged container after the list; todo-diff-model.js omits that container
// when nothing is unchanged. The installed CSS (client.js:1507) fixes the
// geometry the view tests assert.
func TestToolDetailsUnchangedSuppressionAndLabel(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	if got := toolDetailsUnchangedFor(nil); got != nil {
		t.Fatalf("nil unchanged = %#v", got)
	}
	if got := toolDetailsUnchangedFor([]toolDetailItem{}); got != nil {
		t.Fatalf("empty unchanged = %#v", got)
	}
	items := []toolDetailItem{{Title: "Review"}, {Title: "Ship"}}
	for _, locale := range []struct{ name, want string }{
		{i18n.LocaleEN, "2 unchanged"},
		{i18n.LocaleZH, "2 项未变化"},
	} {
		toolDetailsModelLocaleForTest(t, locale.name)
		got := toolDetailsUnchangedFor(items)
		if got == nil || got.Label != locale.want || len(got.Items) != 2 || got.Items[1].Title != "Ship" {
			t.Fatalf("%s unchanged = %#v", locale.name, got)
		}
	}
	// The container keeps the recorded entries it was given.
	if &items[0] != &toolDetailsUnchangedFor(items).Items[0] {
		t.Fatal("unchanged container copied its entries")
	}
}

// The golden pins the whole vocabulary, including the nil-versus-empty fields
// the view branches on and the recorded code language.
func TestToolDetailsGenericEntryModelGolden(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	subtitle, description := "subtitle", " recorded description "
	model := &toolDetailsModel{
		Items: []toolDetailItem{{
			Title: "Workflow", Badge: &toolDetailBadge{Label: "Completed", Tone: "success"},
			Subtitle: &subtitle, Description: &description,
			Fields: []toolDetailField{{Label: "Agents", Value: "2"}},
			Lines:  []string{"first evidence", "second evidence"},
			Code:   &toolDetailCode{Text: "console.log(1)", Language: "javascript"},
			Groups: []toolDetailGroup{{
				Label: "Remaining work",
				Items: []toolDetailItem{{Fields: []toolDetailField{}, Lines: []string{"next step"}}},
			}},
		}},
		Summary: "Workflow", Caption: "Recorded result", Empty: "No results",
	}
	model.Unchanged = toolDetailsUnchangedFor([]toolDetailItem{
		{Title: "Review", Badge: &toolDetailBadge{Label: "Pending", Tone: "warning"}, Fields: []toolDetailField{}},
	})
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"Items":[{"Title":"Workflow","Fields":[{"Label":"Agents","Value":"2"}],"Badge":{"Label":"Completed","Tone":"success"},"Subtitle":"subtitle","Description":" recorded description ","Code":{"Text":"console.log(1)","Language":"javascript"},"Lines":["first evidence","second evidence"],"Groups":[{"Label":"Remaining work","Items":[{"Title":"","Fields":[],"Badge":null,"Subtitle":null,"Description":null,"Code":null,"Lines":["next step"],"Groups":null,"Markdown":null,"Location":null,"Status":null,"PreviousStatus":null,"Change":null}]}],"Markdown":null,"Location":null,"Status":null,"PreviousStatus":null,"Change":null}],"Empty":"No results","Summary":"Workflow","Caption":"Recorded result","ExpandedSummary":"","Unchanged":{"Label":"1 unchanged","Items":[{"Title":"Review","Fields":[],"Badge":{"Label":"Pending","Tone":"warning"},"Subtitle":null,"Description":null,"Code":null,"Lines":null,"Groups":null,"Markdown":null,"Location":null,"Status":null,"PreviousStatus":null,"Change":null}]}}`
	if string(raw) != want {
		t.Fatalf("generic entries = %s", raw)
	}
}

// The per-tool families shipped by round125 must not gain the generic parts, and
// an invalid recorded value still falls back to the evidence body.
func TestToolDetailsFamilyModelsStayOnTheirOwnEntries(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	reminder := `{"id":"s-1","prompt":"Recorded prompt","scheduledAt":"2026-10-25T01:30:00.000Z","deliveryMode":"host","state":"scheduled","kind":"at"}`
	models := map[string]*toolDetailsModel{}
	for name, call := range map[string]*session.ToolCallView{
		"job_output":    toolDetailsModelCallForTest(t, false, "job_output", `{"target":"job-1"}`, "out\n[status: completed]"),
		"job_list":      toolDetailsModelCallForTest(t, false, "job_list", `{}`, "job-1 [bash] running — npm test"),
		"goal":          toolDetailsModelCallForTest(t, false, "get_goal", `{}`, toolDetailsModelValueForTest(toolDetailsModelGoalForTest)),
		"schedule":      toolDetailsModelCallForTest(t, false, "schedule_create", `{}`, reminder),
		"schedule_list": toolDetailsModelCallForTest(t, false, "schedule_list", `{}`, `[`+reminder+`]`),
	} {
		models[name] = toolDetailsModelFor(call)
	}
	for name, model := range models {
		if model == nil || len(model.Items) == 0 {
			t.Fatalf("%s = %#v", name, model)
		}
		if model.Unchanged != nil {
			t.Fatalf("%s unchanged = %#v", name, model.Unchanged)
		}
		for _, item := range model.Items {
			if item.Lines != nil || item.Groups != nil {
				t.Fatalf("%s item gained generic entries: %#v", name, item)
			}
			if item.Code != nil && item.Code.Language != "" {
				t.Fatalf("%s code language = %q", name, item.Code.Language)
			}
		}
	}
	// Malformed recorded values keep the generic evidence body.
	for _, call := range []*session.ToolCallView{
		toolDetailsModelCallForTest(t, false, "job_output", `{}`, "no status here"),
		toolDetailsModelCallForTest(t, false, "schedule_create", `{}`, `{"id":"s-1"}`),
	} {
		if got := toolDetailsModelFor(call); got != nil {
			t.Fatalf("invalid record produced %#v", got)
		}
	}
}

// control-details-model.ts:22-41,113-160 and the harness expectation spec
// packages/client/ui-tool/tests/details-models.client.spec.ts:56-117 pin these.
func toolDetailsReceiptCallForTest(t *testing.T, name, args, text string) *toolDetailsModel {
	t.Helper()
	call := toolDetailsModelCallForTest(t, false, name, args, text)
	call.Name, call.Arguments = name, args
	return toolDetailsModelFor(call)
}

func TestToolDetailsModelAgentList(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	lines := "a1 [running] — Build\na2 [ready] parent=a1 depth=1 — Review"
	got := toolDetailsReceiptCallForTest(t, "list_agents", `{"scope":"descendants"}`, lines)
	if got == nil || got.Summary != "2 agents" || got.Caption != "Recorded result" || len(got.Items) != 2 {
		t.Fatalf("list_agents = %#v", got)
	}
	first, second := got.Items[0], got.Items[1]
	if first.Title != "Build" || first.Subtitle == nil || *first.Subtitle != "a1" || first.Badge.Label != "Running" || first.Badge.Tone != "info" || len(first.Fields) != 0 {
		t.Fatalf("first agent = %#v", first)
	}
	if second.Title != "Review" || second.Badge.Label != "Ready" || second.Badge.Tone != "neutral" {
		t.Fatalf("second agent = %#v", second)
	}
	if len(second.Fields) != 2 || second.Fields[0].Label != "Parent" || second.Fields[0].Value != "a1" || second.Fields[1].Value != "1" {
		t.Fatalf("parent fields = %#v", second.Fields)
	}
	// A member line without a title keeps the target as its heading and no subtitle.
	bare := toolDetailsReceiptCallForTest(t, "list_agents", `{}`, "a1 [failed]")
	if bare == nil || bare.Items[0].Title != "a1" || bare.Items[0].Subtitle != nil || bare.Items[0].Badge.Label != "Failed" {
		t.Fatalf("bare agent = %#v", bare)
	}
	// A JSON roster projects like any entity list; the empty text keeps a zero count.
	roster := toolDetailsReceiptCallForTest(t, "list_agents", `{}`, `[{"id":"a1","status":"running"},{"id":"a2","status":"failed"}]`)
	if roster == nil || roster.Summary != "2 agents" || roster.Items[0].Title != "a1" || roster.Items[1].Badge.Label != "Failed" {
		t.Fatalf("json roster = %#v", roster)
	}
	empty := toolDetailsReceiptCallForTest(t, "list_agents", `{}`, "(no subagents)")
	if empty == nil || empty.Summary != "0 agents" || empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("empty roster = %#v", empty)
	}
	for _, text := range []string{"", "a1 running — Build", "a1 [running] extra"} {
		if got := toolDetailsReceiptCallForTest(t, "list_agents", `{}`, text); got != nil {
			t.Fatalf("invalid roster %q produced %#v", text, got)
		}
	}
}

func TestToolDetailsModelSpawnAndTeamTasks(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	spawn := toolDetailsReceiptCallForTest(t, "spawn_teammate", `{"name":"A"}`, `{"member":{"id":"a1","status":"running"}}`)
	if spawn == nil || spawn.Summary != "A" || spawn.Items[0].Title != "a1" || spawn.Items[0].Badge.Label != "Running" {
		t.Fatalf("spawn = %#v", spawn)
	}
	// A real member row carries target/role/diagnostics, so the installed
	// result fallback names it.
	member := toolDetailsReceiptCallForTest(t, "spawn_teammate", `{"name":"A"}`, `{"member":{"target":"a1","role":"teammate","status":"provisioning","diagnostics":[]}}`)
	if member == nil || member.Items[0].Title != "Result" || member.Items[0].Badge.Label != "Provisioning" {
		t.Fatalf("member = %#v", member)
	}
	// `target` is not a known field name, so it keeps its spelling; role is.
	if member.Items[0].Fields[0].Label != "target" || member.Items[0].Fields[0].Value != "a1" ||
		member.Items[0].Fields[1].Label != "Role" || member.Items[0].Fields[1].Value != "teammate" || len(member.Items[0].Groups) != 0 {
		t.Fatalf("member fields = %#v", member.Items[0])
	}
	for _, text := range []string{`{}`, `{"member":[]}`, `{"member":"a1"}`, `[]`, `not json`} {
		if got := toolDetailsReceiptCallForTest(t, "spawn_teammate", `{"name":"A"}`, text); got != nil {
			t.Fatalf("invalid spawn %q produced %#v", text, got)
		}
	}

	task := toolDetailsReceiptCallForTest(t, "team_task_create", `{}`, `{"id":"task-1","subject":"Review","status":"pending"}`)
	if task == nil || task.Summary != "Review" || task.Items[0].Title != "Review" || task.Items[0].Badge.Label != "Pending" ||
		len(task.Items[0].Fields) != 1 || task.Items[0].Fields[0].Label != "ID" || task.Items[0].Fields[0].Value != "task-1" {
		t.Fatalf("team_task_create = %#v", task)
	}
	for _, name := range []string{"team_task_get", "team_task_update"} {
		if got := toolDetailsReceiptCallForTest(t, name, `{}`, `{"id":"t","subject":"Edit"}`); got == nil || got.Summary != "Edit" {
			t.Fatalf("%s = %#v", name, got)
		}
	}
	for _, text := range []string{`{"id":"t"}`, `{"subject":1}`, `[]`, `bad`} {
		if got := toolDetailsReceiptCallForTest(t, "team_task_create", `{}`, text); got != nil {
			t.Fatalf("invalid task %q produced %#v", text, got)
		}
	}

	list := toolDetailsReceiptCallForTest(t, "team_task_list", `{}`, `{"tasks":[{"id":"task-1","subject":"Review"}],"nextCursor":3}`)
	if list == nil || list.Summary != "1 team tasks" || list.Caption != "More tasks available; next cursor is 3" || list.Items[0].Title != "Review" {
		t.Fatalf("team_task_list = %#v", list)
	}
	noCursor := toolDetailsReceiptCallForTest(t, "team_task_list", `{}`, `{"tasks":[]}`)
	if noCursor == nil || noCursor.Summary != "0 team tasks" || noCursor.Caption != "Recorded result" || len(noCursor.Items) != 1 ||
		noCursor.Items[0].Description == nil || *noCursor.Items[0].Description != "No results" {
		t.Fatalf("empty task list = %#v", noCursor)
	}
	for _, text := range []string{`{"tasks":[],"nextCursor":"bad"}`, `{"tasks":{}}`, `[]`} {
		if got := toolDetailsReceiptCallForTest(t, "team_task_list", `{}`, text); got != nil {
			t.Fatalf("invalid task list %q produced %#v", text, got)
		}
	}
}

func TestToolDetailsModelAgentReceipts(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	queued := toolDetailsReceiptCallForTest(t, "send_message", `{"agent_id":"a1","message":"hello"}`, `{"status":"queued"}`)
	if queued == nil || queued.Summary != "a1 · Queued" || queued.Items[0].Badge.Tone != "warning" ||
		queued.Items[0].Description == nil || *queued.Items[0].Description != "hello" {
		t.Fatalf("queued message = %#v", queued)
	}
	accepted := toolDetailsReceiptCallForTest(t, "send_message", `{"agent_id":"a1","message":"hello"}`, `{"status":"accepted"}`)
	if accepted == nil || accepted.Summary != "a1 · Message delivered" || accepted.Items[0].Badge.Tone != "success" {
		t.Fatalf("accepted message = %#v", accepted)
	}
	delivered := toolDetailsReceiptCallForTest(t, "send_message", `{"agent_id":"a1","message":"hello"}`, "message delivered to agent a1")
	if delivered == nil || delivered.Summary != "a1 · Message delivered" || *delivered.Items[0].Description != "hello" {
		t.Fatalf("delivered message = %#v", delivered)
	}
	for _, tc := range []struct{ args, text string }{
		{`{"agent_id":"a1"}`, "message delivered to agent a2"},
		{`{"agent_id":"a1"}`, `{"status":"sent"}`},
		{`{}`, "delivered"},
	} {
		if got := toolDetailsReceiptCallForTest(t, "send_message", tc.args, tc.text); got != nil {
			t.Fatalf("invalid message %q/%s produced %#v", tc.text, tc.args, got)
		}
	}

	interrupt := toolDetailsReceiptCallForTest(t, "interrupt_agent", `{"agent_id":"a1"}`, `{"previousStatus":"running"}`)
	if interrupt == nil || interrupt.Summary != "a1 · Interrupt requested" || interrupt.Items[0].Badge.Tone != "warning" ||
		len(interrupt.Items[0].Fields) != 1 || interrupt.Items[0].Fields[0].Label != "Previous status" || interrupt.Items[0].Fields[0].Value != "Running" {
		t.Fatalf("interrupt = %#v", interrupt)
	}
	plain := toolDetailsReceiptCallForTest(t, "interrupt_agent", `{"agent_id":"a1"}`, "interrupt requested for agent a1")
	if plain == nil || plain.Summary != "a1 · Interrupt requested" || len(plain.Items[0].Fields) != 0 {
		t.Fatalf("plain interrupt = %#v", plain)
	}
	if got := toolDetailsReceiptCallForTest(t, "interrupt_agent", `{"agent_id":"a1"}`, "unexpected"); got != nil {
		t.Fatalf("invalid interrupt produced %#v", got)
	}

	timeout := toolDetailsReceiptCallForTest(t, "wait_agent", `{}`, `{"timedOut":true}`)
	if timeout == nil || timeout.Summary != "Subagent activity · Wait timed out" || timeout.Items[0].Title != "Subagent activity" || timeout.Items[0].Badge.Tone != "neutral" {
		t.Fatalf("wait timeout = %#v", timeout)
	}
	changed := toolDetailsReceiptCallForTest(t, "wait_agent", `{}`, `{"timedOut":false}`)
	if changed == nil || changed.Summary != "Subagent activity · Change detected" {
		t.Fatalf("wait changed = %#v", changed)
	}
	stalled := toolDetailsReceiptCallForTest(t, "wait_agent", `{}`, `{"timedOut":false,"noProgress":{"message":"No peer"}}`)
	if stalled == nil || stalled.Summary != "No active subagents" || stalled.Items[0].Title != "No active subagents" ||
		stalled.Items[0].Description == nil || *stalled.Items[0].Description != "No peer" {
		t.Fatalf("wait stalled = %#v", stalled)
	}
	for _, text := range []string{`{"timedOut":"yes"}`, `{}`, `[]`} {
		if got := toolDetailsReceiptCallForTest(t, "wait_agent", `{}`, text); got != nil {
			t.Fatalf("invalid wait %q produced %#v", text, got)
		}
	}
}

func TestToolDetailsModelSubagent(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	job := toolDetailsReceiptCallForTest(t, "subagent", `{"prompt":"Do work"}`, "started background subagent job job-1")
	if job == nil || job.Summary != "Do work · Started" || job.Items[0].Badge.Tone != "info" ||
		len(job.Items[0].Fields) != 1 || job.Items[0].Fields[0].Label != "Job ID" || job.Items[0].Fields[0].Value != "job-1" {
		t.Fatalf("subagent job = %#v", job)
	}
	sync := toolDetailsReceiptCallForTest(t, "subagent", `{"prompt":"Do work"}`, "started subagent a1")
	if sync == nil || sync.Items[0].Fields[0].Label != "Agent ID" || sync.Items[0].Fields[0].Value != "a1" {
		t.Fatalf("sync subagent = %#v", sync)
	}
	reply := toolDetailsReceiptCallForTest(t, "subagent", `{"prompt":"Do work"}`, "A useful reply")
	if reply == nil || reply.Summary != "Agent response" || reply.Items[0].Title != "Agent response" ||
		reply.Items[0].Description == nil || *reply.Items[0].Description != "A useful reply" {
		t.Fatalf("subagent reply = %#v", reply)
	}
	if len(reply.Items[0].Groups) != 1 || reply.Items[0].Groups[0].Label != "Task" ||
		reply.Items[0].Groups[0].Items[0].Description == nil || *reply.Items[0].Groups[0].Items[0].Description != "Do work" {
		t.Fatalf("subagent task group = %#v", reply.Items[0].Groups)
	}

	models := toolDetailsReceiptCallForTest(t, "list_subagent_models", `{"model":"deepseek-chat"}`, "deepseek-chat — Fast\nunknown")
	if models == nil || models.Summary != "deepseek-chat" || len(models.Items) != 2 {
		t.Fatalf("subagent models = %#v", models)
	}
	if models.Items[0].Title != "deepseek-chat" || models.Items[0].Description == nil || *models.Items[0].Description != "Fast" {
		t.Fatalf("model row = %#v", models.Items[0])
	}
	if models.Items[1].Title != "" || models.Items[1].Description == nil || *models.Items[1].Description != "unknown" {
		t.Fatalf("model note = %#v", models.Items[1])
	}
	fallback := toolDetailsReceiptCallForTest(t, "list_subagent_models", `{}`, "m — note")
	if fallback == nil || fallback.Summary != "Available models" {
		t.Fatalf("model fallback = %#v", fallback)
	}
	provider := toolDetailsReceiptCallForTest(t, "list_subagent_models", `{"provider":"p"}`, "m — note")
	if provider == nil || provider.Summary != "p" {
		t.Fatalf("provider fallback = %#v", provider)
	}
}

// The generic projector behind spawn_teammate and the team task cards mirrors
// inspectionItems: scalars, localized null/booleans, nested disclosures, a 40
// row cap and JSON code past the depth limit.
func TestToolDetailsModelInspectionProjection(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	scalars := toolDetailsInspectionItems(mustToolDetailsJSONForTest(t, `[null,true,"x",[]]`), 0)
	if len(scalars) != 4 || *scalars[0].Description != "None" || *scalars[1].Description != "Yes" ||
		*scalars[2].Description != "x" || *scalars[3].Description != "No results" {
		t.Fatalf("scalars = %#v", scalars)
	}
	record := toolDetailsInspectionItems(mustToolDetailsJSONForTest(t, `{"title":"Record","description":"Description","status":"running","inputSchema":{"type":"object"},"arguments":"{\"x\":1}","empty":[],"count":2}`), 0)[0]
	if record.Title != "Record" || record.Description == nil || *record.Description != "Description" || record.Badge.Label != "Running" {
		t.Fatalf("record = %#v", record)
	}
	if len(record.Fields) != 1 || record.Fields[0].Label != "count" || record.Fields[0].Value != "2" {
		t.Fatalf("record fields = %#v", record.Fields)
	}
	if len(record.Groups) != 2 || record.Groups[0].Label != "Input schema" || record.Groups[0].Items[0].Code == nil ||
		record.Groups[0].Items[0].Code.Language != "json" || record.Groups[1].Label != "Input" ||
		record.Groups[1].Items[0].Fields[0].Label != "x" || record.Groups[1].Items[0].Fields[0].Value != "1" {
		t.Fatalf("record groups = %#v", record.Groups)
	}
	capped := make([]any, 41)
	for index := range capped {
		capped[index] = map[string]any{"id": "t" + strconv.Itoa(index)}
	}
	raw, _ := json.Marshal(capped)
	items := toolDetailsInspectionItems(mustToolDetailsJSONForTest(t, string(raw)), 0)
	if len(items) != 41 || items[40].Description == nil || *items[40].Description != "1 more items available in Inspect" {
		t.Fatalf("capped = %d %#v", len(items), items[40])
	}
	deep := toolDetailsInspectionItems(mustToolDetailsJSONForTest(t, `{"a":{"b":{"c":{"d":{"e":{"f":1}}}}}}`), 0)[0]
	depth := 0
	for len(deep.Groups) > 0 {
		deep = deep.Groups[0].Items[0]
		depth++
	}
	if deep.Code == nil || deep.Code.Language != "json" || depth < 5 {
		t.Fatalf("deep = %#v depth %d", deep, depth)
	}
}

func mustToolDetailsJSONForTest(t *testing.T, text string) *toolDetailsValue {
	t.Helper()
	value, ok := toolDetailsDecodeJSON(text)
	if !ok {
		t.Fatalf("decode %q", text)
	}
	return value
}

// inspection-details-model.ts:19-34,36-72,92-150 and the harness expectation
// spec packages/client/ui-tool/tests/details-models.client.spec.ts:120-219 pin
// these cards.
func toolDetailsInspectionLocaleForTest(t *testing.T, locale string) {
	t.Helper()
	toolDetailsModelLocaleForTest(t, locale)
	previous := time.Local
	time.Local = time.FixedZone("GMT+8", 8*60*60)
	t.Cleanup(func() { time.Local = previous })
}

func TestToolDetailsModelInspectionCordis(t *testing.T) {
	toolDetailsInspectionLocaleForTest(t, i18n.LocaleEN)
	list := toolDetailsReceiptCallForTest(t, "cordis_inspect_list", `{}`, `{"providers":[{"name":"Service","status":"ready"}]}`)
	if list == nil || list.Summary != "1 inspect providers" || list.Items[0].Title != "Service" || list.Items[0].Badge.Label != "Ready" {
		t.Fatalf("cordis list = %#v", list)
	}
	query := toolDetailsReceiptCallForTest(t, "cordis_inspect_query", `{}`, `{"provider":"Service","method":"get","data":{"value":1}}`)
	if query == nil || query.Summary != "Service.get" || len(query.Items) != 1 || query.Items[0].Fields[0].Label != "value" || query.Items[0].Fields[0].Value != "1" {
		t.Fatalf("cordis query = %#v", query)
	}
	plugins := toolDetailsReceiptCallForTest(t, "cordis_inspect_self", `{}`, `{"mode":"client","plugins":[]}`)
	if plugins == nil || plugins.Summary != "0 dynamic plugins" || len(plugins.Items) != 1 || *plugins.Items[0].Description != "No results" {
		t.Fatalf("cordis plugins = %#v", plugins)
	}
	mode := toolDetailsReceiptCallForTest(t, "cordis_inspect_self", `{}`, `{"mode":"host","packageId":"pkg","value":1}`)
	if mode == nil || mode.Summary != "host" {
		t.Fatalf("cordis mode = %#v", mode)
	}
	plugin := toolDetailsReceiptCallForTest(t, "cordis_inspect_self", `{"pluginId":"plugin-1"}`, `{"mode":"host","value":1}`)
	if plugin == nil || plugin.Summary != "plugin-1" {
		t.Fatalf("cordis plugin arg = %#v", plugin)
	}
	for _, text := range []string{`[]`, `{"providers":{}}`, `bad`} {
		if got := toolDetailsReceiptCallForTest(t, "cordis_inspect_list", `{}`, text); got != nil {
			t.Fatalf("invalid cordis list %q produced %#v", text, got)
		}
	}
}

func TestToolDetailsModelInspectionWorkflowAndRalph(t *testing.T) {
	toolDetailsInspectionLocaleForTest(t, i18n.LocaleEN)
	workflow := toolDetailsReceiptCallForTest(t, "workflow", `{"code":"return true"}`, "workflow \"flow\" completed (1 agent).\nReturn value:\n{\"ok\":true}")
	if workflow == nil || workflow.Summary != "flow" || len(workflow.Items) != 2 {
		t.Fatalf("workflow = %#v", workflow)
	}
	header := workflow.Items[0]
	if header.Title != "flow" || header.Badge.Label != "Completed" || header.Fields[0].Label != "Agents started" || header.Fields[0].Value != "1" {
		t.Fatalf("workflow header = %#v", header)
	}
	if len(header.Groups) != 1 || header.Groups[0].Label != "Workflow script" || header.Groups[0].Items[0].Code.Language != "javascript" ||
		header.Groups[0].Items[0].Code.Text != "return true" {
		t.Fatalf("workflow script = %#v", header.Groups)
	}
	if workflow.Items[1].Fields[0].Label != "ok" || workflow.Items[1].Fields[0].Value != "Yes" {
		t.Fatalf("workflow value = %#v", workflow.Items[1])
	}
	for _, text := range []string{"bad", "workflow \"flow\" completed (1 agent).\nReturn value:\nnot-json"} {
		if got := toolDetailsReceiptCallForTest(t, "workflow", `{}`, text); got != nil {
			t.Fatalf("invalid workflow %q produced %#v", text, got)
		}
	}

	report := `{"summary":"Done","evidence":["checked"],"nextSteps":["ship"],"blocker":""}`
	ralph := toolDetailsReceiptCallForTest(t, "ralph", `{"objective":"Ship"}`, "Ralph worker reported completion after 2 rounds.\nFinal report:\n"+report)
	if ralph == nil || ralph.Summary != "Done" || ralph.Items[0].Badge.Label != "Worker reported completion" ||
		ralph.Items[0].Badge.Tone != "success" || len(ralph.Items[0].Lines) != 1 || ralph.Items[0].Lines[0] != "checked" {
		t.Fatalf("ralph = %#v", ralph)
	}
	if ralph.Items[0].Fields[0].Label != "Rounds" || ralph.Items[0].Fields[0].Value != "2" {
		t.Fatalf("ralph rounds = %#v", ralph.Items[0].Fields)
	}
	if len(ralph.Items[0].Groups) != 2 || ralph.Items[0].Groups[0].Label != "Remaining work" ||
		ralph.Items[0].Groups[0].Items[0].Lines[0] != "ship" || ralph.Items[0].Groups[1].Label != "Task" ||
		*ralph.Items[0].Groups[1].Items[0].Description != "Ship" {
		t.Fatalf("ralph groups = %#v", ralph.Items[0].Groups)
	}
	blocked := toolDetailsReceiptCallForTest(t, "ralph", `{}`, `Ralph worker reported a blocker after 1 round.`+"\nFinal report:\n"+`{"summary":"Blocked","evidence":[],"nextSteps":[],"blocker":"No key"}`)
	if blocked == nil || blocked.Items[0].Badge.Label != "Worker reported a blocker" ||
		len(blocked.Items[0].Fields) != 2 || blocked.Items[0].Fields[0].Label != "Rounds" ||
		blocked.Items[0].Fields[1].Label != "Blocker" || blocked.Items[0].Fields[1].Value != "No key" ||
		blocked.Items[0].Lines != nil || len(blocked.Items[0].Groups) != 0 {
		t.Fatalf("ralph blocker = %#v", blocked)
	}
	limited := toolDetailsReceiptCallForTest(t, "ralph", `{}`, `Ralph reached its 3 rounds limit; the worker reported work remaining.`+"\nFinal report:\n"+`{"summary":"Limited","evidence":[],"nextSteps":[],"blocker":""}`)
	if limited == nil || limited.Items[0].Badge.Label != "Round limit reached" {
		t.Fatalf("ralph limit = %#v", limited)
	}
	for _, text := range []string{
		"Ralph worker reported completion after 1 round.\nFinal report:\nnot-json",
		`Ralph worker reported completion after 1 round.` + "\nFinal report:\n" + `{"summary":"x","evidence":[1],"nextSteps":[],"blocker":""}`,
		`Ralph worker reported completion after 1 round.` + "\nFinal report:\n" + `{"summary":"x","evidence":[],"nextSteps":[1],"blocker":""}`,
		`Unknown result` + "\nFinal report:\n" + `{"summary":"x","evidence":[],"nextSteps":[],"blocker":""}`,
	} {
		if got := toolDetailsReceiptCallForTest(t, "ralph", `{}`, text); got != nil {
			t.Fatalf("invalid ralph produced %#v", got)
		}
	}
}

func TestToolDetailsModelInspectionSearches(t *testing.T) {
	toolDetailsInspectionLocaleForTest(t, i18n.LocaleEN)
	const sessionHit = "1. Session s1 — Session one\n   Created: 2026-01-01T00:00:00.000Z\n   Snippet: needle\n\nResult cap reached."
	search := toolDetailsReceiptCallForTest(t, "session_search", `{}`, sessionHit)
	if search == nil || search.Summary != "1 matches" || search.Caption != "Result limit reached; narrow the search for more" {
		t.Fatalf("session search = %#v", search)
	}
	if search.Items[0].Title != "Session one" || search.Items[0].Subtitle == nil || *search.Items[0].Subtitle != "s1" ||
		search.Items[0].Description == nil || *search.Items[0].Description != "needle" {
		t.Fatalf("session hit = %#v", search.Items[0])
	}
	if search.Items[0].Fields[0].Label != "Time" || search.Items[0].Fields[0].Value != "Jan 1, 2026, 8:00 AM" {
		t.Fatalf("session created = %#v", search.Items[0].Fields)
	}
	empty := toolDetailsReceiptCallForTest(t, "session_search", `{}`, "No prior session matches found.")
	if empty == nil || empty.Summary != "0 matches" || empty.Caption != "Recorded result" || len(empty.Items) != 0 {
		t.Fatalf("empty search = %#v", empty)
	}
	noEventHits := toolDetailsReceiptCallForTest(t, "session_event_search", `{}`, "Session s1 — Session one\n\nNo prior event matches found.")
	if noEventHits == nil || noEventHits.Summary != "0 matches" {
		t.Fatalf("empty event search = %#v", noEventHits)
	}
	const eventHit = "Session s1 — Session one\n\n1. seq 2 | tool/call | current | 2026-01-01T00:00:00.000Z\n   Snippet: needle"
	eventSearch := toolDetailsReceiptCallForTest(t, "session_event_search", `{}`, eventHit)
	if eventSearch == nil || eventSearch.Items[0].Title != "needle" ||
		*eventSearch.Items[0].Subtitle != "tool/call · #2" ||
		eventSearch.Items[0].Fields[0].Label != "Time" || eventSearch.Items[0].Fields[0].Value != "Jan 1, 2026, 8:00 AM" ||
		eventSearch.Items[0].Fields[1].Value != "current" {
		t.Fatalf("event search = %#v", eventSearch)
	}
	if search.Items[0].Fields == nil {
		t.Fatal("session hit lost its field slice")
	}
	for _, tc := range []struct{ name, text string }{
		{"session_search", "not a result"},
		{"session_search", "1. Session — title"},
		{"session_event_search", "1. seq bad | tool/call | current | 2026-01-01T00:00:00.000Z"},
	} {
		if got := toolDetailsReceiptCallForTest(t, tc.name, `{}`, tc.text); got != nil {
			t.Fatalf("invalid %s %q produced %#v", tc.name, tc.text, got)
		}
	}
}

func TestToolDetailsModelInspectionReadsAndTraces(t *testing.T) {
	toolDetailsInspectionLocaleForTest(t, i18n.LocaleEN)
	const read = "Session s1 — Session one\nTarget event seq 2:\n```json\n{\"seq\":2,\"time\":2000,\"type\":\"tool/call\",\"data\":{\"name\":\"x\"}}\n```\nAdjacent events: 1, 3"
	event := toolDetailsReceiptCallForTest(t, "session_event_read", `{}`, read)
	if event == nil || event.Summary != "tool/call · #2" || len(event.Items) != 3 {
		t.Fatalf("event read = %#v", event)
	}
	head := event.Items[0]
	if head.Title != "tool/call" || *head.Subtitle != "Session one · s1" || head.Fields[0].Label != "Sequence" && head.Fields[0].Label != "Event sequence" {
		t.Fatalf("event head = %#v", head)
	}
	if head.Fields[1].Label != "Time" || head.Fields[1].Value != "Jan 1, 1970, 8:00 AM" {
		t.Fatalf("event time = %#v", head.Fields)
	}
	if event.Items[1].Title != "x" {
		t.Fatalf("event data = %#v", event.Items[1])
	}
	if len(event.Items[2].Groups) != 1 || event.Items[2].Groups[0].Label != "Surrounding events" ||
		*event.Items[2].Groups[0].Items[0].Description != "Adjacent events: 1, 3" {
		t.Fatalf("event neighbors = %#v", event.Items[2])
	}
	for _, text := range []string{"not a result", "Session s1 — t\nTarget event seq 2:\n```json\nnot-json\n```"} {
		if got := toolDetailsReceiptCallForTest(t, "session_event_read", `{}`, text); got != nil {
			t.Fatalf("invalid event read %q produced %#v", text, got)
		}
	}

	const traceBody = "Session s1 — Session one\nCreated: 1970-01-01T00:00:00.000Z\nAvailability: unavailable\n\nAncestors (nearest first):\n- none (target is a root session)\n\nDescendants:\n- s2 — Child | 1970-01-01T00:00:01.000Z | live, persisted\n  - s3 — Grandchild | 1970-01-01T00:00:01.000Z | live, persisted"
	trace := toolDetailsReceiptCallForTest(t, "session_trace", `{}`, traceBody)
	if trace == nil || trace.Summary != "Session one" || len(trace.Items) != 3 {
		t.Fatalf("trace = %#v", trace)
	}
	if trace.Items[0].Title != "Session one" || *trace.Items[0].Subtitle != "s1" ||
		trace.Items[0].Fields[0].Label != "Time" || trace.Items[0].Fields[0].Value != "Jan 1, 1970, 8:00 AM" ||
		trace.Items[0].Fields[1].Value != "unavailable" {
		t.Fatalf("trace head = %#v", trace.Items[0])
	}
	if trace.Items[1].Title != "Ancestor sessions" || *trace.Items[1].Description != "None" {
		t.Fatalf("trace ancestors = %#v", trace.Items[1])
	}
	if trace.Items[2].Title != "Descendant sessions" || len(trace.Items[2].Lines) != 2 ||
		trace.Items[2].Lines[0] != "s2 — Child | 1970-01-01T00:00:01.000Z | live, persisted" ||
		trace.Items[2].Lines[1] != "  s3 — Grandchild | 1970-01-01T00:00:01.000Z | live, persisted" {
		t.Fatalf("trace descendants = %#v", trace.Items[2].Lines)
	}
	eventTrace := toolDetailsReceiptCallForTest(t, "session_event_trace", `{}`, "Session s1 — title\nCreated: invalid\nAvailability: unavailable")
	if eventTrace == nil || len(eventTrace.Items) != 1 || eventTrace.Items[0].Fields[0].Value != "invalid" {
		t.Fatalf("event trace = %#v", eventTrace)
	}
	// A spill notice keeps the generic evidence body, and an unknown section or
	// malformed Created line rejects a full trace.
	spilled := "Session s1 — title\nCreated: 1970-01-01T00:00:00.000Z\n\n(Omitted 50000 bytes. Full formatted result stored at: /spill/output.txt. Read the file.)"
	if got := toolDetailsReceiptCallForTest(t, "session_event_trace", `{}`, spilled); got != nil {
		t.Fatalf("spilled trace produced %#v", got)
	}
	if got := toolDetailsReceiptCallForTest(t, "session_trace", `{}`, "Session s1 — title\nCreated: x\n\nOther:\n- y"); got != nil {
		t.Fatalf("unknown trace section produced %#v", got)
	}
}

// control-details-model.ts:56-74 (terminal roster) and :76-98 (language server).
func TestToolDetailsModelTerminalList(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	text := "pty-1 [shell] running pid=4\npty-2 (done) [shell] exited code=2 signal=null pid=5\npty-3 [shell] exited code=0 signal=TERM"
	list := toolDetailsReceiptCallForTest(t, "terminal_list", `{}`, text)
	if list == nil || list.Summary != "3 terminals" || len(list.Items) != 3 {
		t.Fatalf("terminal list = %#v", list)
	}
	running := list.Items[0]
	if running.Title != "pty-1" || running.Subtitle != nil || running.Badge.Label != "Running" || running.Badge.Tone != "info" ||
		len(running.Fields) != 2 || running.Fields[0].Label != "Type" || running.Fields[0].Value != "shell" ||
		running.Fields[1].Label != "Process ID" || running.Fields[1].Value != "4" {
		t.Fatalf("running terminal = %#v", running)
	}
	exited := list.Items[1]
	if exited.Title != "done" || exited.Subtitle == nil || *exited.Subtitle != "pty-2" ||
		exited.Badge.Label != "Exited" || exited.Badge.Tone != "neutral" ||
		len(exited.Fields) != 3 || exited.Fields[1].Label != "Process ID" ||
		exited.Fields[2].Label != "Exit code" || exited.Fields[2].Value != "2" {
		t.Fatalf("exited terminal = %#v", exited)
	}
	// A null signal is dropped; a zero exit code reads as success.
	clean := list.Items[2]
	if clean.Badge.Tone != "success" || len(clean.Fields) != 3 || clean.Fields[1].Label != "Exit code" || clean.Fields[1].Value != "0" ||
		clean.Fields[2].Label != "Signal" || clean.Fields[2].Value != "TERM" {
		t.Fatalf("clean exit = %#v", clean)
	}
	empty := toolDetailsReceiptCallForTest(t, "terminal_list", `{}`, "(no terminal sessions)")
	if empty == nil || empty.Summary != "0 terminals" || empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("empty terminal list = %#v", empty)
	}
	for _, bad := range []string{"", "pty-1 [shell]", "pty-1 shell running", "pty-1 [shell] running pid=x"} {
		if got := toolDetailsReceiptCallForTest(t, "terminal_list", `{}`, bad); got != nil {
			t.Fatalf("invalid terminal list %q produced %#v", bad, got)
		}
	}
}

func TestToolDetailsModelLSP(t *testing.T) {
	toolDetailsInspectionLocaleForTest(t, i18n.LocaleEN)
	args := `{"file_path":"src/main.ts","line":2,"character":3,"operation":"findReferences"}`
	references := "src/a.ts:2:3\nhttps://example.test/a.ts:4:5\nuntitled:Untitled-1:6:7\nC:\\src\\a.ts:8:9\nD:/lib/a.ts:10:11"
	list := toolDetailsReceiptCallForTest(t, "lsp", args, references)
	if list == nil || list.Summary != "src/main.ts · 5 locations" || len(list.Items) != 5 {
		t.Fatalf("lsp references = %#v", list)
	}
	first := list.Items[0]
	if first.Title != "src/a.ts" || first.Subtitle == nil || *first.Subtitle != "Line 2, column 3" ||
		first.Location == nil || first.Location.Path != "src/a.ts" || first.Location.Line == nil || *first.Location.Line != 2 {
		t.Fatalf("first reference = %#v", first)
	}
	// URI rows and Windows drive paths keep their recorded form; only the URI
	// loses its file link.
	if list.Items[1].Location != nil || list.Items[2].Location != nil {
		t.Fatalf("uri rows stayed linked: %#v %#v", list.Items[1].Location, list.Items[2].Location)
	}
	if list.Items[3].Location == nil || list.Items[3].Location.Path != "C:\\src\\a.ts" || list.Items[4].Location == nil {
		t.Fatalf("drive rows lost their link: %#v %#v", list.Items[3].Location, list.Items[4].Location)
	}

	hover := toolDetailsReceiptCallForTest(t, "lsp", `{"file_path":"src/main.ts","line":2,"character":3,"operation":"hover"}`, "hover text")
	if hover == nil || hover.Summary != "src/main.ts:2:3" || hover.Items[0].Markdown == nil || *hover.Items[0].Markdown != "hover text" ||
		hover.Items[0].Location == nil || *hover.Items[0].Location.Line != 2 {
		t.Fatalf("lsp hover = %#v", hover)
	}
	empty := toolDetailsReceiptCallForTest(t, "lsp", args, "No results.")
	if empty == nil || empty.Summary != "0 locations" || len(empty.Items) != 0 {
		t.Fatalf("lsp empty = %#v", empty)
	}
	// The symbol list keeps its ellipsis row and counts only titled rows.
	symbols := toolDetailsReceiptCallForTest(t, "lsp", args, "… 3 more\nsrc/a.ts:2:3")
	if symbols == nil || symbols.Summary != "src/main.ts · 1 locations" || symbols.Items[0].Description == nil ||
		*symbols.Items[0].Description != "… 3 more" {
		t.Fatalf("lsp symbols = %#v", symbols)
	}
	for _, tc := range []struct {
		args, text string
	}{
		{`{"file_path":"","line":2,"character":3,"operation":"hover"}`, "x"},
		{`{"file_path":"a.ts","line":"2","character":3,"operation":"hover"}`, "x"},
		{`{"file_path":"a.ts","character":3}`, "x"},
		{args, "not a location"},
	} {
		if got := toolDetailsReceiptCallForTest(t, "lsp", tc.args, tc.text); got != nil {
			t.Fatalf("invalid lsp %s produced %#v", tc.text, got)
		}
	}
}

// The vocabulary kinds carry the installed fields for the todo row markers and
// the hover/location entries.
func TestToolDetailsModelVocabularyKindsGolden(t *testing.T) {
	line := 12
	markdown := "**bold** reply"
	previous := "Pending"
	model := &toolDetailsModel{Summary: "Review", Items: []toolDetailItem{{
		Title: "Review", Fields: []toolDetailField{},
		Status:         &toolDetailStatus{Value: "in_progress", Label: "In progress"},
		PreviousStatus: &previous,
		Change:         &toolDetailChange{Value: "updated", Label: "Status changed"},
		Markdown:       &markdown,
		Location:       &toolDetailLocation{Path: "src/a.ts", Line: &line},
	}}}
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"Items":[{"Title":"Review","Fields":[],"Badge":null,"Subtitle":null,"Description":null,"Code":null,"Lines":null,"Groups":null,"Markdown":"**bold** reply","Location":{"Path":"src/a.ts","Line":12},"Status":{"Value":"in_progress","Label":"In progress"},"PreviousStatus":"Pending","Change":{"Value":"updated","Label":"Status changed"}}],"Empty":"","Summary":"Review","Caption":"","ExpandedSummary":"","Unchanged":null}`
	if string(raw) != want {
		t.Fatalf("vocabulary kinds = %s", raw)
	}
}

// todo-diff-model.ts:17-79 and todo-history.ts:26-56 pin the write card: the
// list precedes each write, changed rows carry a change marker and the unchanged
// container holds the rest.
func toolDetailsTodoCallForTest(t *testing.T, todos string) *session.ToolCallView {
	t.Helper()
	args := `{"todos":` + todos + `}`
	call := toolDetailsModelCallForTest(t, false, "todo_write", args, "Updated todo list")
	call.Name, call.Arguments = "todo_write", args
	return call
}

func toolDetailsTodoBaselineForTest(todos string) *todoBaseline {
	baseline := &todoBaseline{Known: true}
	if todos == "" {
		return baseline
	}
	value, ok := toolDetailsDecodeJSON(todos)
	if !ok || value.kind != 'a' {
		return baseline
	}
	for _, entry := range value.list {
		content, _ := entry.get("content")
		status, _ := entry.get("status")
		baseline.Todos = append(baseline.Todos, session.PlanTodo{Content: content.text, Status: status.text})
	}
	return baseline
}

func TestToolDetailsModelTodoDiff(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	todos := func(entries ...string) string { return "[" + strings.Join(entries, ",") + "]" }
	entry := func(content, status string) string { return `{"content":"` + content + `","status":"` + status + `"}` }

	// The first loaded write is the initial list; every row counts as added.
	initial := toolDetailsTodoFor(toolDetailsTodoCallForTest(t, todos(entry("Write docs", "pending"), entry("Ship", "completed"))), nil)
	if initial == nil || initial.Caption != "Initial list" || initial.Summary != "2 added" ||
		initial.Empty != "No changes to the list" || len(initial.Items) != 2 ||
		initial.Items[0].Change == nil || initial.Items[0].Change.Value != "added" ||
		initial.Items[0].Status.Value != "pending" || initial.Unchanged != nil {
		t.Fatalf("initial = %#v", initial)
	}

	// An older unloaded page may hide the predecessor.
	unavailable := toolDetailsTodoFor(toolDetailsTodoCallForTest(t, todos(entry("Write docs", "pending"))), &todoBaseline{Known: false, HasMore: true})
	if unavailable == nil || unavailable.Caption != "Previous list unavailable" || unavailable.Summary != "" ||
		len(unavailable.Items) != 1 || unavailable.Items[0].Change != nil {
		t.Fatalf("unavailable = %#v", unavailable)
	}

	// One added row, one status change, one removal, two unchanged rows.
	baseline := toolDetailsTodoBaselineForTest(todos(entry("Write docs", "pending"), entry("Ship", "pending"), entry("Review", "completed")))
	current := todos(entry("Write docs", "completed"), entry("Ship", "pending"), entry("Audit", "pending"))
	diff := toolDetailsTodoFor(toolDetailsTodoCallForTest(t, current), baseline)
	if diff == nil || diff.Caption != "Changes since the previous list" || diff.Summary != "1 added · 1 updated · 1 removed" ||
		len(diff.Items) != 3 {
		t.Fatalf("diff = %#v", diff)
	}
	updated := diff.Items[0]
	if updated.Title != "Write docs" || updated.Change.Value != "updated" || updated.Change.Label != "Status changed" ||
		updated.PreviousStatus == nil || *updated.PreviousStatus != "Pending" || updated.Status.Label != "Completed" {
		t.Fatalf("updated row = %#v", updated)
	}
	added := diff.Items[1]
	if added.Title != "Audit" || added.Change.Value != "added" || added.Change.Label != "Added" {
		t.Fatalf("added row = %#v", added)
	}
	removed := diff.Items[2]
	if removed.Title != "Review" || removed.Change.Value != "removed" || removed.Change.Label != "Removed" ||
		removed.Status.Value != "completed" || removed.PreviousStatus != nil {
		t.Fatalf("removed row = %#v", removed)
	}
	if diff.Unchanged == nil || diff.Unchanged.Label != "1 unchanged" || len(diff.Unchanged.Items) != 1 || diff.Unchanged.Items[0].Title != "Ship" {
		t.Fatalf("unchanged = %#v", diff.Unchanged)
	}

	// Reordering alone marks rows as reordered rather than status changes.
	moved := toolDetailsTodoFor(toolDetailsTodoCallForTest(t, todos(entry("Ship", "pending"), entry("Write docs", "pending"))),
		toolDetailsTodoBaselineForTest(todos(entry("Write docs", "pending"), entry("Ship", "pending"))))
	if moved == nil || moved.Summary != "2 updated" || len(moved.Items) != 2 ||
		moved.Items[0].Change.Label != "Reordered" || moved.Items[0].PreviousStatus != nil || moved.Unchanged != nil {
		t.Fatalf("moved = %#v", moved)
	}

	// Identical lists keep the rows in the unchanged disclosure only.
	same := toolDetailsTodoFor(toolDetailsTodoCallForTest(t, todos(entry("Ship", "completed"))),
		toolDetailsTodoBaselineForTest(todos(entry("Ship", "completed"))))
	if same == nil || same.Summary != "No changes to the list" || len(same.Items) != 0 ||
		same.Empty != "No changes to the list" || same.Unchanged == nil || same.Unchanged.Label != "1 unchanged" {
		t.Fatalf("same = %#v", same)
	}

	// A recorded empty first list reports the empty to-do copy.
	empty := toolDetailsTodoFor(toolDetailsTodoCallForTest(t, `[]`), &todoBaseline{Known: false})
	if empty == nil || empty.Caption != "Initial list" || empty.Empty != "The to-do list is empty" ||
		len(empty.Items) != 0 || empty.Summary != "No changes to the list" {
		t.Fatalf("empty = %#v", empty)
	}

	// Malformed argument lists keep the raw evidence body.
	for _, bad := range []string{
		`{"todos":"x"}`,
		`{"todos":[{"content":"","status":"pending"}]}`,
		`{"todos":[{"content":"   ","status":"pending"}]}`,
		`{"todos":[{"content":"A","status":"done"}]}`,
		`{"todos":[{"content":"A"}]}`,
		`{"todos":[{"content":1,"status":"pending"}]}`,
		`{"todos":[]} {"todos":[]}`,
		`{}`,
	} {
		call := toolDetailsModelCallForTest(t, false, "todo_write", bad, "Updated todo list")
		call.Name, call.Arguments = "todo_write", bad
		if got := toolDetailsTodoFor(call, nil); got != nil {
			t.Fatalf("malformed %s produced %#v", bad, got)
		}
	}
	duplicate := todos(entry("Same", "pending"), entry(" Same ", "pending"))
	duplicateCall := toolDetailsModelCallForTest(t, false, "todo_write", `{"todos":`+duplicate+`}`, "x")
	duplicateCall.Name, duplicateCall.Arguments = "todo_write", `{"todos":`+duplicate+`}`
	if got := toolDetailsTodoFor(duplicateCall, nil); got != nil {
		t.Fatalf("duplicate titles produced %#v", got)
	}
	// Other tools are not this family.
	if got := toolDetailsTodoFor(toolDetailsModelCallForTest(t, false, "todo_read", `{}`, "x"), nil); got != nil {
		t.Fatalf("todo_read produced %#v", got)
	}
}

func TestToolDetailsModelTodoHistoryIndex(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	message := func(eventType, raw string) session.Message {
		return session.Message{EventType: eventType, Raw: json.RawMessage(raw)}
	}
	todoWrite := func(content string) string {
		return `{"type":"todo/write","data":{"todos":[{"content":"` + content + `","status":"pending"}]}}`
	}
	call := func(id string) string {
		return `{"type":"tool/call","data":{"name":"todo_write","arguments":"{}","callId":"` + id + `"}}`
	}
	nested := func(id string) string {
		return `{"type":"tool/ptc-dispatch-start","data":{"name":"todo_write","arguments":{},"subCallId":"` + id + `"}}`
	}
	messages := []session.Message{
		message("todo/write", todoWrite("First")),
		message("tool/call", call("call-2")),
		message("todo/write", todoWrite("Second")),
		message("tool/ptc-dispatch-start", nested("sub-3")),
	}
	baselines := todoBaselinesFor(messages, false)
	if baselines["call-2"] == nil || !baselines["call-2"].Known || len(baselines["call-2"].Todos) != 1 ||
		baselines["call-2"].Todos[0].Content != "First" || baselines["call-2"].HasMore {
		t.Fatalf("call-2 baseline = %#v", baselines["call-2"])
	}
	if baselines["sub-3"] == nil || baselines["sub-3"].Todos[0].Content != "Second" {
		t.Fatalf("sub-3 baseline = %#v", baselines["sub-3"])
	}
	// A truncated window that starts after the earlier write reports hasMore.
	truncated := todoBaselinesFor([]session.Message{message("tool/call", call("call-1"))}, true)
	if truncated["call-1"] == nil || truncated["call-1"].Known || !truncated["call-1"].HasMore {
		t.Fatalf("truncated baseline = %#v", truncated["call-1"])
	}
}

func TestToolDetailsModelTodoCopyChinese(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleZH)
	call := toolDetailsTodoCallForTest(t, `[{"content":"写文档","status":"pending"}]`)
	initial := toolDetailsTodoFor(call, nil)
	if initial == nil || initial.Caption != "首次记录" || initial.Summary != "新增 1" || initial.Empty != "清单没有变化" ||
		initial.Items[0].Status.Label != "待处理" || initial.Items[0].Change.Label != "新增" {
		t.Fatalf("zh initial = %#v", initial)
	}
	baseline := toolDetailsTodoBaselineForTest(`[{"content":"写文档","status":"completed"},{"content":"评审","status":"pending"}]`)
	diff := toolDetailsTodoFor(call, baseline)
	if diff == nil || diff.Summary != "更新 1 · 移除 1" || diff.Caption != "与上次清单相比" ||
		diff.Items[0].PreviousStatus == nil || *diff.Items[0].PreviousStatus != "已完成" ||
		diff.Items[1].Change.Label != "移除" {
		t.Fatalf("zh diff = %#v", diff)
	}
	unavailable := toolDetailsTodoFor(call, &todoBaseline{Known: false, HasMore: true})
	if unavailable == nil || unavailable.Caption != "旧清单不可用" {
		t.Fatalf("zh unavailable = %#v", unavailable)
	}
}

// AskQuestionCard.tsx and toolviews/ask-question-row.tsx pair the call's
// questions with the result's answers by id; the skipped label and the
// cancelled/interrupted verdicts come from the installed dictionary.
func askQuestionCallForTest(t *testing.T, questions, result string) *session.ToolCallView {
	t.Helper()
	args := `{"questions":` + questions + `}`
	call := toolDetailsModelCallForTest(t, false, "ask_user_question", args, result)
	call.Name, call.Arguments = "ask_user_question", args
	return call
}

func askQuestionErrorCallForTest(t *testing.T, questions, code string) *session.ToolCallView {
	t.Helper()
	call := askQuestionCallForTest(t, questions, `{"pending":true,"callId":"goal-123"}`)
	raw := string(call.Result.Raw)
	raw = strings.Replace(raw, `"data":{`, `"data":{"isError":true,"error":{"name":"UserQuestionError","code":"`+code+`"},`, 1)
	call.Result.Raw = json.RawMessage(raw)
	call.State = "error"
	return call
}

func TestToolDetailsModelAskQuestionAnswered(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	questions := `[{"id":"scope","question":"Which project?"},{"id":"notes","question":"Anything else?"}]`
	answers := `{"answers":[{"id":"scope","selected":["deepseek-harness","whaleisle"]},{"id":"notes","selected":[]}]}`
	card := askQuestionCardFor(askQuestionCallForTest(t, questions, answers))
	if card == nil || card.Kind != "answered" || card.SkippedLabel != "Not answered" || len(card.Questions) != 2 {
		t.Fatalf("answered card = %#v", card)
	}
	if card.Questions[0].Question != "Which project?" || len(card.Questions[0].Answers) != 2 ||
		card.Questions[0].Answers[0] != "deepseek-harness" || card.Questions[0].Answers[1] != "whaleisle" {
		t.Fatalf("paired answers = %#v", card.Questions[0])
	}
	if len(card.Questions[1].Answers) != 0 {
		t.Fatalf("skipped question = %#v", card.Questions[1])
	}
	// A custom answer appends after the selected labels; an empty custom does not.
	custom := `{"answers":[{"id":"scope","selected":["a"],"custom":"b"},{"id":"notes","selected":[],"custom":""}]}`
	card = askQuestionCardFor(askQuestionCallForTest(t, questions, custom))
	if card == nil || len(card.Questions[0].Answers) != 2 || card.Questions[0].Answers[1] != "b" ||
		len(card.Questions[1].Answers) != 0 {
		t.Fatalf("custom answers = %#v", card)
	}
}

func TestToolDetailsModelAskQuestionFallbacks(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	questions := `[{"id":"a","question":"A?"},{"id":"b","question":"B?"}]`
	for _, tc := range []struct{ name, questions, result string }{
		{"empty questions", `[]`, `{"answers":[]}`},
		{"duplicate question ids", `[{"id":"a","question":"A?"},{"id":"a","question":"B?"}]`, `{"answers":[]}`},
		{"missing question text", `[{"id":"a"}]`, `{"answers":[]}`},
		{"missing answers", questions, `{"pending":true}`},
		{"answers not an array", questions, `{"answers":{}}`},
		{"fewer answers", questions, `{"answers":[{"id":"a","selected":["x"]}]}`},
		{"unknown answer id", questions, `{"answers":[{"id":"a","selected":["x"]},{"id":"c","selected":[]}]}`},
		{"duplicate answer id", questions, `{"answers":[{"id":"a","selected":["x"]},{"id":"a","selected":[]}]}`},
		{"non-string selected", questions, `{"answers":[{"id":"a","selected":[1]},{"id":"b","selected":[]}]}`},
		{"non-string custom", questions, `{"answers":[{"id":"a","selected":["x"],"custom":1},{"id":"b","selected":[]}]}`},
		{"not json", questions, `not json`},
	} {
		if card := askQuestionCardFor(askQuestionCallForTest(t, tc.questions, tc.result)); card != nil {
			t.Fatalf("%s produced %#v", tc.name, card)
		}
	}
	if card := askQuestionCardFor(askQuestionErrorCallForTest(t, questions, "TOOL_FAILED")); card != nil {
		t.Fatalf("unknown error code produced %#v", card)
	}
	// Other tools never take this card.
	if card := askQuestionCardFor(toolDetailsModelCallForTest(t, false, "get_goal", `{}`, "x")); card != nil {
		t.Fatalf("goal produced %#v", card)
	}
}

func TestToolDetailsModelAskQuestionUnanswered(t *testing.T) {
	toolDetailsModelLocaleForTest(t, i18n.LocaleEN)
	questions := `[{"id":"a","question":"Which project?"}]`
	cancelled := askQuestionCardFor(askQuestionErrorCallForTest(t, questions, "ASK_CANCELLED"))
	if cancelled == nil || cancelled.Kind != "unanswered" ||
		cancelled.Verdict != "This question set was cancelled before answers were submitted." ||
		len(cancelled.Questions) != 1 || cancelled.Questions[0].Question != "Which project?" {
		t.Fatalf("cancelled card = %#v", cancelled)
	}
	aborted := askQuestionCardFor(askQuestionErrorCallForTest(t, questions, "ASK_ABORTED"))
	if aborted == nil || aborted.Kind != "unanswered" ||
		aborted.Verdict != "This question set was interrupted before answers were submitted." {
		t.Fatalf("aborted card = %#v", aborted)
	}

	toolDetailsModelLocaleForTest(t, i18n.LocaleZH)
	zh := askQuestionCardFor(askQuestionErrorCallForTest(t, questions, "ASK_CANCELLED"))
	if zh == nil || zh.Verdict != "本轮已取消，未提交回答" {
		t.Fatalf("zh cancelled = %#v", zh)
	}
	zhAnswered := askQuestionCardFor(askQuestionCallForTest(t, questions, `{"answers":[{"id":"a","selected":[]}]}`))
	if zhAnswered == nil || zhAnswered.SkippedLabel != "未回答" {
		t.Fatalf("zh skipped = %#v", zhAnswered)
	}
}
