package platform

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// BrowserTextKind permits only native key, committed text and IME operations.
type BrowserTextKind uint8

const (
	BrowserTextKeyDown BrowserTextKind = iota
	BrowserTextKeyUp
	BrowserTextInsertText
	BrowserTextComposition
	BrowserTextCancelComposition
)

// BrowserTextModifiers uses CDP bits, independently of any UI toolkit's flags.
type BrowserTextModifiers uint8

const (
	BrowserTextAlt BrowserTextModifiers = 1 << iota
	BrowserTextCtrl
	BrowserTextMeta
	BrowserTextShift
)

// BrowserTextEvent has separate key and text forms. KeyDown/KeyUp use Key, Code,
// virtual key codes and Modifiers; only KeyDown permits Repeat. They never insert
// Text. InsertText commits Text. Composition uses Text and UTF-16 selection
// offsets (which must not split a surrogate pair). Optional replacement offsets
// refer to the existing editable text, not Text; both nil leaves them unknown.
// Replacement offsets must both be supplied, nonnegative and ordered. The
// caller owns focus/document policy and maps native toolkit modifier flags.
// CancelComposition accepts only Kind and cancels without committing text.
type BrowserTextEvent struct {
	Kind                                        BrowserTextKind
	Key, Code                                   string
	WindowsVirtualKeyCode, NativeVirtualKeyCode int
	Modifiers                                   BrowserTextModifiers
	Repeat                                      bool
	Text                                        string
	SelectionStart, SelectionEnd                int
	ReplacementStart, ReplacementEnd            *int
}

// CloneBrowserTextEvent detaches optional ranges before asynchronous submission.
func CloneBrowserTextEvent(e BrowserTextEvent) BrowserTextEvent {
	if e.ReplacementStart != nil {
		v := *e.ReplacementStart
		e.ReplacementStart = &v
	}
	if e.ReplacementEnd != nil {
		v := *e.ReplacementEnd
		e.ReplacementEnd = &v
	}
	return e
}

func ValidateBrowserTextEvent(e BrowserTextEvent) error {
	bad := func(reason string) error { return fmt.Errorf("mygo: invalid browser text input: %s", reason) }
	if e.Kind > BrowserTextCancelComposition || e.Modifiers & ^BrowserTextModifiers(15) != 0 {
		return bad("kind or modifier bits")
	}
	if len(e.Text) > 64<<10 || !utf8.ValidString(e.Text) {
		return bad("text must be valid UTF-8 of at most 64 KiB")
	}
	key := e.Kind == BrowserTextKeyDown || e.Kind == BrowserTextKeyUp
	if key {
		for _, s := range []string{e.Key, e.Code} {
			if s == "" || len(s) > 128 || !utf8.ValidString(s) {
				return bad("key and code must be nonempty bounded UTF-8")
			}
			for _, r := range s {
				if unicode.IsControl(r) {
					return bad("control character in key or code")
				}
			}
		}
		if e.WindowsVirtualKeyCode < 0 || e.WindowsVirtualKeyCode > 255 || e.NativeVirtualKeyCode < 0 || e.NativeVirtualKeyCode > 65535 || e.Kind == BrowserTextKeyUp && e.Repeat {
			return bad("virtual key code or repeated key-up")
		}
		if e.Text != "" || e.SelectionStart != 0 || e.SelectionEnd != 0 || e.ReplacementStart != nil || e.ReplacementEnd != nil {
			return bad("key events cannot carry text or composition ranges")
		}
		return nil
	}
	if e.Key != "" || e.Code != "" || e.WindowsVirtualKeyCode != 0 || e.NativeVirtualKeyCode != 0 || e.Modifiers != 0 || e.Repeat {
		return bad("text operations cannot carry key payload")
	}
	if e.Kind != BrowserTextComposition {
		if e.SelectionStart != 0 || e.SelectionEnd != 0 || e.ReplacementStart != nil || e.ReplacementEnd != nil || e.Kind == BrowserTextCancelComposition && e.Text != "" {
			return bad("unexpected composition payload")
		}
		return nil
	}
	if e.SelectionStart < 0 || e.SelectionEnd < e.SelectionStart {
		return bad("composition selection must be nonnegative and ordered")
	}
	units, startOK, endOK := 0, e.SelectionStart == 0, e.SelectionEnd == 0
	for _, r := range e.Text {
		units++
		if r > 0xffff {
			units++
		}
		startOK = startOK || units == e.SelectionStart
		endOK = endOK || units == e.SelectionEnd
	}
	if !startOK || !endOK {
		return bad("selection outside text or splitting a UTF-16 surrogate pair")
	}
	if (e.ReplacementStart == nil) != (e.ReplacementEnd == nil) {
		return bad("replacement offsets must both be supplied")
	}
	if e.ReplacementStart != nil && (*e.ReplacementStart < 0 || *e.ReplacementEnd < *e.ReplacementStart || *e.ReplacementEnd > 0x7fffffff) {
		return bad("replacement offsets must be nonnegative ordered 32-bit offsets")
	}
	return nil
}

// BrowserTextInputWindow is optional; unsupported backends need no new Window
// methods. Calls and completions run on the UI thread. A provider never queues
// an operation for a future WebView or document.
type BrowserTextInputWindow interface {
	BrowserTextInputAvailable() bool
	DispatchBrowserTextInput(BrowserTextEvent, func(error))
}

func BrowserTextInputProvider(w any) BrowserTextInputWindow {
	n, ok := w.(BrowserTextInputWindow)
	if !ok || !n.BrowserTextInputAvailable() {
		return nil
	}
	return n
}
