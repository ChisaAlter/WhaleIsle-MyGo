//go:build (windows || darwin || linux) && (amd64 || arm64)

package browserrecording

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/storelocale"
)

//go:embed helper.js
var helperScript string

type nativeEncoder struct {
	win  *mygo.Window
	once sync.Once
}

func newEncoder(ctx context.Context) (result encoder, err error) {
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	e := &nativeEncoder{}
	// Window creation can fail before a native handle exists. Do not turn a
	// missing runtime into a crash, a shared profile fallback, or an inert button.
	defer func() {
		if r := recover(); r != nil {
			e.Close()
			result = nil
			err = errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreBrowserrecordingTheVideoEncoderWindowIsUnavailableV), r))
		}
	}()
	e.win = mygo.NewWindow(mygo.WindowOptions{
		Title: "Browser video encoder", Width: 1280, Height: 720, UseContentSize: true,
		Hidden: true, SkipTaskbar: true, Frameless: true, DisableResize: true,
		Page: mygo.PageOptions{IsolatedPreview: true, DisableBindings: true, DevTools: mygo.DevToolsDisabled},
	})
	if e.win == nil || e.win.IsDestroyed() || e.win.Page() == nil {
		e.Close()
		return nil, errors.New(storelocale.T(storelocale.KeyStoreBrowserrecordingCouldNotCreateTheVideoEncoderWindow))
	}
	initCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err = e.loadBlank(initCtx); err != nil {
		e.Close()
		return nil, errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreBrowserrecordingCouldNotStartTheIsolatedVideoEncoding), err))
	}
	value, err := e.call(initCtx, helperScript)
	if err != nil {
		e.Close()
		return nil, errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreBrowserrecordingTheSystemWebviewHasNoUsableVideoEncod), err))
	}
	if ready, ok := value.(bool); !ok || !ready {
		e.Close()
		return nil, errors.New(storelocale.T(storelocale.KeyStoreBrowserrecordingTheVideoEncoderIsNotReady))
	}
	return e, nil
}
func (e *nativeEncoder) loadBlank(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan error, 1)
	deliver := func(err error) {
		select {
		case done <- err:
		default:
		}
	}
	pg := e.win.Page()
	offLoad := pg.OnDidFinishLoad(func() { deliver(nil) })
	defer offLoad()
	offFail := pg.OnDidFailLoad(func(err *mygo.LoadError) { deliver(err) })
	defer offFail()
	offGone := pg.OnRenderProcessGone(func(string) { deliver(errors.New("encoder renderer exited")) })
	defer offGone()
	// Existing isolated policy permits only this programmatic blank navigation.
	// No HTML, resource server, file URL, browser cookie or Host session is used.
	if err := pg.LoadURL("about:blank"); err != nil {
		return err
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (e *nativeEncoder) call(ctx context.Context, code string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e.win == nil || e.win.IsDestroyed() {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreBrowserrecordingTheVideoEncoderWindowWasClosed))
	}
	timeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return e.win.Page().EvalContext(timeout, code)
}
func (e *nativeEncoder) Start(ctx context.Context, frame []byte) (string, error) {
	value, err := e.frameCall(ctx, "start", frame)
	if err != nil {
		return "", err
	}
	mime, ok := value.(string)
	if !ok || !validMIME(mime) {
		return "", errors.New(storelocale.T(storelocale.KeyStoreBrowserrecordingTheVideoEncoderReturnedAnInvalidFormat))
	}
	return mime, nil
}
func (e *nativeEncoder) Frame(ctx context.Context, frame []byte) error {
	value, err := e.frameCall(ctx, "frame", frame)
	if err != nil {
		return err
	}
	if accepted, ok := value.(bool); !ok || !accepted {
		return errors.New(storelocale.T(storelocale.KeyStoreBrowserrecordingTheVideoEncoderAcceptedNoFrames))
	}
	return nil
}
func (e *nativeEncoder) frameCall(ctx context.Context, method string, frame []byte) (any, error) {
	if err := validateFrame(frame); err != nil {
		return nil, err
	}
	// method is private and always a constant; the only variable argument is a
	// JSON-quoted base64 PNG, never page script, markup or any user-provided URL.
	encoded, _ := json.Marshal(base64.StdEncoding.EncodeToString(frame))
	return e.call(ctx, "globalThis.__whaleBrowserRecorder."+method+"("+string(encoded)+")")
}
func (e *nativeEncoder) Finish(ctx context.Context) (Artifact, error) {
	// Each bridge request has its own bounded timeout. A fixed total transfer
	// deadline would truncate legitimate larger artifacts before the 512 MiB cap.
	value, err := e.call(ctx, "globalThis.__whaleBrowserRecorder.finish()")
	if err != nil {
		return Artifact{}, err
	}
	var info struct {
		MIME  string `json:"mime"`
		Bytes int    `json:"bytes"`
	}
	if err = decodeResult(value, &info); err != nil {
		return Artifact{}, err
	}
	if !validMIME(info.MIME) || info.Bytes < 64 || info.Bytes > MaxBytes {
		return Artifact{}, errors.New(storelocale.T(storelocale.KeyStoreBrowserrecordingInvalidVideoEncodingLengthOrFormat))
	}
	result := Artifact{MIME: info.MIME, Data: make([]byte, 0, info.Bytes)}
	for len(result.Data) < info.Bytes {
		offset := len(result.Data)
		value, err = e.call(ctx, "globalThis.__whaleBrowserRecorder.read("+strconv.Itoa(offset)+")")
		if err != nil {
			return Artifact{}, err
		}
		var part struct {
			Offset int    `json:"offset"`
			Length int    `json:"length"`
			Data   string `json:"data"`
		}
		if err = decodeResult(value, &part); err != nil {
			return Artifact{}, err
		}
		expected := min(TransferChunkBytes, info.Bytes-offset)
		if part.Offset != offset || part.Length != expected || len(part.Data) != base64.StdEncoding.EncodedLen(expected) {
			return Artifact{}, errors.New(storelocale.T(storelocale.KeyStoreBrowserrecordingInvalidVideoOutputChunk))
		}
		data, err := base64.StdEncoding.DecodeString(part.Data)
		if err != nil || len(data) != expected {
			return Artifact{}, errors.New(storelocale.T(storelocale.KeyStoreBrowserrecordingTheVideoOutputChunkCouldNotBeDecoded))
		}
		result.Data = append(result.Data, data...)
	}
	if err = result.Validate(); err != nil {
		return Artifact{}, err
	}
	return result, nil
}
func decodeResult(value any, target any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
func (e *nativeEncoder) Close() {
	// Destroying the private renderer releases its canvas stream, MediaRecorder,
	// blob buffers and pending evaluations. The existing private-profile lifetime
	// owns eventual folder cleanup after BrowserProcessExited.
	e.once.Do(func() {
		if e.win != nil && !e.win.IsDestroyed() {
			e.win.Destroy()
		}
	})
}
