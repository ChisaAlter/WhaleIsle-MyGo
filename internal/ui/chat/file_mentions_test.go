package chat

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/widgets/markdown"
)

func TestInlineFileMentionsFixture(t *testing.T) {
	event := func(kind string, seq float64, data map[string]any) session.Message {
		raw, _ := json.Marshal(map[string]any{"type": kind, "seq": seq, "surfaceOp": "append", "data": data})
		return session.Message{Role: "assistant", Kind: session.MessageKindText, Seq: seq, Raw: raw}
	}
	delivered := func(seq, turn float64, paths ...string) session.Message {
		files := []map[string]string{}
		for _, p := range paths {
			files = append(files, map[string]string{"path": p})
		}
		return event("deliverables/presented", seq, map[string]any{"turn": turn, "files": files})
	}
	call := func(id, name, args string, seq float64) session.Message {
		return event("tool/call", seq, map[string]any{"turn": 0, "callId": id, "name": name, "arguments": args})
	}
	result := func(id string, seq float64, failed bool) session.Message {
		return event("tool/result", seq, map[string]any{"turn": 0, "message": map[string]any{"isError": failed, "source": map[string]string{"callId": id}}})
	}
	closing := event("assistant/message", 20, map[string]any{"turn": 0})
	messages := []session.Message{
		delivered(2, 0, "out/report.md", "a/same.md", "b/same.md", `C:\output\win.txt`),
		delivered(3, 1, "other-turn.md"), delivered(21, 0, "future.md"),
		call("write", "write", `{"file_path":"out/written.md","content":"ok"}`, 4), result("write", 5, false),
		call("read", "read", `{"file_path":"read.md"}`, 6), result("read", 7, false),
		call("failed", "write", `{"file_path":"failed.md","content":"ok"}`, 8), result("failed", 9, true),
		call("bad", "edit", `{"file_path":"bad.md","old_string":"same","new_string":"same"}`, 10), result("bad", 11, false),
		call("pending", "write", `{"file_path":"pending.md","content":"ok"}`, 12),
		event("tool/ptc-dispatch", 13, map[string]any{"rootCallId": "write", "name": "write", "arguments": map[string]string{"file_path": "located.md", "content": "ok"}}),
		event("deliverables/presented", 14, map[string]any{"files": []map[string]string{{"path": "missing-turn.md"}}}),
	}
	resolve := FileMentionResolver(messages, closing)
	if resolve == nil {
		t.Fatal("no resolver")
	}
	for token, want := range map[string]string{"report.md": "out/report.md", "out/report.md": "out/report.md", "written.md": "out/written.md", "win.txt": `C:\output\win.txt`, "same.md": "", "a/same.md": "a/same.md", "other-turn.md": "", "future.md": "", "read.md": "", "failed.md": "", "bad.md": "", "pending.md": "", "located.md": "located.md", "missing-turn.md": "", "guessed.pdf": ""} {
		if got := resolve(token); got != want {
			t.Fatalf("%q => %q, want %q", token, got, want)
		}
	}
	unknown := closing
	unknown.Raw = json.RawMessage(`{"type":"assistant/message","seq":20,"data":{}}`)
	unknown.TurnSeq = 1
	if FileMentionResolver(messages, unknown) != nil {
		t.Fatal("inferred turn from edit projection")
	}
	opened := ""
	resolved := map[string]bool{}
	tt := ui.NewTester(func(c *ui.Context) {
		markdown.View(c, "Ready `report.md`, ambiguous `same.md`.\n\n```\nwritten.md\n```", markdown.Style{FontSize: 14, ResolveFile: func(value string) string { resolved[value] = true; return resolve(value) }, OnOpenFile: func(p string) { opened = p }})
	}, 800, 400)
	tt.Frame()
	if opened != "" {
		t.Fatal("render opened file")
	}
	if err := tt.Click("report.md"); err != nil {
		t.Fatal(err)
	}
	if opened != "out/report.md" {
		t.Fatalf("inline click opened %q", opened)
	}
	if resolved["written.md"] || resolved["written.md\n"] {
		t.Fatal("fenced code entered mention resolution")
	}
	if !resolved["same.md"] {
		t.Fatal("ambiguous inline token was not checked")
	}
}
