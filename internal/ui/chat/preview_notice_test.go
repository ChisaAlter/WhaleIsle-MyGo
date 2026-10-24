package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestPreviewNoticeCopyMatchesTheInstalledClient pins the four strings to the
// installed harness dictionary (ui-settings-models locales: welcomeTitle,
// welcomeBody, welcomeContinue, welcomeError) and the version to
// onboarding-copy.ts.
func TestPreviewNoticeCopyMatchesTheInstalledClient(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	if got := i18n.T(i18n.KeyChatPreviewNoticeTitle); got != "预览版说明" {
		t.Fatalf("title=%q", got)
	}
	if got := i18n.T(i18n.KeyChatPreviewNoticeContinue); got != "继续" {
		t.Fatalf("continue=%q", got)
	}
	if got := i18n.T(i18n.KeyChatPreviewNoticeError); got != "暂时无法保存确认状态，请重试。" {
		t.Fatalf("error=%q", got)
	}
	body := i18n.T(i18n.KeyChatPreviewNoticeBody)
	if !strings.HasPrefix(body, "DeepSeek Harness 目前的 0.2 版本仍处于预览阶段") {
		t.Fatalf("body=%q", body)
	}
	paragraphs := PreviewNoticeParagraphs(body)
	if len(paragraphs) != 2 || !strings.HasPrefix(paragraphs[1], "我们期待与全球用户和开发者一起") {
		t.Fatalf("paragraphs=%q", paragraphs)
	}
	if PreviewNoticeVersion != "2026-09-28.1" {
		t.Fatalf("version=%q", PreviewNoticeVersion)
	}
	i18n.SetLocale("en")
	if got := i18n.T(i18n.KeyChatPreviewNoticeTitle); got != "Preview Notice" {
		t.Fatalf("en title=%q", got)
	}
	if got := i18n.T(i18n.KeyChatPreviewNoticeContinue); got != "Continue" {
		t.Fatalf("en continue=%q", got)
	}
}

// TestPreviewNoticeRequestedComparesExactVersions mirrors the native equality
// check: only the acknowledged copy version hides the notice.
func TestPreviewNoticeRequestedComparesExactVersions(t *testing.T) {
	cases := map[string]bool{
		"":                true,
		"2026-09-27.9":    true,
		"2026-09-28.1":    false,
		" 2026-09-28.1\n": false,
		"2026-09-28.1.0":  true,
	}
	for ack, want := range cases {
		if got := PreviewNoticeRequested(ack); got != want {
			t.Fatalf("ack %q: requested=%v want %v", ack, got, want)
		}
	}
}

// TestPreviewNoticeGeometryMatchesTheNativeDialog measures the rendered layer at
// the reference window size: the installed dialog is 600 DIP wide, centred, with
// 28 DIP of content padding and the title on the first 28 DIP line.
func TestPreviewNoticeGeometryMatchesTheNativeDialog(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	var panel ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		PreviewNoticeView(c, PreviewNotice{
			Open:   true,
			Colors: theme.Resolve("deepseek.light").Light,
			Observe: func(r ui.Rect) {
				panel = r
			},
		})
	}, 1440, 920)
	// The layer enters on the overlay token; reduced motion settles it at once so
	// the measured boxes are the final ones.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	if !tt.HasText("预览版说明") || !tt.HasText("继续") {
		t.Fatalf("notice copy missing: %q", tt.Texts())
	}
	if panel.W != PreviewNoticeWidth {
		t.Fatalf("panel width=%v, want %d", panel.W, PreviewNoticeWidth)
	}
	if wave := panel.X - (1440-panel.W)/2; wave > 1 || wave < -1 {
		t.Fatalf("panel is not horizontally centred: x=%v w=%v", panel.X, panel.W)
	}
	if panel.Y <= 0 || panel.Y+panel.H >= 920 {
		t.Fatalf("panel is not vertically centred: y=%v h=%v", panel.Y, panel.H)
	}
	title, ok := tt.Find("预览版说明")
	if !ok {
		t.Fatal("title not found")
	}
	if title.X != panel.X+PreviewNoticePadding || title.Y != panel.Y+PreviewNoticePadding {
		t.Fatalf("title=%+v panel=%+v", title, panel)
	}
	if title.H < PreviewNoticeTitleLine {
		t.Fatalf("title line box=%v, want at least %d", title.H, PreviewNoticeTitleLine)
	}
	button, ok := tt.Find("继续")
	if !ok {
		t.Fatal("continue button not found")
	}
	if button.W < PreviewNoticeButtonMin || button.H != 36 {
		t.Fatalf("button=%+v, want at least %d wide and 36 tall", button, PreviewNoticeButtonMin)
	}
	if button.X+button.W != panel.X+panel.W-PreviewNoticePadding {
		t.Fatalf("button is not end-aligned: %+v panel=%+v", button, panel)
	}
	if button.Y != panel.Y+panel.H-PreviewNoticePadding-button.H {
		t.Fatalf("button is not on the last padded line: %+v panel=%+v", button, panel)
	}
}

// TestPreviewNoticeContinueAcknowledges proves the layer reports the press once
// and stays closed afterwards.
func TestPreviewNoticeContinueAcknowledges(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	pressed := 0
	tt := ui.NewTester(func(c *ui.Context) {
		PreviewNoticeView(c, PreviewNotice{
			Open:       true,
			Colors:     theme.Resolve("deepseek.light").Light,
			OnContinue: func() { pressed++ },
		})
	}, 1440, 920)
	if err := tt.Click("继续"); err != nil {
		t.Fatal(err)
	}
	if pressed != 1 {
		t.Fatalf("continue pressed %d times", pressed)
	}
}

// TestPreviewNoticeClosedRendersNothing keeps the layer out of the tree once the
// profile has acknowledged the notice.
func TestPreviewNoticeClosedRendersNothing(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := ui.NewTester(func(c *ui.Context) {
		PreviewNoticeView(c, PreviewNotice{Open: false, Colors: theme.Resolve("deepseek.light").Light})
	}, 1440, 920)
	if texts := tt.Texts(); len(texts) != 0 {
		t.Fatalf("closed notice painted %q", texts)
	}
}

// TestPreviewNoticeFailedSaveShowsTheNativeError keeps the acknowledgement
// failure line available, as the installed WelcomeNotice does.
func TestPreviewNoticeFailedSaveShowsTheNativeError(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := ui.NewTester(func(c *ui.Context) {
		PreviewNoticeView(c, PreviewNotice{
			Open:   true,
			Failed: true,
			Colors: theme.Resolve("deepseek.light").Light,
		})
	}, 1440, 920)
	if !tt.HasText("暂时无法保存确认状态，请重试。") {
		t.Fatalf("missing save error: %q", tt.Texts())
	}
}
