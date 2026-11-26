package diff

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
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

func BenchmarkDiffViewSteady(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("--- a/file.go\n+++ b/file.go\n@@ -1,60 +1,60 @@\n")
	for i := 0; i < 60; i++ {
		sb.WriteString(" context line unchanged\n")
		sb.WriteString("-old removed line with payload\n")
		sb.WriteString("+new added line with payload\n")
	}
	src := sb.String()
	st := Style{
		FontSize: 12,
		FG:       ui.RGB(20, 20, 20),
		Add:      ui.RGB(0, 140, 60),
		Del:      ui.RGB(180, 30, 30),
		Hunk:     ui.RGB(40, 80, 180),
		Muted:    ui.RGB(120, 120, 120),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, src, st)
	}, 720, 480)
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}
