package composer

import "github.com/egoist/mygo/ui"

func paintPermissionCommand(c *ui.Context, m Model, panel, border, fg, muted ui.Color) {
	ui.Column(c.Key("local-permission-picker:" + m.Snapshot.SelectedID)).FillWidth().Padding(12).Gap(8).Background(panel).BorderColor(border).BorderWidth(1).Radius(12).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, "权限预设 / Permission preset").TextColor(fg).Grow(1)
			if ui.Button(c, "关闭 / Close").Clicked() {
				m.State.PermissionCommand.Open = false
			}
		})
		if len(m.PermissionOptions) == 0 {
			ui.Text(c, "当前会话尚无可用权限预设 / No permission presets available for this session").TextColor(muted)
			if m.OnRetryPermissions != nil && ui.Button(c, "刷新 / Retry").Clicked() {
				m.OnRetryPermissions()
			}
			return
		}
		caption := m.Snapshot.SelectedPermission
		presetSelect(c, caption, permissionChoices(m.PermissionOptions), func(value string) {
			m.State.PermissionCommand.Open = false
			if m.OnPermissionChange != nil {
				m.OnPermissionChange(value)
			}
		})
	})
}
