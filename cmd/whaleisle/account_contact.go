package main

import (
	"github.com/egoist/mygo"
	"runtime"
	"whaleisle/internal/accountcontact"
	"whaleisle/internal/accountdetails"
	desktopwebview "whaleisle/internal/desktop/webview"
)

// The account launcher and Account settings share this explicit contact action.
func (s *mainState) contactAccountSupport() {
	s.mu.Lock()
	store := s.usage
	s.mu.Unlock()
	context := accountcontact.Context{Version: buildVersion, DeviceInfo: runtime.GOOS + "; " + runtime.GOARCH, PixelRatio: 1}
	if store != nil {
		profile := store.Snapshot().Details.Profile
		if profile.Status == accountdetails.Ready && profile.Value.ID != nil {
			context.UID = *profile.Value.ID
		}
	}
	display := mygo.Screen.PrimaryDisplay()
	context.Width, context.Height, context.PixelRatio = float64(display.Bounds.Width), float64(display.Bounds.Height), display.ScaleFactor
	s.reportAccountAction(accountcontact.Open(context, nil, desktopwebview.OpenExternal))
}
