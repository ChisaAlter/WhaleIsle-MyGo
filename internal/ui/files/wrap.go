package files

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/ui/theme"
)

type sourceViewState struct {
	scroll    ui.ScrollState
	viewport  float32
	caret     int
	caretText string
}

func previewPreferences(m Model) *PreviewPreferences {
	if m.Tabs == nil {
		return &PreviewPreferences{}
	}
	if m.Tabs.Preferences == nil {
		m.Tabs.Preferences = &PreviewPreferences{}
	}
	return m.Tabs.Preferences
}
func paintWrapToggle(c *ui.Context, m Model) {
	prefs := previewPreferences(m)
	caption := label("启用折行", "Wrap lines")
	if prefs.WordWrap {
		caption = label("关闭折行", "Unwrap lines")
	}
	if ui.Button(c, caption).FontSize(11).Clicked() {
		prefs.WordWrap = !prefs.WordWrap
		if m.Tabs != nil && m.Tabs.floating != nil && m.Tabs.floating.win != nil {
			m.Tabs.floating.win.Invalidate()
		}
	}
}
func sourceView(m Model, key string) *sourceViewState {
	t := selectedTab(m)
	if t == nil {
		return &sourceViewState{}
	}
	if t.sourceViews == nil {
		t.sourceViews = map[string]*sourceViewState{}
	}
	v := t.sourceViews[key]
	if v == nil {
		v = &sourceViewState{caret: -1}
		t.sourceViews[key] = v
	}
	return v
}
func sourceMeasuredWidth(m Model, text string) float32 {
	font := theme.CurrentPreferences().Code
	cacheKey := text + "\x00" + fmt.Sprint(font)
	t := selectedTab(m)
	if t != nil && t.sourceWidthValid && t.sourceWidthText == cacheKey {
		return t.sourceWidth
	}
	width, _ := ui.ShapeText(text, ui.Font{Family: theme.CurrentPreferences().Code.Family, Size: theme.CurrentPreferences().Code.Size}, 0).Size()
	if t != nil {
		t.sourceWidthText, t.sourceWidth, t.sourceWidthValid = cacheKey, width, true
	}
	return width
}

// TextArea currently scrolls vertically only. Its unwrapped native text stays
// fully selectable inside a real horizontal scroll container of measured width.
// The editor keeps the same parent/key when wrap changes, retaining native edits.
func sourceTextArea(c *ui.Context, m Model, text *string, readOnly bool, key string, fg ui.Color) ui.Element {
	view := sourceView(m, key)
	wrap := previewPreferences(m).WordWrap
	if wrap {
		view.scroll.X = 0
	}
	scroll := ui.ScrollHorizontal(c).Fill().MinWidth(0).MinHeight(0).TrackScroll(&view.scroll)
	viewport := scroll.Bounds().W
	if viewport <= 0 {
		viewport = max(240, paneWidthOf(m))
		c.Invalidate()
	}
	view.viewport = viewport
	width := viewport
	if !wrap {
		width = sourceAreaWidth(false, viewport, sourceMeasuredWidth(m, *text))
	}
	var editor ui.Element
	scroll.Children(func() {
		editor = ui.TextArea(c, text).Width(width).Shrink(0).FillHeight().MinHeight(0).Font(theme.CurrentPreferences().Code.Family).FontSize(theme.CurrentPreferences().Code.Size).TextColor(fg).Padding(8, 8, 8, 8).ReadOnly(readOnly)
		if !wrap {
			editor.NoWrap()
		}
	})
	return editor
}
func revealSourceCaret(c *ui.Context, m Model, editor ui.Element, text, key string) {
	if !editor.Focused() || editor.Composing() || previewPreferences(m).WordWrap {
		return
	}
	view := sourceView(m, key)
	_, end := editor.TextSelection()
	if view.caret == end && view.caretText == text {
		return
	}
	view.caret, view.caretText = end, text
	runes := []rune(text)
	prefix := string(runes[:min(max(end, 0), len(runes))])
	if line := strings.LastIndexAny(prefix, "\r\n"); line >= 0 {
		prefix = prefix[line+1:]
	}
	x, _ := ui.ShapeText(prefix, ui.Font{Family: theme.CurrentPreferences().Code.Family, Size: theme.CurrentPreferences().Code.Size}, 0).Size()
	x += 8
	next := view.scroll.X
	if x < next {
		next = x
	} else if x > next+view.viewport-24 {
		next = max(0, x-view.viewport+24)
	}
	if next != view.scroll.X {
		view.scroll.X = next
		c.Invalidate()
	}
}
