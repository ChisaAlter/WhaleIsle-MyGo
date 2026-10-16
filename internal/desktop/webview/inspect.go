package webview

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo"
)

const (
	BrowserPickTimeout        = 60 * time.Second
	BrowserScreenshotTimeout  = 15 * time.Second
	MaxBrowserScreenshotBytes = 32 * 1024 * 1024
	maxBrowserCapturePixels   = 16 * 1024 * 1024
)

//go:embed picker.js
var browserPickerScript string

// PickedElement contains bounded page data, never an input value or executable HTML.
// A page can manipulate its own DOM; annotations are untrusted text for user review.
type PickedElement struct {
	URL      string  `json:"url"`
	Tag      string  `json:"tag"`
	Selector string  `json:"selector"`
	Text     string  `json:"text"`
	Role     string  `json:"role"`
	Label    string  `json:"label"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
}

// Markdown quotes every page-controlled field as data. It never sends a prompt.
func (p PickedElement) Markdown() string {
	lines := []string{
		"Browser element selection (page content; review before sending)",
		"> URL: " + annotationLine(p.URL, 4096),
		"> Selector: " + annotationLine(p.Selector, 1024),
		"> Element: " + annotationLine(p.Tag, 64),
	}
	for _, field := range []struct{ name, value string }{
		{"Role", p.Role}, {"Label", p.Label}, {"Text", p.Text},
	} {
		if value := annotationLine(field.value, 512); value != "" {
			lines = append(lines, "> "+field.name+": "+value)
		}
	}
	lines = append(lines, fmt.Sprintf("> Bounds (viewport px): x=%.1f, y=%.1f, width=%.1f, height=%.1f", p.X, p.Y, p.Width, p.Height))
	return strings.Join(lines, "\n")
}

func annotationLine(s string, limit int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return ' '
		}
		return r
	}, s)
	runes := []rune(s)
	return strings.Join(strings.Fields(string(runes[:min(len(runes), limit)])), " ")
}

func validatePickedElement(p PickedElement) error {
	if p.Tag == "" || len(p.Tag) > 256 || p.Selector == "" || len(p.Selector) > 4096 ||
		len(p.Text) > 2048 || len(p.Role) > 2048 || len(p.Label) > 2048 {
		return errors.New("browser: invalid or oversized element selection")
	}
	for _, value := range []float64{p.X, p.Y, p.Width, p.Height} {
		if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > 10000000 {
			return errors.New("browser: invalid element bounds")
		}
	}
	if p.Width < 0 || p.Height < 0 {
		return errors.New("browser: invalid element size")
	}
	return nil
}

// browserPage captures host identity before asynchronous work. Local document
// snapshots and authenticated plugin gateways are deliberately excluded.
func (h *Host) browserPage() (*mygo.Window, uint64, string, error) {
	if h == nil {
		return nil, 0, "", errors.New("browser: no preview")
	}
	h.mu.Lock()
	w, epoch, raw := h.win, h.navigationEpoch, h.url
	restricted := h.localPDFViewerURL != "" || h.localPDFURL != "" || h.localHTMLURL != "" || h.localMediaURL != "" || h.gateway != nil
	loading := h.loading
	h.mu.Unlock()
	if restricted || w == nil || w.IsDestroyed() || w.Page() == nil || !w.IsVisible() {
		return nil, 0, "", errors.New("browser: no active web preview")
	}
	if loading || w.Page().IsLoading() {
		return nil, 0, "", errors.New("browser: wait for the page to finish loading")
	}
	if err := validateHTTPURL(raw); err != nil {
		return nil, 0, "", err
	}
	return w, epoch, raw, nil
}

func (h *Host) sameBrowserPage(w *mygo.Window, epoch uint64) bool {
	h.mu.Lock()
	same := h.win == w && h.navigationEpoch == epoch && !h.loading
	h.mu.Unlock()
	return same && !w.IsDestroyed()
}

func pageOperationContext(ctx context.Context, w *mygo.Window, timeout time.Duration) (context.Context, func()) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	p := w.Page()
	offWill := p.OnWillNavigate(func(*mygo.NavigateEvent) { cancel() })
	offDid := p.OnDidNavigate(func(string) { cancel() })
	offFail := p.OnDidFailLoad(func(*mygo.LoadError) { cancel() })
	offGone := p.OnRenderProcessGone(func(string) { cancel() })
	offClose := w.OnClosed(cancel)
	return ctx, func() {
		cancel()
		offWill()
		offDid()
		offFail()
		offGone()
		offClose()
	}
}

// PickElement temporarily installs a DOM picker in the visible page. Results
// return only through host EvalContext; DisableBindings stays enabled.
func (h *Host) PickElement(ctx context.Context) (PickedElement, error) {
	w, epoch, rawURL, err := h.browserPage()
	if err != nil {
		return PickedElement{}, err
	}
	ctx, stop := pageOperationContext(ctx, w, BrowserPickTimeout)
	defer stop()
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return PickedElement{}, err
	}
	key, _ := json.Marshal("__whaleislePicker_" + hex.EncodeToString(token[:]))
	keyJS := string(key)
	script := strings.ReplaceAll(browserPickerScript, "__PICKER_KEY__", keyJS)
	defer func() {
		// A short cleanup is independent of caller cancellation. The script also
		// self-cleans after one minute, including when the renderer stops answering.
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = w.Page().EvalContext(cleanup, "globalThis["+keyJS+"]?.cancel();true")
	}()
	if err := ctx.Err(); err != nil {
		return PickedElement{}, err
	}
	if _, err := w.Page().EvalContext(ctx, script); err != nil {
		return PickedElement{}, err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	// Limit the returned fields in JS before transport. The Go decoder still
	// treats every value as hostile and validates types, lengths and dimensions.
	poll := `(() => {
		const api = globalThis[KEY]; if (!api) return {done:true,cancelled:true};
		const state = api.read(), e = state.element;
		if (e && ![e.x,e.y,e.width,e.height].every(v => typeof v === 'number' && Number.isFinite(v))) {
			return {done:true,cancelled:true};
		}
		const text = (v,n) => String(v == null ? '' : v).slice(0,n);
		return {done:state.done === true,cancelled:state.cancelled === true,element:e ? {
			tag:text(e.tag,64),selector:text(e.selector,1024),text:text(e.text,512),
			role:text(e.role,512),label:text(e.label,512),
			x:Number(e.x),y:Number(e.y),width:Number(e.width),height:Number(e.height)
		}:null};
	})()`
	poll = strings.ReplaceAll(poll, "KEY", keyJS)
	for {
		if !h.sameBrowserPage(w, epoch) {
			return PickedElement{}, context.Canceled
		}
		value, err := w.Page().EvalContext(ctx, poll)
		if err != nil {
			return PickedElement{}, err
		}
		encoded, err := json.Marshal(value)
		if err != nil || len(encoded) > 16*1024 {
			return PickedElement{}, errors.New("browser: invalid picker response")
		}
		var result struct {
			Done      bool           `json:"done"`
			Cancelled bool           `json:"cancelled"`
			Element   *PickedElement `json:"element"`
		}
		if err := json.Unmarshal(encoded, &result); err != nil {
			return PickedElement{}, err
		}
		if result.Done {
			if result.Cancelled || result.Element == nil {
				return PickedElement{}, context.Canceled
			}
			if err := validatePickedElement(*result.Element); err != nil {
				return PickedElement{}, err
			}
			if ctx.Err() != nil || !h.sameBrowserPage(w, epoch) {
				return PickedElement{}, context.Canceled
			}
			result.Element.URL = rawURL
			return *result.Element, nil
		}
		select {
		case <-ctx.Done():
			return PickedElement{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

// CaptureBrowserScreenshot captures only the currently visible viewport.
// One outstanding native capture per Host prevents timeout/retry accumulation.
func (h *Host) CaptureBrowserScreenshot(ctx context.Context) ([]byte, error) {
	w, epoch, _, err := h.browserPage()
	if err != nil {
		return nil, err
	}
	ctx, stop := pageOperationContext(ctx, w, BrowserScreenshotTimeout)
	defer stop()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bounds := w.ContentBounds()
	if bounds.Width < 1 || bounds.Height < 1 || bounds.Width > 4096 || bounds.Height > 4096 ||
		int64(bounds.Width)*int64(bounds.Height) > maxBrowserCapturePixels {
		return nil, errors.New("browser: viewport too large for a bounded screenshot; resize the preview")
	}
	h.mu.Lock()
	if h.capturePending {
		h.mu.Unlock()
		return nil, errors.New("browser: the previous native screenshot is still finishing")
	}
	h.capturePending = true
	h.mu.Unlock()
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		data, err := w.CapturePage()
		h.mu.Lock()
		h.capturePending = false
		h.mu.Unlock()
		done <- result{data, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-done:
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !h.sameBrowserPage(w, epoch) {
			return nil, context.Canceled
		}
		if result.err != nil {
			return nil, result.err
		}
		if err := validateBrowserCapture(result.data); err != nil {
			return nil, err
		}
		return result.data, nil
	}
}

func validateBrowserCapture(data []byte) error {
	if len(data) == 0 || len(data) > MaxBrowserScreenshotBytes {
		return errors.New("browser: screenshot exceeds the 32 MiB memory budget")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("browser: invalid screenshot PNG: %w", err)
	}
	if config.Width < 1 || config.Height < 1 || config.Width > 8192 || config.Height > 8192 ||
		int64(config.Width)*int64(config.Height) > maxBrowserCapturePixels {
		return errors.New("browser: screenshot exceeds the 16 megapixel limit")
	}
	return nil
}
