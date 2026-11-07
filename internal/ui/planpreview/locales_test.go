package planpreview

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestPlanPreviewLocalesParity(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	i18n.SetLocale(i18n.LocaleZH)
	if got := LocaleText("preview.document"); got != "计划 · Markdown" {
		t.Fatalf("zh preview.document = %q", got)
	}
	if got := FormatOpenNamed("架构设计"); got != "打开计划：架构设计" {
		t.Fatalf("zh FormatOpenNamed = %q", got)
	}
	if got := StatusMessage("invalidAddress"); got != "计划地址无效" {
		t.Fatalf("zh StatusMessage(invalidAddress) = %q", got)
	}
	if got := StatusMessage("historyUnavailable"); got != "无法读取会话历史" {
		t.Fatalf("zh StatusMessage(historyUnavailable) = %q", got)
	}
	if got := StatusMessage("exitFailed"); got != "退出计划模式失败" {
		t.Fatalf("zh StatusMessage(exitFailed) = %q", got)
	}

	i18n.SetLocale(i18n.LocaleEN)
	if got := LocaleText("preview.open"); got != "Open plan in sidebar" {
		t.Fatalf("en preview.open = %q", got)
	}
	if got := LocaleText("preview.full"); got != "View full plan" {
		t.Fatalf("en preview.full = %q", got)
	}
	if got := FormatOpenNamed("Architecture"); got != "Open plan: Architecture" {
		t.Fatalf("en FormatOpenNamed = %q", got)
	}
	if got := StatusMessage("loading"); got != "Loading plan…" {
		t.Fatalf("en StatusMessage(loading) = %q", got)
	}
	if got := StatusMessage("failed"); got != "Could not load plan" {
		t.Fatalf("en StatusMessage(failed) = %q", got)
	}
	if got := StatusMessage("unavailable"); got != "Plan preview is unavailable" {
		t.Fatalf("en StatusMessage(unavailable) = %q", got)
	}
}
