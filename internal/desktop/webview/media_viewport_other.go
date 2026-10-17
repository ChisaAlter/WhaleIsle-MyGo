//go:build !windows && !darwin

package webview

import (
	"errors"

	"github.com/egoist/mygo"
)

func clipMediaNativeWindow(_ *mygo.Window, _, _ int, _ []mygo.Rectangle) error {
	return errors.New("media: private inline browser is supported only on Windows and macOS")
}
