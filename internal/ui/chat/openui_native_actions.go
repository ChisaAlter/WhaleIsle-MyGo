package chat

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/desktop/webview"
)

// executeOpenUINativeAction executes only an owned-runtime, one-use request
// while its exact block/session and original owner are still current. The
// owned viewer synchronously checks the real UI event; an arbitrary renderer's
// userGesture boolean alone is never authority to reach this path.
func executeOpenUINativeAction(ctx context.Context, b *openUIBlock, handle *previewprocess.Session, valid func() bool, event openui.Event) openui.APIResponse {
	status := 409
	err := errors.New("OpenUI action owner is no longer active")
	current := func() bool {
		if ctx.Err() != nil || valid == nil || !valid() {
			return false
		}
		select {
		case <-handle.Done():
			return false
		default:
		}
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.handle == handle && b.valid != nil
	}
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		status, err = 501, errors.New("OpenUI native actions are unavailable on this platform")
	} else if current() {
		err = errors.New("the application UI loop is unavailable")
		status = 503
		mygo.RunOnMain(func() {
			if !current() {
				status, err = 409, errors.New("OpenUI action owner changed before execution")
				return
			}
			if claimErr := handle.ClaimOpenUINativeAction(event); claimErr != nil {
				status, err = 403, claimErr
				return
			}
			if !current() {
				status, err = 409, errors.New("OpenUI action owner changed before execution")
				return
			}
			switch event.Type {
			case "clipboard":
				// WriteText discards its native error; use the error-returning API.
				err = mygo.Clipboard.Write(transfer.TextData(event.Text))
			case "open-url":
				// Existing HTTP(S)-only product route calls MyGo's native shell.
				err = webview.OpenExternal(event.URL)
			default:
				err = errors.New("unsupported OpenUI native action")
			}
			if err != nil {
				status = 500
			} else {
				status = 200
			}
		})
	}
	body := struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}{OK: err == nil}
	if err != nil {
		body.Error = err.Error()
		if len(body.Error) > 4096 {
			body.Error = body.Error[:4096]
			for !utf8.ValidString(body.Error) {
				body.Error = body.Error[:len(body.Error)-1]
			}
		}
	}
	raw, _ := json.Marshal(body)
	return openui.APIResponse{InstanceID: event.InstanceID, RequestID: event.RequestID, Status: status, Body: raw}
}
