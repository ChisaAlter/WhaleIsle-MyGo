package composer

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestComposerLocaleCatalogsZHAndEN(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() {
		i18n.SetLocale(prev)
	})

	i18n.SetLocale(i18n.LocaleZH)
	if got := CommandSearchAriaLabel(); got != "筛选选项" {
		t.Fatalf("CommandSearchAriaLabel() zh = %q, want %q", got, "筛选选项")
	}
	if got := CommandLoadingStatusText(); got != "正在加载选项…" {
		t.Fatalf("CommandLoadingStatusText() zh = %q, want %q", got, "正在加载选项…")
	}
	if got := CommandEmptyStatusText(); got != "无选项" {
		t.Fatalf("CommandEmptyStatusText() zh = %q, want %q", got, "无选项")
	}
	if got := FormatCommandListboxAria("/compact"); got != "/compact 匹配项" {
		t.Fatalf("FormatCommandListboxAria() zh = %q, want %q", got, "/compact 匹配项")
	}
	if got := InputTriggerDrillAriaLabel(); got != "进入目录" {
		t.Fatalf("InputTriggerDrillAriaLabel() zh = %q, want %q", got, "进入目录")
	}
	if got := InputTriggerCrumbsAriaLabel(); got != "目录导航" {
		t.Fatalf("InputTriggerCrumbsAriaLabel() zh = %q, want %q", got, "目录导航")
	}
	if got := InputTriggerSuggestionsAriaLabel(); got != "触发候选建议" {
		t.Fatalf("InputTriggerSuggestionsAriaLabel() zh = %q, want %q", got, "触发候选建议")
	}
	if got := FormatModelTriggerAria("DeepSeek V3", "高"); got != "选择模型，当前 DeepSeek V3，推理等级 高" {
		t.Fatalf("FormatModelTriggerAria() zh = %q", got)
	}
	if got := FormatModelCatalogLoadError("timeout"); got != "目录加载失败：timeout" {
		t.Fatalf("FormatModelCatalogLoadError() zh = %q", got)
	}
	if got := ReferenceLocaleText("section.files"); got != "文件与文件夹" {
		t.Fatalf("ReferenceLocaleText(section.files) zh = %q", got)
	}
	if got := FormatReferenceFileOpenFailed("main.go"); got != "无法打开“main.go”，请重试或关闭此提示" {
		t.Fatalf("FormatReferenceFileOpenFailed() zh = %q", got)
	}
	if got := FormatReferenceRelativeDays(3); got != "3天" {
		t.Fatalf("FormatReferenceRelativeDays(3) zh = %q", got)
	}
	if got := FormatReferenceRelativeYears(2); got != "2年" {
		t.Fatalf("FormatReferenceRelativeYears(2) zh = %q", got)
	}
	if got := GoalCommandHintText(false); got != "输入目标，智能体将持续执行" {
		t.Fatalf("GoalCommandHintText(false) zh = %q", got)
	}
	if got := GoalCommandHintText(true); got != "当前目标进行中。可输入 edit 修改 / pause 暂停 / resume 继续 / clear 清除" {
		t.Fatalf("GoalCommandHintText(true) zh = %q", got)
	}
	if got := FormatSessionCostLabel("$0.12", true); got != "当前会话费用：$0.12（部分模型未定价）" {
		t.Fatalf("FormatSessionCostLabel() zh = %q", got)
	}
	if got := FormatAutoReviewNotExecuted(""); got != "工具未执行。原因：Auto review 未授权此次操作" {
		t.Fatalf("FormatAutoReviewNotExecuted() zh = %q", got)
	}
	if got := FormatVoiceDownloadBytes("12", "50", 24); got != "下载：12 / 50 MB，24%" {
		t.Fatalf("FormatVoiceDownloadBytes() zh = %q", got)
	}
	if got := FormatVoiceSystemMicrophoneNamed("USB Mic"); got != "系统默认（USB Mic）" {
		t.Fatalf("FormatVoiceSystemMicrophoneNamed() zh = %q", got)
	}
	errMsg, advice := VoiceDownloadFailureAdvice("network", "VAD", "")
	if errMsg != "无法下载 VAD：连接下载服务失败。" || advice == "" {
		t.Fatalf("VoiceDownloadFailureAdvice(network) zh = (%q, %q)", errMsg, advice)
	}

	i18n.SetLocale(i18n.LocaleEN)
	if got := CommandSearchAriaLabel(); got != "Filter options" {
		t.Fatalf("CommandSearchAriaLabel() en = %q, want %q", got, "Filter options")
	}
	if got := FormatCommandListboxAria("goal"); got != "/goal matches" {
		t.Fatalf("FormatCommandListboxAria() en = %q, want %q", got, "/goal matches")
	}
	if got := InputTriggerSuggestionsAriaLabel(); got != "Trigger suggestions" {
		t.Fatalf("InputTriggerSuggestionsAriaLabel() en = %q, want %q", got, "Trigger suggestions")
	}
	if got := FormatModelTriggerAria("DeepSeek V3", ""); got != "Select model, current DeepSeek V3" {
		t.Fatalf("FormatModelTriggerAria() en = %q", got)
	}
	if got := FormatModelActionError("denied"); got != "Model operation failed: denied" {
		t.Fatalf("FormatModelActionError() en = %q", got)
	}
	if got := ReferenceLocaleText("candidate.noCwd"); got != "(no cwd)" {
		t.Fatalf("ReferenceLocaleText(candidate.noCwd) en = %q", got)
	}
	if got := FormatSessionCostHitRateInline(85); got != " (85%)" {
		t.Fatalf("FormatSessionCostHitRateInline(85) en = %q", got)
	}
	if got := VoiceInputLocaleText("dictate"); got != "Dictate" {
		t.Fatalf("VoiceInputLocaleText(dictate) en = %q", got)
	}
	errMsgEN, adviceEN := VoiceDownloadFailureAdvice("http", "ASR", "503")
	if errMsgEN != "Cannot download ASR: the download service returned HTTP 503." || adviceEN == "" {
		t.Fatalf("VoiceDownloadFailureAdvice(http) en = (%q, %q)", errMsgEN, adviceEN)
	}

	for key, pair := range conversationLocaleCatalog {
		if pair.ZH == "" || pair.EN == "" {
			t.Fatalf("conversationLocaleCatalog[%q] has empty translation: %+v", key, pair)
		}
	}
	for key, pair := range voiceInputLocaleCatalog {
		if pair.ZH == "" || pair.EN == "" {
			t.Fatalf("voiceInputLocaleCatalog[%q] has empty translation: %+v", key, pair)
		}
	}
}
