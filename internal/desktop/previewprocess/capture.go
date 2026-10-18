package previewprocess

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image/png"
	"io"

	"whaleisle/internal/desktop/htmlpreview"
)

const maxCaptureBytes = 90 * 1024 * 1024

// CaptureExperimental renders one authorized snapshot in the existing dedicated
// helper. It never enables in-process capture or changes global readiness.
func CaptureExperimental(ctx context.Context, opt Options, doc Document) (htmlpreview.CaptureResult, error) {
	ctx, cancel := context.WithTimeout(ctx, htmlpreview.CaptureTimeout)
	defer cancel()
	doc.capture = true
	s, err := OpenExperimental(ctx, opt, doc)
	if err != nil {
		return htmlpreview.CaptureResult{}, err
	}
	result := s.capture
	closeErr := s.Close()
	if err := ctx.Err(); err != nil {
		return htmlpreview.CaptureResult{}, err
	}
	if closeErr != nil {
		return htmlpreview.CaptureResult{}, closeErr
	}
	if result == nil {
		return htmlpreview.CaptureResult{}, errors.New("preview helper returned no capture")
	}
	return *result, nil
}

func sendCapture(w io.Writer, nonce string, result htmlpreview.CaptureResult) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if len(raw) == 0 || len(raw) > maxCaptureBytes {
		return errors.New("capture result exceeds IPC limit")
	}
	if err := writePacket(w, packet{Version: protocolVersion, Kind: "capture", Nonce: nonce, Bytes: len(raw)}); err != nil {
		return err
	}
	return writeAll(w, raw)
}

func receiveCapture(r io.Reader, nonce string, width int) (htmlpreview.CaptureResult, error) {
	var result htmlpreview.CaptureResult
	p, err := readPacket(r)
	if err != nil {
		return result, err
	}
	if err := expect(p, "capture", nonce); err != nil {
		return result, err
	}
	if p.Bytes < 1 || p.Bytes > maxCaptureBytes {
		return result, errors.New("invalid capture result size")
	}
	raw := make([]byte, p.Bytes)
	if _, err := io.ReadFull(r, raw); err != nil {
		return result, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, err
	}
	if result.Width != width || result.ContentHeight < 1 || result.CapturedHeight != min(result.ContentHeight, htmlpreview.MaxCaptureHeight) || len(result.ConsoleMessages) > 21 {
		return result, errors.New("invalid capture result dimensions or console count")
	}
	data, err := base64.StdEncoding.DecodeString(result.PNG)
	if err != nil {
		return result, err
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return result, err
	}
	if cfg.Width != width || cfg.Height != result.CapturedHeight {
		return result, errors.New("capture PNG dimensions differ from metadata")
	}
	return result, nil
}
