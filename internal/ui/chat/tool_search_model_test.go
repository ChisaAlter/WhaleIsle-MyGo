package chat

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

const (
	toolSearchModelGlobMetaForTest = `{"shape":"paths","paths":[],"total":0,"truncated":false}`
	toolSearchModelGrepMetaForTest = `{"shape":"matches","files":[],"total":0,"truncated":false}`
)

func toolSearchModelCallForTest(name, args, meta, content string) *session.ToolCallView {
	return &session.ToolCallView{
		CallID: "search-120", Name: name, Arguments: args, State: "ok",
		Call: &session.Message{
			EventType: "tool/call", Raw: json.RawMessage(`{"type":"tool/call","data":{"callId":"search-120"}}`),
			Body: "unrelated flattened head", Args: `{"pattern":"wrong-head"}`,
		},
		Result: toolSearchModelReceiptForTest(meta, content, ""),
	}
}

func toolSearchModelReceiptForTest(meta, content, extra string) *session.Message {
	return &session.Message{
		Kind: session.MessageKindTool, EventType: "tool/result",
		Raw:  json.RawMessage(`{"type":"tool/result","data":{"meta":` + meta + `,"message":{"content":` + content + `}` + extra + `}}`),
		Body: "unrelated flattened result", Args: `{"pattern":"wrong-result"}`,
	}
}

func toolSearchModelJSONForTest(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// These goldens were recorded by executing the CURRENT installed modules in an
// isolated node:vm SourceTextModule, with no UI/browser or reimplementation:
// search-card-model.js SHA256 83e6591201af2bf04f3dfb0c6f193d16927cf41581c27044d31434a45df592b1
// raw-tool-call.js SHA256 c97f8fa5e1b85e63b2bb4c65c1b7bbb265793dd0631ec0a3f32c38b3c15b44a8
// Recorded card/recovery fields are represented directly; absent recovery is "".
// The tests themselves need no installed source, Node runtime, filesystem or host.
func TestToolSearchModelInstalledGoldens(t *testing.T) {
	cases := []struct {
		name, tool, args, meta, content string
		want                            *toolSearchModel
	}{
		{
			"arbitrary paths and duplicate ordering", "glob", `{"pattern":"**/*"}`,
			`{"shape":"paths","paths":[""," \t ","a/../b","https://example.test/\u0000\n","dup","dup"],"total":0,"truncated":false}`,
			`[{"type":"text","text":"ignored"}]`,
			&toolSearchModel{Kind: "paths", Paths: []string{"", " \t ", "a/../b", "https://example.test/\x00\n", "dup", "dup"}},
		},
		{
			"unordered duplicate matches and inconsistent count", "grep", `{"pattern":"x","include":"*.{go,{js,ts}}"}`,
			`{"shape":"matches","files":[{"path":"","matches":[{"lineNumber":9,"line":"before"},{"lineNumber":2,"line":""},{"lineNumber":2,"line":"dup"}]},{"path":"https://example.test/\u0000","matches":[]}],"total":0,"truncated":false}`,
			`[]`,
			&toolSearchModel{Kind: "matches", Files: []toolSearchFile{
				{Path: "", Matches: []toolSearchMatch{{9, "before"}, {2, ""}, {2, "dup"}}},
				{Path: "https://example.test/\x00", Matches: []toolSearchMatch{}},
			}},
		},
		{
			"no native integer bound", "grep", `{"pattern":"x"}`,
			`{"shape":"matches","files":[{"path":"huge","matches":[{"lineNumber":1e300,"line":"huge"}]}],"total":1e300,"truncated":false}`,
			`[]`, &toolSearchModel{Kind: "matches", Files: []toolSearchFile{{"huge", []toolSearchMatch{{1e300, "huge"}}}}, Total: 1e300},
		},
		{
			"IEEE rounding", "grep", `{"pattern":"x"}`,
			`{"shape":"matches","files":[{"path":"rounded","matches":[{"lineNumber":9007199254740993,"line":"unsafe"},{"lineNumber":1.99999999999999999,"line":"two"}]}],"total":1.00000000000000001,"truncated":false}`,
			`[]`, &toolSearchModel{Kind: "matches", Files: []toolSearchFile{{"rounded", []toolSearchMatch{{9007199254740992, "unsafe"}, {2, "two"}}}}, Total: 1},
		},
		{
			"underflow total", "glob", `{"pattern":"x"}`,
			`{"shape":"paths","paths":[],"total":-1e-9999,"truncated":false}`,
			`[]`, &toolSearchModel{Kind: "paths", Paths: []string{}},
		},
		{
			"multi-block recovery", "glob", `{"pattern":"x"}`,
			`{"shape":"paths","paths":[],"total":0,"truncated":true}`,
			`[{"type":"text","text":""},{"type":"image","text":"no"},{"type":"text","text":"recover\r\n\u0000"},{"type":"text","text":1},{"type":"Text","text":"no"},{"type":"text","text":""},{"type":"text","text":"tail"}]`,
			&toolSearchModel{Kind: "paths", Paths: []string{}, Truncated: true, Recovery: "\nrecover\r\n\x00\n\ntail"},
		},
		{
			"two empty text blocks retain LF", "grep", `{"pattern":"x"}`,
			`{"shape":"matches","files":[],"total":0,"truncated":true}`,
			`[{"type":"text","text":""},{"type":"text","text":""}]`,
			&toolSearchModel{Kind: "matches", Files: []toolSearchFile{}, Truncated: true, Recovery: "\n"},
		},
		{
			"single empty text gives no recovery", "glob", `{"pattern":"x"}`,
			`{"shape":"paths","paths":[],"total":0,"truncated":true}`,
			`[{"type":"text","text":""}]`, &toolSearchModel{Kind: "paths", Paths: []string{}, Truncated: true},
		},
		{
			"grep JavaScript blank pattern accepted", "grep", `{"pattern":" \u00a0\ufeff"}`,
			toolSearchModelGrepMetaForTest, `[]`, &toolSearchModel{Kind: "matches", Files: []toolSearchFile{}},
		},
		{
			"glob JavaScript blank pattern declined", "glob", `{"pattern":" \u00a0\ufeff"}`,
			toolSearchModelGlobMetaForTest, `[]`, nil,
		},
		{
			"NEL is not trim whitespace", "glob", `{"pattern":"\u0085","path":"\u0085"}`,
			toolSearchModelGlobMetaForTest, `[]`, &toolSearchModel{Kind: "paths", Paths: []string{}},
		},
		{
			"space before literal exclamation accepted", "grep", `{"pattern":"x","include":" !*.go"}`,
			toolSearchModelGrepMetaForTest, `[]`, &toolSearchModel{Kind: "matches", Files: []toolSearchFile{}},
		},
		{
			"unclosed brace permits inner comma", "grep", `{"pattern":"x","include":"{a,b"}`,
			toolSearchModelGrepMetaForTest, `[]`, &toolSearchModel{Kind: "matches", Files: []toolSearchFile{}},
		},
		{
			"closing brace resets depth", "grep", `{"pattern":"x","include":"}a,b"}`,
			toolSearchModelGrepMetaForTest, `[]`, nil,
		},
		{
			"escaped outer comma still declined", "grep", `{"pattern":"x","include":"a\\,b"}`,
			toolSearchModelGrepMetaForTest, `[]`, nil,
		},
		{
			"glob ignores wrong include type", "glob", `{"pattern":"x","include":null}`,
			toolSearchModelGlobMetaForTest, `[]`, &toolSearchModel{Kind: "paths", Paths: []string{}},
		},
		{
			"duplicate keys last value", "grep", `{"pattern":null,"pattern":"x","include":null,"include":"*.go"}`,
			`{"shape":"paths","shape":"matches","files":null,"files":[{"path":null,"path":"last","matches":[{"lineNumber":0,"lineNumber":1e1,"line":null,"line":""}]}],"total":null,"total":10.0,"truncated":true,"truncated":false}`,
			`[]`, &toolSearchModel{Kind: "matches", Files: []toolSearchFile{{"last", []toolSearchMatch{{10, ""}}}}, Total: 10},
		},
		{
			"overflow total declined", "glob", `{"pattern":"x"}`,
			`{"shape":"paths","paths":[],"total":1e9999,"truncated":false}`, `[]`, nil,
		},
		{
			"overflow line declined", "grep", `{"pattern":"x"}`,
			`{"shape":"matches","files":[{"path":"a","matches":[{"lineNumber":1e9999,"line":""}]}],"total":0,"truncated":false}`, `[]`, nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			call := toolSearchModelCallForTest(tc.tool, tc.args, tc.meta, tc.content)
			if got := toolSearchModelFor(call); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("model = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestToolSearchModelCallValidation(t *testing.T) {
	cases := []struct {
		name, args string
		grep, glob bool
	}{
		{"minimum", `{"pattern":"x"}`, true, true},
		{"empty pattern", `{"pattern":""}`, false, false},
		{"JS whitespace pattern", `{"pattern":" \t\r\n\u00a0\ufeff"}`, true, false},
		{"NEL pattern", `{"pattern":"\u0085"}`, true, true},
		{"control URL pattern and path", `{"pattern":"https://example.test/\n\u0000","path":"https://example.test/\n\u0000"}`, true, true},
		{"NEL path", `{"pattern":"x","path":"\u0085"}`, true, true},
		{"unrelated extension", `{"pattern":"x","extra":1e9999}`, true, true},
		{"duplicate final valid fields", `{"pattern":null,"pattern":"x","path":null,"path":"a"}`, true, true},
		{"duplicate final invalid pattern", `{"pattern":"x","pattern":null}`, false, false},
		{"missing", ``, false, false},
		{"incomplete", `{`, false, false},
		{"null", `null`, false, false},
		{"array", `[{"pattern":"x"}]`, false, false},
		{"string", `"pattern"`, false, false},
		{"boolean", `true`, false, false},
		{"number", `1`, false, false},
		{"missing pattern", `{}`, false, false},
		{"wrong case", `{"Pattern":"x"}`, false, false},
		{"trailing JSON", `{"pattern":"x"} {}`, false, false},
		{"trailing comma", `{"pattern":"x",}`, false, false},
	}
	for _, value := range []string{`null`, `1`, `true`, `[]`, `{}`} {
		cases = append(cases, struct {
			name, args string
			grep, glob bool
		}{"pattern/" + value, `{"pattern":` + value + `}`, false, false})
	}
	for _, value := range []string{`null`, `1`, `true`, `[]`, `{}`, `""`, `" \t\u00a0\ufeff"`} {
		cases = append(cases, struct {
			name, args string
			grep, glob bool
		}{"path/" + value, `{"pattern":"x","path":` + value + `}`, false, false})
	}
	for _, tc := range cases {
		for _, tool := range []string{"grep", "glob"} {
			t.Run(tool+"/"+tc.name, func(t *testing.T) {
				meta, valid := toolSearchModelGlobMetaForTest, tc.glob
				if tool == "grep" {
					meta, valid = toolSearchModelGrepMetaForTest, tc.grep
				}
				got := toolSearchModelFor(toolSearchModelCallForTest(tool, tc.args, meta, `[]`))
				if (got != nil) != valid {
					t.Fatalf("valid=%v, model=%#v", valid, got)
				}
			})
		}
	}
}

func TestToolSearchModelIncludeBraceDepthAndLiteralPrefix(t *testing.T) {
	valid := []string{
		"*.go", "*.{js,ts}", "*.{go,{js,ts}}", "{a,b", "{{a,b}", "a}", "}", "}}{a,b}",
		" !*.go", "\u00a0!*.go", "\ufeff!*.go", "{!a,b}", "\\!*.go", "[!a]*", "[abc]", "\u0085", "\x00", "https://example.test/*.go",
	}
	invalid := []string{
		"", " \t\r\n", "\u00a0\ufeff", "!*.go", "!{a,b}", "!", "a,b", "*.go,*.js", "}a,b", "{a,b},c", "{{a,b}},c", "a\\,b", "[a,b]", ",", "{a}b,c",
	}
	for _, tc := range []struct {
		values []string
		valid  bool
	}{{valid, true}, {invalid, false}} {
		for _, include := range tc.values {
			t.Run(toolSearchModelJSONForTest(t, include), func(t *testing.T) {
				args := toolSearchModelJSONForTest(t, map[string]any{"pattern": "x", "include": include})
				got := toolSearchModelFor(toolSearchModelCallForTest("grep", args, toolSearchModelGrepMetaForTest, `[]`))
				if (got != nil) != tc.valid {
					t.Fatalf("include=%q valid=%v, model=%#v", include, tc.valid, got)
				}
			})
		}
	}
	for _, value := range []string{`null`, `1`, `true`, `[]`, `{}`} {
		t.Run("wrong type/"+value, func(t *testing.T) {
			args := `{"pattern":"x","include":` + value + `}`
			if got := toolSearchModelFor(toolSearchModelCallForTest("grep", args, toolSearchModelGrepMetaForTest, `[]`)); got != nil {
				t.Fatalf("grep accepted non-string include: %#v", got)
			}
			if got := toolSearchModelFor(toolSearchModelCallForTest("glob", args, toolSearchModelGlobMetaForTest, `[]`)); got == nil {
				t.Fatal("glob validated a grep-only include")
			}
		})
	}
}

func TestToolSearchModelJavaScriptTrimWhitespace(t *testing.T) {
	for _, whitespace := range []string{"\t", "\n", "\v", "\f", "\r", " ", "\u00a0", "\u1680", "\u2000", "\u2001", "\u2002", "\u2003", "\u2004", "\u2005", "\u2006", "\u2007", "\u2008", "\u2009", "\u200a", "\u2028", "\u2029", "\u202f", "\u205f", "\u3000", "\ufeff"} {
		t.Run(toolSearchModelJSONForTest(t, whitespace), func(t *testing.T) {
			for _, key := range []string{"pattern", "path", "include"} {
				args := toolSearchModelJSONForTest(t, map[string]any{"pattern": "x", key: whitespace})
				glob := toolSearchModelFor(toolSearchModelCallForTest("glob", args, toolSearchModelGlobMetaForTest, `[]`))
				grep := toolSearchModelFor(toolSearchModelCallForTest("grep", args, toolSearchModelGrepMetaForTest, `[]`))
				if (glob != nil) != (key == "include") || (grep != nil) != (key == "pattern") {
					t.Fatalf("whitespace=%q field=%s, glob=%#v grep=%#v", whitespace, key, glob, grep)
				}
			}
		})
	}
	for _, literal := range []string{"\u0085", "\u180e", "\u200b", "\u0000"} {
		args := toolSearchModelJSONForTest(t, map[string]any{"pattern": literal, "path": literal, "include": literal})
		for _, tc := range []struct{ tool, meta string }{{"grep", toolSearchModelGrepMetaForTest}, {"glob", toolSearchModelGlobMetaForTest}} {
			if got := toolSearchModelFor(toolSearchModelCallForTest(tc.tool, args, tc.meta, `[]`)); got == nil {
				t.Fatalf("%s treated %q as JavaScript trim whitespace", tc.tool, literal)
			}
		}
	}
}

func TestToolSearchModelDeclinesMalformedMetadata(t *testing.T) {
	cases := []struct{ name, tool, meta string }{}
	for _, tool := range []string{"glob", "grep"} {
		for _, meta := range []string{`null`, `[]`, `"meta"`, `true`, `1`, `{}`} {
			cases = append(cases, struct{ name, tool, meta string }{"object/" + meta, tool, meta})
		}
		base := toolSearchModelGlobMetaForTest
		arrayKey := "paths"
		if tool == "grep" {
			base, arrayKey = toolSearchModelGrepMetaForTest, "files"
		}
		for _, key := range []string{"shape", "truncated", "total", arrayKey} {
			meta := map[string]json.RawMessage{}
			if err := json.Unmarshal([]byte(base), &meta); err != nil {
				t.Fatal(err)
			}
			delete(meta, key)
			cases = append(cases, struct{ name, tool, meta string }{"missing/" + key, tool, toolSearchModelJSONForTest(t, meta)})
		}
		for _, field := range []struct {
			key    string
			values []string
		}{
			{"shape", []string{`null`, `1`, `true`, `[]`, `{}`, `""`, `"Paths"`, `"unknown"`}},
			{"truncated", []string{`null`, `0`, `1`, `"false"`, `[]`, `{}`}},
			{"total", []string{`null`, `"0"`, `true`, `[]`, `{}`, `-1`, `0.5`, `1.1`, `1e9999`, `-1e9999`}},
			{arrayKey, []string{`null`, `1`, `true`, `"rows"`, `{}`, `[null]`, `[1]`, `[true]`, `[[]]`, `[{}]`}},
		} {
			for _, value := range field.values {
				meta := map[string]json.RawMessage{}
				if err := json.Unmarshal([]byte(base), &meta); err != nil {
					t.Fatal(err)
				}
				meta[field.key] = json.RawMessage(value)
				cases = append(cases, struct{ name, tool, meta string }{field.key + "/" + value, tool, toolSearchModelJSONForTest(t, meta)})
			}
		}
	}
	cases = append(cases,
		struct{ name, tool, meta string }{"swapped shape", "glob", toolSearchModelGrepMetaForTest},
		struct{ name, tool, meta string }{"swapped shape", "grep", toolSearchModelGlobMetaForTest},
		struct{ name, tool, meta string }{"correct array wrong shape", "glob", `{"shape":"matches","paths":[],"total":0,"truncated":false}`},
		struct{ name, tool, meta string }{"correct array wrong shape", "grep", `{"shape":"paths","files":[],"total":0,"truncated":false}`},
	)
	for _, file := range []string{
		`{"matches":[]}`, `{"path":"a"}`, `{"path":null,"matches":[]}`, `{"path":1,"matches":[]}`,
		`{"path":[],"matches":[]}`, `{"path":{},"matches":[]}`, `{"path":true,"matches":[]}`,
		`{"path":"a","matches":null}`, `{"path":"a","matches":{}}`, `{"path":"a","matches":"rows"}`, `{"path":"a","matches":true}`,
	} {
		cases = append(cases, struct{ name, tool, meta string }{"file/" + file, "grep", `{"shape":"matches","files":[` + file + `],"total":0,"truncated":false}`})
	}
	for _, match := range []string{
		`null`, `[]`, `1`, `true`, `"match"`, `{}`, `{"line":"missing number"}`, `{"lineNumber":1}`,
		`{"lineNumber":null,"line":""}`, `{"lineNumber":"1","line":""}`, `{"lineNumber":true,"line":""}`, `{"lineNumber":{},"line":""}`,
		`{"lineNumber":0,"line":""}`, `{"lineNumber":-0,"line":""}`, `{"lineNumber":-1,"line":""}`, `{"lineNumber":1.5,"line":""}`,
		`{"lineNumber":1e9999,"line":""}`, `{"lineNumber":1e-9999,"line":""}`,
		`{"lineNumber":1,"line":null}`, `{"lineNumber":1,"line":1}`, `{"lineNumber":1,"line":true}`, `{"lineNumber":1,"line":[]}`, `{"lineNumber":1,"line":{}}`,
	} {
		cases = append(cases, struct{ name, tool, meta string }{"match/" + match, "grep", `{"shape":"matches","files":[{"path":"a","matches":[` + match + `]}],"total":0,"truncated":false}`})
	}
	for _, tc := range cases {
		t.Run(tc.tool+"/"+tc.name, func(t *testing.T) {
			if got := toolSearchModelFor(toolSearchModelCallForTest(tc.tool, `{"pattern":"x"}`, tc.meta, `[]`)); got != nil {
				t.Fatalf("malformed metadata produced model: %#v", got)
			}
		})
	}
}

func TestToolSearchModelNumbersRowsAndRecoveryAreIndependent(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want float64
	}{
		{"1e1", 10}, {"10.0", 10}, {"1.00000000000000001", 1}, {"0.99999999999999999", 1},
		{"1.99999999999999999", 2}, {"9007199254740993", 9007199254740992},
		{"9223372036854775807", 9223372036854775808}, {"1e300", 1e300}, {"1.7976931348623157e308", 1.7976931348623157e308},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			meta := `{"shape":"matches","files":[{"path":"","matches":[{"lineNumber":` + tc.raw + `,"line":"a\r\n\u0000\u2028b"},{"lineNumber":1,"line":""},{"lineNumber":` + tc.raw + `,"line":"duplicate"}]},{"path":"","matches":[]}],"total":` + tc.raw + `,"truncated":false,"extra":1e9999}`
			got := toolSearchModelFor(toolSearchModelCallForTest("grep", `{"pattern":"x"}`, meta, `[{"type":"text","text":"not matched against metadata"}]`))
			want := &toolSearchModel{Kind: "matches", Files: []toolSearchFile{
				{"", []toolSearchMatch{{tc.want, "a\r\n\x00\u2028b"}, {1, ""}, {tc.want, "duplicate"}}},
				{"", []toolSearchMatch{}},
			}, Total: tc.want}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Number or row normalization changed model: %#v, want %#v", got, want)
			}
		})
	}
	for _, raw := range []string{"0", "-0.0", "1e-9999", "-1e-9999"} {
		meta := `{"shape":"paths","paths":["b","a","b"],"total":` + raw + `,"truncated":false}`
		got := toolSearchModelFor(toolSearchModelCallForTest("glob", `{"pattern":"x"}`, meta, `[]`))
		if got == nil || got.Total != 0 || !reflect.DeepEqual(got.Paths, []string{"b", "a", "b"}) {
			t.Fatalf("zero/underflow total %s or inconsistent rows declined: %#v", raw, got)
		}
	}
	contents := []struct{ raw, recovery string }{
		{`[]`, ""}, {`[{"type":"image","text":"ignored"}]`, ""},
		{`[{"type":"text","text":null},{"type":"text"},{"type":"text","text":false},{"type":"text","text":1}]`, ""},
		{`[{"type":"text","text":" \t\ufeff"}]`, " \t\ufeff"},
		{`[{"type":"text","text":"left\n"},{"type":"Text","text":"ignored"},{"type":"text","text":"right"}]`, "left\n\nright"},
	}
	for _, content := range contents {
		for _, truncated := range []bool{false, true} {
			meta := `{"shape":"matches","files":[{"path":"independent","matches":[{"lineNumber":3,"line":"actual metadata"}]}],"total":999,"truncated":` + toolSearchModelJSONForTest(t, truncated) + `}`
			got := toolSearchModelFor(toolSearchModelCallForTest("grep", `{"pattern":"x"}`, meta, content.raw))
			wantRecovery := ""
			if truncated {
				wantRecovery = content.recovery
			}
			if got == nil || got.Recovery != wantRecovery || got.Total != 999 || !reflect.DeepEqual(got.Files, []toolSearchFile{{"independent", []toolSearchMatch{{3, "actual metadata"}}}}) {
				t.Fatalf("content=%s truncated=%v, model=%#v, want recovery=%q", content.raw, truncated, got, wantRecovery)
			}
		}
	}
}

func TestToolSearchModelSuccessfulSettledRootReceiptOnly(t *testing.T) {
	cases := []struct {
		name   string
		change func(*session.ToolCallView)
	}{
		{"missing retained head", func(c *session.ToolCallView) { c.Call = nil }},
		{"missing result", func(c *session.ToolCallView) { c.Result = nil }},
		{"projected parent", func(c *session.ToolCallView) { c.ParentCallID = "parent" }},
		{"projected other root", func(c *session.ToolCallView) { c.RootCallID = "other-root" }},
		{"raw PTC despite root projection", func(c *session.ToolCallView) {
			c.Result.EventType = "tool/ptc-dispatch"
			c.Result.Raw = json.RawMessage(strings.Replace(string(c.Result.Raw), `"tool/result"`, `"tool/ptc-dispatch"`, 1))
		}},
		{"event type mismatch", func(c *session.ToolCallView) { c.Result.EventType = "tool/ptc-dispatch" }},
		{"wire failed despite ok state", func(c *session.ToolCallView) {
			c.Result.Raw = json.RawMessage(strings.Replace(string(c.Result.Raw), `"message":{`, `"message":{"isError":true,`, 1))
		}},
		{"wire interrupted despite ok state", func(c *session.ToolCallView) {
			c.Result = toolSearchModelReceiptForTest(toolSearchModelGlobMetaForTest, `[]`, `,"error":{"code":"interrupted"}`)
		}},
	}
	for _, state := range []string{"preparing", "running", "error", "failed", "stopped", "cancelled", "canceled", "OK", "success", ""} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"state/" + state, func(c *session.ToolCallView) { c.State = state }})
	}
	for _, name := range []string{"Glob", "Grep", "glob ", "grep ", "functions.glob", "functions.grep", "plugin:glob", "read", "search", ""} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"name/" + name, func(c *session.ToolCallView) { c.Name = name }})
	}
	for _, raw := range []string{"", "{", "null", "[]", `"receipt"`, "true", "1", `{}`, `{"type":"unknown","data":{}}`, `{"type":"tool/result","data":null}`, `{"type":"tool/result","data":[]}`} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"receipt/" + raw, func(c *session.ToolCallView) { c.Result.Raw = json.RawMessage(raw) }})
	}
	// Keep valid metadata alongside malformed data.message so metadata validity
	// cannot accidentally accept a non-message root receipt.
	for _, message := range []string{"", `null`, `[]`, `"message"`, `true`, `1`} {
		raw := `{"type":"tool/result","data":{"meta":` + toolSearchModelGlobMetaForTest
		if message != "" {
			raw += `,"message":` + message
		}
		raw += `}}`
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"data.message/" + message, func(c *session.ToolCallView) { c.Result.Raw = json.RawMessage(raw) }})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			call := toolSearchModelCallForTest("glob", `{"pattern":"x"}`, toolSearchModelGlobMetaForTest, `[]`)
			tc.change(call)
			if got := toolSearchModelFor(call); got != nil {
				t.Fatalf("non-root or unsuccessful call produced model: %#v", got)
			}
		})
	}
	if got := toolSearchModelFor(nil); got != nil {
		t.Fatalf("nil call produced model: %#v", got)
	}
	call := toolSearchModelCallForTest("glob", `{"pattern":"x"}`, toolSearchModelGlobMetaForTest, `[]`)
	call.RootCallID, call.Result.EventType = call.CallID, ""
	if got := toolSearchModelFor(call); got == nil || got.Kind != "paths" {
		t.Fatalf("self root with raw receipt type should be accepted: %#v", got)
	}
}

func TestToolSearchModelSourceImmutabilityAndIndependentSlices(t *testing.T) {
	for _, tc := range []struct{ tool, meta string }{
		{"glob", `{"shape":"paths","paths":["first","second"],"total":5,"truncated":true}`},
		{"grep", `{"shape":"matches","files":[{"path":"first","matches":[{"lineNumber":3,"line":"one"},{"lineNumber":1,"line":"two"}]},{"path":"second","matches":[]}],"total":5,"truncated":true}`},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			call := toolSearchModelCallForTest(tc.tool, `{"pattern":"x"}`, tc.meta, `[{"type":"text","text":"recovery"}]`)
			call.Events = []session.Message{*call.Call, *call.Result}
			call.Children = []*session.ToolCallView{{CallID: "child", Name: "grep", Arguments: `{"pattern":"child"}`}}
			before := toolSearchModelJSONForTest(t, call)
			first, second := toolSearchModelFor(call), toolSearchModelFor(call)
			if first == nil || !reflect.DeepEqual(first, second) {
				t.Fatalf("repeated derivation changed model: %#v, %#v", first, second)
			}
			originalSecond := toolSearchModelJSONForTest(t, second)
			first.Kind, first.Total, first.Truncated, first.Recovery = "changed", 0, false, "changed"
			if tc.tool == "glob" {
				first.Paths[0] = "changed"
				first.Paths = append(first.Paths, "appended")
			} else {
				first.Files[0].Path = "changed"
				first.Files[0].Matches[0] = toolSearchMatch{999, "changed"}
				first.Files[0].Matches = append(first.Files[0].Matches, toolSearchMatch{1000, "appended"})
				first.Files[1].Matches = append(first.Files[1].Matches, toolSearchMatch{1, "was empty"})
				first.Files = append(first.Files, toolSearchFile{Path: "appended", Matches: []toolSearchMatch{}})
			}
			if toolSearchModelJSONForTest(t, second) != originalSecond {
				t.Fatal("returned slices aliased an independently derived model")
			}
			if got := toolSearchModelFor(call); !reflect.DeepEqual(got, second) {
				t.Fatalf("model mutation affected later derivation: %#v, want %#v", got, second)
			}
			if after := toolSearchModelJSONForTest(t, call); after != before {
				t.Fatal("derivation or returned-model mutation changed source call/events")
			}
			call.Arguments = `{"pattern":null}`
			declinedBefore := toolSearchModelJSONForTest(t, call)
			if got := toolSearchModelFor(call); got != nil {
				t.Fatalf("changed invalid arguments reused stale model: %#v", got)
			}
			if after := toolSearchModelJSONForTest(t, call); after != declinedBefore {
				t.Fatal("declining a model changed source call/events")
			}
		})
	}
}
