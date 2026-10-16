package browserpreview

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// PDFReadingPosition contains only viewer coordinates, never a capability URL.
// The product-owned reader publishes real scroll, visible-page, zoom and text
// selection state through same-document URL changes, without native bindings.
type PDFReadingPosition struct {
	Page      int    `json:"page"`
	Zoom      int    `json:"zoom"`
	Left      int    `json:"left,omitempty"`
	Top       int    `json:"top,omitempty"`
	FitWidth  bool   `json:"fitWidth,omitempty"`
	Selection [4]int `json:"selection,omitempty"`
}

func (p PDFReadingPosition) normalized() PDFReadingPosition {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Zoom < 25 || p.Zoom > 500 {
		p.Zoom = 100
	}
	p.Left = min(1000000000, max(0, p.Left))
	p.Top = min(1000000000, max(0, p.Top))
	if p.Page > 5000 {
		p.Page = 5000
	}
	if !validPDFSelection(p.Selection) {
		p.Selection = [4]int{}
	}
	return p
}

func (t *Tab) pdfViewerURL(base string) string {
	if t.pdfPosition == nil {
		return base
	}
	p := t.pdfPosition.normalized()
	raw := fmt.Sprintf("%s#page=%d&zoom=%d,%d,%d&fit=%d", base, p.Page, p.Zoom, p.Left, p.Top, map[bool]int{false: 0, true: 1}[p.FitWidth])
	if validPDFSelection(p.Selection) {
		raw += fmt.Sprintf("&selection=%d,%d,%d,%d", p.Selection[0], p.Selection[1], p.Selection[2], p.Selection[3])
	}
	return raw
}

func pdfPositionFromURL(base, raw string) (PDFReadingPosition, bool) {
	p := PDFReadingPosition{Page: 1, Zoom: 100}
	parts := strings.SplitN(raw, "#", 2)
	if len(raw) > len(base)+1025 || len(parts) != 2 || parts[0] != base {
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
	p.FitWidth = values.Get("fit") == "1"
	if selection := values.Get("selection"); selection != "" {
		coords := strings.Split(selection, ",")
		if len(coords) != 4 {
			return p, false
		}
		for i, value := range coords {
			n, e := strconv.Atoi(value)
			if e != nil {
				return p, false
			}
			p.Selection[i] = n
		}
		if !validPDFSelection(p.Selection) {
			return p, false
		}
	}
	if p.Page > 5000 || p.Left > 1000000000 || p.Top > 1000000000 {
		return p, false
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
	if p, ok := pdfPositionFromURL(t.Document.ViewerURL(), state.URL); ok {
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
	if err := t.Host.Navigate(t.pdfViewerURL(t.Document.ViewerURL())); err != nil {
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
	b.capturePDFPosition(t)
	ui.Text(c, "PDF 阅读器 · 页码、缩放、滚动与文字选择随标签保存").FontSize(12)
}

func validPDFSelection(v [4]int) bool {
	return v[0] >= 1 && v[0] <= 5000 && v[2] >= 1 && v[2] <= 5000 && v[1] >= 0 && v[1] <= 1000000000 && v[3] >= 0 && v[3] <= 1000000000
}

// The visible floating reader is authoritative while it borrows this tab.
func (b *Controller) capturePDFPosition(t *Tab) {
	if f := b.floatingDocument; f != nil && !f.closed && f.tab == t && f.pdf != nil && f.pdf == t.Document && f.host != nil {
		if p, ok := pdfPositionFromURL(f.pdf.ViewerURL(), f.host.State().URL); ok {
			t.pdfPosition = &p
			return
		}
	}
	t.capturePDFPosition()
}
