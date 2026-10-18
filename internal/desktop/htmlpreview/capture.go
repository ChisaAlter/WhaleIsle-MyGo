package htmlpreview

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"math"
	"time"
)

type ConsoleMessage struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}
type CaptureResult struct {
	PNG             string           `json:"png"`
	Width           int              `json:"width"`
	ContentHeight   int              `json:"contentHeight"`
	CapturedHeight  int              `json:"capturedHeight"`
	ConsoleMessages []consoleMessage `json:"consoleMessages"`
}

// captureWindow has no filesystem, URL-opening, credential or shared-browser
// methods. Only the isolated native adapter may construct a production value.
type captureResult = CaptureResult
type consoleMessage = ConsoleMessage

type captureWindow interface {
	LoadBlank(context.Context) error
	Eval(context.Context, string) (any, error)
	Resize(int, int)
	Capture() ([]byte, error)
	Destroy()
}

type measuredState struct {
	Ready           bool             `json:"ready"`
	Height          float64          `json:"height"`
	Error           string           `json:"error"`
	ConsoleMessages []consoleMessage `json:"consoleMessages"`
	Omitted         int              `json:"omitted"`
}

// captureInWindow owns the window until termination, including timeout and
// caller cancellation. All bytes stay in memory. No local paths are resolved.
func captureInWindow(ctx context.Context, w captureWindow, input request, palette previewPalette) (captureResult, error) {
	ctx, cancel := context.WithTimeout(ctx, CaptureTimeout)
	defer cancel()
	defer w.Destroy()
	stopped := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { w.Destroy(); close(stopped) })
	defer func() {
		if !stop() {
			<-stopped
		}
	}()
	if err := ctx.Err(); err != nil {
		return captureResult{}, err
	}
	if err := w.LoadBlank(ctx); err != nil {
		return captureResult{}, err
	}
	shell, _ := json.Marshal(buildShell(input, palette))
	if _, err := w.Eval(ctx, "document.open();document.write("+string(shell)+");document.close();true"); err != nil {
		return captureResult{}, err
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	var state measuredState
	for {
		value, err := w.Eval(ctx, "window.__previewState")
		if err != nil {
			return captureResult{}, err
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return captureResult{}, err
		}
		if err = json.Unmarshal(raw, &state); err != nil {
			return captureResult{}, err
		}
		if state.Error != "" {
			return captureResult{}, errors.New(state.Error)
		}
		if state.Ready {
			break
		}
		select {
		case <-ctx.Done():
			return captureResult{}, ctx.Err()
		case <-ticker.C:
		}
	}
	if math.IsNaN(state.Height) || math.IsInf(state.Height, 0) || state.Height < 1 || state.Height > 9007199254740991 {
		return captureResult{}, errors.New("Page returned an invalid content height.")
	}
	height := int(math.Ceil(state.Height))
	captured := min(height, MaxCaptureHeight)
	w.Resize(input.Width, captured)
	if _, err := w.Eval(ctx, "new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))"); err != nil {
		return captureResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return captureResult{}, err
	}
	data, err := w.Capture()
	if err != nil {
		return captureResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return captureResult{}, err
	}
	// Validate native output before attachment and normalize device scale to the
	// requested CSS dimensions. Reject unexpectedly large images before decoding.
	if len(data) > 64*1024*1024 {
		return captureResult{}, errors.New("Capture PNG exceeds memory budget.")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return captureResult{}, err
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxWidth*4 || cfg.Height > MaxCaptureHeight*4 || int64(cfg.Width)*int64(cfg.Height) > MaxWidth*MaxCaptureHeight*4 {
		return captureResult{}, errors.New("Invalid capture dimensions.")
	}
	if cfg.Width != input.Width || cfg.Height != captured {
		src, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return captureResult{}, err
		}
		dst := image.NewNRGBA(image.Rect(0, 0, input.Width, captured))
		// Native screenshots are usually device-scale multiples. Nearest-neighbor
		// resampling keeps this memory-only path independent of external tools.
		for y := 0; y < captured; y++ {
			if err := ctx.Err(); err != nil {
				return captureResult{}, err
			}
			for x := 0; x < input.Width; x++ {
				dst.Set(x, y, src.At(x*cfg.Width/input.Width, y*cfg.Height/captured))
			}
		}
		var out bytes.Buffer
		if err := png.Encode(&out, dst); err != nil {
			return captureResult{}, err
		}
		data = out.Bytes()
	}
	if err := ctx.Err(); err != nil {
		return captureResult{}, err
	}
	if state.ConsoleMessages == nil {
		state.ConsoleMessages = []consoleMessage{}
	}
	if len(state.ConsoleMessages) > 20 {
		state.ConsoleMessages = state.ConsoleMessages[:20]
	}
	if state.Omitted > 0 {
		state.ConsoleMessages = append(state.ConsoleMessages, consoleMessage{Level: "warning", Text: "Additional console messages omitted."})
	}
	return captureResult{PNG: base64.StdEncoding.EncodeToString(data), Width: input.Width, ContentHeight: height, CapturedHeight: captured, ConsoleMessages: state.ConsoleMessages}, nil
}
