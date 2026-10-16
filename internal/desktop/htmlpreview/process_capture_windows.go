//go:build windows && (amd64 || arm64)

package htmlpreview

import (
	"context"

	"github.com/egoist/mygo"
	"whaleisle/internal/ui/theme"
)

// CaptureProcessDocument is only for the dedicated helper after its token/job
// handshake and blank-page initialization. The helper owns the window until
// the parent receives the result and closes the job; this grants no readiness.
func CaptureProcessDocument(ctx context.Context, w *mygo.Window, html string, width int, appearance string) (CaptureResult, error) {
	if _, err := ProcessDocumentShell(html, width, appearance); err != nil {
		return CaptureResult{}, err
	}
	colors := theme.Default().Dark
	if appearance == "light" {
		colors = theme.Default().Light
	}
	palette := previewPalette{Background: colors.BG, Surface: colors.Field, Text: colors.FG, Secondary: colors.Muted, Border: colors.Border, Accent: colors.Accent}
	return captureInWindow(ctx, processCaptureWindow{&nativeCaptureWindow{w: w}}, request{HTML: html, Width: width, Appearance: appearance}, palette)
}

type processCaptureWindow struct{ *nativeCaptureWindow }

func (w processCaptureWindow) LoadBlank(context.Context) error { return nil }
func (w processCaptureWindow) Destroy()                        {} // Parent job owns cancellation and cleanup.
