//go:build windows

package components

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
	"whaleisle/internal/store/storelocale"
)

var bridgeQueryImage = syscall.NewLazyDLL("kernel32.dll").NewProc("QueryFullProcessImageNameW")

type bridgeWindowsProcess struct{ handle syscall.Handle }

func openBridgeProcess(pid int, exe string) (bridgeProcess, error) {
	if pid <= 0 {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsInvalidWhalebridgeProcessIdentity))
	}
	// Query limited information, synchronize, terminate. The same handle remains
	// open through status verification and termination, preventing PID reuse races.
	h, err := syscall.OpenProcess(0x1000|0x100000|0x1, false, uint32(pid))
	if err != nil {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsCouldNotVerifyTheWhalebridgeProcessSo))
	}
	p := &bridgeWindowsProcess{handle: h}
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	ok, _, _ := bridgeQueryImage.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	actual := syscall.UTF16ToString(buf[:size])
	if ok == 0 || !sameWindowsExePath(actual, exe) {
		p.Close()
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeProcessIsNotTheInstalled))
	}
	return p, nil
}

func normalizeWindowsExePath(p string) (string, error) {
	resolved, err := filepath.EvalSymlinks(p)
	if err == nil {
		p = resolved
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	abs = strings.TrimPrefix(abs, `\\?\`)
	return filepath.Clean(abs), nil
}

func sameWindowsExePath(actual, expected string) bool {
	expNorm, expErr := normalizeWindowsExePath(expected)
	actNorm, actErr := normalizeWindowsExePath(actual)
	if expErr != nil || actErr != nil {
		return false
	}
	return strings.EqualFold(actNorm, expNorm)
}

func (p *bridgeWindowsProcess) Close() { _ = syscall.CloseHandle(p.handle) }
func (p *bridgeWindowsProcess) Terminate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := syscall.TerminateProcess(p.handle, 1)
	deadline := time.Now().Add(10 * time.Second)
	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		status, waitErr := syscall.WaitForSingleObject(p.handle, 100)
		if waitErr == nil && status == syscall.WAIT_OBJECT_0 {
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil || waitErr != nil || time.Now().After(deadline) {
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeProcessExitCouldNotBeCo))
		}
	}
}
