package ui

import (
	"testing"
)

func TestTextSelectionStreamingAppendPreservesScopedSelection(t *testing.T) {
	first := "Static header"
	body := "Streaming prefix"
	showExtra := false

	tt := coreNewTester(func(c *context) {
		coreColumn(c).Selectable().Padding(16).Gap(8).Children(func() {
			coreText(c, first).Key("first")
			coreText(c, body).Key("stream")
			if showExtra {
				coreText(c, "Trailing paragraph").Key("extra")
			}
		})
	}, 480, 240)

	selectBetween(t, tt, first, 7, body, len([]rune("Streaming")))
	tt.SetClipboard("")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "header\nStreaming" {
		t.Fatalf("initial copy=%q, want %q", got, "header\\nStreaming")
	}

	// Streaming append with multi-byte Unicode runes must keep the prior selection prefix intact.
	body += " 你好🌟 tail"
	tt.Frame()
	tt.SetClipboard("")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "header\nStreaming" {
		t.Fatalf("after streaming append copy=%q, want %q", got, "header\\nStreaming")
	}

	// Appending another paragraph downstream during streaming also preserves the prefix selection.
	showExtra = true
	tt.Frame()
	tt.SetClipboard("")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "header\nStreaming" {
		t.Fatalf("after new paragraph append copy=%q, want %q", got, "header\\nStreaming")
	}

	// Non-prefix replacement still clears the scoped selection.
	body = "Completely replaced text"
	tt.Frame()
	tt.SetClipboard("stale")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "stale" {
		t.Fatalf("non-prefix edit left selection alive, copy=%q", got)
	}
}

func TestTextSelectionStreamingAppendPreservesStandaloneSelection(t *testing.T) {
	body := "中文前缀 alpha"
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(16).Children(func() {
			coreText(c, body).Key("solo").Selectable()
		})
	}, 420, 160)

	x0, y0 := selectableAt(t, tt, body, 0)
	x1, y1 := selectableAt(t, tt, body, 4)
	tt.Press(x0, y0)
	tt.Move(x1, y1)
	tt.Release(x1, y1)

	tt.SetClipboard("")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "中文前缀" {
		t.Fatalf("initial standalone copy=%q, want %q", got, "中文前缀")
	}

	body += " beta 新增后缀"
	tt.Frame()
	tt.SetClipboard("")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "中文前缀" {
		t.Fatalf("standalone copy after streaming append=%q, want %q", got, "中文前缀")
	}

	body = "不同文本内容"
	tt.Frame()
	tt.SetClipboard("stale")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "stale" {
		t.Fatalf("standalone non-prefix replacement should clear selection, copy=%q", got)
	}
}

func TestTextSelectionStreamingAppendInRichText(t *testing.T) {
	chunks := []string{"Hello ", "world"}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Selectable().Padding(16).Children(func() {
			coreRichText(c).Key("rich").Selectable().Children(func() {
				for _, chunk := range chunks {
					coreText(c, chunk)
				}
			})
		})
	}, 420, 160)

	initial := "Hello world"
	x0, y0 := selectableAt(t, tt, initial, 0)
	x1, y1 := selectableAt(t, tt, initial, 9)
	tt.Press(x0, y0)
	tt.Move(x1, y1)
	tt.Release(x1, y1)

	tt.SetClipboard("")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "Hello wor" {
		t.Fatalf("initial rich text copy=%q, want %q", got, "Hello wor")
	}

	chunks = append(chunks, " and 中文流")
	tt.Frame()
	tt.SetClipboard("")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "Hello wor" {
		t.Fatalf("rich text copy after streaming append=%q, want %q", got, "Hello wor")
	}
}
