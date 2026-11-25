package sidebar

import (
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

const (
	sbAckDuration    = 6 * time.Second
	sbDotSize        = 10
	sbStatusPulseDur = 900 * time.Millisecond
)

func tok(c *ui.Context) *design.Tokens {
	return design.For(c)
}

func fs(size float32) float32 {
	return theme.InterfaceSize(size)
}

func sbText(c *ui.Context, label string, size float32, weight int, color ui.Color) ui.Element {
	return ui.Text(c, label).FontSize(fs(size)).FontWeight(weight).TextColor(color)
}

func sidebarDark(c *ui.Context, m Model) bool {
	if design.For(c).IsDark() {
		return true
	}
	if m.ThemeID != "" {
		switch theme.Resolve(m.ThemeID).Scheme {
		case theme.SchemeDark:
			return true
		case theme.SchemeLight:
			return false
		}
	}
	return m.SystemDark
}

// withSidebarTheme temporarily activates dark mode on contexts whose standalone
// Model requests a dark theme without calling design.Bind or tt.SetDark(true).
func withSidebarTheme(c *ui.Context, m Model) func() {
	if c == nil || design.For(c).IsDark() || !sidebarDark(c, m) {
		return func() {}
	}
	if th := c.Theme(); th != nil && !th.Dark {
		th.Dark = true
		return func() { th.Dark = false }
	}
	return func() {}
}

func sbSelectedBg(t *design.Tokens) ui.Color {
	if t.SidebarAccent.A > 0 {
		return t.SidebarAccent
	}
	return t.Accent
}

func sbHoverBg(t *design.Tokens) ui.Color {
	base := sbSelectedBg(t)
	if t.IsDark() {
		return base.Alpha(0.45)
	}
	return base.Alpha(0.5)
}

// sbActiveIndicator paints a subtle 3px primary indicator bar along the leading
// edge of an active sidebar row so active and hovered rows are unmistakably
// distinct without shifting layout.
func sbActiveIndicator(e ui.Element, t *design.Tokens) {
	col := t.Primary
	if col.A == 0 {
		col = t.SidebarPrimary
	}
	if col.A == 0 {
		return
	}
	e.DrawOver(func(p *ui.Painter, r ui.Rect) {
		h := max(float32(10), r.H-12)
		y := r.Y + (r.H-h)/2
		p.Fill(ui.Rect{X: r.X + 2, Y: y, W: 3, H: h}, col, 1.5)
	})
}

func buildActiveScheduleMap(view schedstore.ViewState) map[string][]schedstore.Entry {
	if !view.ClientReady || !view.CatalogSettled || len(view.Rows) == 0 {
		return nil
	}
	var out map[string][]schedstore.Entry
	for _, row := range view.Rows {
		if row.Origin != view.Origin || row.SessionID == "" || row.Status != "active" {
			continue
		}
		if out == nil {
			out = make(map[string][]schedstore.Entry)
		}
		out[row.SessionID] = append(out[row.SessionID], row)
	}
	for id, rows := range out {
		out[id] = schedstore.FilterEntries(rows, "", "active")
	}
	return out
}

func activeSchedulesForSession(m Model, sessionID string) []schedstore.Entry {
	if sessionID == "" || !m.Schedules.ClientReady || !m.Schedules.CatalogSettled || len(m.Schedules.Rows) == 0 {
		return nil
	}
	if m.schedulesBySession != nil {
		return m.schedulesBySession[sessionID]
	}
	return schedstore.ActiveForSession(m.Schedules, sessionID)
}

func sbPopoverSurface(e ui.Element, t *design.Tokens, radius float32) ui.Element {
	e.Background(t.Popover).Border(design.BorderWidth, t.Border).Radius(radius).TextColor(t.PopoverForeground)
	for _, s := range design.ShadowLg {
		if s.Blur > 0 || s.Spread > 0 {
			e.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
	}
	return e
}

func sbGhostIconButton(c *ui.Context, label string, size float32, disabled bool, icon *ui.SVG, iconSize float32, ink ui.Color, key string) ui.Element {
	btn := coss.Button(c, "", coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     coss.ButtonIconSM,
		Disabled: disabled,
		Label:    label,
		Key:      key,
		Content:  func() {},
	}).Size(size, size).Radius(design.RadiusMD).Padding(0).BorderWidth(0).Center()
	if label != "" {
		btn.Tooltip(label)
	}
	btn.Children(func() {
		ui.Icon(c, icon).Size(iconSize, iconSize).TextColor(ink)
	})
	return btn
}

type sbAckState struct {
	text  string
	until time.Time
}

func sbAckLine(c *ui.Context, id, text string) {
	t := tok(c)
	seat := ui.Box(c.Key("ack/"+id)).Absolute().Size(0, 0).PassThrough()
	st := ui.Local(seat, "state", func() sbAckState { return sbAckState{} })
	now := c.Now()
	if st.text != text {
		st.text = text
		if text != "" {
			st.until = now.Add(sbAckDuration)
		}
	}
	if text == "" || !now.Before(st.until) {
		return
	}
	c.After(st.until.Sub(now))
	ui.Text(c, text).FillWidth().FontSize(fs(12)).TextColor(t.MutedForeground)
}

func sbAlert(c *ui.Context, id, message, detail, dismissLabel string, onDismiss func()) {
	if message == "" {
		return
	}
	t := tok(c)
	box := ui.Column(c.Key("alert/"+id)).Label(message).FillWidth().MinWidth(0).
		Gap(design.Sp(1.5)).Padding(design.Sp(2.5), design.Sp(3)).Radius(design.RadiusMD).
		Border(design.BorderWidth, t.Destructive.Alpha(0.4)).Background(t.Destructive.Alpha(0.07))
	open := ui.Local(box, "detail", func() bool { return false })
	box.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Start).Children(func() {
			ui.Text(c, message).Grow(1).MinWidth(0).FontSize(fs(13)).LineHeight(20.0 / 13).TextColor(t.Foreground)
			if detail != "" {
				label := i18n.Pick("详情", "Details")
				if *open {
					label = i18n.Pick("收起", "Hide")
				}
				if coss.Button(c, label, coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonXS,
					Key:     id + "-alert-detail",
				}).BorderWidth(0).Padding(0, design.Sp(2)).TextColor(t.MutedForeground).Clicked() {
					*open = !*open
				}
			}
			if dismissLabel != "" && onDismiss != nil {
				if coss.Button(c, dismissLabel, coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonXS,
					Key:     id + "-alert-dismiss",
				}).BorderWidth(0).Padding(0, design.Sp(2)).TextColor(t.MutedForeground).Clicked() {
					onDismiss()
				}
			}
		})
		if *open && detail != "" {
			ui.Text(c, detail).FillWidth().MinWidth(0).FontSize(fs(12)).LineHeight(18.0 / 12).
				TextColor(t.MutedForeground).Selectable()
		}
	})
}
