package chat

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
)

func fileToolFixture(name, args string, meta any, text string) *session.ToolCallView {
	raw, _ := json.Marshal(map[string]any{"type": "tool/result", "data": map[string]any{"meta": meta, "message": map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}}}})
	return &session.ToolCallView{CallID: "file-call", Name: name, Arguments: args, State: "ok", Call: &session.Message{EventType: "tool/call"}, Result: &session.Message{EventType: "tool/result", Raw: raw}}
}
func TestToolFileReadContract(t *testing.T) {
	meta := map[string]any{"path": "src/a.go", "offset": 12, "lines": []any{map[string]any{"number": 12, "text": "hello"}}, "totalLines": 20}
	call := fileToolFixture("read", `{"file_path":"src/a.go","offset":12,"limit":2}`, meta, "<path>src/a.go</path>\n<type>file</type>\n<content>\n12: hello\n</content>")
	card := toolFileCardFor(call)
	if card.Path != "src/a.go" || card.Line != 12 || card.Read == nil || card.Read.Lines[0].Text != "hello" {
		t.Fatalf("read contract lost: %+v", card)
	}
	for _, args := range []string{`{"file_path":"src/a.go","offset":0}`, `{"file_path":"src/a.go","offset":1.5}`, `{"file_path":"src/a.go","offset":"12"}`, `{"file_path":"src/a.go","limit":null}`, `{"file_path":"src/a.go"`} {
		call.Arguments = args
		if got := toolFileCardFor(call); got.Path != "" {
			t.Fatalf("invalid read became link: %s", args)
		}
	}
}
func TestToolFileMetadataFallback(t *testing.T) {
	for _, meta := range []any{
		map[string]any{"path": "a", "offset": 1, "lines": []any{map[string]any{"number": 1}}, "totalLines": 1},
		map[string]any{"path": "a", "offset": 2, "lines": []any{map[string]any{"number": 1, "text": "backwards"}}, "totalLines": 2},
		map[string]any{"path": "a", "offset": 1, "lines": []any{}, "totalLines": nil},
	} {
		call := fileToolFixture("read", `{"file_path":"a"}`, meta, "<path>a</path>\n<type>file</type>\n<content>\nx\n</content>")
		if got := toolFileCardFor(call); got.Read != nil || got.Path != "a" {
			t.Fatalf("bad metadata lost generic fallback: %+v", got)
		}
	}
}
func TestToolFileMutationContract(t *testing.T) {
	call := fileToolFixture("edit", `{"file_path":"a","old_string":"old","new_string":"new"}`, map[string]any{"diffs": []any{map[string]any{"path": "a", "oldText": "actual old", "newText": "actual new"}}}, "done")
	got := toolFileCardFor(call)
	if len(got.Diffs) != 1 || *got.Diffs[0].OldText != "actual old" || got.Intended {
		t.Fatalf("applied diff missing: %+v", got)
	}
	call.Result = nil
	call.State = "running"
	got = toolFileCardFor(call)
	if !got.Intended || len(got.Diffs) != 1 || *got.Diffs[0].OldText != "old" {
		t.Fatalf("running intended diff missing: %+v", got)
	}
	call = fileToolFixture("edit", call.Arguments, map[string]any{"diffs": []any{map[string]any{"path": "a", "newText": "new"}}}, "done")
	if got = toolFileCardFor(call); len(got.Diffs) != 0 {
		t.Fatal("malformed applied metadata fabricated edit")
	}
	call = fileToolFixture("write", `{"file_path":"a","content":"whole file"}`, map[string]any{"diffs": []any{}}, "done")
	if got = toolFileCardFor(call); len(got.Diffs) != 1 || got.Diffs[0].NewText != "whole file" {
		t.Fatal("new write fallback missing")
	}
}
func TestToolFileAuthorityBoundaries(t *testing.T) {
	call := fileToolFixture("bash", `{"file_path":"secret","command":"cat secret"}`, nil, "/tmp/secret:99")
	if got := toolFileCardFor(call); got.Path != "" {
		t.Fatal("unknown tool became file link")
	}
	call.Name = "read"
	for _, state := range []string{"preparing", "error", "stopped"} {
		call.State = state
		if got := toolFileCardFor(call); got.Path != "" {
			t.Fatalf("%s became file link", state)
		}
	}
	call.State = "ok"
	call.Call = nil
	if got := toolFileCardFor(call); got.Path != "" {
		t.Fatal("orphan result became file link")
	}
	for _, path := range []string{"", "\n/tmp/a", "https://example.org/a"} {
		if toolFilePath(path) {
			t.Fatalf("unsafe path %q", path)
		}
	}
}

func TestToolFileExplicitClickAndReadOnlyBoundary(t *testing.T) {
	call := fileToolFixture("read", `{"file_path":"src/a.go","offset":12}`, nil, "raw result")
	opened := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ToolMessageView(c, session.Message{Tool: call}, ToolCardStyle{FontSize: 12}, ToolImageGalleryOptions{OnOpenFile: func(path string) { opened = path }})
	}, 800, 600)
	tt.Frame()
	if opened != "" {
		t.Fatal("render opened file")
	}
	if err := tt.Click("src/a.go:12"); err != nil {
		t.Fatal(err)
	}
	if opened != "src/a.go:12" {
		t.Fatalf("lost read offset: %q", opened)
	}
	opened = ""
	tt = ui.NewTester(func(c *ui.Context) {
		ToolMessageView(c, session.Message{Tool: call}, ToolCardStyle{FontSize: 12}, ToolImageGalleryOptions{TextOnly: true, OnOpenFile: func(path string) { opened = path }})
	}, 800, 600)
	tt.Frame()
	if err := tt.Click("src/a.go:12"); err == nil {
		t.Fatal("read-only resource exposed active file link")
	}
	if opened != "" {
		t.Fatal("read-only resource opened file")
	}
}
