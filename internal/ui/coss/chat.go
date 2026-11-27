package coss

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// ChatSide is who wrote a message.
type ChatSide uint8

const (
	ChatIncoming ChatSide = iota
	ChatOutgoing
)

// chatText is coss's message text: 14 px at line-height 1.55.
var chatText = design.TextStyle{Size: design.TextSM.Size, LineHeight: 21.7}

// ChatBubbleOptions configure a ChatBubble.
type ChatBubbleOptions struct {
	Side ChatSide
	// Grouped continues a run of messages: no squared top corner.
	Grouped bool
	// Highlighted rings the bubble (a jump to a replied message).
	Highlighted bool
	// Solid keeps the bubble opaque without backdrop blur (e.g. Codex/Claude/Minimal).
	Solid bool
	// BubbleTint overrides the theme bubble color when glass or solid skin tinting is used.
	BubbleTint ui.Color
}

// ChatBubble is coss's ChatBubble: radius 16 with the top corner on the
// author's side squared to 8 (first in a group), 10×14 padding; outgoing on
// foreground/12, incoming on foreground/4.
func ChatBubble(c *ui.Context, o ChatBubbleOptions, fn func()) ui.Element {
	t := tok(c)
	r := design.RadiusXL2
	b := ui.Column(c).MinWidth(0).Gap(design.Sp(2)).Padding(design.Sp(2.5), design.Sp(3.5)).TextColor(t.Foreground)
	if o.Side == ChatOutgoing {
		if !o.Grouped {
			b.Radius(r, design.RadiusMD, r, r)
		} else {
			b.Radius(r)
		}
		if t.Glass && !o.Solid {
			base := o.BubbleTint
			if base.A == 0 {
				base = ui.RGB(237, 243, 254)
				if t.IsDark() {
					base = ui.RGB(44, 44, 46)
				}
			}
			b.Background(base.Alpha(0.40)).Border(design.BorderWidth*0.5, t.Border)
		} else if o.Solid && o.BubbleTint.A > 0 {
			b.Background(o.BubbleTint.Alpha(1))
		} else {
			b.Background(t.Foreground.Alpha(0.12))
		}
	} else {
		if !o.Grouped {
			b.Radius(design.RadiusMD, r, r, r)
		} else {
			b.Radius(r)
		}
		if t.Glass && !o.Solid {
			tint := ui.RGBA(252, 253, 255, 0.40)
			if t.IsDark() {
				tint = ui.RGBA(35, 35, 36, 0.40)
			}
			b.Background(tint).Border(design.BorderWidth*0.5, t.Border)
		} else {
			b.Background(t.Foreground.Alpha(0.04))
		}
	}
	if o.Highlighted {
		var sh shadowList
		sh.add(0, 0, 0, 2, t.Background)
		sh.add(0, 0, 0, 4, t.Foreground.Alpha(0.25))
		finish(b, &sh, nil)
	}
	return b.Children(fn)
}

// ChatMessageText is coss's message paragraph (14/21.7, wrapping, selectable).
func ChatMessageText(c *ui.Context, s string) ui.Element {
	return para(c, s, chatText, design.WeightNormal, tok(c).Foreground).Selectable()
}

// ChatReply is the message a reply quotes.
type ChatReply struct {
	Author, Text string
}

// ChatMessageOptions configure a ChatMessage.
type ChatMessageOptions struct {
	Side ChatSide
	// Author heads the first message of a group; Avatar names its avatar
	// (incoming only).
	Author string
	Avatar string
	// Grouped continues the previous message's group: no header, no avatar.
	Grouped bool
	Text    string
	// Content replaces Text with custom bubble content.
	Content func()
	// Footer is the 10 px muted line below (time, delivery).
	Footer string
	// Reply quotes the message this one answers.
	Reply *ChatReply
	// Actions shows the hover toolbar (reply, more) beside the bubble.
	Actions     bool
	Highlighted bool
	Key         any
}

// ChatMessageEvents report what the user did with a message this frame.
type ChatMessageEvents struct {
	Reply, More, OpenReply bool
}

// ChatMessage is coss's ChatMessage layout: a header (12 px muted medium
// author), an optional 28 DIP avatar, the bubble at most 83% wide with the
// hover action toolbar beside it, and a footer.
func ChatMessage(c *ui.Context, o ChatMessageOptions) ChatMessageEvents {
	t := tok(c)
	var ev ChatMessageEvents
	own := o.Side == ChatOutgoing
	withKey(c, o.Key)
	msg := ui.Column(c).FillWidth().Gap(design.Sp(1)).Radius(design.RadiusXL).Role(ui.RoleGroup)
	hovered := msg.Hovered() || forced.Has(StateHover)
	msg.Children(func() {
		showAvatar := !own && o.Avatar != ""
		if !o.Grouped && o.Author != "" {
			h := ui.Row(c).FillWidth().PaddingX(design.Sp(1))
			if own {
				h.Justify(ui.End)
			} else if showAvatar {
				h.Padding(0, design.Sp(1), 0, 28+design.Sp(2)+design.Sp(1))
			}
			h.Children(func() { text(c, o.Author, design.TextXS, design.WeightMedium, t.MutedForeground) })
		}
		ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(design.Sp(2)).Children(func() {
			if !own && (showAvatar || o.Grouped && o.Avatar != "") {
				ui.Box(c).Width(28).Shrink(0).Children(func() {
					if !o.Grouped {
						Avatar(c, o.Avatar, AvatarOptions{Size: 28})
					}
				})
			}
			content := ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(1))
			if own {
				content.AlignItems(ui.End)
			} else {
				content.AlignItems(ui.Start)
			}
			content.Children(func() {
				if o.Reply != nil {
					if ChatReplyReference(c, o.Reply.Author, o.Reply.Text).Clicked() {
						ev.OpenReply = true
					}
				}
				line := ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(design.Sp(2))
				if own {
					line.Reverse()
				}
				line.Children(func() {
					ui.Column(c).MinWidth(0).MaxWidthPercent(83).Shrink(1).Children(func() {
						ChatBubble(c, ChatBubbleOptions{Side: o.Side, Grouped: o.Grouped, Highlighted: o.Highlighted}, func() {
							if o.Content != nil {
								o.Content()
							} else {
								ChatMessageText(c, o.Text)
							}
						})
					})
					if o.Actions {
						r, m := ChatMessageActions(c, hovered)
						ev.Reply, ev.More = r, m
					}
				})
				if o.Footer != "" {
					ui.Row(c).Gap(design.Sp(1.5)).PaddingX(design.Sp(1)).Padding(design.Sp(0.5), design.Sp(1), 0, design.Sp(1)).Children(func() {
						text(c, o.Footer, design.TextStyle{Size: 10, LineHeight: 13.33}, design.WeightNormal, t.MutedForeground)
					})
				}
			})
		})
	})
	return ev
}

// ChatMessageActions is coss's message action toolbar: ghost 24 DIP round
// reply and more buttons, shown while the message is hovered or one of them
// has the focus (150 ms fade). It reports which was clicked.
func ChatMessageActions(c *ui.Context, visible bool) (reply, more bool) {
	bar := ui.Row(c).Shrink(0).AlignSelf(ui.Center).Padding(design.Sp(0.5)).Radius(design.RadiusFull).Role(ui.RoleToolbar).Label("Message actions")
	show := visible || bar.FocusWithin()
	a := b2f(show)
	if forced == 0 {
		a = bar.AnimateWith("coss.actions", a, design.DurationDefault, design.EaseInOut)
	}
	bar.Opacity(max(a, 0.001)).Children(func() {
		reply = roundIconButton(c, IconReply, "Reply").Clicked()
		more = roundIconButton(c, IconEllipsis, "More actions").Clicked()
	})
	return reply, more
}

func roundIconButton(c *ui.Context, i Icon, label string) ui.Element {
	return Button(c, "", ButtonOptions{Variant: ButtonGhost, Size: ButtonIconXS, Icon: i, Label: label, Key: label}).Radius(design.RadiusFull)
}

// ChatReplyReference is coss's reply reference above a bubble: a 12 px
// corner arrow, the quoted author (medium) and two lines of the quote at
// foreground/80; primary while hovered. Clicked jumps to the quote.
func ChatReplyReference(c *ui.Context, author, quote string) ui.Element {
	t := tok(c)
	b := ui.ButtonBase(c).AlignItems(ui.Start).Gap(design.Sp(1.5)).FocusRing(false).Cursor(ui.CursorPointer).MinWidth(0).Justify(ui.Start)
	s := interact(b, false)
	fg, ia := t.Foreground, float32(design.OpacityDisabled)
	if s.Has(StateHover) || s.Has(StateFocusVisible) {
		fg, ia = t.Primary, 1
	}
	if s.Has(StateFocusVisible) {
		b.Radius(design.RadiusSM)
		var sh shadowList
		sh.add(0, 0, 0, 2, fg)
		finish(b, &sh, nil)
	}
	b.Children(func() {
		icon(c, IconReply, 14, fg, true).Opacity(ia).Margin(2, 0, 0, 0)
		ui.Column(c).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
			text(c, author, design.TextXS, design.WeightMedium, fg)
			para(c, quote, design.TextXS, design.WeightNormal, t.Foreground.Alpha(0.8)).MaxLines(2)
		})
	})
	return b
}

// ChatReplyPreview shows the message being replied to above the composer:
// coss's lifted reply target (popover-colored 16 DIP bubble with the author
// label) with an outline 24 DIP round cancel button on its corner. It
// reports a cancel.
func ChatReplyPreview(c *ui.Context, label, quote string, own bool) (cancel bool) {
	t := tok(c)
	ui.Column(c).FillWidth().Gap(design.Sp(1)).Padding(0, design.Sp(4)).Children(func() {
		h := ui.Row(c).FillWidth().PaddingX(design.Sp(1))
		if own {
			h.Justify(ui.End)
		}
		h.Children(func() { text(c, label, design.TextXS, design.WeightMedium, t.MutedForeground).Role(ui.RoleStatus) })
		row := ui.Row(c).FillWidth()
		if own {
			row.Justify(ui.End)
		}
		row.Children(func() {
			b := ui.Column(c).MaxWidthPercent(83).Padding(design.Sp(2.5), design.Sp(3.5)).Background(t.Popover)
			r := design.RadiusXL2
			if own {
				b.Radius(r, design.RadiusMD, r, r)
			} else {
				b.Radius(design.RadiusMD, r, r, r)
			}
			b.Children(func() {
				para(c, quote, chatText, design.WeightNormal, t.Foreground).MaxLines(3)
				pos := ui.Box(c).Absolute().Top(-design.Sp(2.5)).Radius(design.RadiusFull).Background(t.Popover)
				if own {
					pos.Left(-design.Sp(2.5))
				} else {
					pos.Right(-design.Sp(2.5))
				}
				pos.Children(func() {
					cancel = Button(c, "", ButtonOptions{Variant: ButtonOutline, Size: ButtonIconXS, Icon: IconX, Label: "Cancel reply"}).Radius(design.RadiusFull).Clicked()
				})
			})
		})
	})
	return cancel
}

// ChatDatePill is coss's date separator: a 24 DIP round popover/90 pill of
// 12 px medium text, centered in its row. It returns the pill.
func ChatDatePill(c *ui.Context, label string) ui.Element {
	t := tok(c)
	var pill ui.Element
	ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
		pill = ui.Row(c).Height(24).PaddingX(design.Sp(2)).Radius(design.RadiusFull).Border(design.BorderWidth, t.Border).Background(t.Popover.Alpha(0.9)).
			Center().Children(func() { text(c, label, design.TextXS, design.WeightMedium, t.Foreground) })
	})
	return pill
}

// ChatUnreadDivider is coss's unread marker: 12 px medium label between
// input-colored hairlines, 16 DIPs above and below.
func ChatUnreadDivider(c *ui.Context, label string) ui.Element {
	t := tok(c)
	lineCol := t.Input
	drawLine := func(p *ui.Painter, r ui.Rect) { design.DrawHairline(p, r, false, lineCol) }
	return ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(design.Sp(3)).MarginY(design.Sp(4)).Children(func() {
		ui.Box(c).Height(1).Grow(1).Draw(drawLine)
		text(c, label, design.TextXS, design.WeightMedium, t.Foreground)
		ui.Box(c).Height(1).Grow(1).Draw(drawLine)
	})
}

// ChatTypingIndicator is coss's typing indicator: three 6 DIP dots rising
// in turn (1.4 s, 0.16 s apart) in a muted/72 bubble, and a 12 px label.
func ChatTypingIndicator(c *ui.Context, label string) ui.Element {
	t := tok(c)
	return ui.Row(c).AlignItems(ui.Center).Gap(design.Sp(2.5)).Role(ui.RoleStatus).Label(label).Children(func() {
		b := ui.Row(c).AlignItems(ui.Center).Gap(design.Sp(1)).Padding(design.Sp(3.5), design.Sp(3)).
			Radius(design.RadiusXL2, design.RadiusXL2, design.RadiusXL2, design.RadiusMD).Background(t.Muted.Alpha(0.72))
		phase := float32(0)
		if forced == 0 && !c.Preferences().ReduceMotion {
			phase = b.Loop("coss.typing", 1400*1e6, design.EaseLinear)
		}
		b.Children(func() {
			for i := 0; i < 3; i++ {
				p := phase - float32(i)*0.16/1.4
				if p < 0 {
					p++
				}
				// times [0, .22, .44, 1]: up, back, rest.
				k := float32(0)
				switch {
				case p < 0.22:
					k = p / 0.22
				case p < 0.44:
					k = 1 - (p-0.22)/0.22
				}
				if forced != 0 {
					k = 0
				}
				ui.Box(c).Key(i).Size(6, 6).Radius(design.RadiusFull).Background(t.MutedForeground).
					Opacity(0.4+0.6*k).Margin(0, 0, 0, 0).Top(-3 * k)
			}
		})
		if label != "" {
			text(c, label, design.TextXS, design.WeightNormal, t.MutedForeground)
		}
	})
}

// ChatJumpToLatest is coss's jump-to-latest button: an outline xs pill
// with a down arrow on a popover/90 backdrop.
func ChatJumpToLatest(c *ui.Context, label string) ui.Element {
	t := tok(c)
	var b ui.Element
	ui.Row(c).Radius(design.RadiusFull).Background(t.Popover.Alpha(0.9)).Shrink(0).Children(func() {
		b = Button(c, label, ButtonOptions{Variant: ButtonOutline, Size: ButtonXS, Icon: IconArrowDown}).Radius(design.RadiusFull).Gap(design.Sp(2))
	})
	return b
}

// ChatStateKind is a conversation state.
type ChatStateKind uint8

const (
	ChatStateEmpty ChatStateKind = iota
	ChatStateLoading
	ChatStateReconnecting
	ChatStateError
	ChatStateNoAccess
)

// ChatState is coss's conversation states: an Empty (min 256 high, icon on
// popover) for empty and no-access, skeleton rows for loading, and a status
// bar (muted/72 between border/60 lines, 12 px medium title and muted
// description) for reconnecting and errors. Copy for errors must come from
// internal/errcopy.
func ChatState(c *ui.Context, kind ChatStateKind, title, description string, action func()) ui.Element {
	t := tok(c)
	switch kind {
	case ChatStateLoading:
		return ui.Column(c).FillWidth().Gap(design.Sp(5)).PaddingY(design.Sp(4)).Role(ui.RoleStatus).Label(firstNonEmpty(title, "Loading")).Children(func() {
			for i, side := range []ChatSide{ChatIncoming, ChatOutgoing, ChatIncoming} {
				ui.Column(c).Key(i).FillWidth().Gap(design.Sp(1)).AlignItems(map[bool]ui.Align{true: ui.End, false: ui.Start}[side == ChatOutgoing]).Children(func() {
					Skeleton(c).Size(80, 12)
					ui.Row(c).Gap(design.Sp(2)).Children(func() {
						if side == ChatIncoming {
							Skeleton(c).Size(28, 28).Radius(design.RadiusFull)
						}
						r := design.RadiusXL2
						s := Skeleton(c).Size(224, 44)
						if side == ChatIncoming {
							s.Radius(design.RadiusMD, r, r, r)
						} else {
							s.Radius(r, design.RadiusMD, r, r)
						}
					})
					Skeleton(c).Size(36, 8).MarginX(design.Sp(1))
				})
			}
		})
	case ChatStateReconnecting, ChatStateError:
		return ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(design.Sp(3)).Padding(design.Sp(2.5), design.Sp(5)).
			Background(t.Muted.Alpha(0.72)).BorderWidth(design.BorderWidth, 0, design.BorderWidth, 0).BorderColor(t.Border.Alpha(0.6)).Role(ui.RoleStatus).Children(func() {
			ui.Column(c).Shrink(1).MinWidth(0).Children(func() {
				para(c, title, design.TextXS, design.WeightMedium, t.Foreground)
				if description != "" {
					para(c, description, design.TextXS, design.WeightNormal, t.MutedForeground).Margin(design.Sp(1), 0, 0, 0)
				}
			})
			if action != nil {
				ui.Row(c).Shrink(0).Gap(design.Sp(2)).Children(action)
			}
		})
	}
	ic := IconMessage
	if kind == ChatStateNoAccess {
		ic = IconLock
	}
	return ui.Column(c).FillWidth().MinHeight(256).AlignItems(ui.Center).Justify(ui.Center).Gap(design.Sp(4)).Padding(design.Sp(10), design.Sp(6)).Role(ui.RoleStatus).Children(func() {
		ui.Column(c).MaxWidth(384).AlignItems(ui.Center).Children(func() {
			tile := ui.Box(c).Size(36, 36).Center().Radius(design.RadiusMD).Background(t.Popover).Border(design.BorderWidth, t.Border).Margin(0, 0, design.Sp(2), 0)
			var sh shadowList
			sh.scale(design.ShadowSm, black, 0.05)
			d := deco{border: design.BorderWidth, radius: design.RadiusMD}
			d.highlight(t, 0.04, 0.06)
			finish(tile, &sh, &d)
			tile.Children(func() { icon(c, ic, design.IconMD, t.Foreground, false) })
			para(c, title, design.TextXL, design.WeightBold, t.Foreground).TextAlign(ui.Center)
			if description != "" {
				para(c, description, design.TextSM, design.WeightNormal, t.MutedForeground).TextAlign(ui.Center).Margin(design.Sp(1), 0, 0, 0)
			}
		})
		if action != nil {
			ui.Row(c).Gap(design.Sp(2)).Children(action)
		}
	})
}

// ChatMessageInfo is coss's message info view: the message on top (right
// aligned, at most 83% wide, 28 DIPs above the details) and detail sections
// 28 DIPs apart below.
func ChatMessageInfo(c *ui.Context, message func(), details ...func()) ui.Element {
	return ui.Column(c).FillWidth().Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.End).Gap(design.Sp(1)).Margin(0, 0, design.Sp(7), 0).Children(func() {
			ui.Column(c).MinWidth(0).MaxWidthPercent(83).Children(message)
		})
		ui.Column(c).FillWidth().Gap(design.Sp(7)).Children(func() {
			for i, d := range details {
				ui.Column(c).Key(i).FillWidth().Children(d)
			}
		})
	})
}
