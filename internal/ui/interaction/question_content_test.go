package interaction

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestQuestionDetailMarkdownCodeCopyAndWrap(t *testing.T) {
	code := "fmt.Println(\"" + strings.Repeat("long-code-", 12) + "\")\n\n"
	questions, err := json.Marshal([]map[string]any{{"id": "q", "header": "Heading", "question": "Question", "detail": "**Important**\n\n```go\n" + code + "```\n\nAfter code"}})
	if err != nil {
		t.Fatal(err)
	}
	store, tt, outcome := questionEditorFixture(t, string(questions))
	tt.SetSize(340, 1500)
	if _, ok := tt.Find("Copy code"); !ok {
		t.Fatal("question detail has no Markdown code toolbar")
	}
	if tt.HasText("**Important**") || !tt.HasText("Important") || !tt.HasText("After code") {
		t.Fatal("question detail is raw Markdown or lost surrounding prose")
	}
	if err := tt.Click("Copy code"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != code {
		t.Fatalf("code copy changed source bytes: %q", got)
	}
	if _, ok := tt.Find("Unwrap lines"); !ok {
		t.Fatal("question code should initially wrap like the original composer")
	}
	if err := tt.Click("Unwrap lines"); err != nil {
		t.Fatal(err)
	}
	before, ok := tt.Find("Question answer: Question")
	if !ok {
		t.Fatal("answer missing below Markdown")
	}
	if err := tt.Click("Wrap lines"); err != nil {
		t.Fatal(err)
	}
	after, _ := tt.Find("Question answer: Question")
	if _, ok := tt.Find("Unwrap lines"); !ok || after.Y <= before.Y {
		t.Fatalf("long code did not wrap: before=%+v after=%+v", before, after)
	}
	if err := tt.Click("Unwrap lines"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Question answer: Question"); err != nil {
		t.Fatal(err)
	}
	tt.Type("line one")
	tt.Key(ui.Shift, ui.KeyEnter)
	tt.Type("line two")
	if got := store.SnapshotForAgent("A").QuestionCards["e"].Rows[0].Custom; got != "line one\nline two" {
		t.Fatalf("Markdown interrupted multiline editor: %q", got)
	}
	select {
	case got := <-outcome:
		t.Fatalf("reading detail submitted the request: %+v", got)
	default:
	}
	store.ResolveQuestions("e")
	tt.Frame()
	_ = tt.Click("Wrap lines")
	if _, ok := tt.Find("Wrap lines"); !ok {
		t.Fatal("Sending changed question detail controls")
	}
	_ = tt.Click("Question answer: Question")
	tt.Type("forbidden")
	if got := store.SnapshotForAgent("A").QuestionCards["e"].Rows[0].Custom; got != "line one\nline two" {
		t.Fatalf("Sending changed answer: %q", got)
	}
}

func TestQuestionLongHeaderQuestionAndEveryOptionDescriptionVisible(t *testing.T) {
	header := strings.Repeat("Complete heading ", 5)
	question := strings.Repeat("A complete question remains readable. ", 4)
	var options []map[string]string
	for i := range 6 {
		options = append(options, map[string]string{
			"label":       fmt.Sprintf("Option %d %s", i+1, strings.Repeat("full label ", 5)),
			"description": fmt.Sprintf("Description %d %s", i+1, strings.Repeat("fully visible detail ", 5)),
		})
	}
	questions, err := json.Marshal([]map[string]any{{"id": "q", "header": header, "question": question, "options": options}})
	if err != nil {
		t.Fatal(err)
	}
	_, tt, _ := questionEditorFixture(t, string(questions))
	tt.SetSize(340, 2400)
	texts := []string{header, question}
	for _, option := range options {
		texts = append(texts, option["label"], option["description"])
	}
	for _, text := range texts {
		r, ok := tt.Find(text)
		// Content below the visible body stays reachable by its own scrollport;
		// the title and footer do not need to leave the window.
		for attempt := 0; attempt < 100 && (!ok || r.H <= 20); attempt++ {
			body, found := tt.Find("Question content")
			if !found {
				break
			}
			tt.Scroll(body.X+body.W/2, body.Y+body.H/2, 0, 40)
			r, ok = tt.Find(text)
		}
		if !ok {
			t.Fatalf("complete text missing: %q", text)
		}
		if r.X < 0 || r.X+r.W > 340.5 || r.H <= 20 {
			t.Fatalf("long text is not wrapped inside the card: %q rect=%+v", text, r)
		}
	}
}

func TestQuestionContentLongUnbrokenOptionFitsNarrowCard(t *testing.T) {
	label := strings.Repeat("LongUnbrokenIdentifier", 12)
	questions, _ := json.Marshal([]map[string]any{{"id": "q", "question": "Question", "options": []map[string]string{{"label": label}}}})
	_, tt, _ := questionEditorFixture(t, string(questions))
	tt.SetSize(340, 900)
	r, ok := tt.Find(label)
	if !ok || r.X+r.W > 340.5 || r.H <= 20 {
		t.Fatalf("unbroken option overflows the answer area: %+v found=%v", r, ok)
	}
}

func TestQuestionContentLongBodyScrollKeepsFooterReachable(t *testing.T) {
	var options []map[string]string
	for i := range 20 {
		options = append(options, map[string]string{"label": fmt.Sprintf("Option %d", i+1), "description": strings.Repeat("Full option description ", 5)})
	}
	questions, _ := json.Marshal([]map[string]any{{"id": "q", "header": "Heading", "question": "Question", "options": options}})
	_, tt, _ := questionEditorFixture(t, string(questions))
	tt.SetSize(600, 700)
	footer, ok := tt.Find("提交 / Submit")
	if !ok || footer.Y+footer.H > 700 {
		t.Fatalf("long body pushed footer beyond the window: %+v found=%v", footer, ok)
	}
	first, _ := tt.Find("Option 1")
	tt.Scroll(first.X+20, first.Y+10, 0, 4000)
	last, ok := tt.Find("Option 20")
	if !ok || last.Y < 0 || last.Y+last.H > footer.Y {
		t.Fatalf("last option cannot be reached by scrolling: %+v found=%v", last, ok)
	}
	if err := tt.Click("收起问题 / Minimize question"); err != nil {
		t.Fatal(err)
	}
	_, hasHeader := tt.Find("Heading")
	_, hasQuestion := tt.Find("Question")
	if !hasHeader || !hasQuestion {
		t.Fatal("minimized card dropped its header or question title")
	}
}
