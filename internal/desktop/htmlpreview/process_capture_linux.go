//go:build linux && (amd64 || arm64)

package htmlpreview

import (
	"context"
	"errors"
	"github.com/egoist/mygo"
	"whaleisle/internal/ui/theme"
)

// CaptureProcessDocument is exclusively for the separate namespace-contained
// helper. It does not enable captureNative or change in-process readiness.
func CaptureProcessDocument(ctx context.Context, w *mygo.Window, html string, width int, appearance string) (CaptureResult, error) {
	if _, err := ProcessDocumentShell(html, width, appearance); err != nil {
		return CaptureResult{}, err
	}
	colors := theme.Default().Dark
	if appearance == "light" {
		colors = theme.Default().Light
	}
	palette := previewPalette{Background: colors.BG, Surface: colors.Field, Text: colors.FG, Secondary: colors.Muted, Border: colors.Border, Accent: colors.Accent}
	return captureInWindow(ctx, &linuxProcessWindow{w: w}, request{HTML: html, Width: width, Appearance: appearance}, palette)
}

type linuxProcessWindow struct{ w *mygo.Window }

func (w *linuxProcessWindow) Destroy()                 {} // Namespace parent owns lifetime until result acknowledgement.
func (w *linuxProcessWindow) Resize(width, height int) { w.w.SetContentSize(width, height) }
func (w *linuxProcessWindow) Capture() ([]byte, error) { return w.w.CapturePage() }
func (w *linuxProcessWindow) Eval(ctx context.Context, code string) (any, error) {
	return w.w.Page().EvalContext(ctx, code)
}
func (w *linuxProcessWindow) LoadBlank(ctx context.Context) error {
	done := make(chan error, 1)
	deliver := func(err error) {
		select {
		case done <- err:
		default:
		}
	}
	off := w.w.Page().OnDidFinishLoad(func() { deliver(nil) })
	defer off()
	fail := w.w.Page().OnDidFailLoad(func(err *mygo.LoadError) { deliver(err) })
	defer fail()
	gone := w.w.Page().OnRenderProcessGone(func(reason string) { deliver(errors.New("preview renderer exited: " + reason)) })
	defer gone()
	if err := w.w.Page().LoadURL("about:blank"); err != nil {
		return err
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
