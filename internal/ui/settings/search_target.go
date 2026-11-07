package settings

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

type settingsSearchJump struct {
	Section, Active, BaseActive SectionID
	Target                      string
	Reveal                      []string
	Pending                     bool
	HighlightUntil              time.Time
	Revealed                    map[string]bool
	FilterQuery                 string
	HighlightedKey              string
	HighlightedTitle            string
}

func settingsSearchJumpState(c *ui.Context) *settingsSearchJump {
	return ui.Local(c.Root(), "settings-search-jump", func() settingsSearchJump { return settingsSearchJump{} })
}

func isInlineDisclosureKey(key string) bool {
	switch key {
	case "typography-advanced", "general-font-settings", "about-local-details", "general-local-runtime", "settings-advanced":
		return true
	default:
		return false
	}
}

func settingsFilterQueryMatchesText(query string, fields ...string) bool {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return false
	}
	combined := strings.ToLower(strings.Join(fields, " "))
	for _, term := range terms {
		if !strings.Contains(combined, term) {
			return false
		}
	}
	return true
}

func settingsFilterMatchesDisclosure(query, key string) bool {
	if !isInlineDisclosureKey(key) || strings.TrimSpace(query) == "" {
		return false
	}
	for _, entry := range nestedSearchEntries() {
		if slices.Contains(entry.reveal, key) && settingsFilterQueryMatchesText(query, entry.title, entry.detail, entry.target) {
			return true
		}
	}
	return false
}

func settingsFilterMatchesRow(query, key, title string) bool {
	if strings.TrimSpace(query) == "" {
		return false
	}
	if title != "" && settingsFilterQueryMatchesText(query, title) {
		return true
	}
	for _, entry := range nestedSearchEntries() {
		if (entry.target == key || entry.title == title) && settingsFilterQueryMatchesText(query, entry.title, entry.detail) {
			return true
		}
	}
	return false
}

func settingsSearchReveal(c *ui.Context, key string) bool {
	jump := settingsSearchJumpState(c)
	if (jump.Active == "" || jump.Section == jump.Active) && slices.Contains(jump.Reveal, key) {
		if jump.Revealed == nil {
			jump.Revealed = map[string]bool{}
		}
		if !jump.Revealed[key] {
			jump.Revealed[key] = true
			return true
		}
	}
	if settingsFilterMatchesDisclosure(jump.FilterQuery, key) {
		return true
	}
	return false
}

func settingsSearchHighlighted(c *ui.Context, target string) bool {
	jump := settingsSearchJumpState(c)
	return target != "" && (jump.HighlightedKey == target || jump.HighlightedTitle == target)
}

// Called by the real row, after any owning disclosure/modal has been revealed.
// ScrollIntoView registers against this frame's settled layout, not an old y.
func settingsSearchTarget(c *ui.Context, key, title string, row ui.Element) {
	jump := settingsSearchJumpState(c)
	jumpMatch := (jump.Active == "" || jump.Section == jump.Active) && jump.Target != "" && (jump.Target == key || jump.Target == title)
	filterMatch := settingsFilterMatchesRow(jump.FilterQuery, key, title)
	if !jumpMatch && !filterMatch {
		return
	}
	if jumpMatch && jump.Pending {
		row.ScrollIntoView()
		jump.Pending = false
		jump.HighlightUntil = c.Now().Add(1600 * time.Millisecond)
	}
	if filterMatch || (jumpMatch && c.Now().Before(jump.HighlightUntil)) {
		if jump.HighlightedKey == "" || jumpMatch {
			jump.HighlightedKey = key
			jump.HighlightedTitle = title
		}
		row.Background(c.Theme().Accent.Alpha(.14)).Radius(8)
		row.DrawOver(func(p *ui.Painter, bounds ui.Rect) {
			p.Stroke(bounds, c.Theme().Accent.Alpha(.55), 1, 8)
			p.Fill(ui.Rect{X: bounds.X, Y: bounds.Y + 4, W: 3, H: max(0, bounds.H-8)}, c.Theme().Accent, 1.5)
			if jumpMatch && c.Now().Before(jump.HighlightUntil) {
				p.After(50 * time.Millisecond)
			}
		})
	}
}

func searchRowTarget(section SectionID, zh, title string) string {
	if section == SectionGeneral {
		switch zh {
		case "语言":
			return "general-language"
		case "快捷键":
			return "general-shortcuts-row"
		case "关闭窗口时":
			return "general-close"
		case "启动时":
			return "general-autostart"
		case "Harness 自动恢复":
			return "harness-auto-recovery"
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
		case "背景特效":
			return "background-effect-mode"
		case "指针特效":
			return "cursor-enabled"
		case "玻璃透明度":
			return "appearance-glass"
		case "终端透明度":
			return "terminal-opacity"
		case "透明主题":
			return "transparent-theme"
		case "隐藏侧栏遮罩":
			return "sidebar-mask"
		case "界面字体":
			return "font-interface"
		case "等宽字体":
			return "font-code"
		case "字号":
			return "font-size-interface"
		case "代码字号":
			return "font-size-code"
		case "字号大小":
			return "general-size-text"
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
	addDetail := func(section SectionID, target, zh, en, zhDetail, enDetail string, reveal ...string) {
		entries = append(entries, searchEntry{section: section, target: target, title: providerCopy(zh, en), detail: providerCopy(zhDetail, enDetail), reveal: reveal})
	}
	add("skin-wallpaper", "使用主题自带壁纸", "Use theme wallpaper")
	add("wallpaper-blur", "毛玻璃程度", "Frosted glass")
	add("wallpaper-pixelate", "像素化程度", "Pixelation")
	add("wallpaper-library", "壁纸图库与来源", "Wallpaper library and sources", "wallpaper-library")
	addDetail(SectionAppearance, "general-family-text", "文本字体", "Text font", "正文字体 · 用于界面与会话正文", "Body text font · Used for the interface and conversation text", "general-font-settings")
	addDetail(SectionAppearance, "general-family-code", "代码字体", "Code font", "正文中的代码字体 · 用于代码块、行内代码与工具输出", "Code font in the conversation · Used for code blocks, inline code, and tool output", "general-font-settings")
	add("general-size-code", "正文中的代码字号", "Code size in the conversation", "general-font-settings")
	addDetail(SectionAppearance, "general-family-terminal", "终端字体", "Terminal font", "用于侧栏终端 · 填写本机已安装的等宽字体名", "Used for the sidebar terminal · Type a monospace font name installed on this computer", "general-font-settings", "typography-advanced")
	add("general-size-terminal", "终端字号", "Terminal font size", "general-font-settings")
	addDetail(SectionAppearance, "font-composer", "输入框字体", "Composer font", "高级字体 · 填写本机已安装的字体名。留空则跟随界面字体。", "Advanced typography · Type a font name installed on this computer. Leave empty to follow the interface font.", "typography-advanced")
	addDetail(SectionAppearance, "font-terminal", "高级终端字体", "Advanced terminal font", "高级字体 · 填写本机已安装的等宽字体名。留空则跟随等宽字体。", "Advanced typography · Type a monospace font name installed on this computer. Leave empty to follow the monospace font.", "typography-advanced", "general-font-settings")
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
		add(fmt.Sprintf("cursor-color-%d", i), fmt.Sprintf("Pointer color %d", i+1), fmt.Sprintf("Pointer color %d", i+1), "pointer-effects-settings")
		entries[len(entries)-1].title = providerCopy(fmt.Sprintf("指针颜色 %d", i+1), fmt.Sprintf("Pointer color %d", i+1))
	}
	for _, typing := range []bool{true, false} {
		prefix, title, reveal := "思考炫光 · ", "Thinking glow · ", "composerBeam-settings"
		choices, numbers := beamChoices, beamNumbers
		if typing {
			prefix, title, reveal = "输入特效 · ", "Typing effect · ", "typingFx-settings"
			choices = typingChoices
			numbers = []effectNumber{{"speed", "回显速度", "Echo speed", 40, 240, 10}}
			add("effect-toggle-cursorBlink", prefix+"光标闪烁", title+"Caret blink", reveal)
		} else {
			for _, toggle := range []struct{ path, zh, en string }{{"breathing.enabled", "呼吸亮度", "Brightness breathing"}, {"hueCycle.enabled", "色相循环", "Hue cycling"}, {"night.enabled", "启用夜间时段", "Enable night window"}} {
				add("effect-toggle-"+toggle.path, prefix+toggle.zh, title+toggle.en, reveal)
			}
			numbers = append(append([]effectNumber(nil), numbers...), effectNumber{"night.fromMinute", "开始时间", "Start time", 0, 1439, 1}, effectNumber{"night.toMinute", "结束时间", "End time", 0, 1439, 1})
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
	// Diagnostics and local runtime controls inside collapsed disclosures.
	addDetail(SectionAbout, "about-local-details", "应用数据与诊断", "Application data and diagnostics", "凭据存储、应用数据目录与本地运行诊断", "Credential storage, user data directory, and local runtime diagnostics", "about-local-details")
	entries = append(entries,
		searchEntry{section: SectionAbout, target: "about-detail-id", title: i18n.T(i18n.KeySettingsAboutID), detail: providerCopy("应用数据与诊断", "Application data and diagnostics"), reveal: []string{"about-local-details"}},
		searchEntry{section: SectionAbout, target: "about-detail-user-data", title: i18n.T(i18n.KeySettingsAboutUserData), detail: providerCopy("应用数据与诊断", "Application data and diagnostics"), reveal: []string{"about-local-details"}},
		searchEntry{section: SectionAbout, target: "about-detail-vendor", title: i18n.T(i18n.KeySettingsAboutVendor), detail: providerCopy("应用数据与诊断", "Application data and diagnostics"), reveal: []string{"about-local-details"}},
		searchEntry{section: SectionAbout, target: "general-login", title: i18n.T(i18n.KeySettingsDesktopOpenAtLogin), detail: providerCopy("应用数据与诊断 · 登录系统时自动启动", "Application data and diagnostics · Launch automatically at login"), reveal: []string{"about-local-details", "settings-advanced"}},
		searchEntry{section: SectionAbout, target: "general-local-runtime", title: providerCopy("本地运行与诊断", "Local runtime and diagnostics"), detail: providerCopy("默认工作目录、本地服务端口与内核崩溃自动重启", "Default workspace, local server port, and Harness crash auto-restart"), reveal: []string{"about-local-details", "settings-advanced", "general-local-runtime"}},
		searchEntry{section: SectionAbout, target: "desktop-row-" + i18n.T(i18n.KeySettingsDesktopWorkspace), title: i18n.T(i18n.KeySettingsDesktopWorkspace), detail: providerCopy("本地运行与诊断", "Local runtime and diagnostics"), reveal: []string{"about-local-details", "settings-advanced", "general-local-runtime"}},
		searchEntry{section: SectionAbout, target: "desktop-row-" + i18n.T(i18n.KeySettingsDesktopPort), title: i18n.T(i18n.KeySettingsDesktopPort), detail: providerCopy("本地运行与诊断", "Local runtime and diagnostics"), reveal: []string{"about-local-details", "settings-advanced", "general-local-runtime"}},
		searchEntry{section: SectionAbout, target: "harness-restart-policy", title: providerCopy("内核崩溃自动重启", "Harness crash auto-restart"), detail: providerCopy("本地运行与诊断 · 允许崩溃后自动重启", "Local runtime and diagnostics · Restart automatically after a crash"), reveal: []string{"about-local-details", "settings-advanced", "general-local-runtime"}},
		searchEntry{section: SectionAbout, target: "desktop-row-" + providerCopy("最多重试次数（1–10）", "Maximum retry attempts (1–10)"), title: providerCopy("最多重试次数（1–10）", "Maximum retry attempts (1–10)"), detail: providerCopy("本地运行与诊断 · 内核崩溃自动重启", "Local runtime and diagnostics · Harness crash auto-restart"), reveal: []string{"about-local-details", "settings-advanced", "general-local-runtime"}},
		searchEntry{section: SectionAbout, target: "desktop-row-" + providerCopy("基础延迟（500–30000 毫秒）", "Base delay (500–30000 ms)"), title: providerCopy("基础延迟（500–30000 毫秒）", "Base delay (500–30000 ms)"), detail: providerCopy("本地运行与诊断 · 内核崩溃自动重启", "Local runtime and diagnostics · Harness crash auto-restart"), reveal: []string{"about-local-details", "settings-advanced", "general-local-runtime"}},
	)
	return entries
}
