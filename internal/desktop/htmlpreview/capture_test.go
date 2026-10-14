package htmlpreview

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixtureCaptureWindow struct {
	done          chan struct{}
	once          sync.Once
	block         bool
	width, height int
}

func (w *fixtureCaptureWindow) Destroy()                        { w.once.Do(func() { close(w.done) }) }
func (w *fixtureCaptureWindow) LoadBlank(context.Context) error { return nil }
func (w *fixtureCaptureWindow) Resize(width, height int)        { w.width, w.height = width, height }
func (w *fixtureCaptureWindow) Eval(_ context.Context, code string) (any, error) {
	if code == "window.__previewState" {
		return map[string]any{"ready": true, "height": 9000, "consoleMessages": []any{}}, nil
	}
	return nil, nil
}
func (w *fixtureCaptureWindow) Capture() ([]byte, error) {
	if w.block {
		<-w.done
		return nil, context.Canceled
	}
	var b bytes.Buffer
	err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, w.width, w.height)))
	return b.Bytes(), err
}
func TestCaptureFixtureLifecycle(t *testing.T) {
	input := request{HTML: "<p>fixture</p>", Width: 240, Appearance: "dark"}
	w := &fixtureCaptureWindow{done: make(chan struct{})}
	result, err := captureInWindow(context.Background(), w, input, previewPalette{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 240 || result.ContentHeight != 9000 || result.CapturedHeight != 4000 || result.PNG == "" {
		t.Fatalf("invalid dimensions or PNG: %+v", result)
	}
	select {
	case <-w.done:
	default:
		t.Fatal("window not destroyed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	blocked := &fixtureCaptureWindow{done: make(chan struct{}), block: true}
	if _, err := captureInWindow(ctx, blocked, input, previewPalette{}); !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("capture cancellation missing:", err)
	}
	select {
	case <-blocked.done:
	default:
		t.Fatal("cancelled window survived")
	}
}
func TestShellOpaqueSandboxFixture(t *testing.T) {
	shell := buildShell(request{HTML: `</script><script>/* fixture */</script>`, Width: 240, Appearance: "dark"}, previewPalette{})
	for _, required := range []string{"sandbox','allow-scripts'", "event.source!==frame.contentWindow", "event.origin!=='null'", "connect-src", "worker-src", "frame.srcdoc="} {
		if !strings.Contains(shell, required) {
			t.Fatal("missing shell boundary", required)
		}
	}
	if strings.Contains(shell, "allow-same-origin") || strings.Contains(shell, "/* fixture */</script>") {
		t.Fatal("supplied HTML escaped its encoded document")
	}
}
