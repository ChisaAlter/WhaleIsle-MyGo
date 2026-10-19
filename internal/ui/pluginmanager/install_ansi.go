package pluginmanager

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"golang.org/x/text/width"
)

// This is a transcript projection, following ui-primitives/ansi.ts: replay CR,
// backspace and erase-in-line before folding visible lines. It does not execute
// OSC links, title changes, or terminal commands.
type installSGR struct {
	fg, bg                             ui.Color
	bold, dim, italic, reverse, hidden bool
	decorations                        string
}
type installCell struct {
	text         string
	state        installSGR
	wide, spacer bool
}

var installOSC = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)?`)

func installANSI(raw string, theme *ui.Theme) [][]ui.Span {
	raw = installOSC.ReplaceAllString(raw, "")
	state := installSGR{}
	lines := [][]ui.Span{}
	for _, line := range strings.Split(raw, "\n") {
		text := []rune(strings.TrimRight(line, "\r"))
		columns := []installCell{}
		cursor := 0
		ensure := func(index int) {
			for len(columns) <= index {
				columns = append(columns, installCell{text: " "})
			}
		}
		clear := func(index int) {
			ensure(index)
			if columns[index].spacer && index > 0 {
				columns[index-1] = installCell{text: " ", state: state}
			} else if columns[index].wide && index+1 < len(columns) && columns[index+1].spacer {
				columns[index+1] = installCell{text: " ", state: state}
			}
			columns[index] = installCell{text: " ", state: state}
		}
		for at := 0; at < len(text); at++ {
			char := text[at]
			if char == 27 {
				if at+1 >= len(text) {
					continue
				}
				if text[at+1] == ']' {
					at += 2
					for at < len(text) && text[at] != 7 && text[at] != 27 {
						at++
					}
					if at < len(text) && text[at] == 27 && at+1 < len(text) && text[at+1] == '\\' {
						at++
					}
					continue
				}
				if text[at+1] == '[' {
					end := at + 2
					for end < len(text) && text[end] >= 0x30 && text[end] <= 0x3f {
						end++
					}
					params := string(text[at+2 : end])
					for end < len(text) && text[end] >= 0x20 && text[end] <= 0x2f {
						end++
					}
					if end < len(text) && text[end] >= 0x40 && text[end] <= 0x7e {
						switch text[end] {
						case 'm':
							state = foldInstallSGR(state, params)
						case 'K':
							mode := strings.Split(params, ";")[0]
							if mode == "1" {
								for n := 0; n <= cursor; n++ {
									clear(n)
								}
							} else if mode == "2" {
								columns = nil
							} else if cursor < len(columns) {
								columns = columns[:cursor]
							}
						}
						at = end
					} else {
						at = end - 1
					}
					continue
				}
				at++
				for at < len(text) && text[at] >= 0x20 && text[at] <= 0x2f {
					at++
				}
				// A non-CSI escape consumes at most one final byte.
				if at < len(text) && (text[at] < 0x30 || text[at] > 0x7e) {
					at--
				}
				continue
			}
			switch char {
			case '\r':
				cursor = 0
				continue
			case '\b':
				cursor = max(0, cursor-1)
				continue
			case '\t':
				stop := cursor + 8 - cursor%8
				for ; cursor < stop; cursor++ {
					ensure(cursor)
				}
				continue
			}
			if char < 32 || char == 127 {
				continue
			}
			if unicode.Is(unicode.Mn, char) || unicode.Is(unicode.Me, char) || unicode.Is(unicode.Cf, char) {
				if cursor > 0 && cursor-1 < len(columns) {
					columns[cursor-1].text += string(char)
				}
				continue
			}
			clear(cursor)
			wide := installWideRune(char)
			columns[cursor] = installCell{text: string(char), state: state, wide: wide}
			cursor++
			if wide {
				ensure(cursor)
				columns[cursor] = installCell{state: state, spacer: true}
				cursor++
			}
		}
		spans := []ui.Span{}
		var previous installSGR
		for index, cell := range columns {
			if cell.spacer && (index == 0 || !columns[index-1].wide) {
				cell.text = " "
			}
			if len(spans) > 0 && cell.state == previous {
				spans[len(spans)-1].Text += cell.text
			} else {
				span := installANSISpan(cell.state, theme)
				span.Text = cell.text
				spans = append(spans, span)
				previous = cell.state
			}
		}
		lines = append(lines, spans)
	}
	if len(lines) > 1 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func installWideRune(char rune) bool {
	kind := width.LookupRune(char).Kind()
	return kind == width.EastAsianWide || kind == width.EastAsianFullwidth || unicode.Is(unicode.Han, char) || unicode.Is(unicode.Hiragana, char) || unicode.Is(unicode.Katakana, char) || unicode.Is(unicode.Hangul, char)
}
func foldInstallSGR(state installSGR, params string) installSGR {
	codes := strings.Split(params, ";")
	for n := 0; n < len(codes); n++ {
		code, _ := strconv.Atoi(codes[n])
		switch {
		case code == 0:
			state = installSGR{}
		case code == 1:
			state.bold = true
		case code == 2:
			state.dim = true
		case code == 3:
			state.italic = true
		case code == 4 || code == 9:
			marker := string(rune('0' + code))
			if !strings.Contains(state.decorations, marker) {
				state.decorations += marker
			}
		case code == 7:
			state.reverse = true
		case code == 8:
			state.hidden = true
		case code == 22:
			state.bold = false
			state.dim = false
		case code == 23:
			state.italic = false
		case code == 24:
			state.decorations = strings.ReplaceAll(state.decorations, "4", "")
		case code == 27:
			state.reverse = false
		case code == 28:
			state.hidden = false
		case code == 29:
			state.decorations = strings.ReplaceAll(state.decorations, "9", "")
		case code == 39:
			state.fg = ui.Color{}
		case code == 49:
			state.bg = ui.Color{}
		case code >= 30 && code <= 37:
			state.fg = installPalette(code - 30)
		case code >= 40 && code <= 47:
			state.bg = installPalette(code - 40)
		case code >= 90 && code <= 97:
			state.fg = installPalette(code - 90 + 8)
		case code >= 100 && code <= 107:
			state.bg = installPalette(code - 100 + 8)
		case code == 38 || code == 48:
			color := ui.Color{}
			if n+2 < len(codes) && codes[n+1] == "5" {
				value, err := strconv.Atoi(codes[n+2])
				if err == nil && value >= 0 && value <= 255 {
					color = installPalette(value)
				}
				n += 2
			} else if n+4 < len(codes) && codes[n+1] == "2" {
				r, _ := strconv.Atoi(codes[n+2])
				g, _ := strconv.Atoi(codes[n+3])
				b, _ := strconv.Atoi(codes[n+4])
				color = ui.RGB(uint8(max(0, min(255, r))), uint8(max(0, min(255, g))), uint8(max(0, min(255, b))))
				n += 4
			}
			if code == 38 {
				state.fg = color
			} else {
				state.bg = color
			}
		}
	}
	return state
}
func installPalette(index int) ui.Color {
	basic := [16]ui.Color{ui.RGB(0, 0, 0), ui.RGB(187, 0, 0), ui.RGB(0, 187, 0), ui.RGB(187, 187, 0), ui.RGB(0, 0, 187), ui.RGB(187, 0, 187), ui.RGB(0, 187, 187), ui.RGB(187, 187, 187), ui.RGB(85, 85, 85), ui.RGB(255, 85, 85), ui.RGB(0, 255, 0), ui.RGB(255, 255, 85), ui.RGB(85, 85, 255), ui.RGB(255, 85, 255), ui.RGB(85, 255, 255), ui.RGB(255, 255, 255)}
	if index < 16 {
		return basic[index]
	}
	if index >= 232 {
		v := uint8(8 + 10*(index-232))
		return ui.RGB(v, v, v)
	}
	cube := [6]uint8{0, 95, 135, 175, 215, 255}
	index -= 16
	return ui.RGB(cube[index/36], cube[index/6%6], cube[index%6])
}
func installANSISpan(state installSGR, t *ui.Theme) ui.Span {
	fg, bg := state.fg, state.bg
	if state.reverse {
		fg, bg = bg, fg
	}
	if bg.A == 0 {
		switch fg {
		case installPalette(0), installPalette(15):
			fg = t.Text
		case installPalette(8):
			fg = t.TextMuted
		case installPalette(1), installPalette(9):
			fg = t.Danger
		case installPalette(2), installPalette(10):
			fg = t.Success
		case installPalette(3), installPalette(11):
			fg = t.Warning
		case installPalette(4):
			fg = t.Accent
		case installPalette(12):
			fg = ui.RGB(96, 165, 250)
		case installPalette(6):
			fg = ui.RGB(37, 99, 235)
		case installPalette(14):
			fg = ui.RGB(59, 130, 246)
		}
	}
	if state.hidden {
		if bg.A == 0 {
			fg = t.Surface
		} else {
			fg = bg
		}
	}
	if state.dim {
		if fg.A == 0 {
			fg = t.TextMuted
		}
		fg.A = uint8(float32(fg.A) * .7)
	}
	span := ui.Span{Color: fg, Background: bg, Italic: state.italic}
	if state.bold {
		span.Weight = 700
	}
	if strings.HasSuffix(state.decorations, "4") {
		span.Underline = true
	} else if strings.HasSuffix(state.decorations, "9") {
		span.Strikethrough = true
	}
	return span
}
