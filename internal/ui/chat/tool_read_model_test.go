package chat

import (
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

const (
	toolReadModelArgsForTest = `{"file_path":"/argument/ignored","offset":1,"limit":2}`
	toolReadModelMetaForTest = `{"path":"/work/actual.go","offset":10,"lines":[{"number":10,"text":"first"},{"number":12,"text":""}],"totalLines":20,"lang":"go"}`
	toolReadModelTextForTest = "<path>/envelope/ignored</path>\n<type>file</type>\n<content>\n999: never displayed\n</content>"
)

func toolReadModelCallForTest(t *testing.T, args, meta, text string) *session.ToolCallView {
	t.Helper()
	content, err := json.Marshal([]any{map[string]any{"type": "text", "text": text}})
	if err != nil {
		t.Fatal(err)
	}
	return &session.ToolCallView{
		CallID: "read-118", Name: "read", Arguments: args, State: "ok",
		Call: &session.Message{
			EventType: "tool/call", Raw: json.RawMessage(`{"type":"tool/call","data":{"callId":"read-118"}}`),
			Body: "flattened head ignored", Args: `{"file_path":"/wrong-head"}`,
		},
		Result: toolReadModelReceiptForTest(meta, string(content), ""),
	}
}

func toolReadModelReceiptForTest(meta, content, extra string) *session.Message {
	return &session.Message{
		Kind: session.MessageKindTool, EventType: "tool/result",
		Raw:  json.RawMessage(`{"type":"tool/result","data":{"meta":` + meta + `,"message":{"content":` + content + `}` + extra + `}}`),
		Body: toolReadModelTextForTest, Args: `{"file_path":"/wrong-result"}`,
	}
}

// These expected models follow the current installed read-card-model and
// raw-tool-call contracts. They need no installed source, filesystem or host.
func TestToolReadModelExactMetadataModels(t *testing.T) {
	cases := []struct {
		name, meta, text string
		want             toolReadModel
	}{
		{
			"metadata owns content and label", toolReadModelMetaForTest, toolReadModelTextForTest,
			toolReadModel{Label: "actual.go", Lines: []toolReadLine{{10, "first"}, {12, ""}}, TotalLines: 20, Lang: "go"},
		},
		{
			"empty metadata and envelope paths are valid", `{"path":"","offset":1,"lines":[],"totalLines":0}`,
			"<path></path>\n<type>file</type>\n<content>\n\n</content>",
			toolReadModel{Lines: []toolReadLine{}, TotalLines: 0},
		},
		{
			"empty file beyond requested offset", `{"path":"empty","offset":500,"lines":[],"totalLines":0,"lang":""}`, toolReadModelTextForTest,
			toolReadModel{Label: "empty", Lines: []toolReadLine{}},
		},
		{
			"gaps and metadata exceeding argument limit", `{"path":"relative/./a","offset":2,"lines":[{"number":3,"text":"a\r\nb"},{"number":6,"text":"\u0000"},{"number":9,"text":"last"}],"totalLines":9}`, toolReadModelTextForTest,
			toolReadModel{Label: "relative/./a", Lines: []toolReadLine{{3, "a\r\nb"}, {6, "\x00"}, {9, "last"}}, TotalLines: 9},
		},
		{
			"integral exponent and decimal JSON numbers", `{"path":"/work/numeric","offset":1e1,"lines":[{"number":1e1,"text":""},{"number":12.0,"text":"twelve"}],"totalLines":2e1}`, toolReadModelTextForTest,
			toolReadModel{Label: "numeric", Lines: []toolReadLine{{10, ""}, {12, "twelve"}}, TotalLines: 20},
		},
		{
			"metadata duplicate keys last value", `{"path":"wrong","path":"/work/right","offset":0,"offset":2,"lines":null,"lines":[{"number":1,"number":2,"text":null,"text":""}],"totalLines":0,"totalLines":2,"lang":null,"lang":"text"}`, toolReadModelTextForTest,
			toolReadModel{Label: "right", Lines: []toolReadLine{{2, ""}}, TotalLines: 2, Lang: "text"},
		},
		{
			"unrelated overflowing numbers stay irrelevant", `{"path":"a","offset":1,"lines":[{"number":1,"text":"x","extension":1e9999}],"totalLines":1,"extension":1e9999}`, toolReadModelTextForTest,
			toolReadModel{Label: "a", Lines: []toolReadLine{{1, "x"}}, TotalLines: 1},
		},
		{
			"negative zero total follows JavaScript", `{"path":"zero","offset":1,"lines":[],"totalLines":-0.0}`, toolReadModelTextForTest,
			toolReadModel{Label: "zero", Lines: []toolReadLine{}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			call := toolReadModelCallForTest(t, toolReadModelArgsForTest, tc.meta, tc.text)
			if got := toolReadModelFor(call, "/work", "/home/me"); !reflect.DeepEqual(got, &tc.want) {
				t.Fatalf("model = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestToolReadModelReadArguments(t *testing.T) {
	cases := []struct {
		name, args string
		valid      bool
	}{
		{"minimum", `{"file_path":"a"}`, true},
		{"positive exponent", `{"file_path":"a","offset":1e1,"limit":2e1}`, true},
		{"integer decimal", `{"file_path":"a","offset":1.0,"limit":2.0}`, true},
		{"large argument numbers have no int bound", `{"file_path":"a","offset":1e300,"limit":1e300}`, true},
		{"rounded JavaScript integer", `{"file_path":"a","offset":1.00000000000000001,"limit":1.99999999999999999}`, true},
		{"duplicate final values", `{"file_path":null,"file_path":"a","offset":null,"offset":1,"limit":0,"limit":2}`, true},
		{"NEL not JavaScript blank", `{"file_path":"\u0085"}`, true},
		{"control and URL strings allowed", `{"file_path":"https://example.test/a\n\u0000"}`, true},
		{"unrelated nonfinite number", `{"file_path":"a","extra":1e9999}`, true},
		{"missing args", "", false},
		{"incomplete", "{", false},
		{"null", "null", false},
		{"array", `[{"file_path":"a"}]`, false},
		{"string", `"a"`, false},
		{"number", "1", false},
		{"bool", "true", false},
		{"trailing JSON", `{"file_path":"a"} {}`, false},
		{"trailing comma", `{"file_path":"a",}`, false},
		{"missing file_path", `{}`, false},
		{"path alias", `{"path":"a"}`, false},
		{"wrong case", `{"File_Path":"a"}`, false},
		{"null file_path", `{"file_path":null}`, false},
		{"number file_path", `{"file_path":1}`, false},
		{"array file_path", `{"file_path":["a"]}`, false},
		{"empty file_path", `{"file_path":""}`, false},
		{"blank file_path", `{"file_path":" \t\r\n\u00a0\u2028\u2029\ufeff"}`, false},
		{"duplicate invalid final path", `{"file_path":"a","file_path":null}`, false},
	}
	for _, key := range []string{"offset", "limit"} {
		for _, value := range []string{"null", `"1"`, "true", "[]", "{}", "0", "-0", "-1", "0.5", "1.5", "1e9999", "-1e9999", "1e-9999"} {
			cases = append(cases, struct {
				name, args string
				valid      bool
			}{key + "/" + value, `{"file_path":"a","` + key + `":` + value + `}`, false})
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			call := toolReadModelCallForTest(t, tc.args, toolReadModelMetaForTest, toolReadModelTextForTest)
			got := toolReadModelFor(call, "/work", "")
			if (got != nil) != tc.valid {
				t.Fatalf("valid=%v, model=%#v", tc.valid, got)
			}
		})
	}
}

func TestToolReadModelDeclinesMalformedMetadata(t *testing.T) {
	invalid := []struct{ name, meta string }{
		{"null", "null"}, {"array", "[]"}, {"string", `"meta"`}, {"number", "1"}, {"empty object", `{}`},
		{"trailing JSON", toolReadModelMetaForTest + ` {}`},
	}
	for _, key := range []string{"path", "offset", "lines", "totalLines"} {
		meta, _ := terminalModelObject(json.RawMessage(toolReadModelMetaForTest))
		delete(meta, key)
		raw, err := json.Marshal(meta)
		if err != nil {
			t.Fatal(err)
		}
		invalid = append(invalid, struct{ name, meta string }{"missing " + key, string(raw)})
	}
	badFields := []struct {
		key    string
		values []string
	}{
		{"path", []string{"null", "1", "true", "[]", "{}"}},
		{"offset", []string{"null", `"10"`, "true", "[]", "{}", "0", "-1", "1.5", "1e9999", "1e-9999"}},
		{"totalLines", []string{"null", `"20"`, "true", "[]", "{}", "-1", "20.5", "1e9999"}},
		{"lines", []string{"null", "1", "true", `"lines"`, "{}", "[null]", "[1]", "[[]]", "[{}]"}},
		{"lang", []string{"null", "1", "true", "[]", "{}"}},
	}
	for _, field := range badFields {
		for _, value := range field.values {
			meta, _ := terminalModelObject(json.RawMessage(toolReadModelMetaForTest))
			meta[field.key] = json.RawMessage(value)
			raw, err := json.Marshal(meta)
			if err != nil {
				t.Fatal(err)
			}
			invalid = append(invalid, struct{ name, meta string }{field.key + "/" + value, string(raw)})
		}
	}
	for _, lines := range []string{
		`[{"text":"missing number"}]`,
		`[{"number":10}]`,
		`[{"number":0,"text":"x"}]`,
		`[{"number":-1,"text":"x"}]`,
		`[{"number":9,"text":"before offset"}]`,
		`[{"number":21,"text":"past total"}]`,
		`[{"number":10.5,"text":"fraction"}]`,
		`[{"number":"10","text":"string"}]`,
		`[{"number":null,"text":"null"}]`,
		`[{"number":true,"text":"bool"}]`,
		`[{"number":1e9999,"text":"infinite"}]`,
		`[{"number":10,"text":null}]`,
		`[{"number":10,"text":1}]`,
		`[{"number":10,"text":[]}]`,
		`[{"number":10,"text":{}}]`,
		`[{"number":10,"text":"first"},{"number":10,"text":"duplicate"}]`,
		`[{"number":12,"text":"first"},{"number":11,"text":"decreasing"}]`,
		`[{"number":10,"text":"first"},{"number":12,"text":null}]`,
	} {
		invalid = append(invalid, struct{ name, meta string }{
			"line/" + lines, `{"path":"a","offset":10,"lines":` + lines + `,"totalLines":20}`,
		})
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			call := toolReadModelCallForTest(t, toolReadModelArgsForTest, tc.meta, toolReadModelTextForTest)
			if got := toolReadModelFor(call, "", ""); got != nil {
				t.Fatalf("malformed metadata produced model: %#v", got)
			}
		})
	}
}

func TestToolReadModelRootSettlementAndWireFailures(t *testing.T) {
	cases := []struct {
		name   string
		change func(*session.ToolCallView)
	}{
		{"missing head", func(c *session.ToolCallView) { c.Call = nil }},
		{"missing result", func(c *session.ToolCallView) { c.Result = nil }},
		{"preparing", func(c *session.ToolCallView) { c.State = "preparing" }},
		{"running", func(c *session.ToolCallView) { c.State = "running" }},
		{"failed", func(c *session.ToolCallView) { c.State = "failed" }},
		{"error", func(c *session.ToolCallView) { c.State = "error" }},
		{"stopped", func(c *session.ToolCallView) { c.State = "stopped" }},
		{"canceled", func(c *session.ToolCallView) { c.State = "canceled" }},
		{"cancelled", func(c *session.ToolCallView) { c.State = "cancelled" }},
		{"unknown state", func(c *session.ToolCallView) { c.State = "" }},
		{"parent", func(c *session.ToolCallView) { c.ParentCallID = "parent" }},
		{"root relation implies missing parent", func(c *session.ToolCallView) { c.RootCallID = "other-root" }},
		{"raw PTC despite projected root", func(c *session.ToolCallView) {
			c.Result.EventType = "tool/ptc-dispatch"
			c.Result.Raw = json.RawMessage(strings.Replace(string(c.Result.Raw), `"tool/result"`, `"tool/ptc-dispatch"`, 1))
		}},
		{"event type mismatch", func(c *session.ToolCallView) { c.Result.EventType = "tool/ptc-dispatch" }},
		{"wire failed despite ok state", func(c *session.ToolCallView) {
			c.Result.Raw = json.RawMessage(strings.Replace(string(c.Result.Raw), `"message":{`, `"message":{"isError":true,`, 1))
		}},
		{"wire interrupted despite ok state", func(c *session.ToolCallView) {
			c.Result.Raw = json.RawMessage(strings.Replace(string(c.Result.Raw), `"data":{`, `"data":{"error":{"code":"interrupted"},`, 1))
		}},
	}
	for _, name := range []string{"Read", "read_file", "read_image", "functions.read", "plugin:read", "read ", "", "write", "edit"} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"name/" + name, func(c *session.ToolCallView) { c.Name = name }})
	}
	for _, raw := range []string{"", "{", "null", "[]", `"receipt"`, `{}`, `{"type":"unknown","data":{}}`, `{"type":"tool/result","data":null}`, `{"type":"tool/result","data":[]}`, `{"type":"tool/result","data":{"message":null}}`, `{"type":"tool/result","data":{"message":[]}}`} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"receipt/" + raw, func(c *session.ToolCallView) { c.Result.Raw = json.RawMessage(raw) }})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			call := toolReadModelCallForTest(t, toolReadModelArgsForTest, toolReadModelMetaForTest, toolReadModelTextForTest)
			tc.change(call)
			if got := toolReadModelFor(call, "/work", ""); got != nil {
				t.Fatalf("non-root or unsuccessful call produced model: %#v", got)
			}
		})
	}
	if got := toolReadModelFor(nil, "", ""); got != nil {
		t.Fatalf("nil call produced model: %#v", got)
	}
	call := toolReadModelCallForTest(t, toolReadModelArgsForTest, toolReadModelMetaForTest, toolReadModelTextForTest)
	call.RootCallID, call.Result.EventType = call.CallID, ""
	if got := toolReadModelFor(call, "/work", ""); got == nil || got.Label != "actual.go" {
		t.Fatalf("self root with raw event type should be accepted: %#v", got)
	}
}

func TestToolReadModelSingleTextAndExactEnvelope(t *testing.T) {
	for _, content := range []string{
		`null`, `[]`, `{}`, `"text"`, `[null]`, `[[]]`, `[{}]`,
		`[{"type":"image","text":"ignored"}]`, `[{"type":"Text","text":"ignored"}]`,
		`[{"type":"text"}]`, `[{"type":"text","text":null}]`, `[{"type":"text","text":1}]`,
		`[{"type":"text","text":""},{"type":"text","text":""}]`,
	} {
		t.Run("content/"+content, func(t *testing.T) {
			call := toolReadModelCallForTest(t, toolReadModelArgsForTest, toolReadModelMetaForTest, toolReadModelTextForTest)
			call.Result = toolReadModelReceiptForTest(toolReadModelMetaForTest, content, "")
			if got := toolReadModelFor(call, "", ""); got != nil {
				t.Fatalf("invalid content layout produced model: %#v", got)
			}
		})
	}
	cases := []struct {
		name, text string
		valid      bool
	}{
		{"standard", toolReadModelTextForTest, true},
		{"empty path and body", "<path></path>\n<type>file</type>\n<content>\n\n</content>", true},
		{"CR and unicode separators in path", "<path>a\r\u2028\u2029b</path>\n<type>file</type>\n<content>\nbody\n</content>", true},
		{"path need not be navigation path", "<path>https://example.test/\x00</path>\n<type>file</type>\n<content>\nbody\n</content>", true},
		{"tag text within path", "<path>a</path>tail</path>\n<type>file</type>\n<content>\nbody\n</content>", true},
		{"body includes tags and any character", "<path>a</path>\n<type>file</type>\n<content>\n</content>\n\x00\r\u2028\u2029\n</content>", true},
		{"prefix", "prefix" + toolReadModelTextForTest, false},
		{"leading LF", "\n" + toolReadModelTextForTest, false},
		{"BOM prefix", "\ufeff" + toolReadModelTextForTest, false},
		{"suffix", toolReadModelTextForTest + "suffix", false},
		{"trailing LF", toolReadModelTextForTest + "\n", false},
		{"trailing CR", toolReadModelTextForTest + "\r", false},
		{"trailing CRLF", toolReadModelTextForTest + "\r\n", false},
		{"trailing unicode line separator", toolReadModelTextForTest + "\u2028", false},
		{"trailing unicode paragraph separator", toolReadModelTextForTest + "\u2029", false},
		{"path LF", strings.Replace(toolReadModelTextForTest, "ignored</path>", "ignored\nmore</path>", 1), false},
		{"CRLF structural separators", strings.ReplaceAll(toolReadModelTextForTest, "\n", "\r\n"), false},
		{"missing body separator", "<path>a</path>\n<type>file</type>\n<content>\n</content>", false},
		{"directory", strings.Replace(toolReadModelTextForTest, "<type>file</type>", "<type>directory</type>", 1), false},
		{"type case", strings.Replace(toolReadModelTextForTest, "<type>file</type>", "<type>File</type>", 1), false},
		{"tag whitespace", strings.Replace(toolReadModelTextForTest, "<content>", "<content >", 1), false},
		{"missing path", strings.Replace(toolReadModelTextForTest, "<path>/envelope/ignored</path>\n", "", 1), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			call := toolReadModelCallForTest(t, toolReadModelArgsForTest, toolReadModelMetaForTest, tc.text)
			got := toolReadModelFor(call, "/work", "")
			if (got != nil) != tc.valid {
				t.Fatalf("valid=%v, model=%#v", tc.valid, got)
			}
			if got != nil && !reflect.DeepEqual(got.Lines, []toolReadLine{{10, "first"}, {12, ""}}) {
				t.Fatalf("envelope text replaced metadata lines: %#v", got)
			}
		})
	}
}

func TestToolReadModelCrossOSDisplayLabels(t *testing.T) {
	cases := []struct{ name, path, cwd, home, label string }{
		{"POSIX cwd wins home", "/home/me/work/a", "/home/me/work/", "/home/me", "a"},
		{"POSIX home", "/home/me/other/a", "/work", "/home/me/", "~/other/a"},
		{"exact home", "/home/me///", "", "/home/me", "~"},
		{"home boundary", "/home/member/a", "", "/home/me", "/home/member/a"},
		{"root home", "/etc/a", "", "/", "/etc/a"},
		{"cwd equal", "/work", "/work", "", "/work"},
		{"cwd sibling", "/work-other/a", "/work", "", "/work-other/a"},
		{"root cwd", "/a", "/", "", "a"},
		{"dot segments", "/work/sub/../a", "/work", "", "sub/../a"},
		{"Windows cwd", `C:\Work\sub\a`, `C:\Work\`, "", `sub\a`},
		{"Windows mixed separator boundary", `C:\Work/sub/a`, `C:\Work`, "", "sub/a"},
		{"Windows case", `c:\work\a`, `C:\Work`, "", `c:\work\a`},
		{"Windows separator spelling", "C:/Work/a", `C:\Work`, "", "C:/Work/a"},
		{"Windows home", `C:\Users\me\a`, "", `C:\Users\me`, `C:\Users\me\a`},
		{"Windows home suppresses POSIX abbreviation", "/home/me/a", "", `C:\Users\me`, "/home/me/a"},
		{"UNC cwd", `\\server\share\sub\a`, `\\server\share`, "", `sub\a`},
		{"relative metadata", "relative/../a", "/work", "/home/me", "relative/../a"},
		{"drive relative metadata", `C:a`, "", "", `C:a`},
		{"URL metadata", "https://example.test/a", "", "", "https://example.test/a"},
		{"control metadata", "/work/a\n\x00tail", "/work", "", "a\n\x00tail"},
		{"blank metadata", " \t ", "", "", " \t "},
		{"empty metadata", "", "/work", "/home/me", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pathJSON, err := json.Marshal(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			meta := `{"path":` + string(pathJSON) + `,"offset":1,"lines":[],"totalLines":0}`
			call := toolReadModelCallForTest(t, toolReadModelArgsForTest, meta, toolReadModelTextForTest)
			got := toolReadModelFor(call, tc.cwd, tc.home)
			if got == nil || got.Label != tc.label {
				t.Fatalf("label model = %#v, want label %q", got, tc.label)
			}
		})
	}
}

func TestToolReadModelJavaScriptNumbersAndNativeIntLimit(t *testing.T) {
	bound := math.Ldexp(1, strconv.IntSize-1)
	largest := math.Nextafter(bound, 0)
	if strconv.IntSize == 32 {
		largest = bound - 1
	}
	largestJSON := strconv.FormatFloat(largest, 'f', 0, 64)
	boundJSON := strconv.FormatFloat(bound, 'f', 0, 64)
	meta := `{"path":"large","offset":1,"lines":[{"number":` + largestJSON + `,"text":"last"}],"totalLines":` + largestJSON + `}`
	call := toolReadModelCallForTest(t, toolReadModelArgsForTest, meta, toolReadModelTextForTest)
	got := toolReadModelFor(call, "", "")
	want := &toolReadModel{Label: "large", Lines: []toolReadLine{{int(largest), "last"}}, TotalLines: int(largest)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("largest representable JS integer below int bound: %#v, want %#v", got, want)
	}
	for _, total := range []string{boundJSON, "1e30", strconv.FormatInt(int64(^uint64(0)>>1), 10)} {
		// On 64-bit, the textual MaxInt rounds up to the unrepresentable 2^63.
		call := toolReadModelCallForTest(t, toolReadModelArgsForTest, `{"path":"large","offset":1,"lines":[],"totalLines":`+total+`}`, toolReadModelTextForTest)
		if got := toolReadModelFor(call, "", ""); got != nil {
			t.Fatalf("total %s overflowed native int: %#v", total, got)
		}
	}
	call = toolReadModelCallForTest(t, toolReadModelArgsForTest, `{"path":"offset","offset":1e300,"lines":[],"totalLines":0}`, toolReadModelTextForTest)
	if got := toolReadModelFor(call, "", ""); got == nil {
		t.Fatal("validation-only metadata offset gained native-int limit")
	}
	call = toolReadModelCallForTest(t, toolReadModelArgsForTest, `{"path":"rounded","offset":1.00000000000000001,"lines":[{"number":1.99999999999999999,"text":"two"}],"totalLines":1.99999999999999999}`, toolReadModelTextForTest)
	want = &toolReadModel{Label: "rounded", Lines: []toolReadLine{{2, "two"}}, TotalLines: 2}
	if got := toolReadModelFor(call, "", ""); !reflect.DeepEqual(got, want) {
		t.Fatalf("integer validation must use parsed Number: %#v, want %#v", got, want)
	}
	if strconv.IntSize == 64 {
		call = toolReadModelCallForTest(t, toolReadModelArgsForTest, `{"path":"unsafe","offset":1,"lines":[{"number":9007199254740993,"text":"rounded down"}],"totalLines":9007199254740993}`, toolReadModelTextForTest)
		rounded := int64(9007199254740992)
		want = &toolReadModel{Label: "unsafe", Lines: []toolReadLine{{int(rounded), "rounded down"}}, TotalLines: int(rounded)}
		if got := toolReadModelFor(call, "", ""); !reflect.DeepEqual(got, want) {
			t.Fatalf("Number.isInteger is not Number.isSafeInteger: %#v, want %#v", got, want)
		}
		call = toolReadModelCallForTest(t, toolReadModelArgsForTest, `{"path":"duplicate","offset":1,"lines":[{"number":9007199254740992,"text":"first"},{"number":9007199254740993,"text":"rounded duplicate"}],"totalLines":9007199254740994}`, toolReadModelTextForTest)
		if got := toolReadModelFor(call, "", ""); got != nil {
			t.Fatalf("rounded duplicate line numbers must be rejected: %#v", got)
		}
	}
}

func TestToolReadModelSourceImmutability(t *testing.T) {
	call := toolReadModelCallForTest(t, toolReadModelArgsForTest, toolReadModelMetaForTest, toolReadModelTextForTest)
	call.Events = []session.Message{*call.Call, *call.Result}
	call.Children = []*session.ToolCallView{{CallID: "child", Name: "read", Arguments: `{"file_path":"child"}`}}
	before, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	first := toolReadModelFor(call, "/work", "")
	second := toolReadModelFor(call, "/work", "")
	if first == nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated derivation changed model: %#v, %#v", first, second)
	}
	first.Label, first.TotalLines, first.Lang = "mutated", 0, "mutated"
	first.Lines[0] = toolReadLine{Number: 999, Text: "mutated"}
	first.Lines = append(first.Lines, toolReadLine{Number: 1000, Text: "appended"})
	if got := toolReadModelFor(call, "/work", ""); !reflect.DeepEqual(got, second) {
		t.Fatalf("returned data aliased another model or source: %#v, want %#v", got, second)
	}
	after, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("derivation or returned-model mutation changed source call/events")
	}
	call.Arguments = `{"file_path":null}`
	declinedBefore, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	if got := toolReadModelFor(call, "/work", ""); got != nil {
		t.Fatalf("mutated arguments reused stale successful model: %#v", got)
	}
	declinedAfter, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	if string(declinedBefore) != string(declinedAfter) {
		t.Fatal("declining a model changed source call/events")
	}
}
