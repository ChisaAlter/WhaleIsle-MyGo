package settings

import (
	"context"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

func permissionDefaultUnavailable(s store.ViewState) bool {
	return s.SelectedNS == "permission" && (!s.PermissionDefaults.Loaded || len(s.PermissionDefaults.Options) == 0)
}

func permissionDefaultField(c *ui.Context, m Model, s store.ViewState, f store.Field, fg, muted ui.Color) bool {
	if s.SelectedNS != "permission" || f.Key != "defaultPreset" {
		return false
	}
	ui.Text(c, providerCopy("新任务默认权限", "Default permissions for new tasks")).FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg)
	ui.Text(c, providerCopy("只影响以后创建的任务；当前会话权限保持不变。", "Applies to newly created tasks. The current session keeps its permissions.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
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
