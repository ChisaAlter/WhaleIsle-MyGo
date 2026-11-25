package composer

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
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
	caretBorn       time.Time
}

var (
	typingStyleCacheMu    sync.RWMutex
	typingStyleCacheValid bool
	typingStyleCacheRaw   []byte
	typingStyleCacheVal   typingStyle
	defaultTypingStyle    = func() typingStyle {
		style := typingStyle{Effect: "drop", Cursor: "native", CursorBlink: true, Speed: 100}
		style.Colors.Kind = "theme"
		return style
	}()
)

func (client *DraftClient) configureTyping(m Model, reduced bool) {
	if !m.TypingFx || reduced {
		client.typing.enabled = m.TypingFx
		client.typing.reduced = reduced
		client.typing.caretBorn = time.Time{}
		client.typing.echoes = nil
		return
	}
	style := readTypingStyle(m.TypingFxStyle)
	if client.typing.style.CursorBlink != style.CursorBlink || (client.typing.style.Cursor == "native") != (style.Cursor == "native") {
		client.typing.caretBorn = time.Time{}
	}
	if client.typing.style.Effect != style.Effect {
		client.typing.echoes = nil
	}
	client.typing.enabled, client.typing.reduced, client.typing.style = m.TypingFx, reduced, style
}

func readTypingStyle(raw json.RawMessage) typingStyle {
	if len(raw) == 0 {
		return defaultTypingStyle
	}
	typingStyleCacheMu.RLock()
	if typingStyleCacheValid && bytes.Equal(typingStyleCacheRaw, raw) {
		hit := typingStyleCacheVal
		typingStyleCacheMu.RUnlock()
		return hit
	}
	typingStyleCacheMu.RUnlock()

	style := defaultTypingStyle
	_ = json.Unmarshal(raw, &style)
	if style.Effect != "drop" && style.Effect != "rise" && style.Effect != "flash" {
		style.Effect = "drop"
	}
	if style.Cursor != "native" && style.Cursor != "block" && style.Cursor != "underline" {
		style.Cursor = "native"
	}
	style.Speed = min(240, max(40, style.Speed))
	valid := style.Colors.Kind == "theme" || style.Colors.Kind == "preset" && typingPalettes[style.Colors.ID] != ([3]ui.Color{}) || style.Colors.Kind == "custom" && effectHexColor(style.Colors.Echo) && effectHexColor(style.Colors.Caret) && effectHexColor(style.Colors.Text)
	if !valid {
		style.Colors.Kind = "theme"
	}
	style.Colors.Echo = strings.ToLower(style.Colors.Echo)
	style.Colors.Caret = strings.ToLower(style.Colors.Caret)
	style.Colors.Text = strings.ToLower(style.Colors.Text)

	typingStyleCacheMu.Lock()
	typingStyleCacheRaw = append(typingStyleCacheRaw[:0], raw...)
	typingStyleCacheVal = style
	typingStyleCacheValid = true
	typingStyleCacheMu.Unlock()
	return style
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
	client.nativeTyping = true
	client.ReplaceText(target, text)
	client.nativeTyping = false
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
	caret := client.visualCaret(client.selection.Caret())
	caret.Y -= client.scrollY // source absolute echo keeps its spawn-time host coordinates
	fx.echoes = append(fx.echoes, typingEcho{text: inserted, caret: caret, born: time.Now()})
	if len(fx.echoes) > 24 {
		fx.echoes = fx.echoes[len(fx.echoes)-24:]
	}
}

var typingPalettes = map[string][3]ui.Color{
	"ocean":  {design.TW.Sky300, design.TW.Sky400, design.TW.Sky600},
	"sunset": {design.TW.Orange300, design.TW.Orange400, design.TW.Orange600},
	"candy":  {design.TW.Pink300, design.TW.Pink500, design.TW.Pink600},
	"forest": {design.TW.Green300, design.TW.Emerald400, design.TW.Emerald600},
	"violet": {design.TW.Violet300, design.TW.Violet500, design.TW.Violet600},
	"ember":  {design.TW.Amber300, design.TW.Amber500, design.TW.Amber600},
}

func TypingColorScheme(id string) [3]string {
	swatches, ok := typingPalettes[id]
	if !ok {
		return [3]string{}
	}
	return [3]string{formatEffectHex(swatches[0]), formatEffectHex(swatches[1]), formatEffectHex(swatches[2])}
}

func (fx typingEffects) colors(text, caret ui.Color) (ui.Color, ui.Color, ui.Color) {
	if !fx.enabled || fx.style.Colors.Kind == "theme" {
		return text, caret, text
	}
	if fx.style.Colors.Kind == "preset" {
		values := typingPalettes[fx.style.Colors.ID]
		return values[0], values[1], values[2]
	}
	return theme.Color(fx.style.Colors.Echo), theme.Color(fx.style.Colors.Caret), theme.Color(fx.style.Colors.Text)
}

func (client *DraftClient) paintTyping(p *ui.Painter, r ui.Rect, textInk, caretInk ui.Color, focused bool) {
	fx := &client.typing
	echoInk, caretInk, _ := fx.colors(textInk, caretInk)
	now := p.Now()
	p.Clip(r, 0, func() {
		alive := fx.echoes[:0]
		for _, echo := range fx.echoes {
			progress := float64(now.Sub(echo.born)) / float64(typingDuration(fx.style.Speed))
			if progress >= 1 {
				continue
			}
			alive = append(alive, echo)
			p.AnimationFrame()
			alpha, offsetEm, scale := typingKeyframe(fx.style.Effect, progress)
			offset := offsetEm * client.font.Size
			// The outer echo is right-aligned at the caret before the inner
			// glyph scales about its center. Keep that unscaled anchor stable.
			baseLayout := ui.ShapeText(echo.text, client.font, 0)
			baseWidth, baseHeight := baseLayout.Size()
			font := client.font
			font.Size *= scale
			layout := ui.ShapeText(echo.text, font, 0)
			width, height := layout.Size()
			p.TextLayout(layout, r.X+draftInputLeft+echo.caret.X-baseWidth+(baseWidth-width)/2, r.Y+draftInputTop+echo.caret.Y+offset+(baseHeight-height)/2, echoInk.Alpha(alpha))
		}
		fx.echoes = alive
		if !focused {
			fx.caretBorn = time.Time{}
			return
		}
		caret := client.visualCaret(client.selection.Caret())
		left, _, top := client.ins()
		caret.X += r.X + left
		caret.Y += r.Y + top - client.scrollY
		if fx.enabled && fx.style.Cursor != "native" {
			if client.selection.Range.Start != client.selection.Range.End {
				fx.caretBorn = time.Time{}
				return
			}
			if fx.style.CursorBlink && !fx.reduced {
				if fx.caretBorn.IsZero() {
					fx.caretBorn = now
				}
				visible, next := typingCaretBlink(now.Sub(fx.caretBorn))
				p.After(next)
				if !visible {
					return
				}
			}
			caret.W = .6 * client.font.Size
			if fx.style.Cursor == "underline" {
				caret.Y += caret.H - 2
				caret.H = 2
			}
		}
		radius := float32(1)
		if fx.enabled && fx.style.Cursor == "underline" {
			radius = 0
		}
		p.Fill(caret, caretInk, radius)
	})
}
