package webview

import (
	"errors"
	"strings"

	"github.com/egoist/mygo"
)

// MyGo's window constructor represents backend initialization failure as a
// prefixed panic. Translate only that documented call-site failure so an older
// OS or missing WebKit runtime produces a tab error, not an application crash.
// Programmer errors and unrelated panics must retain their original behavior.
func createBrowserHostWindow(opts mygo.WindowOptions) (*mygo.Window, error) {
	return recoverBackendWindowCreation(func() *mygo.Window { return mygo.NewWindow(opts) })
}

func recoverBackendWindowCreation(create func() *mygo.Window) (window *mygo.Window, err error) {
	defer func() {
		if value := recover(); value != nil {
			if message, ok := value.(string); ok && strings.HasPrefix(message, "mygo: cannot create window: ") {
				window, err = nil, errors.New(message)
				return
			}
			panic(value)
		}
	}()
	return create(), nil
}
