package chat

import (
	"encoding/json"
	"reflect"
	"testing"

	"whaleisle/internal/store/session"
)

func toolSummaryModelCallForTest(name, args string) *session.ToolCallView {
	return &session.ToolCallView{
		CallID: "call-117", Name: name, Arguments: args, State: "running",
		Call: &session.Message{EventType: "tool/call", Body: "wrong display", Args: `{"file_path":"/wrong"}`},
	}
}

func toolSummaryModelReceiptForTest(t *testing.T, nested bool, meta, content, failure any, failed bool) *session.Message {
	t.Helper()
	if content == nil {
		content = []any{}
	}
	typ := "tool/result"
	owner := map[string]any{"content": content, "isError": failed}
	data := map[string]any{"message": owner, "meta": meta}
	if nested {
		typ = "tool/ptc-dispatch"
		data = map[string]any{"content": content, "isError": failed, "meta": meta}
	}
	if failure != nil {
		data["error"] = failure
	}
	raw, err := json.Marshal(map[string]any{"type": typ, "data": data})
	if err != nil {
		t.Fatal(err)
	}
	return &session.Message{
		Kind: session.MessageKindTool, EventType: typ, Raw: raw,
		Body: "flattened result must not become a summary", Args: `{"file_path":"/wrong-result"}`,
	}
}

// Golden values are transcribed from the installed model's selection/display
// rules, with ToolRow's error/stopped navigation suppression applied. No host,
// filesystem, UI, browser or installed-source execution is needed by these tests.
func TestToolSummaryModelDisplayGolden(t *testing.T) {
	cases := []struct {
		name, tool, args, cwd, home string
		want                        toolSummaryModel
	}{
		{"complete file_path wins path and description", "read", `{"description":"ignore","path":"/w/old","file_path":"/w/new"}`, "/w", "", toolSummaryModel{Summary: "new", FilePath: "/w/new"}},
		{"file description is only raw fallback", "write", `{"description":"/w/note"}`, "/w", "", toolSummaryModel{Summary: "note"}},
		{"path fallback number file_path", "edit", `{"file_path":2,"path":"/w/fallback"}`, "/w", "", toolSummaryModel{Summary: "fallback", FilePath: "/w/fallback"}},
		{"path fallback empty file_path", "read_image", `{"file_path":"","path":"/w/a.png"}`, "/w", "", toolSummaryModel{Summary: "a.png", FilePath: "/w/a.png"}},
		{"url is summary never path", "web_fetch", `{"description":"ignored","url":"https://example.test/a"}`, "", "", toolSummaryModel{Summary: "https://example.test/a"}},
		{"first line preserves CR", "read", `{"file_path":"/w/a\r\nrest"}`, "/w", "", toolSummaryModel{Summary: "a\r", FilePath: "/w/a\r"}},
		{"first newline only", "write", `{"file_path":"/w/a\u2028tail\nrest"}`, "/w", "", toolSummaryModel{Summary: "a\u2028tail", FilePath: "/w/a\u2028tail"}},
		{"whitespace strings are nonempty", "read", `{"file_path":" \t ","path":"/w/ignored"}`, "/w", "", toolSummaryModel{Summary: " \t "}},
		{"newline file_path present empty first line", "read", `{"path":"/w/fallback","file_path":"\nrest"}`, "/w", "", toolSummaryModel{Summary: "fallback"}},
		{"description primary retains path spelling", "bash", `{"command":"ignore","description":"/w/primary\nrest"}`, "/w", "", toolSummaryModel{Summary: "/w/primary"}},
		{"description wins search queries", "web_search", `{"queries":["ignored"],"description":"Primary"}`, "", "", toolSummaryModel{Summary: "Primary"}},
		{"description empty first line search fallback", "glob", `{"description":"\nrest","pattern":"fallback"}`, "", "", toolSummaryModel{Summary: "fallback"}},
		{"description empty first line shell stays empty", "bash", `{"description":"\nrest","command":"ignored"}`, "", "", toolSummaryModel{}},
		{"description empty first line generic Object values fallback", "unknown", `{"earlier":"fallback","description":"\nrest"}`, "", "", toolSummaryModel{Summary: "unknown · fallback"}},
		{"command fallback gets path display", "pwsh", `{"command":"/w/command"}`, "/w", "", toolSummaryModel{Summary: "command"}},
		{"search filtered queries", "web_search", `{"queries":["first\nrest",null,7,""," ","\nlast"],"query":"ignored"}`, "", "", toolSummaryModel{Summary: "first,  , "}},
		{"search queries fallback", "grep", `{"queries":[null,7,""],"pattern":"needle"}`, "", "", toolSummaryModel{Summary: "needle"}},
		{"Object values numeric key order", "custom_wire", `{"z":"insertion-first","10":"ten","2":"two","01":"not-index"}`, "", "", toolSummaryModel{Summary: "custom_wire · two"}},
		{"duplicate key final value first position", "custom_wire", `{"z":"old","a":"later","z":"updated"}`, "", "", toolSummaryModel{Summary: "custom_wire · updated"}},
		{"duplicate description final empty falls back", "bash", `{"description":"old","command":"Run","description":""}`, "", "", toolSummaryModel{Summary: "Run"}},
		{"array first string", "read", `[null,7,"/w/first\nrest","second"]`, "/w", "", toolSummaryModel{Summary: "first"}},
		{"array without strings", "read", `[1,2]`, "", "", toolSummaryModel{Summary: `[1,2]`}},
		{"scalar remains JSON text", "read", `"/w/scalar"`, "/w", "", toolSummaryModel{Summary: `"/w/scalar"`}},
		{"null stays raw", "read", `null`, "", "", toolSummaryModel{Summary: "null"}},
		{"empty object stays raw", "get_goal", `{}`, "", "", toolSummaryModel{Summary: "{}"}},
		{"known empty args call id", "get_goal", "", "", "", toolSummaryModel{Summary: "call-117"}},
		{"unknown empty args wire and id", "unknown", "", "", "", toolSummaryModel{Summary: "unknown · call-117"}},
		{"unknown blank line keeps wire name", "unknown", "\nsecond", "", "", toolSummaryModel{Summary: "unknown"}},
		{"legacy shell remains generic source name", "shell", `{"command":"pwd"}`, "", "", toolSummaryModel{Summary: "shell · pwd"}},
		{"object fallback is not recursive", "custom_wire", `{"nested":{"file_path":"/wrong"},"fallback":"right"}`, "", "", toolSummaryModel{Summary: "custom_wire · right"}},
		{"invalid JSON literal fallback", "read", "  literal\r\nsecond", "", "", toolSummaryModel{Summary: "  literal\r"}},
		{"invalid complete path trailing JSON", "read", `{"file_path":"/w/a"} {}`, "/w", "", toolSummaryModel{Summary: `{"file_path":"/w/a"} {}`}},
		{"nonfinite unrelated number preserves strings", "read", `{"extra":1e9999,"file_path":"/w/a"}`, "/w", "", toolSummaryModel{Summary: "a", FilePath: "/w/a"}},
		{"cwd wins before home abbreviation", "read", `{"file_path":"/home/me/work/a"}`, "/home/me/work/", "/home/me", toolSummaryModel{Summary: "a", FilePath: "/home/me/work/a"}},
		{"outside cwd POSIX home", "read", `{"file_path":"/home/me/other/a"}`, "/home/me/work", "/home/me/", toolSummaryModel{Summary: "~/other/a", FilePath: "/home/me/other/a"}},
		{"home exact trailing slashes", "read", `{"file_path":"/home/me///"}`, "/work", "/home/me/", toolSummaryModel{Summary: "~", FilePath: "/home/me///"}},
		{"home root is not abbreviated", "read", `{"file_path":"/etc/a"}`, "", "/", toolSummaryModel{Summary: "/etc/a", FilePath: "/etc/a"}},
		{"home prefix boundary", "read", `{"file_path":"/home/member/a"}`, "", "/home/me", toolSummaryModel{Summary: "/home/member/a", FilePath: "/home/member/a"}},
		{"equal cwd stays absolute", "read", `{"file_path":"/work"}`, "/work", "", toolSummaryModel{Summary: "/work", FilePath: "/work"}},
		{"cwd boundary prevents sibling", "read", `{"file_path":"/work-other/a"}`, "/work", "", toolSummaryModel{Summary: "/work-other/a", FilePath: "/work-other/a"}},
		{"root cwd strips one separator", "read", `{"file_path":"/a"}`, "/", "", toolSummaryModel{Summary: "a", FilePath: "/a"}},
		{"dot segments never cleaned", "edit", `{"file_path":"/work/sub/../a"}`, "/work", "", toolSummaryModel{Summary: "sub/../a", FilePath: "/work/sub/../a"}},
		{"Windows cwd retains inner separators", "read", `{"file_path":"C:\\Work\\sub\\a"}`, `C:\Work\`, "", toolSummaryModel{Summary: `sub\a`, FilePath: `C:\Work\sub\a`}},
		{"Windows slash after literal cwd", "read", `{"file_path":"C:\\Work/sub/a"}`, `C:\Work`, "", toolSummaryModel{Summary: "sub/a", FilePath: `C:\Work/sub/a`}},
		{"Windows case stays distinct", "read", `{"file_path":"c:\\work\\a"}`, `C:\Work`, "", toolSummaryModel{Summary: `c:\work\a`, FilePath: `c:\work\a`}},
		{"Windows separator before boundary not normalized", "read", `{"file_path":"C:/Work/a"}`, `C:\Work`, "", toolSummaryModel{Summary: "C:/Work/a", FilePath: "C:/Work/a"}},
		{"Windows home not abbreviated", "read", `{"file_path":"C:\\Users\\me\\a"}`, "", `C:\Users\me`, toolSummaryModel{Summary: `C:\Users\me\a`, FilePath: `C:\Users\me\a`}},
		{"Windows home disables POSIX abbreviation", "read", `{"file_path":"/home/me/a"}`, "", `C:\Users\me`, toolSummaryModel{Summary: "/home/me/a", FilePath: "/home/me/a"}},
		{"UNC cwd and path retained", "read_image", `{"file_path":"\\\\server\\share\\a.png"}`, `\\server\share`, "", toolSummaryModel{Summary: "a.png", FilePath: `\\server\share\a.png`}},
		{"read offset and limit", "read", `{"file_path":"/w/a","offset":12,"limit":3}`, "/w", "", toolSummaryModel{Summary: "a", FilePath: "/w/a", Line: 12}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toolSummaryModelFor(toolSummaryModelCallForTest(tc.tool, tc.args), tc.cwd, tc.home); got != tc.want {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
	if got := toolSummaryModelFor(nil, "/w", "/home/me"); got != (toolSummaryModel{}) {
		t.Fatalf("nil call: %#v", got)
	}
}

func TestToolSummaryModelExactFileVariants(t *testing.T) {
	for _, name := range []string{"read", "read_image", "web_fetch", "cordis_package_inspect", "cordis_runtime_inspect", "write", "edit"} {
		t.Run(name, func(t *testing.T) {
			call := toolSummaryModelCallForTest(name, `{"file_path":"/w/a","offset":7}`)
			want := toolSummaryModel{Summary: "a", FilePath: "/w/a"}
			if name == "read" {
				want.Line = 7
			}
			if got := toolSummaryModelFor(call, "/w", ""); got != want {
				t.Fatalf("exact file variant: %#v, want %#v", got, want)
			}
		})
	}
	for _, name := range []string{"Read", "read_file", "functions.read", "plugin:read", "write_file", "edit_file", "read_image ", "cordis_inspect_query", "cordis_inspect_list", "cordis_inspect_self", "grep", "glob", "bash", "run_code"} {
		t.Run(name, func(t *testing.T) {
			got := toolSummaryModelFor(toolSummaryModelCallForTest(name, `{"file_path":"/w/a","offset":7}`), "/w", "")
			if got.FilePath != "" || got.Line != 0 {
				t.Fatalf("non-file name produced navigation: %#v", got)
			}
		})
	}
}

func TestToolSummaryModelNavigationGolden(t *testing.T) {
	cases := []struct {
		name, args, state, parent, root string
		nested, settled, missingHead    bool
		meta                            any
		want                            toolSummaryModel
	}{
		{name: "relative running", args: `{"file_path":"sub/a"}`, want: toolSummaryModel{Summary: "sub/a"}},
		{name: "absolute running", args: `{"file_path":"/w/sub/a"}`, want: toolSummaryModel{Summary: "sub/a", FilePath: "/w/sub/a"}},
		{name: "absolute nested running", args: `{"file_path":"/other/a"}`, parent: "parent", want: toolSummaryModel{Summary: "/other/a", FilePath: "/other/a"}},
		{name: "relative successful root remains raw", args: `{"file_path":"sub/../a"}`, settled: true, want: toolSummaryModel{Summary: "sub/../a", FilePath: "sub/../a"}},
		{name: "relative successful root without cwd", args: `{"file_path":"a"}`, settled: true, root: "call-117", want: toolSummaryModel{Summary: "a", FilePath: "a"}},
		{name: "relative nested has no root authority", args: `{"file_path":"a"}`, parent: "parent", nested: true, settled: true, want: toolSummaryModel{Summary: "a"}},
		{name: "missing parent relation stays nested", args: `{"file_path":"a"}`, root: "other-root", settled: true, want: toolSummaryModel{Summary: "a"}},
		{name: "recorded overrides absolute argument", args: `{"file_path":"/w/arg","offset":8}`, settled: true, meta: map[string]any{"path": "/operation/./actual"}, want: toolSummaryModel{Summary: "arg", FilePath: "/operation/./actual", Line: 8}},
		{name: "recorded overrides nested relative", args: `{"file_path":"a","offset":8}`, parent: "parent", nested: true, settled: true, meta: map[string]any{"path": `D:\operation\a`}, want: toolSummaryModel{Summary: "a", FilePath: `D:\operation\a`, Line: 8}},
		{name: "recorded UNC retained", args: `{"path":"a"}`, nested: true, settled: true, meta: map[string]any{"path": `\\server\share\a`}, want: toolSummaryModel{Summary: "a", FilePath: `\\server\share\a`}},
		{name: "recorded relative ignored", args: `{"file_path":"a"}`, nested: true, settled: true, meta: map[string]any{"path": "relative"}, want: toolSummaryModel{Summary: "a"}},
		{name: "recorded drive relative ignored", args: `{"file_path":"a"}`, nested: true, settled: true, meta: map[string]any{"path": `C:a`}, want: toolSummaryModel{Summary: "a"}},
		{name: "metadata arrays ignored", args: `{"file_path":"/w/a"}`, settled: true, meta: []any{map[string]any{"path": "/wrong"}}, want: toolSummaryModel{Summary: "a", FilePath: "/w/a"}},
		{name: "metadata path types ignored", args: `{"file_path":"a"}`, settled: true, meta: map[string]any{"path": []any{"/wrong"}}, want: toolSummaryModel{Summary: "a", FilePath: "a"}},
		{name: "metadata whitespace string not absolute", args: `{"file_path":"a"}`, settled: true, meta: map[string]any{"path": " /wrong"}, want: toolSummaryModel{Summary: "a", FilePath: "a"}},
		{name: "metadata without argument cannot create path", args: `{"description":"note"}`, settled: true, meta: map[string]any{"path": "/wrong"}, want: toolSummaryModel{Summary: "note"}},
		{name: "empty argument paths cannot create path", args: `{"path":"","file_path":""}`, settled: true, meta: map[string]any{"path": "/wrong"}, want: toolSummaryModel{Summary: `{"path":"","file_path":""}`}},
		{name: "nonstring paths cannot create path", args: `{"path":true,"file_path":["a"]}`, settled: true, meta: map[string]any{"path": "/wrong"}, want: toolSummaryModel{Summary: `{"path":true,"file_path":["a"]}`}},
		{name: "empty first line still names argument path", args: `{"path":"fallback","file_path":"\nrest"}`, settled: true, meta: map[string]any{"path": "/operation/a"}, want: toolSummaryModel{Summary: "fallback", FilePath: "/operation/a"}},
		{name: "whitespace path root accepted verbatim", args: `{"file_path":" \t "}`, settled: true, want: toolSummaryModel{Summary: " \t ", FilePath: " \t "}},
		{name: "root receipt missing head ignores args and metadata", args: `{"file_path":"/w/fake","offset":4}`, settled: true, missingHead: true, meta: map[string]any{"path": "/wrong"}, want: toolSummaryModel{Summary: "call-117"}},
		{name: "PTC receipt missing head retains args", args: `{"file_path":"relative","offset":4}`, nested: true, settled: true, missingHead: true, meta: map[string]any{"path": "/operation/relative"}, want: toolSummaryModel{Summary: "relative", FilePath: "/operation/relative", Line: 4}},
		{name: "absolute error path suppressed", args: `{"file_path":"/w/a","offset":3}`, settled: true, state: "error", meta: map[string]any{"path": "/operation/a"}, want: toolSummaryModel{Summary: "a"}},
		{name: "absolute stopped path suppressed", args: `{"file_path":"/w/a","offset":3}`, settled: true, state: "stopped", meta: map[string]any{"path": "/operation/a"}, want: toolSummaryModel{Summary: "a"}},
		{name: "Windows drive absolute running", args: `{"path":"C:/w/a"}`, want: toolSummaryModel{Summary: "C:/w/a", FilePath: "C:/w/a"}},
		{name: "drive relative running no link", args: `{"path":"C:a"}`, want: toolSummaryModel{Summary: "C:a"}},
		{name: "single backslash running no link", args: `{"path":"\\a"}`, want: toolSummaryModel{Summary: `\a`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			call := toolSummaryModelCallForTest("read", tc.args)
			call.ParentCallID, call.RootCallID = tc.parent, tc.root
			if tc.settled {
				call.State = "ok"
				call.Result = toolSummaryModelReceiptForTest(t, tc.nested, tc.meta, nil, nil, false)
			}
			if tc.state != "" {
				call.State = tc.state
			}
			if tc.missingHead {
				call.Call = nil
			}
			if got := toolSummaryModelFor(call, "/w", ""); got != tc.want {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestToolSummaryModelReadLineGolden(t *testing.T) {
	for _, tc := range []struct {
		name, args string
		line       int
	}{
		{"positive", `{"file_path":"/w/a","offset":12}`, 12},
		{"integral fraction", `{"file_path":"/w/a","offset":12.0}`, 12},
		{"exponent", `{"file_path":"/w/a","offset":1e2}`, 100},
		{"large valid limit", `{"file_path":"/w/a","offset":2,"limit":1e300}`, 2},
		{"duplicate offset final value", `{"file_path":"/w/a","offset":1,"offset":3}`, 3},
		{"offset absent", `{"file_path":"/w/a"}`, 0},
		{"offset zero", `{"file_path":"/w/a","offset":0}`, 0},
		{"offset negative", `{"file_path":"/w/a","offset":-1}`, 0},
		{"offset fraction", `{"file_path":"/w/a","offset":1.5}`, 0},
		{"offset string", `{"file_path":"/w/a","offset":"12"}`, 0},
		{"offset null", `{"file_path":"/w/a","offset":null}`, 0},
		{"offset bool", `{"file_path":"/w/a","offset":true}`, 0},
		{"offset infinity", `{"file_path":"/w/a","offset":1e9999}`, 0},
		{"offset unrepresentable int", `{"file_path":"/w/a","offset":1e30}`, 0},
		{"offset underflow", `{"file_path":"/w/a","offset":1e-9999}`, 0},
		{"limit zero invalidates offset", `{"file_path":"/w/a","offset":12,"limit":0}`, 0},
		{"limit null invalidates offset", `{"file_path":"/w/a","offset":12,"limit":null}`, 0},
		{"limit fraction invalidates offset", `{"file_path":"/w/a","offset":12,"limit":0.5}`, 0},
		{"limit string invalidates offset", `{"file_path":"/w/a","offset":12,"limit":"2"}`, 0},
		{"path only is not read envelope", `{"path":"/w/a","offset":12}`, 0},
		{"blank file_path suppresses line", `{"file_path":" \ufeff\n","path":"/w/a","offset":12}`, 0},
		{"NEL is not JS blank", `{"file_path":"\u0085","offset":12}`, 12},
		{"newline path nonblank suffix", `{"file_path":"\na","offset":12}`, 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call := toolSummaryModelCallForTest("read", tc.args)
			call.State = "ok"
			call.Result = toolSummaryModelReceiptForTest(t, false, map[string]any{"path": "/operation/a", "offset": 999}, nil, nil, false)
			got := toolSummaryModelFor(call, "/w", "")
			if got.FilePath != "/operation/a" || got.Line != tc.line {
				t.Fatalf("line/metadata authority: %#v, want line %d", got, tc.line)
			}
		})
	}
}

func TestToolSummaryModelErrorSummaryGolden(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, tc := range []struct {
			name, state string
			content     any
			failure     any
			failed      bool
			want        string
		}{
			{"verbatim first line", "error", []any{map[string]any{"type": "text", "text": "  first error\r\nrest"}}, nil, true, "  first error\r"},
			{"empty first line erases summary", "error", []any{map[string]any{"type": "text", "text": "\nrest"}}, nil, true, ""},
			{"blank result retains base", "error", []any{map[string]any{"type": "text", "text": ""}}, map[string]any{"name": "Ignored", "code": "ignored"}, true, "a"},
			{"empty blocks structured fallback", "error", nil, map[string]any{"name": "ReadError", "code": "ENOENT"}, true, "ReadError: ENOENT"},
			{"missing error fields source undefined", "error", nil, map[string]any{}, true, "undefined: undefined"},
			{"content absent no fallback", "error", nil, nil, true, "a"},
			{"nontext pretty JSON first line", "error", []any{map[string]any{"type": "image", "attachment": map[string]any{"attachmentId": "a"}}}, nil, true, "{"},
			{"text stays ahead of later block", "error", []any{map[string]any{"type": "text", "text": "failure"}, map[string]any{"type": "image"}}, nil, true, "failure"},
			{"multiple blank blocks nonempty output", "error", []any{map[string]any{"type": "text", "text": ""}, map[string]any{"type": "text", "text": ""}}, nil, true, ""},
			{"stopped retains argument summary", "stopped", []any{map[string]any{"type": "text", "text": "do not show"}}, map[string]any{"name": "Interrupted", "code": "interrupted"}, true, "a"},
			{"wire interruption precedes projected error", "error", []any{map[string]any{"type": "text", "text": "do not show"}}, map[string]any{"name": "Interrupted", "code": "interrupted"}, true, "a"},
			{"wire failure protects stale state", "ok", []any{map[string]any{"type": "text", "text": "wire failure"}}, nil, true, "wire failure"},
		} {
			t.Run(tc.name+map[bool]string{false: "/root", true: "/ptc"}[nested], func(t *testing.T) {
				call := toolSummaryModelCallForTest("read", `{"file_path":"/w/a","offset":12}`)
				call.State = tc.state
				call.Result = toolSummaryModelReceiptForTest(t, nested, map[string]any{"path": "/operation/a"}, tc.content, tc.failure, tc.failed)
				want := toolSummaryModel{Summary: tc.want}
				if got := toolSummaryModelFor(call, "/w", ""); got != want {
					t.Fatalf("got %#v, want %#v", got, want)
				}
			})
		}
	}
	// JSON.stringify an empty nontext object yields '{}' on one line; this
	// distinguishes actual structured formatting from hardcoding a brace.
	call := toolSummaryModelCallForTest("unknown", `{"description":"base"}`)
	call.State = "error"
	call.Result = toolSummaryModelReceiptForTest(t, false, nil, []any{map[string]any{}}, nil, true)
	if got := toolSummaryModelFor(call, "", ""); got.Summary != "{}" {
		t.Fatalf("empty nontext object: %#v", got)
	}
}

func TestToolSummaryModelPreparingLimitAndImmutability(t *testing.T) {
	for _, tc := range []struct{ name, args, want, target string }{
		{"empty", "", "", ""},
		{"open description raw fallback", `{"description":"Read so far`, `{"description":"Read so far`, ""},
		{"complete path inside open object raw fallback", `{"file_path":"/w/a","content":"open`, `{"file_path":"/w/a","content":"open`, ""},
		{"complete JSON can project primary", `{"file_path":"/w/a","offset":2}`, "a", "/w/a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call := toolSummaryModelCallForTest("read", tc.args)
			call.State, call.Call = "preparing", nil
			want := toolSummaryModel{Summary: tc.want, FilePath: tc.target}
			if got := toolSummaryModelFor(call, "/w", ""); got != want {
				t.Fatalf("documented PartialArguments limitation: %#v, want %#v", got, want)
			}
		})
	}
	call := toolSummaryModelCallForTest("read", `{"file_path":"/w/sub/../a","offset":4}`)
	call.State = "ok"
	call.Result = toolSummaryModelReceiptForTest(t, false, map[string]any{"path": "/operation/./a"}, []any{map[string]any{"type": "text", "text": "body"}}, nil, false)
	call.Events = []session.Message{*call.Call, *call.Result}
	call.Children = []*session.ToolCallView{toolSummaryModelCallForTest("edit", `{"file_path":"child"}`)}
	before, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	want := toolSummaryModel{Summary: "sub/../a", FilePath: "/operation/./a", Line: 4}
	for range 3 {
		if got := toolSummaryModelFor(call, "/w", "/home/me"); !reflect.DeepEqual(got, want) {
			t.Fatalf("stable pure model: %#v, want %#v", got, want)
		}
	}
	after, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("summary derivation mutated source calls, results, events, or children")
	}
}
