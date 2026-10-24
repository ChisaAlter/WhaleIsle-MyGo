package mygo

import "github.com/egoist/mygo/internal/platform"

// KeyInput is one key event before the page handles it: the key's lower-case
// name and the modifiers held. Down is true for a key press and false for a
// release.
type KeyInput struct {
	Key                       string
	Control, Shift, Alt, Meta bool
	Down                      bool
}

// OnBeforeInput calls fn for key events of this page before the page handles
// them; returning true marks the key handled, so the page never sees it. It
// gives the app what Electron calls before-input-event, which is how a
// window-scoped shortcut (Ctrl+Shift+I) is implemented without registering an
// OS-global hotkey.
//
// It reports whether the platform can report such events. Windows does, through
// WebView2's AcceleratorKeyPressed; macOS and Linux web views expose no
// pre-handling key event, so there it returns false and fn never runs.
func (p *Page) OnBeforeInput(fn func(KeyInput) bool) bool {
	if p == nil || p.w == nil || fn == nil {
		return false
	}
	supported := false
	p.w.page(func(n platform.Window) {
		supported = platform.SetBeforeInput(n, func(in platform.KeyInput) bool {
			return fn(KeyInput{
				Key: in.Key, Control: in.Control, Shift: in.Shift,
				Alt: in.Alt, Meta: in.Meta, Down: in.Down,
			})
		})
	})
	return supported
}

// SetDevToolsEnabled turns this page's web inspector on or off at runtime. It
// reports whether the platform supports it. Windows applies it to WebView2's
// AreDevToolsEnabled, which is what the built-in Ctrl+Shift+I accelerator
// consults, so a denied page cannot open the inspector even though the
// accelerator still reaches the webview. Other backends keep the value but do
// not implement the switch yet.
func (p *Page) SetDevToolsEnabled(enabled bool) bool {
	if p == nil || p.w == nil {
		return false
	}
	supported := false
	p.w.page(func(n platform.Window) {
		supported = platform.SetDevToolsEnabled(n, enabled)
	})
	return supported
}
