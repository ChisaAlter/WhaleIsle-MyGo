package sidebar

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

type connectionTriggerState struct {
	initialized                                                  bool
	epoch                                                        uint64
	previous, rendered                                           string
	connectingAt, holdUntil, recoveredUntil, leaveUntil, entered time.Time
}

func (s *connectionTriggerState) project(now time.Time, epoch uint64, phase string) (string, float32) {
	if !s.initialized || s.epoch != epoch {
		*s = connectionTriggerState{initialized: true, epoch: epoch, previous: phase}
		if phase == "connecting" {
			s.connectingAt = now
		}
	}
	if s.previous != phase {
		if phase == "connecting" {
			s.connectingAt = now
		}
		if s.previous == "connecting" {
			s.holdUntil = s.connectingAt.Add(800 * time.Millisecond)
		}
		if phase == "connected" && (s.previous == "connecting" || s.previous == "disconnected") {
			start := now
			if s.holdUntil.After(start) {
				start = s.holdUntil
			}
			s.recoveredUntil = start.Add(2 * time.Second)
		} else if phase != "connected" {
			s.recoveredUntil = time.Time{}
		}
		s.previous = phase
	}
	desired := ""
	if phase == "connecting" || now.Before(s.holdUntil) {
		desired = "connecting"
	} else if phase == "disconnected" {
		desired = "disconnected"
	} else if now.Before(s.recoveredUntil) {
		desired = "recovered"
	}
	if desired != "" {
		if s.rendered != desired {
			s.rendered, s.entered = desired, now
		}
		s.leaveUntil = time.Time{}
	} else if s.rendered != "" {
		if s.leaveUntil.IsZero() {
			s.leaveUntil = now.Add(150 * time.Millisecond)
		}
		if !now.Before(s.leaveUntil) {
			s.rendered = ""
		}
	}
	alpha := float32(1)
	if !s.leaveUntil.IsZero() {
		alpha = float32(s.leaveUntil.Sub(now)) / float32(150*time.Millisecond)
	} else if !s.entered.IsZero() {
		alpha = min(1, float32(now.Sub(s.entered))/float32(150*time.Millisecond))
	}
	return s.rendered, max(0, alpha)
}

var connectionRefresh = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="M13 6a5 5 0 1 0 0 5M13 2v4H9"/></svg>`))
var connectionCheck = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="m3 8 3 3 7-7"/></svg>`))
var updateDownload = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M8 1.95317V10.0469M4.25 6.29688L8 10.0469L11.75 6.29688M1.5 10.0469V13.158C1.5 13.3937 1.60536 13.6198 1.79289 13.7865C1.98043 13.9532 2.23478 14.0469 2.5 14.0469H13.5C13.7652 14.0469 14.0196 13.9532 14.2071 13.7865C14.3946 13.6198 14.5 13.3937 14.5 13.158V10.0469"/></svg>`))
var settingsGear = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"><path d="m8 2-.5 2-2 .9-1.8-.5L2 7.4l1.5 1.5v2.2L2 12.6l1.7 3 1.8-.5 2 .9.5 2h4l.5-2 2-.9 1.8.5 1.7-3-1.5-1.5V8.9L18 7.4l-1.7-3-1.8.5-2-.9-.5-2z"/><circle cx="10" cy="10" r="3"/></svg>`))

// SettingsRoot's expanded trigger and compact rail share the exact physical
// connection producer. A recovered session/list snapshot cannot stand in for it.
func settingsConnectionTrigger(c *ui.Context, m Model, compact bool) {
	t := tok(c)
	height := float32(44)
	if compact {
		height = 80
	}
	row := ui.Box(c.Key("settings-connection-trigger")).FillWidth().Height(height)
	state := ui.Local(row, "connection", func() connectionTriggerState { return connectionTriggerState{} })
	phase := "disconnected"
	if m.ConnectionReady {
		phase = "connected"
	} else if m.ConnectionConnecting {
		phase = "connecting"
	}
	status, opacity := state.project(c.Now(), m.ConnectionEpoch, phase)
	if m.DesktopUpdateInstalling || m.DesktopUpdatePhase == "installing" {
		status, opacity = "installing", 1
	} else if status == "" && m.DesktopUpdatePhase != "" && m.DesktopUpdatePhase != "idle" {
		status, opacity = "update:"+m.DesktopUpdatePhase, 1
	}
	reduced := c.Preferences().ReduceMotion
	if reduced {
		opacity = 1
	}
	for _, deadline := range []time.Time{state.holdUntil, state.recoveredUntil, state.leaveUntil} {
		if deadline.After(c.Now()) {
			c.After(deadline.Sub(c.Now()))
		}
	}
	row.Children(func() {
		combined := !compact && m.OnOpenSettings != nil
		seatLabel := i18n.T(i18n.KeySettingsOpen)
		if combined {
			seatLabel = i18n.T(i18n.KeySettingsAccountSeat)
		}
		button := coss.Button(c, "", coss.ButtonOptions{
			Variant:  coss.ButtonGhost,
			Size:     coss.ButtonMD,
			Disabled: m.OnOpenSettings == nil,
			Label:    seatLabel,
			Key:      "settings-connection-button",
			Content:  func() {},
		}).FillWidth().Height(42).Padding(0, design.Sp(2)).Radius(design.RadiusMD).BorderWidth(0).Label(seatLabel).Tooltip(seatLabel)
		if compact {
			button.Size(36, 36).Absolute().Bottom(0).Padding(0)
		}
		if combined {
			button.Menu(func(menu *ui.Menu) {
				if m.OnOpenSettings != nil && menu.Item(i18n.T(i18n.KeySettingsOpen)).Chosen() {
					m.OnOpenSettings()
				}
				if m.OnContactAccount != nil && menu.Item(workspaceText("联系我们", "Contact us")).Chosen() {
					m.OnContactAccount()
				}
				openAccount := m.Account.OnOpen
				if openAccount == nil {
					openAccount = m.OnOpenSettings
				}
				if openAccount != nil && menu.Item(i18n.T(i18n.KeySettingsAccountTitle)).Chosen() {
					openAccount()
				}
				if m.Account.OnSignOut != nil && menu.Item(i18n.T(i18n.KeySettingsAccountSignOut)).Chosen() {
					m.Account.OnSignOut()
				}
			})
		}
		button.Children(func() {
			ui.Row(c).Fill().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, settingsGear).Size(design.IconLG, design.IconLG).TextColor(t.Foreground)
				if !compact {
					ui.Text(c, seatLabel).Grow(1).SingleLine().FontSize(fs(14)).TextColor(t.Foreground).Margin(0, design.Sp(10), 0, 0)
				}
			})
		})
		if !combined && button.Clicked() && m.OnOpenSettings != nil {
			m.OnOpenSettings()
		}
		if status == "" {
			return
		}
		seat := ui.Box(c).Size(36, 36).Absolute().Right(0).Top(design.Sp(1))
		if compact {
			seat.Top(0)
		}
		if opacity < 1 {
			seat.Opacity(opacity)
		}
		now := c.Now()
		isUpdate := status == "installing" || len(status) > 7 && status[:7] == "update:"
		if !reduced && !isUpdate && ((!state.leaveUntil.IsZero() && now.Before(state.leaveUntil)) || (!state.entered.IsZero() && now.Sub(state.entered) < 150*time.Millisecond)) {
			c.After(16 * time.Millisecond)
		}
		seat.Children(func() {
			if status == "installing" || len(status) > 7 && status[:7] == "update:" {
				phase := m.DesktopUpdatePhase
				if status == "installing" {
					phase = "installing"
				}
				desktopUpdateSeat(c, m, phase)
			} else {
				connectionStatusSeat(c, m, status)
			}
		})
	})
}

func connectionStatusSeat(c *ui.Context, m Model, status string) {
	t := tok(c)
	label, action := workspaceText("连接异常，刷新重试", "Disconnected"), workspaceText("连接异常，点击立即重连", "Disconnected, reconnect now")
	if status == "connecting" {
		label, action = workspaceText("重新连接中", "Reconnecting"), workspaceText("连接中断，正在重试，点击立即重连", "Reconnecting, reconnect now")
	}
	if m.ConnectionError != "" {
		label += "\n" + m.ConnectionError
		action += "\n" + m.ConnectionError
	}
	if status == "recovered" {
		ui.Box(c).Size(36, 36).Radius(design.RadiusMD).
			Background(t.Success.Alpha(0.12)).Border(design.BorderWidth, t.Success.Alpha(0.24)).
			Role(ui.RoleStatus).Label(workspaceText("连接成功", "Connected")).Tooltip(workspaceText("连接成功", "Connected")).
			Center().Children(func() {
			ui.Icon(c, connectionCheck).Size(design.IconXS, design.IconXS).TextColor(t.SuccessForeground)
		})
		return
	}
	fill := t.Warning.Alpha(0.12)
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     coss.ButtonIconLG,
		Disabled: m.OnReconnect == nil,
		Label:    action,
		Key:      "connection-status-seat",
		Content:  func() {},
	}).Size(36, 36).Padding(0).Radius(design.RadiusMD).Center().
		Background(fill).Border(design.BorderWidth, t.Warning.Alpha(0.24)).
		Label(action).Tooltip(label)
	if button.Hovered() {
		button.Background(t.Warning.Alpha(0.2))
	}
	button.Children(func() {
		if status == "connecting" {
			coss.Spinner(c, coss.SpinnerOptions{Size: design.IconXS, Color: t.MutedForeground})
		} else {
			ui.Icon(c, connectionRefresh).Size(design.IconXS, design.IconXS).TextColor(t.WarningForeground)
		}
	})
	if button.Clicked() && m.OnReconnect != nil {
		m.OnReconnect()
	}
}

// The sidebar is a status projection, not the update center. Only actual
// updater changes should wake the workbench, including during a slow check.
func desktopUpdateSeat(c *ui.Context, m Model, phase string) {
	labels := map[string]string{
		"checking":    workspaceText("正在检查更新…", "Checking for updates…"),
		"available":   workspaceText("有可用更新", "Update available"),
		"downloading": workspaceText("正在下载更新…", "Downloading update…"),
		"verifying":   workspaceText("正在校验更新…", "Verifying update…"),
		"installing":  workspaceText("正在准备重启…", "Preparing to restart…"),
		"ready":       workspaceText("安装并重启", "Install and Restart"),
		"error":       workspaceText("重试更新", "Retry update"),
	}
	label := labels[phase]
	if label == "" {
		return
	}
	detail := m.DesktopUpdateDetail
	if detail == "" {
		detail = label
	}
	t := tok(c)
	busy := phase == "checking" || phase == "downloading" || phase == "verifying" || phase == "installing"
	ink := t.Primary
	if busy {
		ink = t.MutedForeground
	}
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     coss.ButtonIconLG,
		Disabled: busy,
		Label:    label,
		Key:      "desktop-update-seat",
		Content:  func() {},
	}).Fill().Padding(0).Radius(design.RadiusMD).BorderWidth(0).Center().
		TextColor(ink).Label(label).Tooltip(detail)
	button.Children(func() {
		if phase == "error" {
			ui.Box(c).Size(6, 6).Radius(design.RadiusFull).Background(ink)
		} else if busy {
			coss.Spinner(c, coss.SpinnerOptions{Size: design.IconXS, Color: ink})
		} else {
			ui.Icon(c, updateDownload).Size(design.IconXS, design.IconXS).TextColor(ink)
		}
	})
	if !busy && button.Clicked() && m.OnOpenDesktopUpdate != nil {
		m.OnOpenDesktopUpdate()
	}
}
