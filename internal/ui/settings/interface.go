package settings

import (
	"encoding/json"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"

	"whaleisle/internal/shortcuts"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// InterfaceAccess is the dedicated Interface settings page (locale + chrome prefs).
type InterfaceAccess struct {
	Project                               *ProjectPreferenceAccess
	Host                                  *BodySettingsAccess
	DshbotEnabled, RemoteWorkspaceEnabled *bool
	OnSetDshbot, OnSetRemoteWorkspace     func(bool)
	Shortcuts                             *shortcuts.Editor
	Locale                                *string
	CloseToTray                           *bool
	OpenAtLogin                           *bool
	// AutoStartDesktop starts harness after boot when workspace is set (launcher toggle).
	AutoStartDesktop *bool
	// ShowViewTabs is only the local fallback when the Host supplies no field.
	ShowViewTabs     *bool
	ShowArchivedList *bool
	// AutoAllowTools mirrors Electron tool auto-allow (interaction store).
	AutoAllowTools        *bool
	OnSetShowArchivedList func(bool) (bool, error)
	OnSetViewTabs         func(bool) (bool, error)
	OnSetAutoAllowTools   func(bool) (bool, error)
	OnSave                func()
	LastAction            string
	Err                   string
}

func interfaceViewTabsField(a *BodySettingsAccess) (string, store.Field, bool) {
	if a == nil || !a.Snapshot.ClientReady || !a.Snapshot.Described {
		return "", store.Field{}, false
	}
	for _, namespace := range []string{"ui-conversation", "conversation", "submissionPolicy", "ui"} {
		for _, ns := range a.Snapshot.Namespaces {
			if ns.NS != namespace {
				continue
			}
			for _, field := range ns.Fields {
				if field.Key == "viewTabs" {
					return namespace, field, true
				}
			}
		}
	}
	return "", store.Field{}, false
}

// InterfaceViewTabsAccepted reads only a typed current Host field. A malformed
// accepted value never turns a compatibility draft into a Host writer.
func InterfaceViewTabsAccepted(a *BodySettingsAccess) (bool, bool) {
	namespace, field, found := interfaceViewTabsField(a)
	if !found || field.Kind != store.FieldBool {
		return false, false
	}
	for _, ns := range a.Snapshot.Namespaces {
		if ns.NS != namespace {
			continue
		}
		var values map[string]json.RawMessage
		if json.Unmarshal(ns.Value, &values) != nil {
			return false, false
		}
		var value *bool
		if json.Unmarshal(values["viewTabs"], &value) != nil || value == nil {
			return false, false
		}
		return *value, true
	}
	return false, false
}

// InterfaceViewTabsFallbackAllowed reserves any described Host field, including
// read-only or unsupported types. A pending description never grants local authority.
func InterfaceViewTabsFallbackAllowed(a *BodySettingsAccess) bool {
	if _, _, found := interfaceViewTabsField(a); found {
		return false
	}
	return a == nil || !a.Snapshot.ClientReady || a.Snapshot.Described && !a.Snapshot.Loading && a.Snapshot.Err == ""
}

func interfacePane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Interface == nil {
		return
	}
	a := m.Interface
	bodyHostState(c, a.Host, muted)
	viewTabsNS, viewTabsField, hostViewTabs := interfaceViewTabsField(a.Host)
	localViewTabs := InterfaceViewTabsFallbackAllowed(a.Host)
	// 0.3.13 Interface: titlebar, composer and sidebar chrome only. Project, Bots
	// and the remote workspace moved to the built-in features page; session cost and
	// peak hours to the usage page; next-prompt to the conversation page; the
	// thinking glow to the appearance page.
	for _, field := range []struct{ ns, key, zh, en, zhDesc, enDesc string }{
		{"session-log-download", "titlebarAction", "会话日志导出", "Session log export",
			"在标题栏显示额外的会话日志按钮。关闭后仍可从会话的更多操作菜单或 /export 下载。",
			"Show an extra Session log button in the titlebar. Turning this off still leaves the Session more-actions menu and /export available."},
		{"ui-git", "titlebarGit", "标题栏 Git 操作", "Titlebar Git actions",
			"显示初始化、切换分支、提交及菜单。关闭后仍可通过进行中的进度与对话框完成操作。",
			"Show init, branch, commit and menus. The in-progress progress and dialogs still work when hidden."},
		{"ui-titlebar", "terminalToggle", "终端抽屉开关", "Terminal drawer toggle",
			"在标题栏显示终端抽屉按钮。关闭后仍可使用 Ctrl+` 切换。",
			"Show the terminal drawer button in the titlebar. Ctrl+` still toggles it when hidden."},
		{"ui-titlebar", "surfacesToggle", "右侧栏按钮", "Right-panel buttons",
			"在标题栏显示右侧栏按钮。关闭后仍可使用 Ctrl+\\ 开合同一条右栏。",
			"Show the right-panel button in the titlebar. Ctrl+\\ still opens the same panel."},
		{"ui-conversation", "composerResize", "拖动调整输入框大小", "Drag to resize the composer",
			"允许拖动输入区上、左、右边框改变高度和宽度。关闭后高度仍按草稿自动增高，宽度回到栏宽。",
			"Drag the composer's top, left and right edges to resize it. When off, height still grows with the draft and width returns to the lane."},
		{"ui-conversation", "statsLine", "会话统计", "Session statistics",
			"在输入框上方显示轮次、耗时、缓存与 token。关闭后隐藏数字，保留这一行间隔。",
			"Show turns, elapsed time, cache and tokens above the composer. When off, the numbers hide but the row keeps its room."},
		{"ui-conversation", "viewTabs", "对话与轨迹页签", "Conversation and trajectory tabs",
			"在会话顶栏显示「对话」「轨迹」切换。关闭后仍停留在当前视图。",
			"Show the Chat / Trajectory switch in the conversation header. When off, the current view stays."},
	} {
		if field.key == "viewTabs" {
			if hostViewTabs && viewTabsField.Kind == store.FieldBool {
				bodyHostPreference(c, a.Host, viewTabsNS, field.key, providerCopy(field.zh, field.en), providerCopy(field.zhDesc, field.enDesc), fg, muted)
			}
			continue
		}
		bodyHostPreference(c, a.Host, field.ns, field.key, providerCopy(field.zh, field.en), providerCopy(field.zhDesc, field.enDesc), fg, muted)
	}
	if !hostViewTabs && !localViewTabs {
		ui.Text(c, providerCopy("会话视图标签：等待当前 Host 的设置定义。", "Conversation view tabs: waiting for this Host's settings description.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	} else if hostViewTabs && viewTabsField.Kind != store.FieldBool {
		ui.Text(c, providerCopy("会话视图标签：此 Host 的设置类型不受支持。", "Conversation view tabs: this Host's field type is unsupported.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	bodyLocalToggle(c, "interface-archive", providerCopy("显示底部归档管理区", "Show the archive manager"), providerCopy("选择“隐藏已归档”时，在侧栏底部显示可折叠的“已归档”管理区。关闭后仍可从视图选项查看全部或仅归档会话，并取消归档或删除。", "When archived sessions are hidden, show a collapsible Archived area at the bottom of the sidebar. When off, you can still view all or only archived sessions from the view options and unarchive or delete them."), a.ShowArchivedList, a.OnSetShowArchivedList == nil, fg, muted, func(next bool) {
		setInterfaceAccepted(c, a.ShowArchivedList, next, a.OnSetShowArchivedList)
	})
	bodyDisclosure(c, "interface-local-preferences", providerCopy("本地偏好", "Local preferences"), func() {
		if localViewTabs && a.ShowViewTabs != nil {
			desktopRow(c, i18n.T(i18n.KeySettingsInterfaceViewTabs), func() {
				if ui.Switch(c, a.ShowViewTabs).Label(i18n.T(i18n.KeySettingsInterfaceViewTabs)).Disabled(a.OnSetViewTabs == nil).Changed() {
					setInterfaceAccepted(c, a.ShowViewTabs, *a.ShowViewTabs, a.OnSetViewTabs)
				}
			}, fg, muted)
			ui.Text(c, i18n.T(i18n.KeySettingsInterfaceViewTabsDesc)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		desktopRow(c, i18n.T(i18n.KeySettingsInterfaceAutoAllow), func() {
			if a.AutoAllowTools == nil {
				return
			}
			if ui.Switch(c, a.AutoAllowTools).Label(i18n.T(i18n.KeySettingsInterfaceAutoAllow)).Disabled(a.OnSetAutoAllowTools == nil).Changed() {
				setInterfaceAccepted(c, a.AutoAllowTools, *a.AutoAllowTools, a.OnSetAutoAllowTools)
			}
		}, fg, muted)
		ui.Text(c, i18n.T(i18n.KeySettingsInterfaceAutoAllowDesc)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	})
}

func setInterfaceAccepted(c *ui.Context, value *bool, next bool, write func(bool) (bool, error)) {
	if value == nil || write == nil {
		return
	}
	accepted, err := write(next)
	*value = accepted
	if err != nil {
		c.AddToast(ui.Toast{Title: err.Error()})
	}
}
