//go:build !windows

package main

// The installed reference is Windows. Other platforms retain their existing
// native-shell single-instance lock; Chromium's Windows lockfile is not a
// portable protocol.
func claimDesktopProfile() (func(), error) { return func() {}, nil }
