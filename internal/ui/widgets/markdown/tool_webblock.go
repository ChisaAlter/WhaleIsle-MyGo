package markdown

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	browserurl "github.com/nlnwa/whatwg-url/url"
	"whaleisle/internal/ui/theme"
)

type WebSource struct{ URL, Title, Snippet, PublishedAt string }
type WebData struct {
	Kind, Answer, URL string
	StatusCode        float64
	Truncated         bool
	Sources           []WebSource
}
type WebLabels struct{ NoResults, SourcesTruncated, HTTP, ContentTruncated string }

// ToolWebBlock renders the installed search/fetch result preset without a copy
// toolbar. Search citations scroll independently of the provider's Markdown answer.
func ToolWebBlock(c *ui.Context, data WebData, st Style, labels WebLabels) ui.Element {
	root := ui.Column(c.Key("web-card")).MinWidth(0).Shrink(0).Padding(12, 14, 12, 14).Radius(16).Background(st.CodeBG)
	prefs := theme.RenderPreferences()
	secondary := st.Muted
	tertiary := st.ToolCodeLanguage
	if tertiary.A == 0 {
		tertiary = secondary
	}
	root.Children(func() {
		if data.Kind == "fetch" {
			root.Gap(6)
			webResultLink(c.Key("fetch-url"), data.URL, data.URL, prefs.Code.Family, 13, 19, true, st)
			ui.Row(c.Key("meta")).MinWidth(0).Gap(12).AlignItems(ui.Start).Children(func() {
				ui.Text(c, labels.HTTP+" "+webStatusNumber(data.StatusCode)).Font(prefs.Interface.Family).FontSize(13).FixedLineHeight(20).TextColor(secondary).NoWrap().Shrink(0)
				if data.Truncated {
					ui.Text(c, labels.ContentTruncated).Font(prefs.Interface.Family).FontSize(13).FixedLineHeight(20).TextColor(tertiary).NoWrap().Shrink(0)
				}
			})
			return
		}
		if data.Answer != "" {
			ui.Column(c.Key("answer")).MinWidth(0).Shrink(0).Margin(0, 0, 8, 0).Children(func() {
				answerStyle := st
				answerStyle.DocumentLayout = true
				answerStyle.FontSize = prefs.Chat.Size
				View(c, data.Answer, answerStyle)
			})
		}
		empty := data.Answer == "" && len(data.Sources) == 0
		if empty {
			ui.Text(c, labels.NoResults).Font(prefs.Interface.Family).FontSize(13).FixedLineHeight(20).TextColor(secondary).MinWidth(0).Shrink(0)
		} else {
			// The marker owns 2.5em of the list's inherited chat font. A separate
			// column preserves the ordered marker when source text wraps or scrolls.
			ui.Scroll(c.Key("sources")).FillWidth().MinWidth(0).MaxHeight(320).Shrink(0).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
					for i, source := range data.Sources {
						ui.Row(c.Key("source:" + strconv.Itoa(i))).FillWidth().MinWidth(0).Shrink(0).AlignItems(ui.Start).Children(func() {
							ui.Text(c, strconv.Itoa(i+1)+".").Font(prefs.Chat.Family).FontSize(prefs.Chat.Size).FixedLineHeight(20).TextColor(st.FG).Width(prefs.Chat.Size*2.5).Padding(0, 5, 0, 0).TextAlign(ui.End).NoWrap().Shrink(0)
							ui.Column(c).Grow(1).MinWidth(0).Shrink(1).Children(func() {
								webResultLink(c.Key("link"), source.URL, WebLinkLabel(source.URL, source.Title), prefs.Interface.Family, 14, 20, false, st)
								pendingMargin := false
								for _, detail := range []struct {
									raw    string
									height float32
									color  ui.Color
								}{{source.Snippet, 19, secondary}, {source.PublishedAt, 20, tertiary}} {
									if detail.raw == "" {
										continue
									}
									text := webInlineText(detail.raw)
									if text == "" {
										pendingMargin = true
										continue
									}
									ui.Text(c, text).Font(prefs.Interface.Family).FontSize(13).FixedLineHeight(detail.height).TextColor(detail.color).MinWidth(0).Shrink(0).Margin(2, 0, 0, 0).Selectable()
									pendingMargin = false
								}
								// Adjacent empty block margins collapse in the installed source li.
								if pendingMargin {
									ui.Box(c).Height(0).Shrink(0).Margin(2, 0, 0, 0)
								}
							})
						})
					}
				})
			})
		}
		if data.Truncated {
			ui.Text(c, labels.SourcesTruncated).Font(prefs.Interface.Family).FontSize(13).FixedLineHeight(20).TextColor(tertiary).Margin(8, 0, 0, 0).MinWidth(0).Shrink(0)
		}
	})
	return root
}

func webStatusNumber(number float64) string {
	if number == 0 {
		return "0"
	}
	if number < 0 {
		return "-" + SearchNumber(-number)
	}
	return SearchNumber(number)
}

func webResultGlyph(raw string) *ui.SVG {
	if parsed, err := browserurl.Parse(raw); err == nil && (parsed.Scheme() == "http" || parsed.Scheme() == "https") {
		host := strings.ToLower(parsed.Hostname())
		longest := 0
		var mark *ui.SVG
		for suffix, symbol := range webSiteHosts {
			if len(suffix) > longest && (host == suffix || strings.HasSuffix(host, "."+suffix)) {
				longest = len(suffix)
				mark = webSiteMarks[symbol]
			}
		}
		if mark != nil {
			return mark
		}
	}
	return webGlobe
}

// The transparent leading span reserves the inline SVG width without indenting
// continuation lines. The glyph is drawn in that first-line seat after layout.
func webResultLink(c *ui.Context, raw, label, font string, size, height float32, breakAll bool, st Style) {
	href := WebHref(raw)
	display := webInlineText(label)
	glyphSize := size * 1.1
	glyphAdvance := glyphSize + 5
	spans := []ui.Span{}
	if href != "" {
		space, _ := c.MeasureText(0, ui.Span{Text: " ", Font: font, Size: size})
		spans = append(spans, ui.Span{Text: "\u00a0", Font: font, Size: size, LetterSpacing: glyphAdvance - space})
	}
	spans = append(spans, ui.Span{Text: display, Font: font, Size: size, Weight: 500})
	element := ui.RichText(c, spans...).Font(font).FontSize(size).FontWeight(500).FixedLineHeight(height).TextColor(st.Accent).MinWidth(0).Shrink(0)
	if breakAll {
		element.BreakAll()
	} else {
		element.AlignSelf(ui.Start)
	}
	if href == "" {
		element.Selectable()
		return
	}
	element.Label(label).Role(ui.RoleLink).Focusable().Cursor(ui.CursorPointer).Tooltip(raw).FocusRing(false)
	if externalURLActivated(element) {
		if st.OnOpenURL != nil {
			st.OnOpenURL(href)
		}
	}
	glyph := webResultGlyph(href)
	focused := element.FocusVisible()
	decorated := element.Hovered() || focused
	element.DrawOver(func(p *ui.Painter, r ui.Rect) {
		metrics := (ui.Font{Family: font, Size: size}).Metrics()
		baseline := (height-metrics.Ascent-metrics.Descent)/2 + metrics.Ascent
		p.Icon(glyph, ui.Rect{X: r.X, Y: r.Y + baseline + .25*size - glyphSize, W: glyphSize, H: glyphSize}, st.Accent)
		if focused {
			p.Stroke(ui.Rect{X: r.X - 3, Y: r.Y - 3, W: r.W + 6, H: r.H + 6}, st.Accent, 3, 2)
		}
		if decorated {
			width, _ := p.MeasureText(0, spans...)
			lines := int(math.Ceil(float64(r.H / height)))
			for line := 0; line < lines; line++ {
				run := min(r.W, max(float32(0), width-float32(line)*r.W))
				if line == 0 {
					run = max(run, glyphAdvance)
				}
				y := r.Y + float32(line)*height + baseline + 3
				for x := r.X; x < r.X+run; x += 3 {
					p.Fill(ui.Rect{X: x, Y: y, W: min(float32(1), r.X+run-x), H: 1}, st.Accent, .5)
				}
			}
		}
	})
}
