package remote

import (
	"github.com/egoist/mygo/ui"
	"strconv"
	"whaleisle/internal/config"
)

// GatewayEditor owns an explicit draft, so runtime polling cannot overwrite edits.
type GatewayEditor struct {
	Open            bool
	Draft, Expected config.RemoteGatewayPreferences
	Error           string
}

func viewGatewayEditor(c *ui.Context, m Model, fg, muted ui.Color) {
	e := m.GatewayEditor
	if e == nil || m.OnSaveGateway == nil {
		return
	}
	if !e.Open {
		if ui.Button(c, label("编辑手机网关", "Edit phone gateway")).Clicked() {
			e.Expected = config.RemoteGatewayPreferences{Mode: m.RemoteMode, Port: strconv.Itoa(m.RemotePort), BindAddress: m.RemoteBindAddress, RelayEndpoint: m.RemoteRelayEndpoint, RelayTLS: m.RemoteRelayUseTLS}
			e.Draft = e.Expected
			e.Error = ""
			e.Open = true
		}
		return
	}
	ui.Text(c, label("更改会重连网关并撤销当前手机 Git 授权；已配对设备保留。", "Changes reconnect the gateway and revoke the current phone Git grant; paired devices are retained.")).TextColor(muted)
	ui.Text(c, label("连接模式", "Connection mode")).TextColor(fg)
	ui.Select(c, &e.Draft.Mode, []string{"relay", "lan", "ssh"}).FillWidth()
	for _, r := range []struct {
		name  string
		value *string
	}{
		{label("端口 (1024–65535)", "Port (1024–65535)"), &e.Draft.Port},
		{label("绑定 IPv4（127.0.0.1 / 0.0.0.0 / 本机地址）", "Bind IPv4 (127.0.0.1 / 0.0.0.0 / local address)"), &e.Draft.BindAddress},
		{label("中继 host[:port]（留空使用默认）", "Relay host[:port] (blank uses default)"), &e.Draft.RelayEndpoint},
	} {
		ui.Text(c, r.name).TextColor(fg)
		ui.TextInput(c, r.value).FillWidth()
	}
	ui.Checkbox(c, &e.Draft.RelayTLS, label("中继 TLS（默认中继始终使用 TLS）", "Relay TLS (always enabled for default relay)"))
	ui.Text(c, label("0.0.0.0 会监听全部网卡；仅填写信任的中继地址。", "0.0.0.0 listens on all interfaces; use only a trusted relay address.")).TextColor(muted)
	if e.Error != "" {
		ui.Text(c, e.Error).TextColor(fg)
	}
	if ui.Button(c, label("保存并重新连接", "Save and reconnect")).Disabled(m.PhoneRuntime.Status == "starting" || m.PhoneRuntime.DeviceAction.Busy).Clicked() {
		if err := m.OnSaveGateway(e.Expected, e.Draft); err != nil {
			e.Error = err.Error()
		} else {
			e.Open = false
			e.Error = ""
		}
	}
	if ui.Button(c, label("取消", "Cancel")).Clicked() {
		e.Open = false
		e.Error = ""
	}
}
