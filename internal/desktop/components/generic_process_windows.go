//go:build windows

package components

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"whaleisle/internal/processutil"
)

func genericPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := syscall.OpenProcess(0x1000|0x100000, false, uint32(pid))
	if err != nil {
		return err != syscall.Errno(87)
	} // Access denied is not evidence of exit.
	defer syscall.CloseHandle(handle)
	result, err := syscall.WaitForSingleObject(handle, 0)
	return err != nil || result == syscall.WAIT_TIMEOUT
}

func genericInspectProcess(pid int) (exePath, cmdLine string, ok bool) {
	if pid <= 0 {
		return "", "", false
	}
	handle, err := syscall.OpenProcess(0x1000, false, uint32(pid))
	if err != nil {
		return "", "", false
	}
	defer syscall.CloseHandle(handle)
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	r1, _, _ := bridgeQueryImage.Call(uintptr(handle), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r1 == 0 {
		return "", "", false
	}
	return syscall.UTF16ToString(buf[:size]), "", true
}

func genericMatchProcessIdentity(pid int, expected genericProcessExpectation, inspect func(int) (string, string, bool)) bool {
	if expected.EntryFile == "" && expected.ID == "" && expected.NodeBin == "" {
		return true
	}
	exePath, cmdLine, ok := inspect(pid)
	if !ok {
		return false
	}
	if cmdLine != "" {
		return matchCommandLineIdentity(exePath, cmdLine, expected)
	}
	// On Windows without command-line reading, verify executable identity when
	// known so an unrelated OS process reusing the PID is rejected.
	if expected.NodeBin != "" && sameWindowsExePath(exePath, expected.NodeBin) {
		return true
	}
	if expected.EntryFile != "" && strings.EqualFold(filepath.Ext(expected.EntryFile), ".exe") && sameWindowsExePath(exePath, expected.EntryFile) {
		return true
	}
	if expected.NodeBin == "" {
		base := strings.ToLower(filepath.Base(exePath))
		if base == "node.exe" || (expected.ID != "" && strings.Contains(base, strings.ToLower(expected.ID))) {
			return true
		}
	}
	return false
}

func genericKillPID(pid int, force bool) bool {
	if pid <= 0 {
		return false
	}
	args := []string{"/PID", strconv.Itoa(pid)}
	if force {
		args = append(args, "/T", "/F")
	}
	cmd := exec.Command("taskkill", args...)
	processutil.Background(cmd)
	return cmd.Run() == nil
}
