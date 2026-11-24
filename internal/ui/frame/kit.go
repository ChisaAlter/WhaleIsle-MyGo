package frame

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// frameStatusDuration matches the 6s status window used for inline ack lines.
const frameStatusDuration = 6 * time.Second

func tk(c *ui.Context) *design.Tokens { return design.For(c) }

func fs(s design.TextStyle) float32 { return theme.InterfaceSize(s.Size) }

func lh(s design.TextStyle) float32 { return theme.InterfaceSize(s.LineHeight) }

func frameText(c *ui.Context, text string, style design.TextStyle, weight int, col ui.Color) ui.Element {
	return ui.Text(c, text).FontSize(fs(style)).FixedLineHeight(lh(style)).FontWeight(weight).TextColor(col)
}

func frameLabel(c *ui.Context, text string, style design.TextStyle, weight int, col ui.Color) ui.Element {
	return ui.Text(c, text).FontSize(fs(style)).FixedLineHeight(lh(style)).FontWeight(weight).TextColor(col).NoWrap()
}

func frameInsetBg(t *design.Tokens) ui.Color {
	if t.IsDark() {
		return t.Input.Alpha(0.32)
	}
	return t.Muted
}

func frameShadow(e ui.Element, sc design.Shadows) ui.Element {
	if len(sc) > 0 {
		s := sc[0]
		e.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	return e
}

func framePopoverPanel(p ui.Element, t *design.Tokens, radius float32) ui.Element {
	frameShadow(p.
		Radius(radius).
		Border(design.BorderWidth, t.Border).
		Background(t.Popover), design.ShadowLg)
	return p.Transition(ui.ElementTransition{
		Duration: design.DurationDefault,
		Ease:     design.EaseInOut,
		Position: true,
	})
}

type frameAckSeen struct {
	text  string
	until time.Time
}

func frameAckLive(c *ui.Context, id, text string, d time.Duration) bool {
	seat := ui.Box(c.Key("ack/"+id)).Absolute().Size(0, 0).PassThrough()
	st := ui.Local(seat, "state", func() frameAckSeen { return frameAckSeen{} })
	now := time.Now()
	if st.text != text {
		st.text = text
		if text != "" {
			st.until = now.Add(d)
		}
	}
	if text == "" || !now.Before(st.until) {
		return false
	}
	c.After(st.until.Sub(now))
	return true
}

func frameAckLine(c *ui.Context, id, text string) {
	if !frameAckLive(c, id, text, frameStatusDuration) {
		return
	}
	t := tk(c)
	frameText(c, text, design.TextXS, design.WeightNormal, t.MutedForeground).FillWidth()
}

type frameAlertOptions struct {
	ID        string
	Title     string
	Message   string
	Detail    string
	Action    string
	OnAction  func()
	Dismiss   string
	OnDismiss func()
}

func frameErrorAlert(c *ui.Context, o frameAlertOptions) {
	if o.Message == "" {
		return
	}
	anchor := ui.Column(c.Key("frame-alert/" + o.ID)).FillWidth().MinWidth(0).Gap(design.Sp(1.5))
	open := ui.Local(anchor, "detail", func() bool { return false })
	anchor.Children(func() {
		var action func()
		hasExtra := o.Detail != "" || (o.Action != "" && o.OnAction != nil) || (o.Dismiss != "" && o.OnDismiss != nil)
		if hasExtra {
			action = func() {
				if o.Detail != "" {
					label := i18n.Pick("详情", "Details")
					if *open {
						label = i18n.Pick("收起", "Hide")
					}
					if coss.Button(c, label, coss.ButtonOptions{
						Variant: coss.ButtonGhost,
						Size:    coss.ButtonXS,
						Key:     "frame-alert-detail/" + o.ID,
					}).Clicked() {
						*open = !*open
					}
				}
				if o.Action != "" && o.OnAction != nil {
					if coss.Button(c, o.Action, coss.ButtonOptions{
						Variant: coss.ButtonOutline,
						Size:    coss.ButtonXS,
						Key:     "frame-alert-action/" + o.ID,
					}).Clicked() {
						o.OnAction()
					}
				}
				if o.Dismiss != "" && o.OnDismiss != nil {
					if coss.Button(c, o.Dismiss, coss.ButtonOptions{
						Variant: coss.ButtonGhost,
						Size:    coss.ButtonXS,
						Key:     "frame-alert-dismiss/" + o.ID,
					}).Clicked() {
						o.OnDismiss()
					}
				}
			}
		}
		coss.Alert(c, o.Title, o.Message, coss.AlertOptions{
			Variant: coss.AlertError,
			Action:  action,
		}).Label(o.Message)
		if *open && o.Detail != "" {
			frameText(c, o.Detail, design.TextXS, design.WeightNormal, tk(c).MutedForeground).FillWidth().MinWidth(0).Selectable() // verbatim-diagnostics
		}
	})
}
