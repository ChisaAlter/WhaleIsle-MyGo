package settings

import (
	"context"
	"fmt"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func permissionDefaultUnavailable(s store.ViewState) bool {
	return s.SelectedNS == "permission" && (!s.PermissionDefaults.Loaded || len(s.PermissionDefaults.Options) == 0)
}

func permissionDefaultField(c *ui.Context, m Model, s store.ViewState, f store.Field, fg, muted ui.Color) bool {
	if s.SelectedNS != "permission" || f.Key != "defaultPreset" {
		return false
	}
	ui.Text(c, providerCopy("权限", "Permission")).FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg)
	ui.Text(c, providerCopy("选择新会话的默认权限模式", "Choose the default permission mode for new sessions")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	defaults := s.PermissionDefaults
	if defaults.Err != "" {
		ui.Text(c, defaults.Err).FontSize(theme.InterfaceSize(11)).TextColor(fg)
	}
	if permissionDefaultUnavailable(s) {
		ui.Text(c, providerCopy("默认权限目录尚不可用。", "Default permission catalog is unavailable.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, providerCopy("刷新目录", "Refresh catalog")).Disabled(s.Loading || s.Saving).Clicked() {
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
		if ui.Select(c, choice, labels).FillWidth().Disabled(s.Loading || s.Saving || !s.Writable).Changed() {
			m.Store.SetDraft(f.Key, byLabel[*choice])
		}
	}
	if description != "" {
		ui.Text(c, description).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if !s.Writable {
		ui.Text(c, providerCopy("此 Host 的设置为只读。", "Settings on this Host are read-only.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if ui.Button(c, providerCopy("重置为继承值", "Reset to inherited")).Disabled(s.Loading || s.Saving || !s.Writable).Clicked() {
		m.Store.ResetDraft(f.Key)
	}
	if s.Resets[f.Key] {
		ui.Text(c, providerCopy("保存后继承默认权限；选择其他项可取消重置。", "Save to inherit default permissions; choose an option to cancel the reset.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	return true
}

func generalPermissionPreference(c *ui.Context, a *BodySettingsAccess, fg, muted ui.Color) {
	_, value, found := bodyHostValue(a, "permission", "defaultPreset")
	if !found {
		return
	}
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
				ui.Text(c, providerCopy("默认权限目录尚不可用。", "Default permission catalog is unavailable.")).FillWidth().TextColor(muted)
				return
			}
			if settingsPreferenceSelect(c, &selected, options).Label(providerCopy("默认权限", "Default permissions")).Disabled(bodyHostDisabled(a)).Changed() {
				if values[selected] == "danger-full-access" {
					*confirming, *acknowledged = true, false
				} else {
					a.OnChange("permission", "defaultPreset", values[selected])
				}
			}
		})
		if defaults.Err != "" {
			ui.Text(c, defaults.Err).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
	})
	if bodyHostDisabled(a) {
		*confirming, *acknowledged = false, false
	}
	if *confirming {
		ui.DialogBase(c, confirming, func(backdrop, panel ui.Element) {
			chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 420, 24).Gap(12).Children(func() {
				bodyHeading(c, providerCopy("确认启用完全权限？", "Enable Full access?"), providerCopy("新任务将可以访问工作区之外的文件并执行命令。", "New tasks can access files outside the workspace and execute commands."), fg, muted)
				ui.Checkbox(c, acknowledged, providerCopy("我了解这些权限的影响", "I understand the impact of these permissions")).Label("permission-full-access-acknowledgement")
				ui.Row(c).Wrap().Gap(8).Children(func() {
					if ui.Button(c, providerCopy("取消", "Cancel")).Clicked() {
						*confirming = false
					}
					if ui.Button(c, providerCopy("启用完全权限", "Enable Full access")).Disabled(!*acknowledged).Clicked() {
						*confirming = false
						a.OnChange("permission", "defaultPreset", "danger-full-access")
					}
				})
			})
		})
	}
}
