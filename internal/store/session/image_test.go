package session

import (
	"encoding/json"
	"testing"
)

func TestExtractTextishImageBlock(t *testing.T) {
	raw := json.RawMessage(`{"content":[{"type":"image","path":"/tmp/shot.png"}]}`)
	got := extractTextish(raw)
	want := "[image] /tmp/shot.png"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExtractTextishMarkdownImage(t *testing.T) {
	raw := json.RawMessage(`"![screenshot](assets/demo.jpg)"`)
	got := extractTextish(raw)
	want := "[image] assets/demo.jpg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExtractTextishImagePathString(t *testing.T) {
	raw := json.RawMessage(`"C:\\Users\\x\\pic.webp"`)
	got := extractTextish(raw)
	want := "[image] C:\\Users\\x\\pic.webp"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestClassifyUserMessageWithImage(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "user/message",
		"seq": 1,
		"data": {"content":[{"type":"image","url":"https://example.com/a.png"}]}
	}`)
	msg, ok := classifyPersistenceEvent(raw)
	if !ok || msg.Body != "[image] https://example.com/a.png" {
		t.Fatalf("%#v ok=%v", msg, ok)
	}
}
