package chat

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/ansi"
)

const (
	terminalLineHeight      = 18
	terminalHeaderMaxHeight = 168 // CSS max-height150 + padding9 on each edge.
	terminalMaxHeight       = 248 // CSS max-height224 + padding12 on each edge.
)

func terminalDescription(m *toolTerminalModel) string {
	if m.SessionID != "" {
		return fmt.Sprintf(i18n.T(i18n.KeyToolTerminalSession), m.SessionID)
	}
	return m.Description
}

func terminalCommand(m *toolTerminalModel) string {
	if m.SessionID != "" && m.Command == "" {
		return i18n.T(i18n.KeyToolTerminalSend)
	}
	return m.Command
}

func terminalExitText(m *toolTerminalModel) string {
	if m == nil || m.Background || m.Running {
		return ""
	}
	if m.HasSignal {
		return fmt.Sprintf(i18n.T(i18n.KeyToolTerminalSignal), m.Signal)
	}
	if m.MissingExit {
		return i18n.T(i18n.KeyToolTerminalNoExit)
	}
	if m.HasExit && m.ExitCode != 0 {
		return fmt.Sprintf(i18n.T(i18n.KeyToolTerminalExit), strconv.FormatFloat(m.ExitCode, 'f', -1, 64))
	}
	return ""
}

func terminalPrompt(cwd, home string) string {
	if cwd == "" {
		return "$"
	}
	trimmed := strings.TrimRight(cwd, `/\`)
	if home != "" && trimmed == strings.TrimRight(home, `/\`) {
		return "~"
	}
	if n := strings.LastIndexAny(trimmed, `/\`); n >= 0 {
		trimmed = trimmed[n+1:]
	}
	if trimmed == "" {
		return cwd
	}
	return trimmed
}

func terminalPalette(st ToolCardStyle) ansi.Palette {
	danger := st.Err
	if danger.A == 0 {
		danger = chrome.Color(st.Colors.Danger)
	}
	background := st.Field
	if st.Colors.CodeBlock != "" {
		background = chrome.Color(st.Colors.CodeBlock)
	}
	return ansi.Palette{
		Primary: st.FG, Tertiary: st.Muted, Background: background, DimBase: st.FG,
		Error: danger, ErrorSecondary: ui.RGB(242, 90, 90),
		Success: ui.RGB(34, 197, 94), SuccessSecondary: ui.RGB(78, 209, 126),
		Warning: ui.RGB(245, 158, 11), WarningSecondary: ui.RGB(247, 173, 49), Business: st.Accent,
	}
}

func terminalVisible(lines [][]ui.Span) bool {
	for _, line := range lines {
		for _, span := range line {
			if strings.TrimSpace(span.Text) != "" {
				return true
			}
		}
	}
	return false
}

// terminalCard draws only validated shell/terminal_send data. The command and
// output scroll independently; clipboard payload is the exact parsed output.
func terminalCard(c *ui.Context, st ToolCardStyle, key string, m *toolTerminalModel, home string, readOnly bool) {
	palette := terminalPalette(st)
	background, stroke, secondary := palette.Background, st.Border.Alpha(.4), st.Muted
	if st.Colors.BorderL1 != "" {
		stroke = chrome.Color(st.Colors.BorderL1)
	}
	if st.Colors.Secondary != "" {
		secondary = chrome.Color(st.Colors.Secondary)
	}
	border := float32(max(1, math.Floor(.5*float64(c.DeviceScale()))) / float64(c.DeviceScale()))
	lines := ansi.Parse(m.Output, palette)
	visible := terminalVisible(lines)
	hasBody := !m.Background && (!m.Running || visible)
	showCopy := !m.Background && !m.Running && visible
	command := terminalCommand(m)
	command = strings.TrimSuffix(command, "\n")
	commandLines := strings.Split(command, "\n")
	font := theme.RenderPreferences().Code.Family
	card := ui.Column(c.Key("terminal:"+key)).MinWidth(0).Shrink(0).Margin(4, 0, 4, 4).
		Border(border, stroke).Radius(16).Background(background).Clip().Label(i18n.T(i18n.KeyToolTerminalOutput))
	copied := ui.Local(card, "copied-until", func() time.Time { return time.Time{} })
	copying := c.Now().Before(*copied)
	if copying {
		c.After(copied.Sub(c.Now()))
	}
	card.Children(func() {
		banner := ui.Box(c.Key("banner")).FillWidth().MinWidth(0).Shrink(0).Clip().Label(i18n.T(i18n.KeyToolTerminalBanner))
		position := ui.Local(banner, "position", func() ui.ScrollState { return ui.ScrollState{} })
		status, copyText := terminalExitText(m), i18n.T(i18n.KeyChatToolCopy)
		if copying {
			copyText = i18n.T(i18n.KeyMarkdownCopied)
		}
		controlsWidth := float32(0)
		if status != "" {
			width, _ := c.MeasureText(0, ui.Span{Text: status, Font: theme.RenderPreferences().Chat.Family, Size: 12})
			controlsWidth += width + 16
		}
		if showCopy {
			width, _ := c.MeasureText(0, ui.Span{Text: copyText, Font: theme.RenderPreferences().Chat.Family, Size: 13})
			if controlsWidth > 0 {
				controlsWidth += 12
			}
			controlsWidth += width
		}
		rightPadding := float32(14)
		if controlsWidth > 0 {
			rightPadding += controlsWidth + 12
		}
		banner.Children(func() {
			ui.Scroll(c.Key("header")).TrackScroll(position).FillWidth().MinWidth(0).MaxHeight(terminalHeaderMaxHeight).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(12).Padding(9, rightPadding, 9, 30).AlignItems(ui.Start).Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Gap(0).Children(func() {
						for index, line := range commandLines {
							ui.Row(c.Key(strconv.Itoa(index))).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
								if index == 0 && (!m.Background || m.Running) {
									terminalRunState(c, m, palette).Absolute().Left(-22).Top((terminalLineHeight - terminalRunStateSize(m)) / 2)
								}
								prompt := "$"
								if index == 0 {
									prompt = terminalPrompt(m.Cwd, home)
								}
								ui.Text(c, prompt).Font(font).FontSize(11).FixedLineHeight(terminalLineHeight).TextColor(st.Muted).NoWrap().Shrink(0)
								ui.Text(c, line).Font(font).FontSize(11).FixedLineHeight(terminalLineHeight).TextColor(st.FG).
									Grow(1).MinWidth(0).Selectable().Description(fmt.Sprintf(i18n.T(i18n.KeyToolTerminalLine), index+1))
							})
						}
					})
				})
			})
			if controlsWidth > 0 {
				// CSS sticky: reserve the normal header width and retain its 9px
				// inset until scrolling passes it, then clamp controls to top0.
				ui.Row(c.Key("controls")).Absolute().Top(max(float32(0), 9-position.Y)).Right(14).Height(18).Gap(12).AlignItems(ui.Start).Children(func() {
					if status != "" {
						ui.Box(c).Shrink(0).Padding(0, 8, 0, 8).Radius(9).Background(terminalPillBackground(st)).Label(i18n.T(i18n.KeyToolTerminalStatus)).Children(func() {
							ui.Text(c, status).Font(theme.RenderPreferences().Chat.Family).FontSize(12).FixedLineHeight(18).NoWrap().TextColor(palette.Error)
						})
					}
					if showCopy {
						text := i18n.T(i18n.KeyChatToolCopy)
						if copying {
							text = i18n.T(i18n.KeyMarkdownCopied)
						}
						button := ui.ButtonBase(c.Key("copy")).Label(text).Tooltip(text).Background(background).TextColor(secondary).
							Height(18).Shrink(0).MinWidth(0).Padding(0).Font(theme.RenderPreferences().Chat.Family).FontSize(13)
						button.Children(func() { ui.Text(c, text).FontSize(13).FixedLineHeight(18).NoWrap().TextColor(secondary) })
						if button.Clicked() && !copying {
							if readOnly {
								c.Services().WriteClipboard(m.Output)
							} else {
								c.WriteClipboard(m.Output)
							}
							*copied = c.Now().Add(time.Second)
							c.Invalidate()
						}
					}
				})
			}
		})
		if hasBody {
			ui.Box(c).FillWidth().Height(border).Shrink(0).Background(st.Border)
			if !visible {
				ui.Box(c).MinWidth(0).Margin(0, 0, 0, 30).Padding(12, 14, 12, 0).Label(i18n.T(i18n.KeyToolTerminalNoOutput)).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyToolTerminalNoOutput)).Font(font).FontSize(11).FixedLineHeight(16).TextColor(st.Muted)
				})
			} else {
				ui.ScrollBoth(c.Key("output")).MinWidth(0).Margin(0, 0, 0, 30).MaxHeight(terminalMaxHeight).Label(i18n.T(i18n.KeyChatToolOutput)).Children(func() {
					spans := []ui.Span{}
					for index, line := range lines {
						spans = append(spans, line...)
						if index+1 < len(lines) {
							spans = append(spans, ui.Span{Text: "\n"})
						}
					}
					ui.RichText(c, spans...).Font(font).FontSize(11).FixedLineHeight(18).TextColor(st.FG).NoWrap().Selectable().
						Padding(12, 14, 12, 0).MinWidth(0).Shrink(0)
				})
			}
		}
	})
}

func terminalPillBackground(st ToolCardStyle) ui.Color {
	if darkSurface(st.Muted) {
		return ui.RGB(44, 44, 46)
	}
	return ui.RGB(255, 255, 255)
}

func terminalRunStateSize(m *toolTerminalModel) float32 {
	if m.Running {
		return 14
	}
	return 10
}

// The loading arc is synchronized to UI time, like the installed spinner's
// document-time origin. Reduced motion draws its static 18/150 dash.
func terminalRunState(c *ui.Context, m *toolTerminalModel, p ansi.Palette) ui.Element {
	size := terminalRunStateSize(m)
	state := i18n.T(i18n.KeyToolTerminalDone)
	ink := p.Success
	if terminalExitText(m) != "" {
		state, ink = toolRowStatusLabel("error"), p.Error
	}
	if m.Running {
		state, ink = toolRowStatusLabel("running"), p.Tertiary
	}
	el := ui.Box(c).Size(size, size).Shrink(0).Role(ui.RoleStatus).Label(state)
	reduced := c.Preferences().ReduceMotion
	el.Draw(func(painter *ui.Painter, r ui.Rect) {
		if !m.Running {
			painter.Fill(ui.Rect{X: r.X + 2, Y: r.Y + 2, W: 6, H: 6}, ink, 3)
			return
		}
		radius := r.W * 9.5 / 24
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		var ring ui.Path
		ring.Circle(cx, cy, radius)
		painter.StrokePath(&ring, r.W*2/24, ink.Alpha(.25))
		turn, dash, offset := float64(0), float64(18), float64(-3)
		if !reduced {
			phase := float64(painter.Now().UnixMilli()%1500) / 1500
			turn = phase * 2 * math.Pi
			bounce := 1 - math.Abs(phase*2-1)
			dash, offset = 12+12*bounce, -6*bounce
			painter.AnimationFrame()
		}
		start := turn - offset/9.5
		var arc ui.Path
		for n := 0; n <= 18; n++ {
			a := start + (dash/9.5)*float64(n)/18
			x, y := cx+radius*float32(math.Cos(a)), cy+radius*float32(math.Sin(a))
			if n == 0 {
				arc.MoveTo(x, y)
			} else {
				arc.LineTo(x, y)
			}
		}
		painter.StrokePath(&arc, r.W*2/24, ink)
	})
	return el
}
