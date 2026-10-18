package htmlpreview

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"
)

// SavedCapture is an in-memory, static rendering of previously authorized HTML.
// It never checks or changes the Host's generation preference.
type SavedCapture struct {
	PNG                                  []byte
	Width, ContentHeight, CapturedHeight int
}

func RenderSaved(ctx context.Context, preparedHTML string, width int, appearance string) (SavedCapture, error) {
	if err := RequireAvailable(); err != nil {
		return SavedCapture{}, err
	}
	if !utf8.ValidString(preparedHTML) || strings.TrimSpace(preparedHTML) == "" || len(preparedHTML) > MaxPreparedHTMLBytes {
		return SavedCapture{}, errors.New("Invalid prepared HTML content")
	}
	if width < MinWidth || width > MaxWidth || (appearance != "light" && appearance != "dark") {
		return SavedCapture{}, errors.New("Invalid preview dimensions or appearance")
	}
	captured, err := capturePrepared(ctx, request{HTML: preparedHTML, Width: width, Appearance: appearance})
	if err != nil {
		return SavedCapture{}, err
	}
	data, err := base64.StdEncoding.DecodeString(captured.PNG)
	if err != nil {
		return SavedCapture{}, err
	}
	return SavedCapture{PNG: data, Width: captured.Width, ContentHeight: captured.ContentHeight, CapturedHeight: captured.CapturedHeight}, nil
}
