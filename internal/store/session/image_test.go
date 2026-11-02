package session

import (
	"encoding/json"
	"testing"
)

func TestExtractTextishImageBlocksKeepTrailingTextSeparate(t *testing.T) {
	raw := json.RawMessage(`{"content":[{"type":"text","text":"Before"},{"type":"image","path":"./a_(1).png"},{"type":"text","text":"After"}]}`)
	if got := extractTextish(raw); got != "Before\n[image] ./a_(1).png\nAfter" {
		t.Fatalf("mixed block order=%q", got)
	}
	raw = json.RawMessage(`"![one](a_(1).png) ![two](b.png)"`)
	if got := extractTextish(raw); got != "[image] a_(1).png\n[image] b.png" {
		t.Fatalf("multiple image-only string=%q", got)
	}
}

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
