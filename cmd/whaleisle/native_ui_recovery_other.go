//go:build !windows

package main

import "fmt"

type nativeRecoveryProcess struct{}

func openNativeRecoveryProcess(pid int, executable string) (*nativeRecoveryProcess, error) {
	return nil, fmt.Errorf("independent native UI recovery is available on Windows")
}
func (*nativeRecoveryProcess) alive() bool { return false }
func (*nativeRecoveryProcess) close()      {}
func (*nativeRecoveryProcess) stopTree(int) error {
	return fmt.Errorf("native process recovery is unavailable on this platform")
}
