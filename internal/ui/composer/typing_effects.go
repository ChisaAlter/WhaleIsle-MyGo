package composer

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
)

type typingStyle struct {
	Effect      string  `json:"effect"`
	Cursor      string  `json:"cursor"`
	CursorBlink bool    `json:"cursorBlink"`
	Speed       float64 `json:"speed"`
	Colors      struct {
		Kind  string `json:"kind"`
		ID    string `json:"id"`
		Echo  string `json:"echo"`
		Caret string `json:"caret"`
		Text  string `json:"text"`
	} `json:"colors"`
}

type typingEcho struct {
	text  string
	caret ui.Rect
	born  time.Time
}

type typingEffects struct {
	enabled         bool
	reduced         bool
	style           typingStyle
	echoes          []typingEcho
	compositionBase string
	composing       bool
}

func (client *DraftClient) configureTyping(m Model, reduced bool) {
	style := typingStyle{Effect: "drop", Cursor: "native", CursorBlink: true, Speed: 100}
	style.Colors.Kind = "theme"
	_ = json.Unmarshal(m.TypingFxStyle, &style)
	client.typing.enabled, client.typing.reduced, client.typing.style = m.TypingFx, reduced, style
	if !m.TypingFx || reduced {
		client.typing.echoes = nil
	}
}

// Only the native text-service adapter emits keystrokes. Clipboard, suggestions,
// voice insertion, chip replacement and undo retain the original DraftClient.
type typingInputClient struct{ *DraftClient }

func (input typingInputClient) ReplaceText(target *ui.TextInputRange, text string) {
	client := input.DraftClient
	before := client.text()
	if client.typing.composing {
		before = client.typing.compositionBase
	}
	client.ReplaceText(target, text)
	client.typing.composing = false
	client.captureTyping(before, client.text())
}

func (input typingInputClient) SetMarkedText(target *ui.TextInputRange, text string, selected ui.TextInputRange) {
	client := input.DraftClient
	if !client.typing.composing {
		client.typing.compositionBase = client.text()
		client.typing.composing = true
	}
	client.SetMarkedText(target, text, selected)
}

func (input typingInputClient) UnmarkText() {
	client := input.DraftClient
	client.UnmarkText()
	if client.typing.composing {
		client.captureTyping(client.typing.compositionBase, client.text())
		client.typing.composing = false
	}
}

func (client *DraftClient) captureTyping(before, after string) {
	fx := &client.typing
	if !fx.enabled || fx.reduced || ui.UTF16Len(after) <= ui.UTF16Len(before) {
		return
	}
	previous, next := []rune(before), []rune(after)
	prefix, suffix := 0, 0
	for prefix < len(previous) && prefix < len(next) && previous[prefix] == next[prefix] {
		prefix++
	}
	for suffix < len(previous)-prefix && suffix < len(next)-prefix && previous[len(previous)-1-suffix] == next[len(next)-1-suffix] {
		suffix++
	}
	inserted := string(next[prefix : len(next)-suffix])
	if strings.TrimSpace(inserted) == "" || ui.UTF16Len(inserted) > 16 {
		return
	}
	fx.echoes = append(fx.echoes, typingEcho{text: inserted, caret: client.visualCaret(client.selection.Caret()), born: time.Now()})
	if len(fx.echoes) > 24 {
		fx.echoes = fx.echoes[len(fx.echoes)-24:]
	}
}

var typingPalettes = map[string][3]string{
	"ocean":  {"#7dd3fc", "#38bdf8", "#0284c7"},
	"sunset": {"#fdba74", "#fb923c", "#ea580c"},
	"candy":  {"#f9a8d4", "#ec4899", "#db2777"},
	"forest": {"#86efac", "#34d399", "#059669"},
	"violet": {"#c4b5fd", "#8b5cf6", "#7c3aed"},
	"ember":  {"#fcd34d", "#f59e0b", "#d97706"},
}

func (fx typingEffects) colors(text, caret ui.Color) (ui.Color, ui.Color, ui.Color) {
	if !fx.enabled || fx.style.Colors.Kind == "theme" {
		return text, caret, text
	}
	values := [3]string{fx.style.Colors.Echo, fx.style.Colors.Caret, fx.style.Colors.Text}
	if fx.style.Colors.Kind == "preset" {
		values = typingPalettes[fx.style.Colors.ID]
	}
	return chrome.Color(values[0]), chrome.Color(values[1]), chrome.Color(values[2])
}

func (client *DraftClient) paintTyping(p *ui.Painter, r ui.Rect, textInk, caretInk ui.Color, focused bool) {
	fx := &client.typing
	echoInk, caretInk, _ := fx.colors(textInk, caretInk)
	now := p.Now()
	p.Clip(r, 0, func() {
		alive := fx.echoes[:0]
		for _, echo := range fx.echoes {
			progress := float32(now.Sub(echo.born).Seconds() / (0.4 * 100 / fx.style.Speed))
			if progress >= 1 {
				continue
			}
			alive = append(alive, echo)
			p.AnimationFrame()
			alpha, offset, scale := float32(1-progress), float32(0), float32(1)
			switch fx.style.Effect {
			case "rise":
				alpha, offset = .9*(1-progress), -.7*client.font.Size*progress
			case "flash":
				alpha = min(1, progress/.2) * (1 - progress)
				scale = 1 + .35*(1-progress)
			default:
				alpha = min(1, progress/.35) * (1 - progress)
				travel := max(0, 1-progress/.55)
				offset, scale = -.55*client.font.Size*travel, 1+.18*travel
			}
			font := client.font
			font.Size *= scale
			layout := ui.ShapeText(echo.text, font, 0)
			width, _ := layout.Size()
			p.TextLayout(layout, r.X+draftInputLeft+echo.caret.X-width, r.Y+draftInputTop+echo.caret.Y-client.scrollY+offset, echoInk.Alpha(alpha))
		}
		fx.echoes = alive
		if !focused {
			return
		}
		caret := client.visualCaret(client.selection.Caret())
		caret.X += r.X + draftInputLeft
		caret.Y += r.Y + draftInputTop - client.scrollY
		if fx.enabled && fx.style.Cursor != "native" {
			if client.selection.Range.Start != client.selection.Range.End {
				return
			}
			if fx.style.CursorBlink && !fx.reduced {
				p.After(450 * time.Millisecond)
				if now.UnixMilli()%1000 >= 450 {
					return
				}
			}
			caret.W = .6 * client.font.Size
			if fx.style.Cursor == "underline" {
				caret.Y += caret.H - 2
				caret.H = 2
			}
		}
		p.Fill(caret, caretInk, 1)
	})
}
