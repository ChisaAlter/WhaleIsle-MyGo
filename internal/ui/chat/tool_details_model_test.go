package chat

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

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
		for _, name := range []string{"", "Create_goal", "get_goal ", "functions.get_goal", "plugin:get_goal", "schedule_create", "schedule_list", "job_list", "cordis_inspect_self", "todo_write", "unknown"} {
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
