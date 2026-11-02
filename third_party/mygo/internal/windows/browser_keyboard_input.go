//go:build windows && (amd64 || arm64)

package windows

import (
	"encoding/json"

	"github.com/egoist/mygo/internal/platform"
)

func (w *window) BrowserTextInputAvailable() bool {
	return !w.closed && !w.destroying && w.surface == nil && w.ready && w.webview != 0
}

// DispatchBrowserTextInput uses only fixed Input commands on this WebView. The
// existing devtools adapter owns async completion and its static COM callbacks;
// no callback trampoline or DevTools window is created for an input operation.
func (w *window) DispatchBrowserTextInput(e platform.BrowserTextEvent, done func(error)) {
	dispatchBrowserTextInput(w.BrowserTextInputAvailable(), e, w.devtools, done)
}

func dispatchBrowserTextInput(ready bool, e platform.BrowserTextEvent, call func(string, string, func(string, error)), done func(error)) {
	if err := platform.ValidateBrowserTextEvent(e); err != nil {
		done(err)
		return
	}
	if !ready {
		done(platform.ErrUnsupported)
		return
	}
	var method string
	var params any
	switch e.Kind {
	case platform.BrowserTextKeyDown, platform.BrowserTextKeyUp:
		kind := "rawKeyDown"
		if e.Kind == platform.BrowserTextKeyUp {
			kind = "keyUp"
		}
		method = "Input.dispatchKeyEvent"
		params = struct {
			Type                  string                        `json:"type"`
			Key                   string                        `json:"key"`
			Code                  string                        `json:"code"`
			WindowsVirtualKeyCode int                           `json:"windowsVirtualKeyCode"`
			NativeVirtualKeyCode  int                           `json:"nativeVirtualKeyCode"`
			Modifiers             platform.BrowserTextModifiers `json:"modifiers"`
			AutoRepeat            bool                          `json:"autoRepeat"`
		}{kind, e.Key, e.Code, e.WindowsVirtualKeyCode, e.NativeVirtualKeyCode, e.Modifiers, e.Repeat}
	case platform.BrowserTextInsertText:
		method = "Input.insertText"
		params = struct {
			Text string `json:"text"`
		}{e.Text}
	case platform.BrowserTextComposition, platform.BrowserTextCancelComposition:
		method = "Input.imeSetComposition"
		params = struct {
			Text             string `json:"text"`
			SelectionStart   int    `json:"selectionStart"`
			SelectionEnd     int    `json:"selectionEnd"`
			ReplacementStart *int   `json:"replacementStart,omitempty"`
			ReplacementEnd   *int   `json:"replacementEnd,omitempty"`
		}{e.Text, e.SelectionStart, e.SelectionEnd, e.ReplacementStart, e.ReplacementEnd}
	}
	raw, err := json.Marshal(params)
	if err != nil {
		done(err)
		return
	}
	call(method, string(raw), func(_ string, err error) { done(err) })
}
