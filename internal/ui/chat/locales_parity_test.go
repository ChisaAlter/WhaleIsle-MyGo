package chat

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestChatAllModulesLocalesParity(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	i18n.SetLocale(i18n.LocaleZH)
	if got := ChatLocale("openui.loading"); got != "正在载入交互式回复…" {
		t.Fatalf("zh openui.loading = %q", got)
	}
	if got := ChatLocale("message.failure.interrupted"); got != "DSH 进程在本轮完成前退出（例如被强制结束、崩溃或关闭超时），重新启动后本轮已中止。未完成的工具调用结果未知。" {
		t.Fatalf("zh message.failure.interrupted = %q", got)
	}
	if got := FormatStepProcessJoinTwo("准备读取文件", "准备编辑文件"); got != "准备读取文件并准备编辑文件" {
		t.Fatalf("zh FormatStepProcessJoinTwo = %q", got)
	}
	if got := FormatStepProcessMore("正在读取文件"); got != "正在读取文件等" {
		t.Fatalf("zh FormatStepProcessMore = %q", got)
	}
	if got := ChatPlanLocale("preview.document"); got != "计划 · Markdown" {
		t.Fatalf("zh ChatPlanLocale(preview.document) = %q", got)
	}
	if got := SkillRowStateTitle("preparing"); got != "准备加载技能" {
		t.Fatalf("zh SkillRowStateTitle(preparing) = %q", got)
	}
	if got := GoalLocale("commandInput.aria"); got != "指令输入" {
		t.Fatalf("zh GoalLocale(commandInput.aria) = %q", got)
	}
	if got := CordisLocale("row.defineTitle"); got != "注册 Cordis 插件" {
		t.Fatalf("zh CordisLocale(row.defineTitle) = %q", got)
	}
	if got := TrajectoryLocale("toolbar.aria"); got != "轨迹工具栏" {
		t.Fatalf("zh TrajectoryLocale(toolbar.aria) = %q", got)
	}
	if got := FormatTrajectoryTurnLabel(3); got != "第 3 轮" {
		t.Fatalf("zh FormatTrajectoryTurnLabel = %q", got)
	}
	if got := VisualReplyLoadError("too_large"); got != "页面超过 25 MiB 的读取上限，无法加载。" {
		t.Fatalf("zh VisualReplyLoadError(too_large) = %q", got)
	}
	if got := FormatVisualReplyFrameTitle("Chart"); got != "视觉回复：Chart" {
		t.Fatalf("zh FormatVisualReplyFrameTitle = %q", got)
	}
	if got := ChatAgentTeamLocale("loading"); got != "正在加载团队…" {
		t.Fatalf("zh ChatAgentTeamLocale(loading) = %q", got)
	}
	if got := CotTranslationLocale("translating"); got != "正在翻译思考内容" {
		t.Fatalf("zh CotTranslationLocale(translating) = %q", got)
	}

	i18n.SetLocale(i18n.LocaleEN)
	if got := ChatLocale("openui.desktopOnly"); got != "Open this interactive reply in Whale Isle MyGo desktop." {
		t.Fatalf("en openui.desktopOnly = %q", got)
	}
	if got := FormatStepProcessJoinTwo("Preparing to read files", "Preparing to edit files"); got != "Preparing to read files and Preparing to edit files" {
		t.Fatalf("en FormatStepProcessJoinTwo = %q", got)
	}
	if got := FormatStepProcessMore("Reading files"); got != "Reading files, etc." {
		t.Fatalf("en FormatStepProcessMore = %q", got)
	}
	if got := SkillRowStateTitle("failed"); got != "Skill load failed" {
		t.Fatalf("en SkillRowStateTitle(failed) = %q", got)
	}
	if got := GoalLocale("commandInput.aria"); got != "Command input" {
		t.Fatalf("en GoalLocale(commandInput.aria) = %q", got)
	}
	if got := CordisLocale("panel.hint"); got != "Run controls live in the Cordis panel above Settings" {
		t.Fatalf("en CordisLocale(panel.hint) = %q", got)
	}
	if got := TrajectoryLocale("details.resizeTitle"); got != "Drag to resize. Double-click to reset." {
		t.Fatalf("en TrajectoryLocale(details.resizeTitle) = %q", got)
	}
	if got := FormatTrajectoryStepLabel(5); got != "Step 5" {
		t.Fatalf("en FormatTrajectoryStepLabel = %q", got)
	}
	if got := VisualReplyLoadError("not_referenced"); got != "This page is not referenced by the current conversation and cannot be read." {
		t.Fatalf("en VisualReplyLoadError(not_referenced) = %q", got)
	}
	if got := ChatAgentTeamLocale("loading"); got != "Loading Team…" {
		t.Fatalf("en ChatAgentTeamLocale(loading) = %q", got)
	}
	if got := CotTranslationLocale("deepseekOfficial"); got != "DeepSeek Flash — Official API (paid)" {
		t.Fatalf("en CotTranslationLocale(deepseekOfficial) = %q", got)
	}
}
