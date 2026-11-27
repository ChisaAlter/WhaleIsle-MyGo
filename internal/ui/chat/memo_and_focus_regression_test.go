package chat

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/workflowrun"
)

func TestChatMarkdownMemoVersionExcludesMathAndCodeBlocks(t *testing.T) {
	msg := session.Message{ID: "m1", Role: "assistant"}
	m := Model{ContentWidth: 640}
	var gotOK bool
	var body string
	tt := ui.NewTester(func(c *ui.Context) {
		_, gotOK = chatMarkdownMemoVersion(c, m, msg, body)
	}, 700, 120)

	cases := []struct {
		name string
		body string
		want bool
	}{
		{"plain prose", "Hello world, settled reply.", true},
		{"single currency dollar", "The price is $5 today.", true},
		{"inline math", "Euler identity $e^{i\\pi}+1=0$ holds.", false},
		{"display math", "Formula:\n$$x^2 + y^2 = z^2$$", false},
		{"paren math", `Compute \(a + b\) now.`, false},
		{"bracket math", `Block \[E = mc^2\] here.`, false},
		{"fenced backtick code", "Example:\n```go\nfmt.Println(1)\n```", false},
		{"fenced tilde code", "Example:\n~~~sh\necho hi\n~~~", false},
		{"indented code block", "Intro:\n\n    return 42", false},
	}
	for _, tc := range cases {
		body = tc.body
		tt.Frame()
		if gotOK != tc.want {
			t.Errorf("%s: chatMarkdownMemoVersion ok=%v, want %v", tc.name, gotOK, tc.want)
		}
	}
}

func TestChatMarkdownMemoVersionTracksThemeDesignAndBackdrop(t *testing.T) {
	prevPrefs := theme.CurrentPreferences()
	t.Cleanup(func() {
		theme.SetPreferences(prevPrefs)
		design.SetGlass(false, nil)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
	})
	design.SetGlass(false, nil)
	chrome.ConfigureBackdrop("", wallpapers.Defaults())

	msg := session.Message{ID: "m2", Role: "assistant"}
	m := Model{ContentWidth: 640, ThemeID: "deepseek"}
	var ver uint64
	var ok bool
	tt := ui.NewTester(func(c *ui.Context) {
		ver, ok = chatMarkdownMemoVersion(c, m, msg, "Plain settled response")
	}, 700, 120)
	if !ok {
		t.Fatal("expected settled plain markdown to memoize")
	}
	base := ver

	// Theme preference change (theme.Revision).
	next := prevPrefs
	next.Dark = "codex"
	theme.SetPreferences(next)
	tt.Frame()
	if ver == base {
		t.Fatal("memo version ignored theme.Revision change")
	}
	afterTheme := ver

	// Design glass toggle (design.Revision + tokens.Glass).
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {})
	tt.Frame()
	if ver == afterTheme {
		t.Fatal("memo version ignored design.SetGlass change")
	}
	afterGlass := ver

	// Backdrop activation & glass solidity change.
	wp := wallpapers.Defaults()
	wp.Glass = 60
	chrome.ConfigureBackdrop("wall.png", wp)
	tt.Frame()
	if ver == afterGlass {
		t.Fatal("memo version ignored chrome.BackdropActive change")
	}
	afterBackdrop := ver

	wp.Transparent = true
	chrome.ConfigureBackdrop("wall.png", wp)
	tt.Frame()
	if ver == afterBackdrop {
		t.Fatal("memo version ignored chrome.BackdropPreferences Transparent change")
	}
}

func TestWorkflowMemberFocusRingZeroLayoutShift(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	member := workflowrun.Member{Seq: 1, Label: "Worker Alpha", Status: "running"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			paintWorkflowMember(c, member, member.Label, true, ToolCardStyle{
				FG: c.Theme().Text, Muted: c.Theme().TextMuted, Accent: c.Theme().Accent,
			}, func() {})
		})
	}, 420, 120)

	labelRest, ok := tt.Find("Worker Alpha")
	if !ok {
		t.Fatal("missing workflow member label")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Open Worker Alpha") {
		t.Fatal("Tab did not focus workflow member row")
	}
	labelFocused, _ := tt.Find("Worker Alpha")
	if labelFocused != labelRest {
		t.Fatalf("focus ring shifted workflow member text: rest=%+v focused=%+v", labelRest, labelFocused)
	}
}
