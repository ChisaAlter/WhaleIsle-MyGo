package settings

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// searchEntry is one row of the settings search index: the page it lives on,
// its title and the line shown under it. A row with a current value (字号 →
// 16px) shows that value where other rows show their description.
type searchEntry struct {
	section SectionID
	title   string
	detail  string
	value   func(Model) string
	target  string
	reveal  []string
	rank    int
	open    func()
}

func searchEntries(m Model) []searchEntry {
	row := func(section SectionID, zh, en, zhDetail, enDetail string) searchEntry {
		title := providerCopy(zh, en)
		return searchEntry{section: section, title: title, detail: providerCopy(zhDetail, enDetail), target: searchRowTarget(section, zh, title)}
	}
	value := func(section SectionID, zh, en string, current func(Model) string) searchEntry {
		title := providerCopy(zh, en)
		return searchEntry{section: section, title: title, value: current, target: searchRowTarget(section, zh, title)}
	}
	fontSize := func(role string) func(Model) string {
		return func(m Model) string {
			prefs := theme.CurrentPreferences()
			if m.Appearance != nil && m.Appearance.Preferences != nil {
				prefs = m.Appearance.Preferences.Saved
			} else if m.Desktop != nil && m.Desktop.Preferences != nil {
				prefs = m.Desktop.Preferences.Saved
			}
			size := prefs.Interface.Size
			if role == "code" {
				size = prefs.Code.Size
			}
			return fmt.Sprintf("%.0fpx", size)
		}
	}
	locale := func(m Model) string {
		if m.Desktop != nil && m.Desktop.Fields.Locale != nil && *m.Desktop.Fields.Locale == "en" || i18n.Locale() == "en" {
			return "English"
		}
		return "中文"
	}
	entries := []searchEntry{
		row(SectionUsageStats, "会话累计费用", "Session cost", "在输入框下方显示峰谷状态条与当前会话累计费用。", "Show the peak/valley strip and this session's accumulated cost below the composer."),
		row(SectionUsageStats, "官方峰谷时", "Official peak and valley hours", "会话累计费用开启时，在会话统计下方显示官方峰谷时段状态条。", "With session cost on, show the official peak/valley strip below the statistics."),
		value(SectionGeneral, "语言", "Language", locale),
		row(SectionGeneral, "显示代码工作视图", "Show coding view", "开启后显示轨迹、本轮代码差异，新对话中的 Agent 预设切换", "Show trajectory, this turn's code diff, and the Agent preset switch for new conversations."),
		row(SectionGeneral, "快捷键", "Shortcuts", "查看和编辑当前可用的快捷键和输入操作", "View and edit the available shortcuts and input actions"),
		row(SectionGeneral, "关闭窗口时", "When closing the window", "点标题栏关闭时：最小化到托盘，或退出并停止本机 Harness 服务。", "The title-bar close button can hide the app in the tray, or quit and stop the local Harness service."),
		row(SectionGeneral, "启动时", "On launch", "直接打开桌面端，或先打开启动器再手动启动桌面端。", "Open the desktop directly, or open the launcher first and start the desktop manually."),
		row(SectionGeneral, "性能与用量", "Performance and usage", "选择性能与用量信息展示的详细程度", "Choose how detailed the performance and usage information is"),
		row(SectionGeneral, "在使用官方模型 API 时上传 Session Log", "Upload Session Log when using the official model API", "帮助改进 DeepSeek 模型与产品", "Help improve DeepSeek models and products"),
		row(SectionGeneral, "Harness 自动恢复", "Harness auto-recovery", "Harness 进程意外退出后自动重启，最多重试有限次数。", "Restart the Harness automatically after an unexpected exit, with a bounded number of retries."),
		row(SectionAppearance, "颜色模式", "Color scheme", "选择浅色、深色或跟随系统。", "Choose light, dark, or follow the system."),
		row(SectionAppearance, "主题", "Theme", "浅色与深色各自选择主题。", "Pick a theme for the light and the dark half."),
		row(SectionAppearance, "背景图", "Wallpaper", "选一张图铺在整个界面后面。图要透过界面才能看见：玻璃透明度越高，图越被盖住。设好之后可以拉毛玻璃和像素化。", "Choose an image to sit behind the whole UI. It shows through the chrome: higher glass opacity hides more of it. Frost and pixelate sliders appear after it is set."),
		row(SectionAppearance, "背景特效", "Background effect", "不设置背景图时，在整个界面后面铺一层缓慢流动的渐变光斑。", "Without a background image, draw slowly moving gradient glows behind the interface."),
		row(SectionAppearance, "指针特效", "Pointer effect", "在指针划过处绘制装饰特效，可选像素拖尾或流体飞溅。", "Draw a decorative effect where the pointer passes: pixel trail or fluid splash."),
		row(SectionAppearance, "按钮悬停光泽", "Button hover shine", "鼠标悬停在按钮上时叠加一层金属扫光。", "Overlay a metallic sweep while the pointer hovers a button."),
		row(SectionAppearance, "玻璃透明度", "Glass opacity", "控制界面表层的实心程度：菜单、对话框、输入框。", "How solid the interface surfaces are: menus, dialogs and inputs."),
		row(SectionAppearance, "终端透明度", "Terminal opacity", "终端面板自己的实心程度，不跟随上面的玻璃滑杆。", "The terminal panel's own solidity; it does not follow the glass slider."),
		row(SectionAppearance, "透明主题", "Transparent theme", "开启后界面表层完全透明，壁纸色彩全量透出。", "Make the interface surfaces fully transparent so the wallpaper shows through."),
		row(SectionAppearance, "隐藏侧栏遮罩", "Hide sidebar mask", "开启后左侧栏不再叠加遮罩，与工作区底色保持一致。", "Stop overlaying a mask on the left sidebar so it matches the workspace."),
		row(SectionAppearance, "界面字体", "Interface font", "填写本机已安装的字体名。留空则用系统默认。", "Name an installed font. Leave blank for the system default."),
		value(SectionAppearance, "字号", "Size", fontSize("interface")),
		row(SectionAppearance, "等宽字体", "Monospace font", "填写本机已安装的等宽字体名。留空则用系统默认。", "Name an installed monospace font. Leave blank for the system default."),
		value(SectionAppearance, "代码字号", "Code size", fontSize("code")),
		row(SectionAppearance, "字号大小", "Font size", "仅影响会话内容的字号", "Only affects the size of conversation content"),
		row(SectionAppearance, "输入特效", "Typing effect", "键入字符时在输入框播放短暂回显，可替换光标样式。", "Play a short echo in the composer while typing; can replace the cursor style."),
		row(SectionAppearance, "发送消息时的思考炫光", "Thinking glow when sending", "智能体发送或思考时，在输入条周围播放彩色边光。", "Play a colored edge glow around the composer while the agent sends or thinks."),
		row(SectionInterface, "标题栏 Git 操作", "Titlebar Git actions", "显示初始化、切换分支、提交及菜单。", "Show init, branch switching, commit and menus."),
		row(SectionInterface, "终端抽屉开关", "Terminal drawer toggle", "在标题栏显示终端抽屉按钮。", "Show the terminal drawer button in the titlebar."),
		row(SectionInterface, "右侧栏按钮", "Right-panel buttons", "在标题栏显示右侧栏按钮。", "Show the right-panel button in the titlebar."),
		row(SectionInterface, "拖动调整输入框大小", "Drag to resize the composer", "允许拖动输入区上、左、右边框改变高度和宽度。", "Drag the composer's top, left and right edges to resize it."),
		row(SectionInterface, "会话统计", "Session statistics", "在输入框上方显示轮次、耗时、缓存与 token。", "Show turns, elapsed time, cache and tokens above the composer."),
		row(SectionInterface, "对话与轨迹页签", "Chat and trajectory tabs", "在会话顶栏显示「对话」「轨迹」切换。", "Show the Chat / Trajectory switch in the conversation header."),
		row(SectionInterface, "显示底部归档管理区", "Show the archive manager", "选择“隐藏已归档”时，在侧栏底部显示可折叠的“已归档”管理区。", "When archived sessions are hidden, show a collapsible Archived area at the bottom of the sidebar."),
		row(SectionConversation, "权限", "Permissions", "选择新会话的默认权限模式", "Choose the default permission mode for new sessions"),
		row(SectionConversation, "繁忙时的发送行为", "Send behavior while busy", "智能体运行时 Enter 键和发送按钮的行为；Cmd/Ctrl+Enter 使用另一行为", "What Enter and Send do while the agent runs; Cmd/Ctrl+Enter uses the other action"),
		row(SectionConversation, "工作步骤展示", "Work step display", "选择希望看到多少工具调用细节", "Choose how much tool-call detail to show"),
		row(SectionConversation, "工作步骤收起时机", "Collapse completed work steps", "选择何时自动收起工作步骤", "Choose when completed work steps collapse"),
		row(SectionConversation, "网页链接默认打开方式", "Default link opening", "对话中网页链接的打开位置", "Where links in the conversation open"),
		row(SectionConversation, "下一句提示", "Next prompt suggestions", "回复结束后建议下一句，可按 Tab 采纳，不会自动发送。", "Suggest the next message after a reply; Tab accepts it, nothing is sent automatically."),
		row(SectionConversation, "视觉回复", "Visual replies", "允许智能体生成交互式页面并在会话内展示。截图检查需要支持图片的模型，或已配置的识图模型；关闭后已生成页面仍可查看。", "Let the agent generate interactive pages and show them in the conversation."),
		row(SectionConversation, "自定义指令", "Custom instructions", "给智能体定几条规则，后续对所有任务都生效", "Give the agent a few rules that apply to every later task"),
		row(SectionModels, "识图模型", "Vision model", "主模型不支持图片时，先由该模型识别图片内容。", "When the main model cannot read images, this model describes them first."),
		row(SectionFeatures, "鲸鱼娘助理", "Whale assistant", "开启她的常驻对话与侧栏入口；切换后会重启 Harness。", "Turn on her resident conversation and sidebar entry; switching restarts the Harness."),
		row(SectionFeatures, "Project", "Project", "让一个主对话带领智能体团队，按需求清单持续完成一个目录里的开发。", "Let one lead conversation guide an agent team through a running list of requirements in a directory."),
		row(SectionFeatures, "远程工作区（SSH）", "Remote workspace (SSH)", "通过 SSH 把远程机器挂为工作区。", "Mount a remote machine as a workspace over SSH."),
		row(SectionRemote, "中继主机", "Relay host", "默认 ayase.cn:443（TLS），一般不用改。", "Defaults to ayase.cn:443 (TLS); rarely needs changing."),
		row(SectionRemote, "连接方式", "Connection mode", "局域网：扫码打开本机网页。服务器：扫码打开公网网页；数据走中继。", "LAN: scan to open the local page. Server: scan to open the public page; data goes through the relay."),
		row(SectionRemote, "监听端口", "Listen port", "本机 daemon 默认 6767；局域网配对页另用 3180。", "The local daemon defaults to 6767; the LAN pairing page uses 3180."),
		row(SectionRemote, "监听范围", "Listen scope", "仅本机监听：局域网设备无法直连。", "Local only: LAN devices cannot connect directly."),
		row(SectionRemote, "传输加密", "Transport encryption", "手机会话使用端到端加密。", "Phone sessions use end-to-end encryption."),
		row(SectionRemote, "刷新配对码", "Refresh pairing code", "只换二维码；已配对设备仍有效，除非解除配对。", "Only replaces the QR code; paired devices stay valid unless unpaired."),
	}
	entries = append(entries, nestedSearchEntries()...)
	if m.Account != nil && m.Account.Status != "credential-stored" && m.OnSignIn != nil {
		entries = append(entries, searchEntry{section: SectionGeneral, title: providerCopy("账户登录", "Sign in"), detail: providerCopy("登录账户", "Sign in to your account"), target: "general-account", open: m.OnSignIn})
	}
	var snap store.ViewState
	if m.Store != nil {
		snap = m.Store.Snapshot()
	}
	for _, page := range primaryNavigation(m, snap) {
		entries = append(entries, searchEntry{section: page.id, title: navigationLabel(m, page.id), detail: providerCopy("设置页面", "Settings page"), open: page.onSelect})
	}
	// Index the live description, not a fixed list of plugin names. Values and
	// secret-like fields never enter search text. Registered dedicated rows keep
	// their richer entry; all other schema fields retain a real form destination.
	for _, ns := range snap.Namespaces {
		for _, field := range ns.Fields {
			if SecretLikeKey(field.Key) {
				continue
			}
			duplicate := false
			for _, entry := range entries {
				if entry.target == ns.NS+"."+field.Key {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}
			name, key := ns.NS, field.Key
			title := field.Label
			if title == "" {
				title = key
			}
			open := func() {
				if m.OnSelectNamespace != nil {
					m.OnSelectNamespace(name)
				} else if m.Store != nil {
					m.Store.Select(name)
				}
			}
			section := SectionForNamespace(name)
			pageTitle := navigationLabel(m, section)
			section = SectionGeneral // the namespace browser is the generic General form destination
			entries = append(entries, searchEntry{section: section, title: title, detail: field.Description + " · " + pageTitle + " · " + name, target: "field-" + key, open: open})
		}
	}
	for _, contribution := range m.ContributedSections {
		for _, field := range contribution.SearchEntries {
			entries = append(entries, searchEntry{section: contribution.ID, title: field.Title, detail: field.Description, target: field.Target, reveal: field.Reveal, open: contribution.OnSelect})
		}
	}
	if m.PublicContributions.Entries != nil {
		for _, page := range primaryNavigation(m, snap) {
			slot := "settings." + string(page.id) + ".item"
			if page.id == SectionUsageStats {
				slot = "settings.usage.item"
			}
			for _, contribution := range m.PublicContributions.Entries(slot, "") {
				if contribution.Native {
					continue
				}
				title, detail := settingsContributionSearchText(contribution.Nodes)
				if label := strings.TrimSpace(contribution.Label); label != "" {
					if title != "" {
						detail = strings.TrimSpace(title + " · " + detail)
					}
					title = label
				}
				if title != "" {
					entries = append(entries, searchEntry{section: page.id, title: title, detail: detail, target: "public-settings-item-" + contribution.Handle, open: page.onSelect})
				}
			}
		}
	}
	filtered := entries[:0]
	for _, entry := range entries {
		slot := "settings." + string(entry.section) + ".item"
		if entry.section == SectionUsageStats {
			slot = "settings.usage.item"
		}
		id := settingsSearchEntryBuiltinID(entry, slot)
		if id != "" && !m.PublicContributions.NativeWinner(slot, "", id) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

// settingsSearchMatches matches every word across the title, description and
// page name, ranking title matches first and keeping the rail's stable order.
func settingsSearchMatches(m Model, available map[SectionID]bool, query string) []searchEntry {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil
	}
	order := map[SectionID]int{}
	for index, id := range planSectionOrder {
		order[id] = index
	}
	var matches []searchEntry
	for _, entry := range searchEntries(m) {
		if !available[entry.section] && entry.open == nil {
			continue
		}
		detail := entry.detail
		if entry.value != nil {
			detail = entry.value(m)
		}
		title, description, page := strings.ToLower(entry.title), strings.ToLower(detail), strings.ToLower(navigationLabel(m, entry.section))
		matched, allTitle := true, true
		for _, term := range terms {
			inTitle := strings.Contains(title, term)
			allTitle = allTitle && inTitle
			if !inTitle && !strings.Contains(description, term) && !strings.Contains(page, term) {
				matched = false
				break
			}
		}
		if matched {
			entry.rank = 2
			if allTitle {
				entry.rank = 1
				if strings.HasPrefix(title, terms[0]) {
					entry.rank = 0
				}
			}
			matches = append(matches, entry)
		}
	}
	// Stable sort by rail order: rows keep their page order inside one section.
	for i := 1; i < len(matches); i++ {
		for j := i; j > 0 && order[matches[j].section] < order[matches[j-1].section]; j-- {
			matches[j], matches[j-1] = matches[j-1], matches[j]
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].rank < matches[j].rank })
	return matches
}

// settingsSearchResults is the options pane while the search box holds text:
// one button per matching row (title, then "page · line"), or the empty state.
func settingsSearchResults(c *ui.Context, m Model, snap store.ViewState, query string, fg, muted, hover ui.Color, clear func()) {
	available := map[SectionID]func(){}
	ids := map[SectionID]bool{}
	for _, entry := range primaryNavigation(m, snap) {
		available[entry.id] = entry.onSelect
		ids[entry.id] = true
	}
	matches := settingsSearchMatches(m, ids, query)
	if len(matches) == 0 {
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeySettingsNavSearchEmpty), strings.TrimSpace(query))).FillWidth().Padding(24, 0, 0, 0).
			FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(muted)
		return
	}
	ui.Column(c.Key("settings-search-results")).FillWidth().MinWidth(0).Padding(10, 0, 0, 0).Gap(4).Children(func() {
		for index, entry := range matches {
			detail := entry.detail
			if entry.value != nil {
				detail = entry.value(m)
			}
			subtitle := navigationLabel(m, entry.section)
			if detail != "" {
				subtitle += " · " + detail
			}
			button := ui.Button(c.Key(fmt.Sprintf("settings-search-%d-%s", index, entry.title)), "").Label(entry.title).
				FillWidth().MinWidth(0).Height(60).Shrink(0).Padding(10, 12).Radius(12).BorderWidth(0).Background(ui.Transparent).
				AlignItems(ui.Start).FocusRing(false)
			button.Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(0).Children(func() {
					ui.Text(c, entry.title).FillWidth().FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).FontWeight(500).TextColor(fg).SingleLine()
					ui.Text(c, subtitle).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted).SingleLine()
				})
			})
			if button.Hovered() {
				button.Background(hover)
			}
			if button.Clicked() {
				if clear != nil {
					clear()
				}
				open := entry.open
				if open == nil {
					open = available[entry.section]
				}
				if open != nil {
					jump := settingsSearchJumpState(c)
					*jump = settingsSearchJump{Section: entry.section, Target: entry.target, Reveal: entry.reveal, Pending: true}
					open()
				}
			}
		}
	})
}
