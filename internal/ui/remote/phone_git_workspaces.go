package remote

import "github.com/egoist/mygo/ui"

// Candidate identities come from the current local Host/Project registry, never
// an arbitrary path field or phone-supplied path.
type PhoneGitWorkspace struct {
	ID, Title, Root string
	Selected        bool
}

func phoneGitWorkspaces(c *ui.Context, m Model) {
	if m.OnTogglePhoneGitWorkspace == nil {
		return
	}
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, label("额外手机 Git 工作区（本次 Host 运行）", "Additional phone Git workspaces (this Host run)")).Bold()
		ui.Text(c, label("默认不授权额外根。勾选后手机可在该已登记根执行 Git；取消立即撤销。重启 Host、关闭 Remote 或撤销授权后需重新选择。", "No extra roots are granted by default. Select a registered root to allow phone Git there; unselect to revoke immediately. Host restart, Remote disable or grant revocation clears selections."))
		if len(m.PhoneGitWorkspaces) == 0 {
			ui.Text(c, label("暂无已验证的本机 Host/Project 工作区；先连接本机 Host 并载入工作区。", "No verified local Host/Project workspaces. Connect the local Host and load its registry first."))
		}
		for _, row := range m.PhoneGitWorkspaces {
			row := row
			selected := row.Selected
			if ui.Checkbox(c.Key(row.ID), &selected, row.Title+" · "+row.Root).Disabled(!m.RemoteEnabled).Changed() {
				m.OnTogglePhoneGitWorkspace(row.ID, selected)
			}
		}
	})
}
