package platform

// KeyInput describes one key event before the page handles it. Backends fill it
// from their own event (Windows: WebView2's AcceleratorKeyPressed), so it is
// deliberately small: the key's lower-case name and the modifiers held.
type KeyInput struct {
	Key                       string
	Control, Shift, Alt, Meta bool
	Down                      bool
}

// keyInputWindow is implemented by backend windows that can report key input
// before the page handles it. A backend without such an event simply does not
// implement it, so adding the capability never breaks a platform.
type keyInputWindow interface {
	SetBeforeInput(fn func(KeyInput) bool)
}

// SetBeforeInput installs fn on w and reports whether the backend supports it.
// Returning true from fn marks the key handled (the page never sees it).
func SetBeforeInput(w any, fn func(KeyInput) bool) bool {
	k, ok := w.(keyInputWindow)
	if !ok {
		return false
	}
	k.SetBeforeInput(fn)
	return true
}
