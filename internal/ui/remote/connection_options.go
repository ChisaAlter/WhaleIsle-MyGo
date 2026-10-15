package remote

import (
	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
)

func viewConnectionOptions(c *ui.Context, e *MachineEditor, busy bool) {
	if e.Draft.Options == nil { e.Draft.Options = &remotestore.ConnectionDraft{ProxyPort: "22"} }
	o := e.Draft.Options
	for _, f := range []struct { title string; value *string }{
		{label("密码（留空保留）", "Password (blank keeps saved)"), &o.Password},
		{label("私钥口令", "Key passphrase"), &o.Passphrase},
	} {
		ui.Row(c).Gap(8).Children(func() { ui.Text(c, f.title).Width(180); ui.TextInput(c, f.value).Password().FillWidth().Disabled(busy) })
	}
	ui.Text(c, label("主机密钥策略", "Host-key policy"))
	ui.Row(c).Gap(6).Children(func() {
		for _, mode := range []struct { id, title string }{
			{"", label("接受新密钥", "Accept new")}, {"verify", label("严格验证", "Verify")}, {"off", label("关闭验证", "Off")},
		} {
			title := mode.title
			if o.HostKeyMode == mode.id { title = "✓ " + title }
			if ui.Button(c, title).Disabled(busy).Clicked() { o.HostKeyMode = mode.id }
		}
	})
	if o.HostKeyMode == "off" { ui.Text(c, label("关闭验证会失去服务器身份保护。", "Disabling verification removes server identity protection.")) }
	ui.Checkbox(c, &o.UseAgent, label("使用 SSH Agent", "Use SSH agent")).Disabled(busy)
	ui.Checkbox(c, &o.KeyboardInteractive, label("键盘交互认证", "Keyboard-interactive authentication")).Disabled(busy)
	ui.Checkbox(c, &o.EncryptPassword, label("使用系统凭据存储保存新密码", "Save new password in the OS credential store")).Disabled(busy)
	ui.Text(c, label("跳板机（可选）", "Jump host (optional)")).Bold()
	for _, f := range []struct { title string; value *string }{
		{label("跳板主机", "Proxy host"), &o.ProxyHost}, {label("跳板端口", "Proxy port"), &o.ProxyPort},
		{label("跳板用户", "Proxy username"), &o.ProxyUsername}, {label("跳板私钥路径", "Proxy key path"), &o.ProxyKeyPath},
	} {
		ui.Row(c).Gap(8).Children(func() { ui.Text(c, f.title).Width(180); ui.TextInput(c, f.value).FillWidth().Disabled(busy) })
	}
	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, label("跳板密码（留空保留）", "Proxy password (blank keeps saved)")).Width(180)
		ui.TextInput(c, &o.ProxyPassword).Password().FillWidth().Disabled(busy)
	})
}
