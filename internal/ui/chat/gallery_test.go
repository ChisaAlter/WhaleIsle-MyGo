package chat

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// The transcript gallery renders components that until now only had tests, so
// a reviewer can look at them. It is software-rendered with ui.NewTester, not a
// live window: fonts and shadows come from the same theme tokens, but native
// text shaping and the real compositor are not exercised.
//
// Run it with a destination directory:
//
//	CHAT_GALLERY_DIR=docs/review-20261020/evidence/after go test ./internal/ui/chat -run TestTranscriptGallery
//
// Every image documented here lives in docs/review-20261020/evidence/after/.
func galleryDir(t *testing.T) string {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("CHAT_GALLERY_DIR"))
	if dir == "" {
		t.Skip("set CHAT_GALLERY_DIR to render the transcript gallery")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("gallery dir: %v", err)
	}
	return dir
}

func writeGallery(t *testing.T, dir, name string, tt *ui.Tester) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		_ = f.Close()
		t.Fatalf("encode %s: %v", name, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", name, err)
	}
}

// galleryTheme resolves the light palette the evidence screenshots use.
func galleryTheme() (theme.Colors, ui.Color, ui.Color) {
	colors := theme.Resolve("deepseek.light").Light
	return colors, chrome.Color(colors.FG), chrome.Color(colors.Muted)
}

func galleryImage(t *testing.T, dir, name string, w, h int, fill color.RGBA) string {
	t.Helper()
	path := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("fixture image: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("fixture png: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("fixture close: %v", err)
	}
	return path
}

// transcript-user-attachments.png — the user row: one durable PDF attachment
// card, one broken image failure card and a message with two 64 DIP tiles.
func TestTranscriptGalleryUserAttachments(t *testing.T) {
	dir := galleryDir(t)
	// Fixtures live outside the evidence folder: only transcript-* belongs there.
	fixtures := t.TempDir()
	shotA := galleryImage(t, fixtures, "fixture-shot-a.png", 320, 180, color.RGBA{58, 122, 196, 255})
	shotB := galleryImage(t, fixtures, "fixture-shot-b.png", 180, 320, color.RGBA{96, 156, 116, 255})
	missing := filepath.Join(fixtures, "fixture-missing.png")
	withFile := session.Message{
		ID: "u1", Role: "user", Kind: session.MessageKindText, Seq: 1, EventTime: 1700000000000,
		Body: "[image] " + missing + "\n请看一下这份季度报告",
		Raw: []byte(`{"type":"user/message","seq":1,"data":{"content":[
			{"type":"text","text":"请看一下这份季度报告"},
			{"type":"file","attachment":{"attachmentId":"a1","name":"quarterly-report.pdf","mediaType":"application/pdf","bytes":2411724}}
		]}}`),
	}
	tiles := session.Message{
		ID: "u2", Role: "user", Kind: session.MessageKindText, Seq: 3, EventTime: 1700000060000,
		Body: "[image] " + shotA + "\n[image] " + shotB + "\n两张截图都在这里",
	}
	colors, _, _ := galleryTheme()
	m := Model{Snapshot: session.ViewState{Messages: []session.Message{withFile, tiles}}, ContentWidth: 720, ThemeID: "deepseek.light"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(0, 32, 0, 32).Gap(4).Children(func() {
			for _, msg := range []session.Message{withFile, tiles} {
				refs, body := userMessageParts(msg.Body)
				userMessageView(c, m, msg, colors, refs, body, nil)
			}
		})
	}, 1200, 760)
	writeGallery(t, dir, "transcript-user-attachments.png", tt)
}

// transcript-assistant-markdown.png — assistant Markdown with the block ladder:
// h2, bullets, a blockquote with its rule, inline code, a fenced Go code card
// with toolbar and a table with a filled header.
func TestTranscriptGalleryAssistantMarkdown(t *testing.T) {
	dir := galleryDir(t)
	body := strings.Join([]string{
		"## 写入失败的三个原因",
		"",
		"先说结论：**权限**、路径与并发写。检查 `fs.writeFile` 的返回值即可。",
		"",
		"- 目标目录只读",
		"- 路径里有 `~` 没有被展开",
		"- 两个进程同时写同一个文件",
		"",
		"> 原版在写入前会先 `stat` 父目录，这一步最容易漏。",
		"",
		"```go",
		"func writeOnce(path string, body []byte) error {",
		"\tif err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {",
		"\t\treturn err",
		"\t}",
		"\treturn os.WriteFile(path, body, 0o644)",
		"}",
		"```",
		"",
		"| 现象 | 根因 | 处理 |",
		"|---|---|---|",
		"| EACCES | 目录只读 | 换到工作区目录 |",
		"| ENOENT | 父目录不存在或无法读取 | 先 MkdirAll |",
		"| 内容被截断 | 两个进程同时写同一个文件 | 加锁或写临时文件再改名 |",
	}, "\n")
	msg := session.Message{ID: "a1", Role: "assistant", Kind: session.MessageKindText, Seq: 4, EventTime: 1700000090000, Body: body}
	colors, fg, muted := galleryTheme()
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(24, 32, 24, 32).Children(func() {
			style := transcriptMarkdownStyle(fg, muted, chrome.Color(colors.Field), chrome.Color(colors.Accent), 14, false)
			// The transcript's message axis: tables only scroll past this width.
			style.AvailableWidth = 720
			markdown.View(c, msg.Body, style)
		})
	}, 1200, 1150)
	writeGallery(t, dir, "transcript-assistant-markdown.png", tt)
}

// transcript-tools.png — the expanded bash tool row (single-line disclosure plus
// its terminal seat holding command and output) above a collapsed read row, and
// a second bash row whose long output scrolls inside the same seat.
func TestTranscriptGalleryToolRows(t *testing.T) {
	dir := galleryDir(t)
	colors, fg, muted := galleryTheme()
	st := ToolCardStyle{FG: fg, Muted: muted, Border: chrome.Color(colors.Border), Field: chrome.Color(colors.Field),
		Accent: chrome.Color(colors.Accent), OK: ui.RGB(61, 153, 112), Err: ui.RGB(200, 70, 70), FontSize: 13}
	bashResult := `{"type":"tool/result","seq":6,"data":{"message":{"content":[{"type":"text","text":"ok  \twhaleisle/internal/ui/chat\t2.41s\nok  \twhaleisle/internal/ui/frame\t9.02s\nFAIL\twhaleisle/internal/ui/composer\t7.57s\nPASS\n"}]}}}`
	bash := session.Message{ID: "t1", Role: "assistant", Kind: session.MessageKindTool, Seq: 5, EventTime: 1700000100000,
		Tool: &session.ToolCallView{CallID: "call-bash", Name: "bash", State: "ok",
			Arguments: `{"command":"go test ./internal/ui/chat ./internal/ui/frame ./internal/ui/composer -count=1"}`,
			Result:    &session.Message{Kind: session.MessageKindTool, Role: "tool", EventType: "tool/result", Raw: []byte(bashResult)}}}
	read := session.Message{ID: "t2", Role: "assistant", Kind: session.MessageKindTool, Seq: 7, EventTime: 1700000110000,
		Tool: &session.ToolCallView{CallID: "call-read", Name: "read", State: "ok",
			Arguments: `{"path":"internal/ui/chat/chat.go","offset":96,"limit":40}`}}
	// One very wide line proves the seat scrolls horizontally instead of
	// widening the transcript.
	wideResult := `{"type":"tool/result","seq":10,"data":{"message":{"content":[{"type":"text","text":"` +
		strings.Repeat("ok      whaleisle/internal/ui/widgets/markdown     1.20s                                      ", 3) + `\nPASS\n"}]}}}`
	wide := session.Message{ID: "t3", Role: "assistant", Kind: session.MessageKindTool, Seq: 9, EventTime: 1700000120000,
		Tool: &session.ToolCallView{CallID: "call-run", Name: "run", State: "ok",
			Arguments: `{"command":"go test ./internal/ui/widgets/markdown -run TestTable -count=1"}`,
			Result:    &session.Message{Kind: session.MessageKindTool, Role: "tool", EventType: "tool/result", Raw: []byte(wideResult)}}}
	expanded := map[string]bool{"t1|tool:call-bash": true, "t3|tool:call-run": true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(24, 32, 24, 32).Gap(10).Children(func() {
			ToolMessageView(c, bash, st, ToolImageGalleryOptions{ID: "t1", Expanded: &expanded})
			ToolMessageView(c, read, st, ToolImageGalleryOptions{ID: "t2", Expanded: &expanded})
			ToolMessageView(c, wide, st, ToolImageGalleryOptions{ID: "t3", Expanded: &expanded})
		})
	}, 1200, 760)
	writeGallery(t, dir, "transcript-tools.png", tt)
}

// transcript-tool-kinds.png — the non-terminal tool families, one expanded row
// each: a grep hit list (path:line + highlighted snippet + overflow count), a web
// search with title/domain/snippet cards, and a todo checklist with state dots.
func TestTranscriptGalleryToolKinds(t *testing.T) {
	dir := galleryDir(t)
	colors, fg, muted := galleryTheme()
	st := ToolCardStyle{FG: fg, Muted: muted, Border: chrome.Color(colors.Border), Field: chrome.Color(colors.Field),
		Accent: chrome.Color(colors.Accent), OK: ui.RGB(61, 153, 112), Err: ui.RGB(200, 70, 70), FontSize: 13}
	result := func(seq int, text string) *session.Message {
		raw := []byte(`{"type":"tool/result","seq":` + itoaFast(seq) + `,"data":{"message":{"content":[{"type":"text","text":` + strconv.Quote(text) + `}]}}}`)
		return &session.Message{Kind: session.MessageKindTool, Role: "tool", EventType: "tool/result", Raw: raw}
	}
	searchOutput := strings.Join([]string{
		"internal/ui/chat/chat.go:145: if len(msgs) == 0 {",
		"internal/ui/chat/chat.go:146: ui.Text(c, i18n.T(i18n.KeyChatEmpty))",
		"internal/ui/chat/transcript_preferences.go:88: if len(msgs) == 0 {",
		"internal/ui/chat/tool_lifecycle.go:64: output := toolLifecycleOutputFor(call, true)",
		"internal/ui/chat/gallery_test.go:12: len(msgs) == 0",
	}, "\n")
	search := session.Message{ID: "k1", Role: "assistant", Kind: session.MessageKindTool, Seq: 11, EventTime: 1700000200000,
		Tool: &session.ToolCallView{CallID: "call-grep", Name: "grep", State: "ok",
			Arguments: `{"pattern":"len(msgs) == 0"}`, Result: result(12, searchOutput)}}
	web := session.Message{ID: "k2", Role: "assistant", Kind: session.MessageKindTool, Seq: 13, EventTime: 1700000210000,
		Tool: &session.ToolCallView{CallID: "call-web", Name: "web_search", State: "ok",
			Arguments: `{"query":"gofmt line endings CRLF"}`,
			Result: &session.Message{Kind: session.MessageKindTool, Role: "tool", EventType: "tool/result",
				Web: &session.WebResult{Kind: "search", Truncated: false, Sources: []session.WebSource{
					{URL: "https://www.example.com/go/gofmt", Title: "gofmt rewrites files in place", Snippet: "gofmt -w rewrites every file, which turns CRLF into LF and shows up as a whole-file diff.", PublishedAt: "2026-02-11"},
					{URL: "https://docs.example.org/text", Title: "Text normalization", Snippet: "Normalize line endings before comparing generated files."},
					}}}}}
	todo := session.Message{ID: "k3", Role: "assistant", Kind: session.MessageKindTool, Seq: 15, EventTime: 1700000220000,
		Tool: &session.ToolCallView{CallID: "call-todo", Name: "todo", State: "ok",
			Arguments: `{"todos":[]}`,
			Result: result(16, "- [x] 修好表格列宽\n- [~] 复核工具行画廊\n- [ ] 补 terminalBlock 深色截图\n- 收尾更新 README")}}
	expanded := map[string]bool{"k1|tool:call-grep": true, "k2|tool:call-web": true, "k3|tool:call-todo": true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(24, 32, 24, 32).Gap(12).Children(func() {
			ToolMessageView(c, search, st, ToolImageGalleryOptions{ID: "k1", Expanded: &expanded, OnOpenURL: func(string) {}})
			ToolMessageView(c, web, st, ToolImageGalleryOptions{ID: "k2", Expanded: &expanded, OnOpenURL: func(string) {}})
			ToolMessageView(c, todo, st, ToolImageGalleryOptions{ID: "k3", Expanded: &expanded})
		})
	}, 1200, 900)
	writeGallery(t, dir, "transcript-tool-kinds.png", tt)
}

// transcript-turn-failure.png — a failed turn with no API key: localized
// headline, the fix action and the raw Host error behind the details toggle.
func TestTranscriptGalleryTurnFailure(t *testing.T) {
	dir := galleryDir(t)
	colors, fg, muted := galleryTheme()
	raw := "llm-deepseek: no API key for provider route deepseek/deepseek-chat MISSING_API_KEY"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(24, 32, 24, 32).Gap(18).Children(func() {
			paintTurnFailure(c, "api-key", classifyTurnFailure(raw, "MISSING_API_KEY"), colors, fg, muted, func() {})
			details := ui.Local(c.Root(), "turn-failure-details", func() bool { return false })
			_ = details
			paintTurnFailure(c, "generic", classifyTurnFailure("HTTP 502 from upstream", ""), colors, fg, muted, nil)
		})
	}, 1200, 320)
	writeGallery(t, dir, "transcript-turn-failure.png", tt)
}
// transcript-queue-dock.png — the composer queue dock with two rows, previews
// and the icon-only edit/steer/remove actions.
func TestTranscriptGalleryQueueDock(t *testing.T) {
	dir := galleryDir(t)
	m := PlanBoardModel{
		ThemeID: "deepseek.light",
		Board: session.PlanBoard{
			Queue: []session.PlanQueueItem{
				{ID: "q1", Preview: "把 writeOnce 的临时文件改名逻辑补上，并加一个单测"},
				{ID: "q2", Preview: "顺手看一下 CI 为什么在 Windows 上慢"},
			},
			QueueActions: session.QueueActionState{Mutable: true, Running: true},
		},
		OnQueueEdit:   func(string, string) {},
		OnQueueSteer:  func(string) {},
		OnQueueRemove: func(string) {},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(24, 32, 24, 32).Children(func() { ui.Column(c).Width(720).Children(func() { QueueDockView(c, m) }) })
	}, 1200, 260)
	writeGallery(t, dir, "transcript-queue-dock.png", tt)
}

// transcript-turn-rail.png — the floating turn rail with four ticks, the last
// one accent-filled as the current turn.
func TestTranscriptGalleryTurnRail(t *testing.T) {
	dir := galleryDir(t)
	var messages []session.Message
	for index, prompt := range []string{"先看写入失败的报错", "确认父目录权限", "补一个并发写的单测", "跑一遍三平台 CI"} {
		messages = append(messages, session.Message{
			ID: "u" + itoaFast(index), Role: "user", Kind: session.MessageKindText,
			Seq: float64(index*2 + 1), Body: prompt,
		})
		messages = append(messages, session.Message{
			ID: "a" + itoaFast(index), Role: "assistant", Kind: session.MessageKindText,
			Seq: float64(index*2 + 2), Body: "收到。",
		})
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).FillWidth().Padding(24, 32, 24, 32).Children(func() {
			ui.Box(c).Grow(1)
			TurnRailView(c, Model{Snapshot: session.ViewState{Messages: messages}, ThemeID: "deepseek.light"}, 900, func(float64) {})
		})
	}, 1200, 300)
	writeGallery(t, dir, "transcript-turn-rail.png", tt)
	_ = i18n.T(i18n.KeyChatTurnRailLabel)
}
