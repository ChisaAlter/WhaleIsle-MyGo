package settings

import (
	"fmt"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

type settingsSearchJump struct {
	Section, Active SectionID
	Target          string
	Reveal          []string
	Pending         bool
	HighlightUntil  time.Time
}

func settingsSearchJumpState(c *ui.Context) *settingsSearchJump {
	return ui.Local(c.Root(), "settings-search-jump", func() settingsSearchJump { return settingsSearchJump{} })
}

func settingsSearchReveal(c *ui.Context, key string) bool {
	jump := settingsSearchJumpState(c)
	return jump.Pending && jump.Section == jump.Active && slices.Contains(jump.Reveal, key)
}

// Called by the real row, after any owning disclosure/modal has been revealed.
// ScrollIntoView registers against this frame's settled layout, not an old y.
func settingsSearchTarget(c *ui.Context, key, title string, row ui.Element) {
	jump := settingsSearchJumpState(c)
	if jump.Section != jump.Active || jump.Target == "" || jump.Target != key && jump.Target != title {
		return
	}
	if jump.Pending {
		row.ScrollIntoView()
		jump.Pending = false
		jump.HighlightUntil = c.Now().Add(1600 * time.Millisecond)
	}
	if c.Now().Before(jump.HighlightUntil) {
		row.Background(c.Theme().Accent.Alpha(.12)).Radius(8)
		row.DrawOver(func(p *ui.Painter, _ ui.Rect) { p.After(50 * time.Millisecond) })
	}
}

func searchRowTarget(section SectionID, zh, title string) string {
	if section == SectionGeneral {
		switch zh {
		case "快捷键":
			return "general-shortcuts-row"
		case "显示代码工作视图":
			return "ui-settings.enabled"
		case "性能与用量":
			return "ui-chat.performanceUsage"
		case "在使用官方模型 API 时上传 Session Log":
			return "session-log-deepseek.enabled"
		}
	}
	if section == SectionInterface {
		keys := map[string]string{"标题栏 Git 操作": "ui-git.titlebarGit", "终端抽屉开关": "ui-titlebar.terminalToggle", "右侧栏按钮": "ui-titlebar.surfacesToggle", "拖动调整输入框大小": "ui-conversation.composerResize", "会话统计": "ui-conversation.statsLine", "对话与轨迹页签": "ui-conversation.viewTabs"}
		if key := keys[zh]; key != "" {
			return key
		}
	}
	if section == SectionConversation {
		keys := map[string]string{"权限": "permission-row", "繁忙时的发送行为": "ui-conversation.busyEnter", "工作步骤展示": "ui-chat.transcriptView", "工作步骤收起时机": "ui-chat.collapseTiming", "网页链接默认打开方式": "ui-chat.linkOpening", "下一句提示": "ui-conversation.nextPrompt", "视觉回复": "visual-replies.enabled", "自定义指令": "custom-instructions"}
		if key := keys[zh]; key != "" {
			return key
		}
	}
	if section == SectionAppearance {
		switch zh {
		case "颜色模式":
			return "appearance-scheme"
		case "主题":
			return "theme-library"
		case "背景图":
			return "appearance-wallpaper"
		case "界面字体":
			return "font-interface"
		case "等宽字体":
			return "font-code"
		case "字号":
			return "font-size-interface"
		case "代码字号":
			return "font-size-code"
		case "按钮悬停光泽":
			return "metallic-buttons"
		case "输入特效":
			return "effect-row-typingFx"
		case "发送消息时的思考炫光":
			return "effect-row-composerBeam"
		}
	}
	return title
}

func nestedSearchEntries() []searchEntry {
	entries := []searchEntry{
		{section: SectionGeneral, target: "harness-restart-attempts-picker", title: providerCopy("最大重启次数", "Max restarts")},
		{section: SectionGeneral, target: "harness-restart-delay-picker", title: providerCopy("基础重启延迟", "Base delay")},
	}
	add := func(target, zh, en string, reveal ...string) {
		entries = append(entries, searchEntry{section: SectionAppearance, target: target, title: providerCopy(zh, en), reveal: reveal})
	}
	add("skin-wallpaper", "使用主题自带壁纸", "Use theme wallpaper")
	add("wallpaper-blur", "毛玻璃程度", "Frosted glass")
	add("wallpaper-pixelate", "像素化程度", "Pixelation")
	add("wallpaper-library", "壁纸图库与来源", "Wallpaper library and sources", "wallpaper-library")
	for _, field := range []struct{ target, zh, en string }{
		{"general-family-text", "文本字体", "Text font"}, {"general-family-code", "正文中的代码字体", "Code font in the conversation"}, {"general-size-code", "正文中的代码字号", "Code size in the conversation"},
		{"general-family-terminal", "终端字体", "Terminal font"}, {"general-size-terminal", "终端字号", "Terminal font size"},
	} {
		add(field.target, field.zh, field.en, "general-font-settings")
	}
	add("font-composer", "输入框字体", "Composer font", "typography-advanced")
	add("font-terminal", "高级终端字体", "Advanced terminal font", "typography-advanced")
	for _, field := range []struct{ target, zh, en string }{
		{"background-preset", "背景配色方案", "Background color preset"}, {"background-variant", "背景动画样式", "Background animation style"}, {"background-speed", "背景速度", "Background speed"}, {"background-count", "背景光团数量", "Background orb count"},
	} {
		add(field.target, field.zh, field.en, "background-effect-settings")
	}
	for _, field := range []struct{ target, zh, en string }{
		{"cursor-effect", "指针效果类型", "Pointer effect kind"}, {"cursor-preset", "指针配色", "Pointer colors"}, {"cursor-speed", "指针速度", "Pointer speed"}, {"cursor-size", "指针大小", "Pointer size"},
	} {
		add(field.target, field.zh, field.en, "pointer-effects-settings")
	}
	for i := 0; i < 7; i++ {
		add(fmt.Sprintf("background-color-%d", i), fmt.Sprintf("背景颜色 %d", i+1), fmt.Sprintf("Background color %d", i+1), "background-effect-settings")
	}
	for i := 0; i < 6; i++ {
		add(fmt.Sprintf("cursor-color-%d", i), fmt.Sprintf("指针颜色 %d", i+1), fmt.Sprintf("Pointer color %d", i+1), "pointer-effects-settings")
	}
	for _, typing := range []bool{true, false} {
		prefix, title, reveal := "思考炫光 · ", "Thinking glow · ", "composerBeam-settings"
		choices, numbers := beamChoices, beamNumbers
		if typing {
			prefix, title, reveal = "输入特效 · ", "Typing effect · ", "typingFx-settings"
			choices = typingChoices
			numbers = []effectNumber{{"speed", "速度", "Speed", 40, 240, 10}}
			add("effect-toggle-cursorBlink", prefix+"光标闪烁", title+"Blink caret", reveal)
		} else {
			for _, toggle := range []struct{ path, zh, en string }{{"breathing.enabled", "呼吸效果", "Breathing"}, {"hueCycle.enabled", "色相循环", "Hue cycling"}, {"night.enabled", "夜间调暗", "Night dimming"}} {
				add("effect-toggle-"+toggle.path, prefix+toggle.zh, title+toggle.en, reveal)
			}
			numbers = append(append([]effectNumber(nil), numbers...), effectNumber{"night.fromMinute", "夜间开始", "Night starts", 0, 1439, 1}, effectNumber{"night.toMinute", "夜间结束", "Night ends", 0, 1439, 1})
		}
		for _, choice := range choices {
			// A preset-id row exists only while that palette source is active.
			// The always-visible source selector remains searchable in every mode.
			if choice.path == "palette.id" || choice.path == "colors.id" {
				continue
			}
			add("effect-choice-"+choice.path, prefix+choice.zh, title+choice.en, reveal)
		}
		for _, number := range numbers {
			add("effect-number-"+number.path, prefix+number.zh, title+number.en, reveal)
		}
	}
	return entries
}
