//go:build windows && (amd64 || arm64)

package windows

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestBrowserTextCDPDispatchMethodsAndParameters(t *testing.T) {
	start, end := 4, 7
	for _, tc := range []struct {
		name, method string
		event        platform.BrowserTextEvent
		params       map[string]any
	}{
		{"down", "Input.dispatchKeyEvent", platform.BrowserTextEvent{Kind: platform.BrowserTextKeyDown, Key: "A", Code: "KeyA", WindowsVirtualKeyCode: 65, NativeVirtualKeyCode: 30, Modifiers: platform.BrowserTextCtrl | platform.BrowserTextShift, Repeat: true}, map[string]any{"type": "rawKeyDown", "key": "A", "code": "KeyA", "windowsVirtualKeyCode": float64(65), "nativeVirtualKeyCode": float64(30), "modifiers": float64(10), "autoRepeat": true}},
		{"up", "Input.dispatchKeyEvent", platform.BrowserTextEvent{Kind: platform.BrowserTextKeyUp, Key: "Enter", Code: "Enter", WindowsVirtualKeyCode: 13}, map[string]any{"type": "keyUp", "key": "Enter", "code": "Enter", "windowsVirtualKeyCode": float64(13), "nativeVirtualKeyCode": float64(0), "modifiers": float64(0), "autoRepeat": false}},
		{"insert", "Input.insertText", platform.BrowserTextEvent{Kind: platform.BrowserTextInsertText, Text: "鲸屿🙂\n\"x\""}, map[string]any{"text": "鲸屿🙂\n\"x\""}},
		{"compose", "Input.imeSetComposition", platform.BrowserTextEvent{Kind: platform.BrowserTextComposition, Text: "中🙂", SelectionStart: 1, SelectionEnd: 3}, map[string]any{"text": "中🙂", "selectionStart": float64(1), "selectionEnd": float64(3)}},
		{"replace", "Input.imeSetComposition", platform.BrowserTextEvent{Kind: platform.BrowserTextComposition, Text: "🙂", SelectionStart: 2, SelectionEnd: 2, ReplacementStart: &start, ReplacementEnd: &end}, map[string]any{"text": "🙂", "selectionStart": float64(2), "selectionEnd": float64(2), "replacementStart": float64(4), "replacementEnd": float64(7)}},
		{"cancel", "Input.imeSetComposition", platform.BrowserTextEvent{Kind: platform.BrowserTextCancelComposition}, map[string]any{"text": "", "selectionStart": float64(0), "selectionEnd": float64(0)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, completions := 0, 0
			var reply func(string, error)
			dispatchBrowserTextInput(true, tc.event, func(method, raw string, done func(string, error)) {
				calls++
				if method != tc.method {
					t.Errorf("method %q want %q", method, tc.method)
				}
				var got map[string]any
				if err := json.Unmarshal([]byte(raw), &got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, tc.params) {
					t.Errorf("params %s, want %+v", raw, tc.params)
				}
				reply = done
			}, func(err error) {
				completions++
				if err != nil {
					t.Errorf("completion %v", err)
				}
			})
			if calls != 1 || completions != 0 || reply == nil {
				t.Fatalf("asynchronous dispatch calls=%d completions=%d", calls, completions)
			}
			reply("{}", nil)
			if completions != 1 {
				t.Fatalf("completion count %d", completions)
			}
		})
	}
}

func TestBrowserTextCDPRefusalAndNativeFailure(t *testing.T) {
	calls := 0
	call := func(string, string, func(string, error)) { calls++ }
	for _, tc := range []struct {
		ready bool
		event platform.BrowserTextEvent
	}{
		{false, platform.BrowserTextEvent{Kind: platform.BrowserTextInsertText, Text: "x"}},
		{true, platform.BrowserTextEvent{Kind: 99}},
		{true, platform.BrowserTextEvent{Kind: platform.BrowserTextComposition, Text: "🙂", SelectionEnd: 1}},
	} {
		var got error
		dispatchBrowserTextInput(tc.ready, tc.event, call, func(err error) { got = err })
		if got == nil {
			t.Fatal("invalid/not-ready dispatch succeeded")
		}
	}
	if calls != 0 {
		t.Fatal("refused input reached CDP")
	}
	failure := errors.New("native input failed")
	var got error
	dispatchBrowserTextInput(true, platform.BrowserTextEvent{Kind: platform.BrowserTextInsertText, Text: "x"}, func(_, _ string, done func(string, error)) { done("", failure) }, func(err error) { got = err })
	if !errors.Is(got, failure) {
		t.Fatalf("native failure lost: %v", got)
	}
	var w window
	if w.BrowserTextInputAvailable() {
		t.Fatal("uninitialized WebView advertised text input")
	}
	w.DispatchBrowserTextInput(platform.BrowserTextEvent{Kind: platform.BrowserTextInsertText, Text: "x"}, func(err error) {
		if !errors.Is(err, platform.ErrUnsupported) {
			t.Errorf("unready backend %v", err)
		}
	})
}
