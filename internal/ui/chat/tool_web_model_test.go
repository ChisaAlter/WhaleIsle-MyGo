package chat

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

const (
	toolWebModelSearchMetaForTest = `{"sources":[],"truncated":false}`
	toolWebModelFetchMetaForTest  = `{"url":"result","statusCode":200,"truncated":false}`
)

func toolWebModelCallForTest(name, args, meta, content string) *session.ToolCallView {
	return &session.ToolCallView{
		CallID: "web-121", Name: name, Arguments: args, State: "ok",
		Call: &session.Message{
			EventType: "tool/call", Raw: json.RawMessage(`{"type":"tool/call","data":{"callId":"web-121"}}`),
			Body: "unrelated flattened head", Args: `{"url":"wrong-head","queries":["wrong-head"]}`,
		},
		Result: toolWebModelReceiptForTest(meta, content, ""),
	}
}

func toolWebModelReceiptForTest(meta, content, extra string) *session.Message {
	return &session.Message{
		Kind: session.MessageKindTool, EventType: "tool/result",
		Raw:  json.RawMessage(`{"type":"tool/result","data":{"meta":` + meta + `,"message":{"content":` + content + `}` + extra + `}}`),
		Body: "unrelated flattened result", Args: `{"url":"wrong-result","queries":["wrong-result"]}`,
	}
}

func toolWebModelJSONForTest(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// These goldens were recorded by executing the CURRENT installed modules in an
// isolated node:vm SourceTextModule, with no browser or model reimplementation:
// web-card-model.js SHA256 b4616554642998851a5e00e3239e99cd0a0d04ae557c5d1b08379323538b24dd
// raw-tool-call.js SHA256 c97f8fa5e1b85e63b2bb4c65c1b7bbb265793dd0631ec0a3f32c38b3c15b44a8
// Absent optional strings are represented as "", equivalent in the card UI.
// The tests need no installed source, Node runtime, filesystem or host access.
func TestToolWebModelInstalledGoldens(t *testing.T) {
	cases := []struct {
		name, tool, args, meta string
		want                   *toolWebModel
	}{
		{"minimal search", "web_search", `{"queries":["x"]}`, toolWebModelSearchMetaForTest, &toolWebModel{Kind: "search", Sources: []toolWebSource{}}},
		{
			"absent and empty optional strings", "web_search", `{"queries":["x"]}`,
			`{"sources":[{"url":""},{"url":"","title":"","snippet":"","publishedAt":""}],"answer":"","truncated":false}`,
			&toolWebModel{Kind: "search", Sources: []toolWebSource{{}, {}}},
		},
		{
			"unordered duplicate unsafe sources", "web_search", `{"queries":["one","two","three","four","five"]}`,
			`{"sources":[{"url":"javascript:alert(1)","title":"Z","snippet":"\r\n\u0000","publishedAt":"not a date"},{"url":""},{"url":"javascript:alert(1)","title":"A"}],"answer":" \ufeff\nanswer ","truncated":true}`,
			&toolWebModel{Kind: "search", Answer: " \ufeff\nanswer ", Truncated: true, Sources: []toolWebSource{
				{URL: "javascript:alert(1)", Title: "Z", Snippet: "\r\n\x00", PublishedAt: "not a date"}, {}, {URL: "javascript:alert(1)", Title: "A"},
			}},
		},
		{"fetch empty result URL", "web_fetch", `{"url":"javascript:alert(1)"}`, `{"url":"","statusCode":-404,"truncated":true}`, &toolWebModel{Kind: "fetch", StatusCode: -404, Truncated: true}},
		{"fetch unsafe result URL", "web_fetch", `{"url":"relative"}`, `{"url":"javascript:alert(1)","statusCode":700,"truncated":false}`, &toolWebModel{Kind: "fetch", URL: "javascript:alert(1)", StatusCode: 700}},
		{"fetch huge negative integer", "web_fetch", `{"url":"x"}`, `{"url":"x","statusCode":-1e300,"truncated":false}`, &toolWebModel{Kind: "fetch", URL: "x", StatusCode: -1e300}},
		{"fetch huge positive integer", "web_fetch", `{"url":"x"}`, `{"url":"x","statusCode":1e300,"truncated":false}`, &toolWebModel{Kind: "fetch", URL: "x", StatusCode: 1e300}},
		{"fetch unsafe rounded integer", "web_fetch", `{"url":"x"}`, `{"url":"x","statusCode":-9007199254740993,"truncated":false}`, &toolWebModel{Kind: "fetch", URL: "x", StatusCode: -9007199254740992}},
		{"fetch rounded fraction", "web_fetch", `{"url":"x"}`, `{"url":"x","statusCode":-1.99999999999999999,"truncated":false}`, &toolWebModel{Kind: "fetch", URL: "x", StatusCode: -2}},
		{"fetch negative underflow", "web_fetch", `{"url":"x"}`, `{"url":"x","statusCode":-1e-9999,"truncated":false}`, &toolWebModel{Kind: "fetch", URL: "x"}},
		{"fetch positive underflow", "web_fetch", `{"url":"x"}`, `{"url":"x","statusCode":1e-9999,"truncated":false}`, &toolWebModel{Kind: "fetch", URL: "x"}},
		{"fetch overflow declined", "web_fetch", `{"url":"x"}`, `{"url":"x","statusCode":1e9999,"truncated":false}`, nil},
		{"fetch fraction declined", "web_fetch", `{"url":"x"}`, `{"url":"x","statusCode":-0.5,"truncated":false}`, nil},
		{"search NBSP BOM blank declined", "web_search", `{"queries":[" \u00a0\ufeff"]}`, toolWebModelSearchMetaForTest, nil},
		{"fetch NBSP BOM blank declined", "web_fetch", `{"url":" \u00a0\ufeff"}`, toolWebModelFetchMetaForTest, nil},
		{"search NEL retained", "web_search", `{"queries":["\u0085"]}`, toolWebModelSearchMetaForTest, &toolWebModel{Kind: "search", Sources: []toolWebSource{}}},
		{"fetch NEL retained", "web_fetch", `{"url":"\u0085"}`, toolWebModelFetchMetaForTest, &toolWebModel{Kind: "fetch", URL: "result", StatusCode: 200}},
		{"search null optional title declined", "web_search", `{"queries":["x"]}`, `{"sources":[{"url":"x","title":null}],"truncated":false}`, nil},
		{"search null optional answer declined", "web_search", `{"queries":["x"]}`, `{"sources":[],"answer":null,"truncated":false}`, nil},
		{"search null source declined", "web_search", `{"queries":["x"]}`, `{"sources":[null],"truncated":false}`, nil},
		{"search array source declined", "web_search", `{"queries":["x"]}`, `{"sources":[[]],"truncated":false}`, nil},
		{"search array queries only", "web_search", `{"queries":"x"}`, toolWebModelSearchMetaForTest, nil},
		{"search full JSON required", "web_search", `{"queries":["x"]} {}`, toolWebModelSearchMetaForTest, nil},
		{"fetch case sensitive", "Web_fetch", `{"url":"x"}`, toolWebModelFetchMetaForTest, nil},
		{
			"duplicate last values", "web_search", `{"queries":null,"queries":["x"]}`,
			`{"sources":null,"sources":[{"url":null,"url":"last","title":null,"title":""}],"answer":null,"answer":"last answer","truncated":true,"truncated":false}`,
			&toolWebModel{Kind: "search", Answer: "last answer", Sources: []toolWebSource{{URL: "last"}}},
		},
		{"search ignores fetch-only metadata", "web_search", `{"queries":["x"],"url":null}`, `{"sources":[],"url":null,"statusCode":null,"truncated":false}`, &toolWebModel{Kind: "search", Sources: []toolWebSource{}}},
		{"fetch ignores search-only metadata", "web_fetch", `{"url":"x","queries":null}`, `{"url":"result","statusCode":0,"sources":null,"answer":null,"truncated":false}`, &toolWebModel{Kind: "fetch", URL: "result"}},
		{"search ignored overflow extension", "web_search", `{"queries":["x"],"extra":1e9999}`, `{"sources":[],"truncated":false,"extra":1e9999}`, &toolWebModel{Kind: "search", Sources: []toolWebSource{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			call := toolWebModelCallForTest(tc.tool, tc.args, tc.meta, `[{"type":"text","text":"unrelated recovery text"}]`)
			if got := toolWebModelFor(call); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("model = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestToolWebModelCallValidation(t *testing.T) {
	for _, tool := range []string{"web_search", "web_fetch"} {
		key, validValue, meta := "queries", `["x"]`, toolWebModelSearchMetaForTest
		if tool == "web_fetch" {
			key, validValue, meta = "url", `"x"`, toolWebModelFetchMetaForTest
		}
		for _, raw := range []string{"", "{", "null", "[]", `"args"`, "true", "1", "{}", `{"Queries":["x"],"URL":"x"}`, `{"` + key + `":` + validValue + `} {}`, `{"` + key + `":` + validValue + `,}`} {
			t.Run(tool+"/object/"+raw, func(t *testing.T) {
				if got := toolWebModelFor(toolWebModelCallForTest(tool, raw, meta, `[]`)); got != nil {
					t.Fatalf("invalid complete argument object produced model: %#v", got)
				}
			})
		}
	}
	for _, value := range []string{`null`, `1`, `true`, `{}`, `"x"`, `[]`, `[null]`, `[1]`, `[true]`, `[{}]`, `[[]]`, `[""]`, `["x",""]`, `["x"," \u00a0\ufeff"]`, `["x",null]`, `["x",1]`} {
		t.Run("queries/"+value, func(t *testing.T) {
			if got := toolWebModelFor(toolWebModelCallForTest("web_search", `{"queries":`+value+`}`, toolWebModelSearchMetaForTest, `[]`)); got != nil {
				t.Fatalf("invalid query array produced model: %#v", got)
			}
		})
	}
	for _, value := range []string{`null`, `1`, `true`, `{}`, `[]`, `""`, `" \u00a0\ufeff"`} {
		t.Run("url/"+value, func(t *testing.T) {
			if got := toolWebModelFor(toolWebModelCallForTest("web_fetch", `{"url":`+value+`}`, toolWebModelFetchMetaForTest, `[]`)); got != nil {
				t.Fatalf("invalid argument URL produced model: %#v", got)
			}
		})
	}
	for _, tc := range []struct{ tool, args, meta string }{
		{"web_search", `{"queries":["  x  ","x","x","four","five","six"],"unrelated":1e9999}`, toolWebModelSearchMetaForTest},
		{"web_search", `{"queries":null,"queries":["x"],"url":null}`, toolWebModelSearchMetaForTest},
		{"web_fetch", `{"url":null,"url":"relative","queries":null,"unrelated":1e9999}`, toolWebModelFetchMetaForTest},
	} {
		if got := toolWebModelFor(toolWebModelCallForTest(tc.tool, tc.args, tc.meta, `[]`)); got == nil {
			t.Fatalf("valid extended or duplicate-last arguments declined: %s %s", tc.tool, tc.args)
		}
	}
	for _, literal := range []string{"relative", "javascript:alert(1)", "file:///recorded", "https://example.test/\r\n\x00", " C:\\recorded\\path "} {
		args := toolWebModelJSONForTest(t, map[string]any{"url": literal})
		if got := toolWebModelFor(toolWebModelCallForTest("web_fetch", args, toolWebModelFetchMetaForTest, `[]`)); got == nil {
			t.Fatalf("fetch arguments imposed URL validation on %q", literal)
		}
	}
	for _, tc := range []struct{ tool, args, meta string }{
		{"web_search", `{"queries":["x"],"queries":null}`, toolWebModelSearchMetaForTest},
		{"web_fetch", `{"url":"x","url":null}`, toolWebModelFetchMetaForTest},
	} {
		if got := toolWebModelFor(toolWebModelCallForTest(tc.tool, tc.args, tc.meta, `[]`)); got != nil {
			t.Fatalf("invalid duplicate-last arguments produced model: %#v", got)
		}
	}
}

func TestToolWebModelJavaScriptTrimWhitespace(t *testing.T) {
	for _, whitespace := range []string{"\t", "\n", "\v", "\f", "\r", " ", "\u00a0", "\u1680", "\u2000", "\u2001", "\u2002", "\u2003", "\u2004", "\u2005", "\u2006", "\u2007", "\u2008", "\u2009", "\u200a", "\u2028", "\u2029", "\u202f", "\u205f", "\u3000", "\ufeff"} {
		t.Run(toolWebModelJSONForTest(t, whitespace), func(t *testing.T) {
			for _, tool := range []string{"web_search", "web_fetch"} {
				for _, nonblank := range []bool{false, true} {
					value := whitespace
					if nonblank {
						value += "x" + whitespace
					}
					args, meta := map[string]any{"queries": []string{value}}, toolWebModelSearchMetaForTest
					if tool == "web_fetch" {
						args, meta = map[string]any{"url": value}, toolWebModelFetchMetaForTest
					}
					got := toolWebModelFor(toolWebModelCallForTest(tool, toolWebModelJSONForTest(t, args), meta, `[]`))
					if (got != nil) != nonblank {
						t.Fatalf("tool=%s value=%q nonblank=%v, model=%#v", tool, value, nonblank, got)
					}
				}
			}
		})
	}
	for _, literal := range []string{"\u0085", "\u180e", "\u200b", "\u0000"} {
		for _, tc := range []struct{ tool, args, meta string }{
			{"web_search", toolWebModelJSONForTest(t, map[string]any{"queries": []string{literal}}), toolWebModelSearchMetaForTest},
			{"web_fetch", toolWebModelJSONForTest(t, map[string]any{"url": literal}), toolWebModelFetchMetaForTest},
		} {
			if got := toolWebModelFor(toolWebModelCallForTest(tc.tool, tc.args, tc.meta, `[]`)); got == nil {
				t.Fatalf("%s treated %q as JavaScript trim whitespace", tc.tool, literal)
			}
		}
	}
}

func TestToolWebModelDeclinesMalformedMetadata(t *testing.T) {
	for _, tool := range []string{"web_search", "web_fetch"} {
		args, base, required := `{"queries":["x"]}`, toolWebModelSearchMetaForTest, []string{"sources", "truncated"}
		if tool == "web_fetch" {
			args, base, required = `{"url":"x"}`, toolWebModelFetchMetaForTest, []string{"url", "statusCode", "truncated"}
		}
		for _, meta := range []string{`null`, `[]`, `"meta"`, `true`, `1`, `{}`} {
			t.Run(tool+"/object/"+meta, func(t *testing.T) {
				if got := toolWebModelFor(toolWebModelCallForTest(tool, args, meta, `[]`)); got != nil {
					t.Fatalf("non-metadata object produced model: %#v", got)
				}
			})
		}
		for _, key := range required {
			fields := map[string]json.RawMessage{}
			if err := json.Unmarshal([]byte(base), &fields); err != nil {
				t.Fatal(err)
			}
			delete(fields, key)
			if got := toolWebModelFor(toolWebModelCallForTest(tool, args, toolWebModelJSONForTest(t, fields), `[]`)); got != nil {
				t.Fatalf("%s missing required %s produced model: %#v", tool, key, got)
			}
		}
		for _, value := range []string{`null`, `0`, `1`, `"false"`, `[]`, `{}`} {
			meta := strings.TrimSuffix(base, "}") + `,"truncated":` + value + `}`
			if got := toolWebModelFor(toolWebModelCallForTest(tool, args, meta, `[]`)); got != nil {
				t.Fatalf("%s accepted non-boolean truncated=%s: %#v", tool, value, got)
			}
		}
	}
	for _, value := range []string{`null`, `1`, `true`, `"sources"`, `{}`, `[null]`, `[1]`, `[true]`, `["source"]`, `[[]]`, `[{}]`, `[{"url":"valid"},null]`} {
		t.Run("sources/"+value, func(t *testing.T) {
			meta := `{"sources":` + value + `,"truncated":false}`
			if got := toolWebModelFor(toolWebModelCallForTest("web_search", `{"queries":["x"]}`, meta, `[]`)); got != nil {
				t.Fatalf("invalid source array produced model: %#v", got)
			}
		})
	}
	for _, key := range []string{"url", "title", "snippet", "publishedAt"} {
		for _, value := range []string{`null`, `1`, `true`, `[]`, `{}`} {
			t.Run("source/"+key+"/"+value, func(t *testing.T) {
				meta := `{"sources":[{"url":"valid","` + key + `":` + value + `}],"truncated":false}`
				if got := toolWebModelFor(toolWebModelCallForTest("web_search", `{"queries":["x"]}`, meta, `[]`)); got != nil {
					t.Fatalf("non-string source field produced model: %#v", got)
				}
			})
		}
	}
	for _, value := range []string{`null`, `1`, `true`, `[]`, `{}`} {
		meta := `{"sources":[],"answer":` + value + `,"truncated":false}`
		if got := toolWebModelFor(toolWebModelCallForTest("web_search", `{"queries":["x"]}`, meta, `[]`)); got != nil {
			t.Fatalf("non-string optional answer=%s produced model: %#v", value, got)
		}
		meta = `{"url":` + value + `,"statusCode":200,"truncated":false}`
		if got := toolWebModelFor(toolWebModelCallForTest("web_fetch", `{"url":"x"}`, meta, `[]`)); got != nil {
			t.Fatalf("non-string result URL=%s produced model: %#v", value, got)
		}
	}
	for _, value := range []string{`null`, `"200"`, `true`, `[]`, `{}`, `1.5`, `-1.5`, `1e9999`, `-1e9999`, `1e-300`, `-1e-300`} {
		meta := `{"url":"result","statusCode":` + value + `,"truncated":false}`
		if got := toolWebModelFor(toolWebModelCallForTest("web_fetch", `{"url":"x"}`, meta, `[]`)); got != nil {
			t.Fatalf("non-integer statusCode=%s produced model: %#v", value, got)
		}
	}
}

func TestToolWebModelSignedJavaScriptIntegersAndRecordedStrings(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want float64
	}{
		{"-404", -404}, {"0", 0}, {"-0.0", 0}, {"99", 99}, {"600", 600},
		{"1e1", 10}, {"-10.0", -10}, {"1.00000000000000001", 1}, {"-0.99999999999999999", -1},
		{"9007199254740993", 9007199254740992}, {"-9007199254740993", -9007199254740992},
		{"9223372036854775807", 9223372036854775808}, {"-9223372036854775809", -9223372036854775808},
		{"1e300", 1e300}, {"-1e300", -1e300},
		{"1.7976931348623157e308", math.MaxFloat64}, {"-1.7976931348623157e308", -math.MaxFloat64},
		{"1e-9999", 0}, {"-1e-9999", 0},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			meta := `{"url":" \tjavascript:alert(1)\r\n\u0000\ufeff","statusCode":` + tc.raw + `,"truncated":true,"extra":1e9999}`
			got := toolWebModelFor(toolWebModelCallForTest("web_fetch", `{"url":"different argument URL"}`, meta, `[{"type":"text","text":"not a body envelope"}]`))
			want := &toolWebModel{Kind: "fetch", URL: " \tjavascript:alert(1)\r\n\x00\ufeff", StatusCode: tc.want, Truncated: true}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("recorded URL or IEEE integer changed: %#v, want %#v", got, want)
			}
		})
	}
	for _, truncated := range []bool{false, true} {
		meta := `{"sources":[{"url":"b","title":" title ","snippet":"\r\n\u0000","publishedAt":"invalid date"},{"url":"a"},{"url":"b"}],"answer":"\n answer \ufeff","truncated":` + toolWebModelJSONForTest(t, truncated) + `}`
		for _, content := range []string{`[]`, `null`, `{}`, `"content"`, `[{"type":"text","text":"unrelated body"},{"type":"image"}]`} {
			got := toolWebModelFor(toolWebModelCallForTest("web_search", `{"queries":["query not copied"]}`, meta, content))
			want := &toolWebModel{Kind: "search", Answer: "\n answer \ufeff", Truncated: truncated, Sources: []toolWebSource{
				{URL: "b", Title: " title ", Snippet: "\r\n\x00", PublishedAt: "invalid date"}, {URL: "a"}, {URL: "b"},
			}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("truncated=%v content=%s changed metadata-owned card: %#v, want %#v", truncated, content, got, want)
			}
		}
	}
}

func TestToolWebModelSuccessfulSettledRootReceiptOnly(t *testing.T) {
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
		{"wire failed despite ok projection", func(c *session.ToolCallView) {
			c.Result.Raw = json.RawMessage(strings.Replace(string(c.Result.Raw), `"message":{`, `"message":{"isError":true,`, 1))
		}},
		{"wire interrupted despite ok projection", func(c *session.ToolCallView) {
			c.Result.Raw = json.RawMessage(strings.TrimSuffix(string(c.Result.Raw), "}}") + `,"error":{"code":"interrupted"}}}`)
		}},
	}
	for _, state := range []string{"preparing", "running", "error", "failed", "stopped", "cancelled", "canceled", "OK", "success", ""} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"state/" + state, func(c *session.ToolCallView) { c.State = state }})
	}
	for _, name := range []string{"Web_search", "Web_fetch", "web_search ", "web_fetch ", "functions.web_search", "functions.web_fetch", "plugin:web_search", "search", "fetch", ""} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"name/" + name, func(c *session.ToolCallView) { c.Name = name }})
	}
	for _, raw := range []string{"", "{", "null", "[]", `"receipt"`, "true", "1", "{}", `{"type":"unknown","data":{}}`, `{"type":"tool/result"}`, `{"type":"tool/result","data":null}`, `{"type":"tool/result","data":[]}`} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"receipt/" + raw, func(c *session.ToolCallView) { c.Result.Raw = json.RawMessage(raw) }})
	}
	for _, message := range []string{"", `null`, `[]`, `"message"`, `true`, `1`} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"data.message/" + message, func(c *session.ToolCallView) {
			meta := toolWebModelSearchMetaForTest
			if c.Name == "web_fetch" {
				meta = toolWebModelFetchMetaForTest
			}
			raw := `{"type":"tool/result","data":{"meta":` + meta
			if message != "" {
				raw += `,"message":` + message
			}
			c.Result.Raw = json.RawMessage(raw + `}}`)
		}})
	}
	for _, location := range []string{"missing", "message.meta", "data.metadata"} {
		cases = append(cases, struct {
			name   string
			change func(*session.ToolCallView)
		}{"metadata location/" + location, func(c *session.ToolCallView) {
			meta := toolWebModelSearchMetaForTest
			if c.Name == "web_fetch" {
				meta = toolWebModelFetchMetaForTest
			}
			raw := `{"type":"tool/result","data":{"message":{}}}`
			if location == "message.meta" {
				raw = `{"type":"tool/result","data":{"message":{"meta":` + meta + `}}}`
			} else if location == "data.metadata" {
				raw = `{"type":"tool/result","data":{"message":{},"metadata":` + meta + `}}`
			}
			c.Result.Raw = json.RawMessage(raw)
		}})
	}
	for _, tool := range []string{"web_search", "web_fetch"} {
		args, meta := `{"queries":["x"]}`, toolWebModelSearchMetaForTest
		if tool == "web_fetch" {
			args, meta = `{"url":"x"}`, toolWebModelFetchMetaForTest
		}
		for _, tc := range cases {
			t.Run(tool+"/"+tc.name, func(t *testing.T) {
				call := toolWebModelCallForTest(tool, args, meta, `[]`)
				tc.change(call)
				if got := toolWebModelFor(call); got != nil {
					t.Fatalf("non-root or unsuccessful call produced model: %#v", got)
				}
			})
		}
		call := toolWebModelCallForTest(tool, args, meta, `[]`)
		call.RootCallID, call.Result.EventType = call.CallID, ""
		call.Result.Raw = json.RawMessage(`{"type":"tool/result","data":{"meta":` + meta + `,"message":{}}}`)
		if got := toolWebModelFor(call); got == nil {
			t.Fatalf("%s self root with raw receipt type and empty message object declined", tool)
		}
	}
	if got := toolWebModelFor(nil); got != nil {
		t.Fatalf("nil call produced model: %#v", got)
	}
}

func TestToolWebModelProjectedWireLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, dataExtra, messageExtra, wantState string
		wantModel                                bool
	}{
		{"success", "", "", "ok", true},
		{"data error", `,"isError":true`, "", "error", false},
		{"message error", "", `,"isError":true`, "error", false},
		{"ordinary error", `,"error":{"code":"unavailable"}`, "", "error", false},
		{"interrupted", `,"error":{"code":"interrupted"}`, "", "stopped", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			head := session.Message{Kind: session.MessageKindTool, EventType: "tool/call", Seq: 1,
				Raw: json.RawMessage(`{"type":"tool/call","data":{"callId":"web-projected","name":"web_search","arguments":"{\"queries\":[\"x\"]}"}}`)}
			receipt := session.Message{Kind: session.MessageKindTool, EventType: "tool/result", Seq: 2,
				Raw: json.RawMessage(`{"type":"tool/result","data":{"meta":` + toolWebModelSearchMetaForTest + `,"message":{"toolCallId":"web-projected","content":[]` + tc.messageExtra + `}` + tc.dataExtra + `}}`)}
			messages := []session.Message{head, receipt}
			before := toolWebModelJSONForTest(t, messages)
			projected := session.ProjectToolMessages(messages)
			if len(projected) != 1 || projected[0].Tool == nil {
				t.Fatalf("wire call was not projected: %#v", projected)
			}
			call := projected[0].Tool
			if got := toolWebModelFor(call); call.State != tc.wantState || (got != nil) != tc.wantModel {
				t.Fatalf("state=%s model=%#v, want state=%s valid=%v", call.State, got, tc.wantState, tc.wantModel)
			}
			if toolWebModelJSONForTest(t, messages) != before {
				t.Fatal("model derivation changed original wire messages")
			}
			resultOnly := session.ProjectToolMessages([]session.Message{receipt})
			if len(resultOnly) != 1 || resultOnly[0].Tool == nil || toolWebModelFor(resultOnly[0].Tool) != nil {
				t.Fatal("result-only history window acquired a web card without a retained head")
			}
		})
	}
}

func TestToolWebModelSourceImmutabilityAndIndependentSlices(t *testing.T) {
	for _, tc := range []struct{ tool, args, meta string }{
		{"web_search", `{"queries":["x"]}`, `{"sources":[{"url":"first","title":"title","snippet":"snippet","publishedAt":"date"},{"url":"second"}],"answer":"answer","truncated":true}`},
		{"web_fetch", `{"url":"x"}`, `{"url":"result","statusCode":-404,"truncated":true}`},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			call := toolWebModelCallForTest(tc.tool, tc.args, tc.meta, `[{"type":"text","text":"body"}]`)
			call.Events = []session.Message{*call.Call, *call.Result}
			call.Children = []*session.ToolCallView{{CallID: "child", Name: "web_fetch", Arguments: `{"url":"child"}`}}
			before := toolWebModelJSONForTest(t, call)
			first, second := toolWebModelFor(call), toolWebModelFor(call)
			if first == nil || !reflect.DeepEqual(first, second) {
				t.Fatalf("repeated derivation changed model: %#v, %#v", first, second)
			}
			originalSecond := toolWebModelJSONForTest(t, second)
			first.Kind, first.Answer, first.URL, first.StatusCode, first.Truncated = "changed", "changed", "changed", 999, false
			if len(first.Sources) != 0 {
				first.Sources[0] = toolWebSource{URL: "changed", Title: "changed", Snippet: "changed", PublishedAt: "changed"}
			}
			first.Sources = append(first.Sources, toolWebSource{URL: "appended"})
			if toolWebModelJSONForTest(t, second) != originalSecond {
				t.Fatal("returned source slices aliased an independently derived model")
			}
			if got := toolWebModelFor(call); !reflect.DeepEqual(got, second) {
				t.Fatalf("model mutation affected later derivation: %#v, want %#v", got, second)
			}
			if toolWebModelJSONForTest(t, call) != before {
				t.Fatal("derivation or returned-model mutation changed source call/events")
			}
			call.Arguments = `{}`
			declinedBefore := toolWebModelJSONForTest(t, call)
			if got := toolWebModelFor(call); got != nil {
				t.Fatalf("changed invalid arguments reused stale model: %#v", got)
			}
			if toolWebModelJSONForTest(t, call) != declinedBefore {
				t.Fatal("declining a model changed source call/events")
			}
		})
	}
}
