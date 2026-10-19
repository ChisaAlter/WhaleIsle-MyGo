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
	reduced := c.Preferences().ReduceMotion
	if reduced {
		opacity = 1
	}
	if status != "" {
		c.After(16 * time.Millisecond)
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
		seat.Children(func() { connectionStatusSeat(c, m, status, colors, reduced) })
	})
}

func connectionStatusSeat(c *ui.Context, m Model, status string, colors theme.Colors, reduced bool) {
	dark := theme.ResolveWithSystem(m.ThemeID, m.SystemDark).Scheme == theme.SchemeDark
	ink, fill := ui.RGB(221, 134, 41), ui.RGB(254, 245, 231)
	if dark {
		fill = ui.RGB(39, 36, 31)
	}
	label, action := sidebarConnectionCopy("连接已断开，点击重试", "Disconnected. Click to reconnect"), sidebarConnectionCopy("重新连接", "Reconnect")
	if status == "connecting" {
		label, action = sidebarConnectionCopy("正在重新连接…", "Reconnecting…"), sidebarConnectionCopy("重新开始连接", "Restart connection attempt")
	}
	if status == "recovered" {
		ink, fill = ui.RGB(34, 197, 94), ui.RGB(230, 250, 237)
		if dark {
			fill = ui.RGB(35, 60, 44)
		}
		ui.Box(c).Size(36, 36).Radius(8).Background(fill).Border(1, ink.Alpha(.2)).Role(ui.RoleStatus).Label(sidebarConnectionCopy("已重新连接", "Reconnected")).Tooltip(sidebarConnectionCopy("已重新连接", "Reconnected")).Center().Children(func() { ui.Icon(c, connectionCheck).Size(14, 14).TextColor(ink) })
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
				start := t * 2 * math.Pi
				length := .4 + .4*(1-math.Cos(t*2*math.Pi))/2
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

func sidebarConnectionCopy(zh, en string) string {
	if i18n.Locale() == "zh" {
		return zh
	}
	return en
}
