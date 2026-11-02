package main

import (
	"github.com/egoist/mygo"

	"whaleisle/internal/product"
)

// configureDesktopName retains native window state and webview data when the
// display name changes. Isolated and portable profiles set their own paths.
func configureDesktopName() {
	if !isolatedDesktopRun() {
		mygo.App.SetName(product.LegacyNativeName)
		for _, key := range []mygo.PathName{mygo.PathUserData, mygo.PathCache, mygo.PathLogs} {
			if path, err := mygo.App.Path(key); err == nil {
				mygo.App.SetPath(key, path)
			}
		}
	}
	mygo.App.SetName(product.Name)
}
