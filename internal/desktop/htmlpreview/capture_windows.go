//go:build windows && (amd64 || arm64)

package htmlpreview

import (
	"context"
	"errors"
	"sync"

	"github.com/egoist/mygo"
	"whaleisle/internal/ui/theme"
)

func capturePrepared(ctx context.Context, input request) (captureResult, error) {
	// Do not silently promote compile-time API availability to a verified safe
	// renderer. This stays unavailable pending a Windows fixture runtime check.
	if !Available() {
		return captureResult{}, errors.New(UnavailableReason)
	}
	ctx, cancel := context.WithTimeout(ctx, CaptureTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return captureResult{}, err
	}
	colors := theme.Default().Dark
	if input.Appearance == "light" {
		colors = theme.Default().Light
	}
	palette := previewPalette{Background: colors.BG, Surface: colors.Field, Text: colors.FG, Secondary: colors.Muted, Border: colors.Border, Accent: colors.Accent}
	w := mygo.NewWindow(mygo.WindowOptions{
		Title: "HTML preview", Width: input.Width, Height: 800, UseContentSize: true,
		Hidden: true, SkipTaskbar: true, Frameless: true, DisableResize: true,
		Page: mygo.PageOptions{IsolatedPreview: true, DisableBindings: true, DevTools: mygo.DevToolsDisabled},
	})
	return captureInWindow(ctx, &nativeCaptureWindow{w: w}, input, palette)
}

type nativeCaptureWindow struct {
	w    *mygo.Window
	once sync.Once
}

func (w *nativeCaptureWindow) Destroy()                 { w.once.Do(w.w.Destroy) }
func (w *nativeCaptureWindow) Resize(width, height int) { w.w.SetContentSize(width, height) }
func (w *nativeCaptureWindow) Capture() ([]byte, error) { return w.w.CapturePage() }
func (w *nativeCaptureWindow) Eval(ctx context.Context, code string) (any, error) {
	return w.w.Page().EvalContext(ctx, code)
}
func (w *nativeCaptureWindow) LoadBlank(ctx context.Context) error {
	done := make(chan error, 1)
	deliver := func(err error) {
		select {
		case done <- err:
		default:
		}
	}
	offLoad := w.w.Page().OnDidFinishLoad(func() { deliver(nil) })
	defer offLoad()
	offFail := w.w.Page().OnDidFailLoad(func(err *mygo.LoadError) { deliver(err) })
	defer offFail()
	offGone := w.w.Page().OnRenderProcessGone(func(reason string) { deliver(errors.New("Preview renderer exited: " + reason)) })
	defer offGone()
	w.w.Page().LoadURL("about:blank")
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
