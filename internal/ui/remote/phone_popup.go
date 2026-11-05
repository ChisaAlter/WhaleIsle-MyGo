package remote

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// PhoneMenuAction is the same shortcut within gateway settings. The shell's
// launcher uses PhonePopup with its own persistent anchor/open state.
func PhoneMenuAction(c *ui.Context, m Model) {
	anchor := ui.Button(c.Key("phone-connect-popup-action"), label("连接手机…", "Connect phone…"))
	open := ui.Local(anchor, "phone-popup-open", func() bool { return false })
	if anchor.Clicked() {
		*open = !*open
	}
	PhonePopup(c, anchor, open, m)
}

type phonePopupState struct{ devices, git bool }

// PhonePopup is a modal-layer shortcut, anchored to the caller's action. An
// inner device layer consumes dismissal before the main connection layer.
func PhonePopup(c *ui.Context, anchor ui.Element, open *bool, m Model) {
	if open == nil || !*open {
		return
	}
	state := ui.Local(anchor, "phone-popup-layers", func() phonePopupState { return phonePopupState{} })
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	field, border, accent := chrome.Color(colors.Field), chrome.Color(colors.Border), chrome.Color(colors.Accent)
	runtime := m.PhoneRuntime
	busy := runtime.DeviceAction.Busy || runtime.Status == "starting"
	_, height := c.Size()
	dismiss := func() {
		if state.devices {
			state.devices = false
			return
		}
		if state.git {
			state.git = false
			return
		}
		*open = false
	}
	ui.DialogBaseWithOptions(c.Key("phone-connect-popup"), open, ui.DialogOptions{DismissDisabled: busy, OnDismiss: dismiss}, func(backdrop, panel ui.Element) {
		chrome.DialogSurfaceSized(c, backdrop, panel, c.Theme(), chrome.DialogSurfaceBox{Width: 280, Radius: 12, Padding: 12}).MaxWidthPercent(94).MaxHeight(height-24).
			AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(8, 0).Gap(10).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label("手机连接", "Phone connection")).Bold().Grow(1).MinWidth(0).TextColor(fg)
				caption := label("启用", "Enable")
				if m.RemoteEnabled {
					caption = label("关闭", "Turn off")
				}
				if ui.Button(c, caption).Disabled(busy || m.OnTogglePairing == nil).Clicked() {
					m.OnTogglePairing(!m.RemoteEnabled)
				}
			})
			ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(max(float32(80), height-160)).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
					if !m.RemoteEnabled {
						ui.Text(c, label("启用后，用手机扫描二维码即可连接。", "Enable, then scan the QR code with your phone.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
						return
					}
					status := runtime.Status
					if status == "" {
						status = "offline"
					}
					snap := phoneSnapshot(m)
					readiness := phoneReadinessLabel(snap.Readiness)
					if runtime.Status == "ready" && runtime.Listening && (runtime.RelayConnected || lanMode(m.RemoteMode)) {
						readiness = i18n.T(i18n.KeyRemotePhoneServiceReady)
					}
					ui.Text(c, readiness).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
					usable := offerUsable(runtime, status, lanMode(m.RemoteMode))
					if usable && runtime.PairingImage != nil {
						ui.Column(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
							ui.Image(c, ui.NewBitmap(runtime.PairingImage)).Size(196, 196).Fit(ui.Contain)
							ui.Text(c, label("用手机扫码连接", "Scan to connect")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
							seconds := max(0, int(runtime.ExpiresAt.Sub(c.Now()).Seconds()))
							ui.Text(c, fmt.Sprintf(label("有效时间 %02d:%02d", "Valid for %02d:%02d"), seconds/60, seconds%60)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						})
						c.After(time.Second)
					} else {
						text := label("正在等待可用的配对码…", "Waiting for a pairing code…")
						if runtime.PairingURL != "" && !c.Now().Before(runtime.ExpiresAt) {
							text = label("配对码已过期，请刷新。", "Pairing code expired. Refresh to continue.")
						}
						if usable && runtime.PairingImage == nil {
							text = label("二维码不可用，请复制配对链接。", "QR image unavailable. Copy the pairing link.")
						}
						ui.Text(c, text).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
					if runtime.Message != "" {
						ui.Text(c, runtime.Message).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(accent)
					}
					if action := runtime.DeviceAction; action.Busy {
						ui.Text(c, label("正在更新配对状态…", "Updating pairing…")).FillWidth().TextColor(muted)
					} else if action.Error != "" {
						ui.Text(c, action.Error).FillWidth().TextColor(c.Theme().Danger)
					}
					if m.PhoneActionMsg != "" && !runtimeStatusWord(m.PhoneActionMsg) {
						ui.Text(c, m.PhoneActionMsg).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(accent)
					}
					ui.Row(c).FillWidth().Wrap().Gap(6).Children(func() {
						if ui.Button(c, label("复制链接", "Copy link")).Disabled(!usable || m.OnCopyConnectURL == nil).Clicked() {
							m.OnCopyConnectURL()
						}
						if ui.Button(c, label("打开", "Open")).Disabled(!usable || m.OnOpenConnectURL == nil).Clicked() {
							m.OnOpenConnectURL()
						}
						if ui.Button(c, label("刷新", "Refresh")).Disabled(busy || m.OnRefreshPhone == nil).Clicked() {
							m.OnRefreshPhone()
						}
					})
					enabled := m.PhoneGitEnabled
					if ui.Checkbox(c, &enabled, label("允许手机 Git 操作", "Allow phone Git")).Disabled(busy || m.OnTogglePhoneGit == nil).Changed() {
						m.OnTogglePhoneGit(enabled)
					}
					if m.OnTogglePhoneGitWorkspace != nil && ui.Button(c, label("管理 Git 工作区…", "Manage Git workspaces…")).Disabled(busy).Clicked() {
						state.git = true
					}
				})
			})
			ui.Box(c).FillWidth().Height(.5).Background(border)
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				count := phoneSnapshot(m).DeviceCount
				if ui.Button(c, fmt.Sprintf(label("已配对设备 (%d)", "Paired devices (%d)"), count)).Grow(1).MinWidth(0).Disabled(busy).Clicked() {
					state.devices = true
				}
				if ui.Button(c, label("收起", "Close")).Disabled(busy).Clicked() {
					dismiss()
				}
			})
		})
	})
	if state.devices {
		visible := true
		ui.DialogBaseWithOptions(c.Key("phone-devices-popup"), &visible, ui.DialogOptions{DismissDisabled: runtime.DeviceAction.Busy, OnDismiss: func() { state.devices = false }}, func(backdrop, panel ui.Element) {
			chrome.DialogSurfaceSized(c, backdrop, panel, c.Theme(), chrome.DialogSurfaceBox{Width: 280, Height: min(float32(420), height-120), Radius: 12, Padding: 12}).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(8, 0).Gap(10).Children(func() {
				ui.Text(c, label("已配对设备", "Paired devices")).Bold().TextColor(fg).Shrink(0)
				ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
					ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() { viewPhoneDevices(c, m, fg, muted, field, border, accent) })
				})
				ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, label("刷新", "Refresh")).Disabled(busy || m.OnRefreshPhone == nil).Clicked() {
						m.OnRefreshPhone()
					}
					if ui.Button(c, label("返回", "Back")).Disabled(runtime.DeviceAction.Busy).Clicked() {
						state.devices = false
					}
				})
			})
		})
	}
	if state.git {
		visible := true
		ui.DialogBaseWithOptions(c.Key("phone-git-workspaces-popup"), &visible, ui.DialogOptions{DismissDisabled: busy, OnDismiss: func() { state.git = false }}, func(backdrop, panel ui.Element) {
			chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 420, 16).MaxHeight(height - 24).Gap(12).Children(func() {
				ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(max(float32(80), height-120)).Children(func() { phoneGitWorkspaces(c, m) })
				if ui.Button(c, label("返回", "Back")).Disabled(busy).Clicked() {
					state.git = false
				}
			})
		})
	}
}
