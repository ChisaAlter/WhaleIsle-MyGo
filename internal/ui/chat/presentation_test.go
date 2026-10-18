package chat

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func presentationMessage(body string) session.Message {
	return session.Message{ID: "user-1", Role: "user", Kind: session.MessageKindText, Seq: 7, Body: body, EventTime: 1700000000000}
}

func messageTester(msg session.Message, width int, configure func(*Model)) *ui.Tester {
	m := Model{Snapshot: session.ViewState{Messages: []session.Message{msg}}, OnQuoteMessage: func(string) {}, OnEditMessage: func(float64, string) {}}
	if configure != nil {
		configure(&m)
	}
	return ui.NewTester(func(c *ui.Context) { View(c, m) }, width, 600)
}

func rectNear(t *testing.T, got, want float32, what string) {
	t.Helper()
	if math.Abs(float64(got-want)) > 0.1 {
		t.Errorf("%s = %.2f, want %.2f", what, got, want)
	}
}

func TestUserMessageKeepsLiteralTextBeforeMarkdownAndDiff(t *testing.T) {
	for _, body := range []string{"hello **world**", "```go\n  fmt.Println(\"hello\")\n```", "```diff\n--- a/old\n+++ b/new\n@@ -1 +1 @@\n-old\n+new\n```", "first\n\n  third"} {
		t.Run(body, func(t *testing.T) {
			tt := messageTester(presentationMessage(body), 800, nil)
			if _, ok := tt.Find(body); !ok {
				t.Fatalf("literal user text missing; got %q", tt.Texts())
			}
		})
	}
}

func TestUserBubbleGeometryAndOutsideActions(t *testing.T) {
	for _, width := range []int{320, 560, 800, 1200} {
		for _, body := range []string{"hello", strings.Repeat("long content ", 30), "first\nsecond"} {
			tt := messageTester(presentationMessage(body), width, nil)
			bubble, ok := tt.Find("User message bubble")
			if !ok {
				t.Fatalf("bubble missing at width %d; texts=%q", width, tt.Texts())
			}
			cap := min(float32(748)*.702, float32(width-32)*.82)
			if bubble.W > cap+.1 {
				t.Errorf("width %d bubble=%v exceeds cap %.2f", width, bubble, cap)
			}
			rectNear(t, bubble.X+bubble.W, float32(width-16), "bubble right edge")
			bodyBox, _ := tt.Find(body)
			rectNear(t, bodyBox.X-bubble.X, 16, "bubble left padding")
			rectNear(t, bodyBox.Y-bubble.Y, 10, "bubble top padding")
			rectNear(t, bubble.H-bodyBox.H, 20, "bubble vertical padding")
			if body == "hello" {
				rectNear(t, bubble.H, 42, "single-line bubble height")
			}
			actions, ok := tt.Find("User message actions")
			if !ok {
				t.Fatal("action strip missing")
			}
			rectNear(t, actions.Y-(bubble.Y+bubble.H), 6, "actions outside bubble gap")
			rectNear(t, actions.H, 28, "action strip height")
			rectNear(t, actions.X+actions.W, float32(width-16), "actions right edge")
			copy, _ := tt.Find(i18n.T(i18n.KeyChatCopy))
			quote, _ := tt.Find(i18n.T(i18n.KeyChatQuote))
			edit, _ := tt.Find(i18n.T(i18n.KeyChatEdit))
			for _, r := range []ui.Rect{copy, quote, edit} {
				rectNear(t, r.W, 28, "action width")
				rectNear(t, r.H, 28, "action height")
				rectNear(t, r.Y, actions.Y, "action baseline")
			}
			rectNear(t, quote.X-(copy.X+copy.W), 8, "copy/quote gap")
			rectNear(t, edit.X-(quote.X+quote.W), 8, "quote/edit gap")
		}
	}
}

func TestUserActionsPreserveOriginalBodies(t *testing.T) {
	body := "```go\ncode\n```"
	quoted, edited := "", ""
	var seq float64
	tt := messageTester(presentationMessage(body), 800, func(m *Model) {
		m.OnQuoteMessage = func(s string) { quoted = s }
		m.OnEditMessage = func(n float64, s string) { seq = n; edited = s }
	})
	for _, label := range []string{i18n.T(i18n.KeyChatCopy), i18n.T(i18n.KeyChatQuote), i18n.T(i18n.KeyChatEdit)} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if tt.Clipboard() != body {
		t.Errorf("copied %q, want original %q", tt.Clipboard(), body)
	}
	if quoted != body || edited != body || seq != 7 {
		t.Errorf("callbacks quote=%q edit=%q seq=%v", quoted, edited, seq)
	}
}

func TestUserAttachmentOnlyHasNoEmptyBubble(t *testing.T) {
	tt := messageTester(presentationMessage("[image] attachment:fixture"), 320, nil)
	if !tt.HasText("attachment:fixture") {
		t.Fatalf("attachment lost: %q", tt.Texts())
	}
	if _, ok := tt.Find("User message bubble"); ok {
		t.Fatal("attachment-only message has empty bubble")
	}
	attachment, _ := tt.Find("[image] attachment:fixture")
	if attachment.X+attachment.W > 304.1 {
		t.Fatalf("attachment exceeds right edge: %v", attachment)
	}
}

func TestAssistantMarkdownUnaffected(t *testing.T) {
	msg := presentationMessage("**bold** and `code`")
	msg.Role = "assistant"
	tt := messageTester(msg, 800, nil)
	if !tt.HasText("bold and code") {
		t.Fatalf("assistant Markdown missing: %q", tt.Texts())
	}
	if tt.HasText("**bold**") {
		t.Fatal("assistant treated as literal user text")
	}
	if _, ok := tt.Find("User message bubble"); ok {
		t.Fatal("assistant acquired user bubble")
	}
}

func TestUserMessageRetainsBoundaryWhitespace(t *testing.T) {
	body := "  hello\n\n  world  "
	tt := messageTester(presentationMessage(body), 800, nil)
	if _, ok := tt.Find(body); !ok {
		t.Fatalf("user whitespace was changed: %q", tt.Texts())
	}
}

func TestUserMessageSharedWidthAxis(t *testing.T) {
	body := strings.Repeat("long content ", 30)
	tt := messageTester(presentationMessage(body), 1200, func(m *Model) { m.ContentWidth = 1000 })
	bubble, ok := tt.Find("User message bubble")
	if !ok {
		t.Fatal("bubble missing")
	}
	rectNear(t, bubble.W, 702, "adaptive bubble cap")
	tt.SetSize(320, 600)
	bubble, _ = tt.Find("User message bubble")
	rectNear(t, bubble.W, 288*.82, "narrow bubble cap")
}

func TestUserBubbleUsesResolvedAndCustomPalette(t *testing.T) {
	for _, colors := range []theme.Colors{theme.Resolve("light").Active(false), theme.Resolve("dark").Active(false), {FG: "#202020", Muted: "#505050", Accent: "#806020", Bubble: "#bfd6ba"}} {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().Padding(12, 16, 8, 16).Children(func() { userMessageView(c, Model{}, presentationMessage("hello"), colors, nil, "hello", nil) })
		}, 800, 120)
		bubble, _ := tt.Find("User message bubble")
		want := chrome.Color(colors.Bubble)
		got := tt.Image().RGBAAt(int(bubble.X+2), int(bubble.Y+bubble.H/2))
		if got.R != want.R || got.G != want.G || got.B != want.B {
			t.Errorf("bubble ignores palette %q: %v", colors.Bubble, got)
		}
		corner := tt.Image().RGBAAt(int(bubble.X+1), int(bubble.Y+1))
		if corner == got {
			t.Fatal("bubble corner did not preserve radius")
		}
	}
}

func TestUserMessageKeepsURLAction(t *testing.T) {
	body := "Read [guide](https://example.com/guide)"
	opened := ""
	tt := messageTester(presentationMessage(body), 800, func(m *Model) { m.OnOpenCitationURL = func(s string) { opened = s } })
	if !tt.HasText(body) {
		t.Fatalf("URL source text changed: %q", tt.Texts())
	}
	if err := tt.Click("↗ guide"); err != nil {
		t.Fatal(err)
	}
	if opened != "https://example.com/guide" {
		t.Fatalf("opened %q", opened)
	}
}

func TestUserLocalAttachmentStaysInsideNarrowStack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.SetRGBA(x, y, color.RGBA{31, 80, 96, 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	opened := ""
	tt := messageTester(presentationMessage("[image] "+path), 320, func(m *Model) { m.OnOpenImageLightbox = func(s string) { opened = s } })
	caption, ok := tt.Find("fixture.png")
	if !ok {
		t.Fatal("local attachment lost")
	}
	if caption.X < 304-288*.82-.1 || caption.X+caption.W > 304.1 {
		t.Errorf("attachment caption outside user cap: %v", caption)
	}
	rendered := tt.Image()
	for y := 0; y < 230; y++ {
		for x := 0; x < 320; x++ {
			if p := rendered.RGBAAt(x, y); p.R == 31 && p.G == 80 && p.B == 96 && (x < 67 || x > 303) {
				t.Fatalf("attachment pixels escaped capped user stack at (%d,%d)", x, y)
			}
		}
	}
	// The square fixture must remain visible, right aligned, and clickable.
	tt.ClickAt(285, caption.Y-80)
	if opened != path {
		t.Errorf("image callback=%q want %q", opened, path)
	}
	if _, ok := tt.Find("User message bubble"); ok {
		t.Fatal("attachment-only message acquired empty bubble")
	}
}

func TestUserFencesKeepLiteralImagePlaceholderLines(t *testing.T) {
	for _, body := range []string{"```text\n[image] attachment:fixture\n```", "~~~text\n[image] attachment:fixture\n~~~", "````text\n```\n[image] attachment:fixture\n````"} {
		tt := messageTester(presentationMessage(body), 800, nil)
		if _, ok := tt.Find(body); !ok {
			t.Errorf("fenced image-like source was changed: %q", tt.Texts())
		}
		if _, ok := tt.Find("[image] attachment:fixture"); ok {
			t.Error("fenced source was rendered as an attachment")
		}
	}
}

func TestUserUnbrokenTextAndLongURLActionStayCapped(t *testing.T) {
	for _, body := range []string{strings.Repeat("unbroken", 50), "Read [" + strings.Repeat("verylonglink", 35) + "](https://example.com)"} {
		opened := ""
		m := Model{ContentWidth: 748, Snapshot: session.ViewState{Messages: []session.Message{{ID: "u", Role: "user", Kind: session.MessageKindText, Body: body}}}, OnOpenCitationURL: func(s string) { opened = s }}
		tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 320, 1600)
		bubble, _ := tt.Find("User message bubble")
		text, _ := tt.Find(body)
		t.Logf("bodyPrefix=%q bubble=%v text=%v", body[:12], bubble, text)
		if bubble.X < 67.8 || bubble.X+bubble.W > 304.1 || text.X < bubble.X || text.X+text.W > bubble.X+bubble.W+.1 {
			t.Errorf("text outside bubble: %v %v", bubble, text)
		}
		if strings.HasPrefix(body, "Read [") {
			r, _ := tt.Find("↗ " + strings.Repeat("verylonglink", 35))
			t.Logf("URL action=%v", r)
			if r.X < 67.8 || r.X+r.W > 304.1 {
				t.Errorf("URL action outside capped stack: %v", r)
			}
			if err := tt.Click("↗ " + strings.Repeat("verylonglink", 35)); err != nil {
				t.Fatal(err)
			}
			if opened != "https://example.com" {
				t.Errorf("long URL target changed to %q", opened)
			}
		}
	}
}
