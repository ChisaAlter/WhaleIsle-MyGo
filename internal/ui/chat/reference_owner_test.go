package chat

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestPTCInlineFileLocationsCanonicalAliasesAndAmbiguity(t *testing.T) {
	event := func(raw string) session.Message { return session.Message{Role: "assistant", Raw: json.RawMessage(raw)} }
	closing := event(`{"type":"assistant/message","seq":20,"data":{"turn":1}}`)
	messages := []session.Message{
		event(`{"type":"tool/ptc-dispatch","seq":1,"data":{"rootCallId":"guessed","name":"write","arguments":{"file_path":"unlocated.md","content":"ok"}}}`),
		event(`{"type":"step/start","seq":2,"data":{"turn":1,"step":0}}`),
		event(`{"type":"tool/ptc-dispatch","seq":3,"data":{"name":"write","arguments":{"file_path":"relative.md","content":"ok"},"meta":{"path":"C:/recorded/relative.md"}}}`),
		event(`{"type":"tool/ptc-dispatch","seq":4,"data":{"name":"write","isError":true,"arguments":{"file_path":"failed.md","content":"ok"}}}`),
		event(`{"type":"tool/ptc-dispatch","seq":5,"data":{"name":"write","arguments":{"file_path":"uncertain.md","content":"ok"},"meta":{"path":"not/absolute"}}}`),
		event(`{"type":"tool/ptc-dispatch","seq":6,"data":{"name":"edit","arguments":{"file_path":"/absolute.md","old_string":"a","new_string":"b"},"meta":{"path":null}}}`),
		event(`{"type":"deliverables/presented","seq":7,"data":{"turn":1,"files":[{"path":"relative.md"}]}}`),
		event(`{"type":"tool/ptc-dispatch","seq":8,"data":{"turn":null,"name":"write","arguments":{"file_path":"session.md","content":"ok"}}}`),
		event(`{"type":"turn/end","seq":9,"data":{"turn":1}}`),
		event(`{"type":"tool/ptc-dispatch","seq":10,"data":{"rootCallId":"guessed","name":"write","arguments":{"file_path":"outside.md","content":"ok"}}}`),
	}
	resolve := FileMentionResolver(messages, closing)
	for token, want := range map[string]string{"relative.md": "", "C:/recorded/relative.md": "C:/recorded/relative.md", "absolute.md": "/absolute.md", "unlocated.md": "", "failed.md": "", "uncertain.md": "", "session.md": "", "outside.md": ""} {
		if got := resolve(token); got != want {
			t.Fatalf("%s => %s, want %s", token, got, want)
		}
	}
}

func TestHistoricalUserReferencesPreserveTextAndOnlyOpenFiles(t *testing.T) {
	text := `Look @"dir/my file.ts", @folder/ @[Alpha](dsh-session:ImEi) @Known title /review /unknown email@host @plain.md!`
	facts := userReferenceFacts{Labels: []string{"Known", "Known title"}, Skills: []string{"review"}}
	var rendered, original strings.Builder
	var files []string
	for _, run := range userReferenceRuns(text, facts) {
		rendered.WriteString(run.Text)
		if run.Raw != "" {
			original.WriteString(run.Raw)
		} else {
			original.WriteString(run.Text)
		}
		if run.Kind == "file" {
			files = append(files, run.Path)
		}
	}
	if original.String() != text || len(files) != 2 || files[0] != "dir/my file.ts" || files[1] != "plain.md" {
		t.Fatalf("reference grammar changed original text: %q %v", original.String(), files)
	}
	if strings.Contains(rendered.String(), "dsh-session:") || strings.Contains(rendered.String(), "@Known") || !strings.Contains(rendered.String(), "email@host") {
		t.Fatal("semantic labels or literal text lost")
	}
	opened := ""
	tt := ui.NewTester(func(c *ui.Context) {
		userReferenceText(c, text, 14, ui.RGB(0, 0, 0), ui.RGB(90, 90, 90), func(path string) { opened = path }, facts)
	}, 1200, 300)
	tt.Frame()
	if opened != "" {
		t.Fatal("render opened a file")
	}
	if err := tt.Click("my file.ts"); err != nil {
		t.Fatal(err)
	}
	if opened != "dir/my file.ts" {
		t.Fatalf("opened %q", opened)
	}
}

func TestHistoricalReferenceFactsUseRecallAdjacencyAndSkillBoundaries(t *testing.T) {
	human := session.Message{Role: "user", Kind: session.MessageKindText, Seq: 5, Body: "@Alpha /review"}
	context := func(seq float64, source string) session.Message {
		return session.Message{Role: "context", Kind: session.MessageKindContext, Seq: seq, Raw: json.RawMessage(`{"type":"user/message","data":{"source":` + source + `}}`)}
	}
	rows := []session.Message{human, context(6, `{"kind":"session-reference","references":[{"label":"Alpha"},{"label":"Alpha"}]}`), context(7, `{"kind":"skill-invocation","name":"review"}`), {Role: "assistant", Kind: session.MessageKindText, Seq: 8}, context(9, `{"kind":"skill-invocation","name":"other"}`)}
	labels := userMessageReferenceLabels(rows, 5)
	if len(labels) != 1 || labels[0] != "Alpha" || len(userMessageReferenceLabels(rows, 4)) != 0 {
		t.Fatal("recall guessed message identity")
	}
	names := userMessageSkillNames(rows, human)
	if len(names) != 1 || names[0] != "review" {
		t.Fatalf("skill crossed contribution boundary: %v", names)
	}
}
