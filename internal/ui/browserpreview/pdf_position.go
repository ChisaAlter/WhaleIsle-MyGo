package browserpreview

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// PDFReadingPosition contains only viewer coordinates, never a capability URL.
// Native PDF scroll changes that do not publish a URL cannot be observed; the
// application controls below provide an explicit, persistent reading position.
type PDFReadingPosition struct {
	Page int `json:"page"`
	Zoom int `json:"zoom"`
	Left int `json:"left,omitempty"`
	Top  int `json:"top,omitempty"`
}

func (p PDFReadingPosition) normalized() PDFReadingPosition {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Zoom < 25 || p.Zoom > 500 {
		p.Zoom = 100
	}
	p.Left = max(0, p.Left)
	p.Top = max(0, p.Top)
	return p
}

func (t *Tab) pdfViewerURL(base string) string {
	if t.pdfPosition == nil {
		return base
	}
	p := t.pdfPosition.normalized()
	return fmt.Sprintf("%s#page=%d&zoom=%d,%d,%d", base, p.Page, p.Zoom, p.Left, p.Top)
}

func pdfPositionFromURL(base, raw string) (PDFReadingPosition, bool) {
	p := PDFReadingPosition{Page: 1, Zoom: 100}
	parts := strings.SplitN(raw, "#", 2)
	if len(parts) != 2 || parts[0] != base {
		return p, false
	}
	values, err := url.ParseQuery(parts[1])
	if err != nil {
		return p, false
	}
	seen := false
	if s := values.Get("page"); s != "" {
		p.Page, err = strconv.Atoi(s)
		if err != nil || p.Page < 1 {
			return p, false
		}
		seen = true
	}
	if s := values.Get("zoom"); s != "" {
		coords := strings.Split(s, ",")
		p.Zoom, err = strconv.Atoi(coords[0])
		if err != nil || p.Zoom < 25 || p.Zoom > 500 {
			return p, false
		}
		seen = true
		for i, target := range []*int{&p.Left, &p.Top} {
			if len(coords) > i+1 {
				*target, err = strconv.Atoi(coords[i+1])
				if err != nil || *target < 0 {
					return p, false
				}
			}
		}
	}
	return p, seen
}

func (t *Tab) capturePDFPosition() {
	if t.Document == nil || t.Host == nil {
		return
	}
	state := t.Host.State()
	if state.Loading {
		return
	}
	if p, ok := pdfPositionFromURL(t.Document.URL(), state.URL); ok {
		t.pdfPosition = &p
	}
}

func (b *Controller) setPDFPosition(t *Tab, p PDFReadingPosition) {
	if t.Document == nil || t.Host == nil {
		return
	}
	p = p.normalized()
	previous := t.pdfPosition
	t.pdfPosition = &p
	if err := t.Host.Navigate(t.pdfViewerURL(t.Document.URL())); err != nil {
		t.pdfPosition = previous
		t.Error = err.Error()
		return
	}
	t.pdfPageDraft = strconv.Itoa(p.Page)
	t.Error = ""
	b.update()
}

func (b *Controller) pdfReadingControls(c *ui.Context, t *Tab) {
	if t.Document == nil {
		return
	}
	t.capturePDFPosition()
	p := PDFReadingPosition{Page: 1, Zoom: 100}
	if t.pdfPosition != nil {
		p = t.pdfPosition.normalized()
	}
	if t.pdfPageDraft == "" {
		t.pdfPageDraft = strconv.Itoa(p.Page)
	}
	disabled := t.Host == nil || !t.Host.Alive()
	ui.ScrollHorizontal(c).Height(28).FillWidth().Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "上一页").Disabled(disabled || p.Page <= 1).Clicked() {
				p.Page--
				p.Left, p.Top = 0, 0
				b.setPDFPosition(t, p)
			}
			submitted := ui.TextInput(c, &t.pdfPageDraft).Width(65).Placeholder("页码").Disabled(disabled).Submitted()
			if ui.Button(c, "跳转").Disabled(disabled).Clicked() || submitted {
				page, err := strconv.Atoi(strings.TrimSpace(t.pdfPageDraft))
				if err != nil || page < 1 {
					t.Error = "请输入大于零的页码"
				} else {
					p.Page = page
					p.Left, p.Top = 0, 0
					b.setPDFPosition(t, p)
				}
			}
			if ui.Button(c, "下一页").Disabled(disabled).Clicked() {
				p.Page++
				p.Left, p.Top = 0, 0
				b.setPDFPosition(t, p)
			}
			zoom := strconv.Itoa(p.Zoom) + "%"
			if ui.Select(c, &zoom, []string{"25%", "50%", "75%", "100%", "125%", "150%", "200%", "300%", "400%", "500%"}).Disabled(disabled).Changed() {
				p.Zoom, _ = strconv.Atoi(strings.TrimSuffix(zoom, "%"))
				b.setPDFPosition(t, p)
			}
			ui.Text(c, "此处页码/缩放会恢复；PDF 内置工具栏及滚动不保证同步。")
		})
	})
}
