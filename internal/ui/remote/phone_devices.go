package remote

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"strings"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// PhoneDeviceEditor is window-local form state. IDs always come from the
// original runtime's public snapshot; no device secret is displayed or cached.
type PhoneDeviceEditor struct {
	EditingID, Name, PendingUnbind                          string
	ConfirmRotate, ConfirmRevoke, Submitted                 bool
	Error                                                   string
	PreviousActionID, SubmittedOperation, SubmittedDeviceID string
}

func viewPhoneDevices(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	s, e := m.PhoneRuntime, m.PhoneDeviceEditor
	action := s.DeviceAction
	if e != nil && e.Submitted && !action.Busy && action.ID != e.PreviousActionID && action.Operation == e.SubmittedOperation && action.DeviceID == e.SubmittedDeviceID {
		e.Submitted = false
		if action.Applied {
			e.EditingID, e.Name, e.PendingUnbind = "", "", ""
			e.ConfirmRevoke = false
			e.ConfirmRotate = false
		}
	}
	if action.Error != "" {
		ui.Text(c, action.Error).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
	} else if action.Message != "" {
		ui.Text(c, action.Message).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if e != nil && e.Error != "" {
		ui.Text(c, e.Error).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
	}
	if !s.DevicesLoaded {
		// Historical credential labels are read-only and cannot identify a
		// device in the original runtime's authoritative registry.
		if len(m.RemoteDeviceNames) > 0 {
			ui.Text(c, label("历史设备标签（刷新后管理）", "Historical device labels (refresh to manage)")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			for _, name := range m.RemoteDeviceNames {
				ui.Text(c, "· "+name).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			}
		}
		ui.Text(c, label("启用手机运行时后自动读取已配对设备；读取失败可刷新重试。", "Enable the phone runtime to load paired devices; refresh to retry if loading fails.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		return
	}
	ui.Text(c, label("已配对手机", "Paired phones")).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
	if len(s.Devices) == 0 {
		ui.Text(c, label("尚无已配对设备。", "No paired devices.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if e == nil || m.OnPhoneDeviceAction == nil {
		for _, device := range s.Devices {
			ui.Text(c, device.Name).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		return
	}
	disabled := action.Busy || !m.RemoteEnabled || !s.Listening
	run := func(operation, id, name string) {
		if err := m.OnPhoneDeviceAction(operation, id, name); err != nil {
			e.Error = errcopy.Sentence(err)
			return
		}
		e.Error = ""
		e.Submitted = true
		e.PreviousActionID, e.SubmittedOperation, e.SubmittedDeviceID = action.ID, operation, id
	}
	for _, device := range s.Devices {
		device := device
		chrome.FrostSurface(ui.Column(c).Border(1, border).Padding(8, 10, 8, 10).Gap(6), colors, m.SystemDark, field, 8).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, device.Name).FillWidth().MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
				if coss.Button(c, label("重命名", "Rename"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled}).Clicked() {
					e.EditingID, e.Name, e.Error = device.ID, device.Name, ""
					e.PendingUnbind, e.ConfirmRotate = "", false
				}
				if coss.Button(c, label("解绑", "Unbind"), coss.ButtonOptions{Variant: coss.ButtonDestructive, Size: coss.ButtonSM, Disabled: disabled}).Clicked() {
					e.PendingUnbind = device.ID
					e.EditingID, e.ConfirmRotate = "", false
				}
			})
			ui.Text(c, device.ID).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
			if device.LastSeenAt != "" {
				ui.Text(c, label("最近连接：", "Last seen: ")+device.LastSeenAt).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
			}
			if e.EditingID == device.ID {
				coss.Input(c, &e.Name, coss.InputOptions{Disabled: disabled})
				ui.Row(c).Gap(8).Children(func() {
					if coss.Button(c, label("保存名称", "Save name"), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Disabled: disabled || strings.TrimSpace(e.Name) == ""}).Clicked() {
						run("rename", device.ID, e.Name)
					}
					if coss.Button(c, label("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Disabled: action.Busy}).Clicked() {
						e.EditingID, e.Name, e.Error = "", "", ""
					}
				})
			}
			if e.PendingUnbind == device.ID {
				ui.Text(c, label("解绑此手机？所有手机连接将暂时中断，此手机需要重新配对；其他手机可重新连接。", "Unbind this phone? All phone connections briefly disconnect. This phone must pair again; other phones can reconnect.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
				ui.Row(c).Gap(8).Children(func() {
					if coss.Button(c, label("确认解绑", "Confirm unbind"), coss.ButtonOptions{Variant: coss.ButtonDestructive, Size: coss.ButtonSM, Disabled: disabled}).Clicked() {
						run("unbind", device.ID, "")
					}
					if coss.Button(c, label("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Disabled: action.Busy}).Clicked() {
						e.PendingUnbind = ""
					}
				})
			}
		})
	}
	if s.PairingLinkURL != "" {
		ui.Text(c, label("长期链接可重复配对，直到在此撤销。已配对设备不受撤销影响。", "A reusable link can pair devices until revoked here. Paired devices stay connected.")).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if coss.Button(c, label("复制长期链接", "Copy reusable link"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled}).Clicked() {
				if err := mygo.Clipboard.Write(transfer.TextData(s.PairingLinkURL), mygo.ClipboardOptions{}); err != nil {
					e.Error = errcopy.Sentence(err)
				} else {
					e.Error = ""
				}
			}
			if coss.Button(c, label("撤销长期链接", "Revoke reusable link"), coss.ButtonOptions{Variant: coss.ButtonDestructive, Size: coss.ButtonSM, Disabled: disabled}).Clicked() {
				e.ConfirmRevoke = true
			}
		})
	}
	if !action.Busy && s.Status == "ready" && s.PairingURL != "" && s.PairingLinkURL == "" {
		ui.Text(c, label("当前手机运行时未提供长期链接；临时二维码仍可使用。", "This phone runtime does not provide a reusable link; the temporary QR remains available.")).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if e.ConfirmRevoke {
		ui.Text(c, label("撤销并生成新长期链接？旧链接不能再配对，已配对设备保持连接。", "Revoke and replace this link? It will stop pairing new devices; paired devices stay connected.")).FillWidth().TextColor(muted)
		ui.Row(c).Gap(8).Children(func() {
			if coss.Button(c, label("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled}).Clicked() {
				e.ConfirmRevoke = false
			}
			if coss.Button(c, label("撤销并生成新链接", "Revoke and replace"), coss.ButtonOptions{Variant: coss.ButtonDestructive, Size: coss.ButtonSM, Disabled: disabled}).Clicked() {
				run("revoke-link", "", "")
			}
		})
	}
	if coss.Button(c, label("更新配对令牌", "Renew pairing token"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: disabled}).AlignSelf(ui.Start).Clicked() {
		e.ConfirmRotate = true
		e.PendingUnbind, e.EditingID = "", ""
	}
	if e.ConfirmRotate {
		ui.Text(c, label("生成新的临时配对链接？已配对设备身份保持不变，旧链接仍遵循原有效期。", "Generate a new temporary pairing link? Paired device identities stay unchanged; older links retain their original expiry.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Row(c).Gap(8).Children(func() {
			if coss.Button(c, label("确认更新", "Confirm renewal"), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Disabled: disabled}).Clicked() {
				run("rotate", "", "")
			}
			if coss.Button(c, label("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Disabled: action.Busy}).Clicked() {
				e.ConfirmRotate = false
			}
		})
	}
}
