package settings

import (
	"context"
	"fmt"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"

	"github.com/egoist/mygo/ui"
)

func permissionDefaultUnavailable(s store.ViewState) bool {
	return s.SelectedNS == "permission" && (!s.PermissionDefaults.Loaded || len(s.PermissionDefaults.Options) == 0)
}

func permissionDefaultField(c *ui.Context, m Model, s store.ViewState, f store.Field, fg, muted ui.Color) bool {
	if s.SelectedNS != "permission" || f.Key != "defaultPreset" {
		return false
	}
	kLabel(c, providerCopy("权限", "Permission"), fg)
	kHint(c, providerCopy("选择新会话的默认权限模式", "Choose the default permission mode for new sessions"))
	defaults := s.PermissionDefaults
	settingsAlert(c, "permission_defaults:25", defaults.Err != "", providerCopy("没能读取默认权限，请重试。", "Default permissions could not be read. Try again."), defaults.Err)
	if permissionDefaultUnavailable(s) {
		kHint(c, providerCopy("默认权限目录尚不可用。", "Default permission catalog is unavailable."))
		if kButtonDisabled(c, providerCopy("刷新目录", "Refresh catalog"), s.Loading || s.Saving).AlignSelf(ui.Start).Clicked() {
			m.Store.RefreshDescribe(context.Background())
		}
		return true
	}
	value := s.Drafts[f.Key]
	if value == "" {
		value = defaults.DefaultPreset
	}
	labels := make([]string, 0, len(defaults.Options))
	byLabel := make(map[string]string)
	selected, description := value, ""
	for _, option := range defaults.Options {
		label := option.Value
		if option.Name != "" && option.Name != option.Value {
			label = option.Name + " (" + option.Value + ")"
		}
		labels = append(labels, label)
		byLabel[label] = option.Value
		if option.Value == value {
			selected, description = label, option.Description
		}
	}
	if m.DraftPtr != nil {
		choice := m.DraftPtr("permission-default-label", selected)
		*choice = selected
		if kSelect(c, choice, labels, providerCopy("权限", "Permission"), s.Loading || s.Saving || !s.Writable).FillWidth().Changed() {
			m.Store.SetDraft(f.Key, byLabel[*choice])
		}
	}
	if description != "" {
		kHint(c, description)
	}
	if !s.Writable {
		kHint(c, providerCopy("此 Host 的设置为只读。", "Settings on this Host are read-only."))
	}
	if kButtonDisabled(c, providerCopy("重置为继承值", "Reset to inherited"), s.Loading || s.Saving || !s.Writable).AlignSelf(ui.Start).Clicked() {
		m.Store.ResetDraft(f.Key)
	}
	if s.Resets[f.Key] {
		kHint(c, providerCopy("保存后继承默认权限；选择其他项可取消重置。", "Save to inherit default permissions; choose an option to cancel the reset."))
	}
	return true
}

func generalPermissionPreference(c *ui.Context, a *BodySettingsAccess, fg, muted ui.Color) {
	_, value, found := bodyHostValue(a, "permission", "defaultPreset")
	if !found {
		return
	}
	clearFailure := bodyPreferenceFieldError(c, a, "permission", "defaultPreset", providerCopy("权限", "Permission"))
	defaults := a.Snapshot.PermissionDefaults
	if value == "" {
		value = defaults.DefaultPreset
	}
	options, values := []string{}, map[string]string{}
	selected := value
	for _, option := range defaults.Options {
		label := hostEnumLabel(option.Value)
		if label == option.Value && option.Name != "" {
			label = option.Name
		}
		if label == "" {
			label = option.Value
		}
		for _, other := range defaults.Options {
			if other.Value != option.Value && other.Name == option.Name {
				label += " (" + option.Value + ")"
				break
			}
		}
		options = append(options, label)
		values[label] = option.Value
		if option.Value == value {
			selected = label
		}
	}
	row := ui.Column(c.Key(fmt.Sprintf("general-permission-%d", a.Owner))).FillWidth().MinWidth(0)
	confirming := ui.Local(row, "confirm", func() bool { return false })
	acknowledged := ui.Local(row, "acknowledged", func() bool { return false })
	row.Children(func() {
		bodyPreferenceRow(c, "permission-row", providerCopy("权限", "Permission"), providerCopy("选择新会话的默认权限模式", "Choose the default permission mode for new sessions"), fg, muted, func() {
			if len(options) == 0 {
				kHint(c, providerCopy("默认权限目录尚不可用。", "Default permission catalog is unavailable."))
				return
			}
			if settingsPreferenceSelect(c, &selected, options).Label(providerCopy("默认权限", "Default permissions")).Disabled(bodyHostFieldDisabled(a, "permission", "defaultPreset")).Changed() {
				if values[selected] == "danger-full-access" {
					*confirming, *acknowledged = true, false
				} else {
					clearFailure()
					a.OnChange("permission", "defaultPreset", values[selected])
				}
			}
		})
		settingsAlert(c, "permission_defaults:128", defaults.Err != "", providerCopy("没能保存默认权限，请重试。", "Default permissions were not saved. Try again."), defaults.Err)
		bodyPreferenceFieldAlert(c, a, "permission", "defaultPreset", providerCopy("默认权限", "Default permissions"))
	})
	if bodyHostFieldDisabled(a, "permission", "defaultPreset") {
		*confirming, *acknowledged = false, false
	}
	if *confirming {
		ui.DialogBase(c, confirming, func(backdrop, panel ui.Element) {
			chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 420, design.Sp(6)).Gap(design.Sp(3)).Children(func() {
				bodyHeading(c, providerCopy("确认启用完全权限？", "Enable Full access?"), providerCopy("启用完全权限后，新会话将减少确认步骤，并且可以直接执行更多操作，包括敏感操作、文件修改或外部命令。仅建议在你信任后续任务时使用。", "Full access lets new sessions reduce confirmation steps and perform more actions directly, including sensitive operations, file changes, or external commands. Only use it when you trust subsequent tasks."), fg, muted)
				kCheckbox(c, acknowledged, providerCopy("我已了解风险，并愿意继续", "I understand the risks and want to continue"), false).Label("permission-full-access-acknowledgement")
				ui.Row(c).Wrap().Gap(design.Sp(2)).Children(func() {
					if kButton(c, providerCopy("取消", "Cancel")).Clicked() {
						*confirming = false
					}
					if kDanger(c, providerCopy("启用完全权限", "Enable Full access"), !*acknowledged || bodyHostFieldDisabled(a, "permission", "defaultPreset")).Clicked() {
						*confirming = false
						clearFailure()
						a.OnChange("permission", "defaultPreset", "danger-full-access")
					}
				})
			})
		})
	}
}
