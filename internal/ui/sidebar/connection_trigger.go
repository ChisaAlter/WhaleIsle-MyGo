package sidebar

import (
	"github.com/egoist/mygo/ui"
	"math"
	"time"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
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

var connectionRefresh = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M13 6a5 5 0 1 0 0 5M13 2v4H9"/></svg>`))
var connectionCheck = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="m3 8 3 3 7-7"/></svg>`))
var updateLoading = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M12.596 12.596C11.687 13.5049 10.5288 14.1239 9.26798 14.3747C8.00716 14.6255 6.70028 14.4968 5.51261 14.0048C4.32494 13.5129 3.30981 12.6798 2.59557 11.611C1.88134 10.5421 1.50008 9.2855 1.5 7.99998C1.50008 6.71446 1.88134 5.45783 2.59557 4.38898C3.30981 3.32013 4.32494 2.48707 5.51261 1.99513C6.70028 1.50319 8.00716 1.37447 9.26798 1.62524C10.5288 1.87602 11.687 2.49502 12.596 3.40398"/></svg>`))
var updateDownload = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M8 1.95317V10.0469M4.25 6.29688L8 10.0469L11.75 6.29688M1.5 10.0469V13.158C1.5 13.3937 1.60536 13.6198 1.79289 13.7865C1.98043 13.9532 2.23478 14.0469 2.5 14.0469H13.5C13.7652 14.0469 14.0196 13.9532 14.2071 13.7865C14.3946 13.6198 14.5 13.3937 14.5 13.158V10.0469"/></svg>`))
var settingsGear = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"><path d="m8 2-.5 2-2 .9-1.8-.5L2 7.4l1.5 1.5v2.2L2 12.6l1.7 3 1.8-.5 2 .9.5 2h4l.5-2 2-.9 1.8.5 1.7-3-1.5-1.5V8.9L18 7.4l-1.7-3-1.8.5-2-.9-.5-2z"/><circle cx="10" cy="10" r="3"/></svg>`))

// SettingsRoot's expanded trigger and compact rail share the exact physical
// connection producer. A recovered session/list snapshot cannot stand in for it.
func settingsConnectionTrigger(c *ui.Context, m Model, colors theme.Colors, compact bool) {
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
	if !reduced && (opacity < 1 || status == "connecting" || status == "installing") {
		c.After(16 * time.Millisecond)
	}
	for _, deadline := range []time.Time{state.holdUntil, state.recoveredUntil, state.leaveUntil} {
		if deadline.After(c.Now()) {
			c.After(deadline.Sub(c.Now()))
		}
	}
	row.Children(func() {
		button := ui.Button(c, "").FillWidth().Height(42).Padding(0, 8).Radius(8).BorderWidth(0).Background(ui.Transparent).Label(i18n.T(i18n.KeySettingsOpen)).Tooltip(i18n.T(i18n.KeySettingsOpen)).Disabled(m.OnOpenSettings == nil)
		if compact {
			button.Size(36, 36).Absolute().Bottom(0).Padding(0)
		}
		if button.Hovered() {
			button.Background(chrome.Color(colors.Hover))
		}
		button.Children(func() {
			ui.Row(c).Fill().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, settingsGear).Size(20, 20).TextColor(chrome.Color(colors.FG))
				if !compact {
					label := ui.Text(c, i18n.T(i18n.KeySettingsOpen)).Grow(1).SingleLine().FontSize(theme.InterfaceSize(14)).TextColor(chrome.Color(colors.FG))
					if status != "" {
						label.Margin(0, 44, 0, 0)
					}
				}
			})
		})
		if button.Clicked() && m.OnOpenSettings != nil {
			m.OnOpenSettings()
		}
		if status == "" {
			return
		}
		seat := ui.Box(c).Size(36, 36).Absolute().Right(0).Top(4).Opacity(opacity)
		if compact {
			seat.Top(0)
		}
		seat.Children(func() {
			if status == "installing" || len(status) > 7 && status[:7] == "update:" {
				phase := m.DesktopUpdatePhase
				if status == "installing" {
					phase = "installing"
				}
				desktopUpdateSeat(c, m, phase, colors, reduced)
			} else {
				connectionStatusSeat(c, m, status, colors, reduced)
			}
		})
	})
}

func connectionStatusSeat(c *ui.Context, m Model, status string, colors theme.Colors, reduced bool) {
	dark := theme.ResolveWithSystem(m.ThemeID, m.SystemDark).Scheme == theme.SchemeDark
	ink, fill := ui.RGB(221, 134, 41), ui.RGB(254, 245, 231)
	if dark {
		fill = ui.RGB(39, 36, 31)
	}
	label, action := sidebarConnectionCopy("连接异常，刷新重试", "Disconnected"), sidebarConnectionCopy("连接异常，点击立即重连", "Disconnected, reconnect now")
	if status == "connecting" {
		label, action = sidebarConnectionCopy("重新连接中", "Reconnecting"), sidebarConnectionCopy("连接中断，正在重试，点击立即重连", "Reconnecting, reconnect now")
	}
	if status == "recovered" {
		ink, fill = ui.RGB(34, 197, 94), ui.RGB(230, 250, 237)
		if dark {
			fill = ui.RGB(35, 60, 44)
		}
		ui.Box(c).Size(36, 36).Radius(8).Background(fill).Border(1, ink.Alpha(.2)).Role(ui.RoleStatus).Label(sidebarConnectionCopy("连接成功", "Connected")).Tooltip(sidebarConnectionCopy("连接成功", "Connected")).Center().Children(func() { ui.Icon(c, connectionCheck).Size(14, 14).TextColor(ink) })
		return
	}
	button := ui.Button(c, "").Size(36, 36).Padding(0).Radius(8).Background(fill).Border(1, ink.Alpha(.2)).Label(action).Tooltip(label).Disabled(m.OnReconnect == nil)
	if button.Hovered() {
		button.Background(ui.RGB(uint8(float32(fill.R)*.94+float32(ink.R)*.06), uint8(float32(fill.G)*.94+float32(ink.G)*.06), uint8(float32(fill.B)*.94+float32(ink.B)*.06)))
	}
	button.Children(func() {
		if status == "connecting" {
			ui.Box(c).Size(14, 14).Draw(func(p *ui.Painter, r ui.Rect) {
				color := chrome.Color(colors.Muted)
				p.Stroke(ui.Rect{X: r.X + 1.5, Y: r.Y + 1.5, W: r.W - 3, H: r.H - 3}, color.Alpha(.25), 7, 1.2)
				t := float64(p.Now().UnixNano()%int64(1500*time.Millisecond)) / float64(1500*time.Millisecond)
				if reduced {
					t = .5
				}
				breath := (1 - math.Cos(t*2*math.Pi)) / 2
				start := t*2*math.Pi + 6/9.5*breath
				length := (12 + 12*breath) / 9.5
				if reduced {
					start = 3 / 9.5
					length = 18 / 9.5
				}
				var path ui.Path
				for i := 0; i <= 20; i++ {
					a := start + length*float64(i)/20
					x, y := r.X+7+float32(math.Cos(a))*5.5, r.Y+7+float32(math.Sin(a))*5.5
					if i == 0 {
						path.MoveTo(x, y)
					} else {
						path.LineTo(x, y)
					}
				}
				p.StrokePath(&path, 1.2, color)
				if !reduced {
					p.AnimationFrame()
				}
			})
		} else {
			ui.Icon(c, connectionRefresh).Size(14, 14).TextColor(ink)
		}
	})
	if button.Clicked() && m.OnReconnect != nil {
		m.OnReconnect()
	}
}

// Phase and detail come from the existing native updater's actual progress;
// opening an actionable status reuses its existing launcher management page.
func desktopUpdateSeat(c *ui.Context, m Model, phase string, colors theme.Colors, reduced bool) {
	labels := map[string]string{"checking": sidebarConnectionCopy("正在检查更新…", "Checking for updates…"), "available": sidebarConnectionCopy("有可用更新", "Update available"), "downloading": sidebarConnectionCopy("正在下载更新…", "Downloading update…"), "verifying": sidebarConnectionCopy("正在校验更新…", "Verifying update…"), "installing": sidebarConnectionCopy("正在准备重启…", "Preparing to restart…"), "ready": sidebarConnectionCopy("安装并重启", "Install and Restart"), "error": sidebarConnectionCopy("重试更新", "Retry update")}
	label := labels[phase]
	if label == "" {
		return
	}
	detail := m.DesktopUpdateDetail
	if detail == "" {
		detail = label
	}
	busy := phase == "checking" || phase == "downloading" || phase == "verifying" || phase == "installing"
	ink := chrome.Color(colors.Accent)
	button := ui.Button(c, "").Size(36, 36).Padding(0).Radius(8).Background(ink.Alpha(.08)).Border(1, ink.Alpha(.2)).TextColor(ink).Label(label).Tooltip(detail)
	if busy {
		button.Disabled(true)
	} else if button.Hovered() {
		button.Background(ink.Alpha(.14))
	}
	button.Children(func() {
		if phase == "error" {
			ui.Box(c).Size(6, 6).Radius(3).Background(ink)
		} else if busy {
			angle := float32(0)
			if !reduced {
				angle = float32(c.Now().UnixNano()%int64(time.Second)) / float32(time.Second) * 360
				c.After(16 * time.Millisecond)
			}
			ui.Icon(c, updateLoading).Size(14, 14).Rotate(angle).TextColor(ink)
		} else {
			ui.Icon(c, updateDownload).Size(14, 14).TextColor(ink)
		}
	})
	if !busy && button.Clicked() && m.OnOpenDesktopUpdate != nil {
		m.OnOpenDesktopUpdate()
	}
}

func sidebarConnectionCopy(zh, en string) string {
	if i18n.Locale() == "zh" {
		return zh
	}
	return en
}
