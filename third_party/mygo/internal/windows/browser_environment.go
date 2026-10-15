//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// This is a persistent Browser-only UDF, not an IsolatedPreview. The existing
// createEnvironment loader and verified Environment7 ABI are reused unchanged.
// https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/webview2-idl?view=webview2-1.0.3537.50#createcorewebview2environmentwithoptions
// https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2environment7?view=webview2-1.0.3537.50
// All fields, callbacks and COM references are confined to MyGo's main thread.
type browserEnvironment struct {
	owners browserEnvironmentOwners[*window]
	env    uintptr
	err    error
}

func browserEnvironmentFolder(env uintptr) (string, error) {
	v7 := queryInterface(env, &iidPreviewEnvironment7)
	if v7 == 0 {
		return "", errors.New("mygo: Browser profile requires WebView2 Environment7")
	}
	defer release(v7)
	var raw uintptr
	hr := comCall(v7, previewGetUserDataFolder, uintptr(unsafe.Pointer(&raw)))
	folder := takeWstr(raw)
	if failed(hr) || !filepath.IsAbs(folder) {
		return "", errors.New("mygo: cannot verify Browser user data folder")
	}
	return filepath.Clean(folder), nil
}

func browserProfileDirectory(appName, shared string) (string, error) {
	// Never change global overrides, silently fall back to the shared profile,
	// or accept an arbitrary caller-supplied path.
	for _, name := range []string{"WEBVIEW2_USER_DATA_FOLDER", "WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "WEBVIEW2_PIPE_FOR_SCRIPT_DEBUGGER", "WEBVIEW2_WAIT_FOR_SCRIPT_DEBUGGER"} {
		if os.Getenv(name) != "" {
			return "", fmt.Errorf("mygo: Browser profile rejects %s override", name)
		}
	}
	base := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(base) {
		return "", errors.New("mygo: Browser profile requires an absolute LOCALAPPDATA directory")
	}
	folder := filepath.Join(base, sanitize(appName), "WebView2-Browser")
	if strings.EqualFold(filepath.Clean(shared), folder) {
		return "", errors.New("mygo: Browser profile overlaps the application's shared profile")
	}
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return "", fmt.Errorf("mygo: create Browser profile directory: %w", err)
	}
	info, err := os.Lstat(folder)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("mygo: Browser profile directory must be a real directory")
	}
	sharedInfo, err := os.Stat(shared)
	if err != nil || os.SameFile(info, sharedInfo) {
		return "", errors.New("mygo: cannot establish a distinct Browser profile directory")
	}
	return folder, nil
}

func (w *window) startBrowserEnvironment() {
	if w.closed || w.destroying {
		return
	}
	// The default environment is already initialized by NewWindow. Comparing
	// its effective folder also detects registry policy redirecting it to the
	// Browser UDF, including when Browser is the first page in this process.
	if w.b.envErr != nil {
		w.webViewFailed(w.b.envErr)
		return
	}
	p := w.b.browser
	if p == nil {
		p = &browserEnvironment{}
		w.b.browser = p
	}
	if !p.owners.add(w) {
		w.webViewFailed(errors.New("mygo: Browser environment has closed"))
		return
	}
	w.browser = p
	if p.env != 0 {
		w.createWebView()
		return
	}
	if p.err != nil {
		w.webViewFailed(p.err)
		return
	}
	if len(p.owners.members) != 1 {
		return // The first owner is already creating this environment.
	}
	shared, err := browserEnvironmentFolder(w.b.env)
	if err != nil {
		p.complete(0, err)
		return
	}
	folder, err := browserProfileDirectory(w.b.name, shared)
	if err != nil {
		p.complete(0, err)
		return
	}
	err = createEnvironment(folder, func(env uintptr, err error) {
		if p.owners.closed {
			if env != 0 {
				release(env)
			}
			return
		}
		if err == nil {
			var actual string
			actual, err = browserEnvironmentFolder(env)
			if err == nil && !strings.EqualFold(actual, folder) {
				err = errors.New("mygo: Browser user data folder was overridden")
			}
		}
		p.complete(env, err)
	})
	if err != nil {
		p.complete(0, err)
	}
}

func (p *browserEnvironment) complete(env uintptr, err error) {
	if p.owners.closed || err != nil {
		if env != 0 {
			release(env)
		}
		if p.owners.closed {
			return
		}
		env = 0
	}
	p.env, p.err = env, err
	for w := range p.owners.members {
		if w.closed || w.destroying || w.browser != p {
			continue
		}
		if err != nil {
			w.webViewFailed(err)
		} else {
			w.createWebView()
		}
	}
}

func (w *window) releaseBrowserEnvironment() {
	p := w.browser
	w.browser = nil
	if p == nil || !p.owners.remove(w) {
		return
	}
	if w.b.browser == p {
		w.b.browser = nil
	}
	// cleanup closed the native controller first. A pending environment callback
	// sees the retired generation and releases its result without creating a page.
	if p.env != 0 {
		release(p.env)
		p.env = 0
	}
	// Persistent profile data is deliberately never removed or migrated here.
}
