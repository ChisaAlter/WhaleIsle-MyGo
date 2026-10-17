package diff

import (
	"strings"
	"testing"
)

func TestIsUnifiedPatch(t *testing.T) {
	patch := "--- a/foo.go\n+++ b/foo.go\n@@ -1,3 +1,3 @@\n line\n-old\n+new\n"
	if !IsUnifiedPatch(patch) {
		t.Fatal("expected patch")
	}
	if IsUnifiedPatch("hello world") {
		t.Fatal("plain text")
	}
}

func TestExtractPatchTextFromFence(t *testing.T) {
	body := "Applied patch:\n```diff\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n```\n"
	got := ExtractPatchText(body)
	if !IsUnifiedPatch(got) {
		t.Fatalf("got %q", got)
	}
}

func TestSpansColors(t *testing.T) {
	src := "--- a\n+++ b\n@@\n+add\n-del\n ctx\n"
	spans := Spans(src, Style{FontSize: 12})
	if len(spans) < 4 {
		t.Fatalf("len %d", len(spans))
	}
	if !strings.Contains(spans[3].Text, "+add") {
		t.Fatalf("%#v", spans[3])
	}
}

func TestLineDiffHelper(t *testing.T) {
	out := LineDiff("alpha\nbeta", "alpha\ngamma")
	if out == "" {
		t.Fatal("empty diff")
	}
}
