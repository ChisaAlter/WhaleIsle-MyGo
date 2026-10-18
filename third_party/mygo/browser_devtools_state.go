package mygo

import "errors"

var ErrBrowserDevToolsUnsupported = errors.New("mygo: developer tools require a ready, verified Browser workspace page without native bindings")

type browserDevToolsBackend interface {
	BrowserDevToolsAvailable() bool
	OpenBrowserDevTools() error
}

// Main-thread policy only; no page bridge or general inspector opt-in is added.
func browserDevToolsProvider(native any, bindingsDisabled bool) browserDevToolsBackend {
	if !bindingsDisabled {
		return nil
	}
	n, ok := native.(browserDevToolsBackend)
	if !ok || !n.BrowserDevToolsAvailable() {
		return nil
	}
	return n
}

func openBrowserDevTools(native any, bindingsDisabled bool) error {
	n := browserDevToolsProvider(native, bindingsDisabled)
	if n == nil {
		return ErrBrowserDevToolsUnsupported
	}
	return n.OpenBrowserDevTools()
}
