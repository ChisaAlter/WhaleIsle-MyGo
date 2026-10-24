package chat

import (
	"encoding/json"
	"reflect"
	"testing"

	"whaleisle/internal/store/session"
)

func toolDiffModelStringForTest(value string) *string { return &value }

func toolDiffModelCallForTest(name, args, meta string, settled bool) *session.ToolCallView {
	call := &session.ToolCallView{
		CallID: "diff-119", Name: name, Arguments: args, State: "running",
		Call: &session.Message{EventType: "tool/call", Raw: json.RawMessage(`{"type":"tool/call","data":{"callId":"diff-119"}}`)},
	}
	if settled {
		call.State = "ok"
		call.Result = &session.Message{
			EventType: "tool/result",
			Raw:       json.RawMessage(`{"type":"tool/result","data":{"meta":` + meta + `,"message":{"content":[{"type":"text","text":"generic output ignored"}]}}}`),
		}
	}
	return call
}

// Expected values were recorded by evaluating the CURRENT installed JS in a
// source-only Node VM (raw-tool-call + recorded-path + diff-card-model), without
// a browser or application. Installed diff-card-model SHA256:
// 75ef6ce33e48194a78c96696e56017fab2b3fe6a57dce4ee3f57de60c4f48e75.
// raw-tool-call SHA256: c97f8fa5e1b85e63b2bb4c65c1b7bbb265793dd0631ec0a3f32c38b3c15b44a8.
// recorded-path SHA256: 57dd73a191a150ff916cfd4efd5a2f1623f176d0a0eaa973fe88f56170569abe.
func TestToolDiffModelInstalledGoldens(t *testing.T) {
	cases := []struct {
		id, name, args, meta string
		settled              bool
		want                 []toolFileDiff
	}{
		{"running-write", "write", `{"file_path":" ./src/a.go ","content":"🙂\r\n\n"}`, "", false, []toolFileDiff{{Path: " ./src/a.go ", NewText: "🙂\r\n\n"}}},
		{"running-edit-empty", "edit", `{"file_path":"./src/a.go","old_string":"","new_string":"intent-new"}`, "", false, []toolFileDiff{{Path: "./src/a.go", NewText: "intent-new"}}},
		{"legacy-create-omitted", "str_replace_editor", `{"command":"create","path":"a"}`, "", false, []toolFileDiff{{Path: "a"}}},
		{"legacy-replace-omitted", "str_replace_editor", `{"command":"str_replace","path":"a"}`, "", false, []toolFileDiff{{Path: "a"}}},
		{"legacy-replace-empty", "str_replace_editor", `{"command":"str_replace","path":"a","old_str":"","new_str":""}`, "", false, []toolFileDiff{{Path: "a", OldText: toolDiffModelStringForTest("")}}},
		{"legacy-null-invalid", "str_replace_editor", `{"command":"str_replace","path":"a","old_str":null}`, "", false, nil},
		{"settled-write-empty", "write", `{"file_path":"a","content":"intended"}`, `{"diffs":[]}`, true, []toolFileDiff{{Path: "a", NewText: "intended"}}},
		{"settled-write-malformed", "write", `{"file_path":"a","content":"intended"}`, `{"diffs":[{"path":"a","newText":"bad"}]}`, true, []toolFileDiff{{Path: "a", NewText: "intended"}}},
		{"settled-edit-rewrite", "edit", `{"file_path":"./src/a.go","old_string":"intent-old","new_string":"intent-new"}`, `{"path":"C:\\work\\a.go","diffs":[{"path":"C:\\work\\a.go","oldText":"applied-old","newText":"applied-new"},{"path":"c:\\work\\a.go","oldText":null,"newText":"other"},{"path":"","oldText":"","newText":""}]}`, true, []toolFileDiff{{Path: "./src/a.go", OldText: toolDiffModelStringForTest("applied-old"), NewText: "applied-new"}, {Path: `c:\work\a.go`, NewText: "other"}, {OldText: toolDiffModelStringForTest("")}}},
		{"settled-edit-no-rewrite-relative", "edit", `{"file_path":"./src/a.go","old_string":"intent-old","new_string":"intent-new"}`, `{"path":"./recorded","diffs":[{"path":"./recorded","oldText":null,"newText":"actual"}]}`, true, []toolFileDiff{{Path: "./recorded", NewText: "actual"}}},
		{"settled-edit-no-applied", "edit", `{"file_path":"a","old_string":"old","new_string":"new"}`, `{}`, true, nil},
		{"settled-legacy", "str_replace_editor", `{"command":"create","path":"a","file_text":"x"}`, `{"diffs":[{"path":"a","oldText":null,"newText":"actual"}]}`, true, nil},
		{"BOM-blank", "write", `{"file_path":"\ufeff","content":"x"}`, "", false, nil},
		{"NEL-nonblank", "write", `{"file_path":"\u0085","content":"x"}`, "", false, []toolFileDiff{{Path: "\u0085", NewText: "x"}}},
		{"unrestricted-path", "edit", `{"file_path":"https://example.test/a\n\u0000","old_string":"intent-old","new_string":"intent-new"}`, "", false, []toolFileDiff{{Path: "https://example.test/a\n\x00", OldText: toolDiffModelStringForTest("intent-old"), NewText: "intent-new"}}},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			got := toolDiffModelFor(toolDiffModelCallForTest(tc.name, tc.args, tc.meta, tc.settled))
			if tc.want == nil {
				if got != nil {
					t.Fatalf("model = %#v, want nil", got)
				}
			} else if got == nil || !reflect.DeepEqual(got.Diffs, tc.want) {
				t.Fatalf("model = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestToolDiffModelIntendedArguments(t *testing.T) {
	cases := []struct {
		name, args string
		valid      bool
	}{
		{"write", `{"file_path":"a","content":""}`, true},
		{"write", `{"file_path":"a","content":"one","extension":1e9999}`, true},
		{"edit", `{"file_path":"a","old_string":"","new_string":"","replace_all":false}`, true},
		{"edit", `{"file_path":"a","old_string":"old","new_string":"new","replace_all":true}`, true},
		{"write", `{"file_path":"wrong","file_path":"a","content":null,"content":"final"}`, true},
		{"str_replace_editor", `{"command":"create","path":"a","file_text":"new","old_str":false}`, true},
		{"str_replace_editor", `{"command":"str_replace","path":"a","new_str":"new","file_text":false}`, true},
		{"str_replace_editor", `{"command":"str_replace","path":"a","old_str":"old"}`, true},
		{"Write", `{"file_path":"a","content":"new"}`, false},
		{"read", `{"file_path":"a","content":"new"}`, false},
		{"str_replace_editor", `{"command":"view","path":"a"}`, false},
		{"str_replace_editor", `{"command":"insert","path":"a","new_str":"new"}`, false},
		{"str_replace_editor", `{"command":null,"path":"a"}`, false},
	}
	for _, args := range []string{"", "{", "null", "[]", `"args"`, "1", "true", `{"file_path":"a","content":"x"} {}`, `{"file_path":"a","content":"x",}`} {
		cases = append(cases, struct {
			name, args string
			valid      bool
		}{"write", args, false})
	}
	for _, value := range []string{`null`, `true`, `1`, `[]`, `{}`} {
		for _, field := range []string{"file_path", "content"} {
			args := map[string]json.RawMessage{"file_path": json.RawMessage(`"a"`), "content": json.RawMessage(`"text"`)}
			args[field] = json.RawMessage(value)
			raw, _ := json.Marshal(args)
			cases = append(cases, struct {
				name, args string
				valid      bool
			}{"write", string(raw), false})
		}
		for _, field := range []string{"old_string", "new_string", "replace_all"} {
			args := map[string]json.RawMessage{"file_path": json.RawMessage(`"a"`), "old_string": json.RawMessage(`"old"`), "new_string": json.RawMessage(`"new"`)}
			args[field] = json.RawMessage(value)
			raw, _ := json.Marshal(args)
			cases = append(cases, struct {
				name, args string
				valid      bool
			}{"edit", string(raw), field == "replace_all" && value == "true"})
		}
		for _, field := range []string{"file_text", "old_str", "new_str"} {
			command := "str_replace"
			if field == "file_text" {
				command = "create"
			}
			args := `{"path":"a","command":"` + command + `","` + field + `":` + value + `}`
			cases = append(cases, struct {
				name, args string
				valid      bool
			}{"str_replace_editor", args, false})
		}
	}
	for _, args := range []string{`{}`, `{"path":"a","content":"x"}`, `{"file_path":"a"}`, `{"file_path":"","content":"x"}`, `{"file_path":" \t\r\n\u00a0\u2028\u2029\ufeff","content":"x"}`} {
		cases = append(cases, struct {
			name, args string
			valid      bool
		}{"write", args, false})
	}
	for i, tc := range cases {
		got := toolDiffModelFor(toolDiffModelCallForTest(tc.name, tc.args, "", false))
		if (got != nil) != tc.valid {
			t.Errorf("case %d %s %s: model=%#v, valid=%v", i, tc.name, tc.args, got, tc.valid)
		}
	}
}

func TestToolDiffModelSettledAppliedAndFallback(t *testing.T) {
	for _, name := range []string{"write", "edit"} {
		args := `{"file_path":"intent","content":"write-intent","old_string":"old-intent","new_string":"new-intent"}`
		for _, meta := range []string{
			`null`, `[]`, `"meta"`, `1`, `true`, `{}`, `{"diffs":null}`, `{"diffs":{}}`, `{"diffs":[]}`,
			`{"diffs":[null]}`, `{"diffs":[1]}`, `{"diffs":[[]]}`, `{"diffs":[{}]}`,
			`{"diffs":[{"path":"a","newText":"new"}]}`,
			`{"diffs":[{"path":null,"oldText":null,"newText":"new"}]}`,
			`{"diffs":[{"path":"a","oldText":false,"newText":"new"}]}`,
			`{"diffs":[{"path":"a","oldText":null,"newText":null}]}`,
			`{"diffs":[{"path":"a","oldText":null,"newText":"valid"},{"path":"b","newText":"invalid"}]}`,
		} {
			got := toolDiffModelFor(toolDiffModelCallForTest(name, args, meta, true))
			if name == "edit" {
				if got != nil {
					t.Errorf("edit meta %s yielded %#v", meta, got)
				}
			} else if got == nil || !reflect.DeepEqual(got.Diffs, []toolFileDiff{{Path: "intent", NewText: "write-intent"}}) {
				t.Errorf("write meta %s fallback = %#v", meta, got)
			}
		}
		meta := `{"diffs":[{"path":"","oldText":null,"newText":""},{"path":"https://example.test/a\n\u0000","oldText":"","newText":"applied"},{"path":"\ufeff","oldText":"old","newText":"new","extension":1e9999}]}`
		got := toolDiffModelFor(toolDiffModelCallForTest(name, args, meta, true))
		want := []toolFileDiff{{}, {Path: "https://example.test/a\n\x00", OldText: toolDiffModelStringForTest(""), NewText: "applied"}, {Path: "\ufeff", OldText: toolDiffModelStringForTest("old"), NewText: "new"}}
		if got == nil || !reflect.DeepEqual(got.Diffs, want) {
			t.Errorf("%s applied display strings = %#v, want %#v", name, got, want)
		}
		// A valid receipt with absent metadata is equivalent to malformed metadata.
		call := toolDiffModelCallForTest(name, args, "null", true)
		call.Result.Raw = json.RawMessage(`{"type":"tool/result","data":{"message":{}}}`)
		if got := toolDiffModelFor(call); (got != nil) != (name == "write") {
			t.Errorf("%s missing metadata = %#v", name, got)
		}
	}
	// Applied hunks cannot rescue invalid intent.
	call := toolDiffModelCallForTest("write", `{"file_path":"a","content":false}`, `{"diffs":[{"path":"a","oldText":null,"newText":"actual"}]}`, true)
	if got := toolDiffModelFor(call); got != nil {
		t.Fatalf("invalid intent rescued by metadata: %#v", got)
	}
}

func TestToolDiffModelRecordedPathExactAndPure(t *testing.T) {
	for _, path := range []string{`/work/./a`, `C:\work\a`, `C:/work/a`, `\\server\share\a`} {
		meta, _ := json.Marshal(map[string]any{"path": path, "diffs": []any{
			map[string]any{"path": path, "oldText": "actual-old", "newText": "actual-new"},
			map[string]any{"path": path + "/", "oldText": nil, "newText": "other"},
			map[string]any{"path": path, "oldText": nil, "newText": "again"},
		}})
		call := toolDiffModelCallForTest("edit", `{"file_path":" ./src/../a ","old_string":"intent-old","new_string":"intent-new"}`, string(meta), true)
		before, _ := json.Marshal(call)
		got := toolDiffModelFor(call)
		if got == nil || len(got.Diffs) != 3 || got.Diffs[0].Path != " ./src/../a " || got.Diffs[1].Path != path+"/" || got.Diffs[2].Path != " ./src/../a " {
			t.Fatalf("recorded %q rewrite = %#v", path, got)
		}
		got.Diffs[0].Path, *got.Diffs[0].OldText = "mutated", "mutated"
		again := toolDiffModelFor(call)
		after, _ := json.Marshal(call)
		if string(before) != string(after) || again.Diffs[0].Path != " ./src/../a " || *again.Diffs[0].OldText != "actual-old" {
			t.Fatal("model derivation or returned model mutated source/later derivation")
		}
	}
	for _, path := range []string{"relative/a", "C:a", `\one-backslash`, "", "https://example.test/a"} {
		meta, _ := json.Marshal(map[string]any{"path": path, "diffs": []any{map[string]any{"path": path, "oldText": nil, "newText": "actual"}}})
		got := toolDiffModelFor(toolDiffModelCallForTest("write", `{"file_path":"intent","content":"intent"}`, string(meta), true))
		if got == nil || got.Diffs[0].Path != path {
			t.Errorf("nonabsolute recorded path %q rewritten: %#v", path, got)
		}
	}
}

func TestToolDiffModelDeclinesLifecycleAndReceipts(t *testing.T) {
	valid := func() *session.ToolCallView {
		return toolDiffModelCallForTest("write", `{"file_path":"a","content":"new"}`, `{"diffs":[{"path":"a","oldText":null,"newText":"actual"}]}`, true)
	}
	if toolDiffModelFor(nil) != nil {
		t.Fatal("nil call acquired model")
	}
	for _, state := range []string{"preparing", "error", "failed", "stopped", "cancelled", "canceled"} {
		for _, settled := range []bool{false, true} {
			call := valid()
			call.State = state
			if !settled {
				call.Result = nil
			}
			if got := toolDiffModelFor(call); got != nil {
				t.Errorf("state %s settled=%v acquired %#v", state, settled, got)
			}
		}
	}
	for _, change := range []func(*session.ToolCallView){
		func(c *session.ToolCallView) { c.ParentCallID = "outer" },
		func(c *session.ToolCallView) { c.RootCallID = "outer" },
		func(c *session.ToolCallView) { c.Call = nil },
		func(c *session.ToolCallView) { c.Result.EventType = "tool/ptc-dispatch" },
		func(c *session.ToolCallView) {
			c.Result.Raw = json.RawMessage(`{"type":"tool/ptc-dispatch","data":{"meta":{"diffs":[{"path":"a","oldText":null,"newText":"nested"}]}}}`)
		},
		func(c *session.ToolCallView) {
			c.Result.Raw = json.RawMessage(`{"type":"tool/result","data":{"message":{"isError":true}}}`)
		},
		func(c *session.ToolCallView) {
			c.Result.Raw = json.RawMessage(`{"type":"tool/result","data":{"message":{},"error":{"code":"interrupted"}}}`)
		},
	} {
		call := valid()
		change(call)
		if got := toolDiffModelFor(call); got != nil {
			t.Errorf("declining lifecycle/receipt acquired %#v", got)
		}
	}
	for _, raw := range []string{"", "{", "null", "[]", `{}`, `{"type":"tool/result"}`, `{"type":"tool/result","data":null}`, `{"type":"tool/result","data":{"message":null}}`, `{"type":"tool/result","data":{"message":[]}}`} {
		call := valid()
		call.Result.Raw = json.RawMessage(raw)
		if got := toolDiffModelFor(call); got != nil {
			t.Errorf("malformed receipt %s acquired %#v", raw, got)
		}
	}
	call := valid()
	call.RootCallID = call.CallID
	call.Result.EventType = ""
	if got := toolDiffModelFor(call); got == nil {
		t.Fatal("valid raw root receipt without redundant EventType declined")
	}
}
