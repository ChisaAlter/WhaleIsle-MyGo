//go:build !windows

package components

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func genericPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err != syscall.ESRCH
}

func genericInspectProcess(pid int) (exePath, cmdLine string, ok bool) {
	if pid <= 0 {
		return "", "", false
	}
	switch runtime.GOOS {
	case "linux":
		raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
		if err == nil {
			cmdLine = strings.TrimSpace(strings.ReplaceAll(string(raw), "\x00", " "))
		}
		exe, _ := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
		if cmdLine == "" && exe == "" {
			return "", "", false
		}
		return exe, cmdLine, true
	case "darwin":
		out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
		if err != nil {
			return "", "", false
		}
		line := strings.TrimSpace(string(out))
		if line == "" {
			return "", "", false
		}
		return "", line, true
	default:
		return "", "", false
	}
}

func genericMatchProcessIdentity(pid int, expected genericProcessExpectation, inspect func(int) (string, string, bool)) bool {
	if expected.EntryFile == "" && expected.ID == "" && expected.NodeBin == "" {
		return true
	}
	exePath, cmdLine, ok := inspect(pid)
	if !ok {
		return false
	}
	return matchCommandLineIdentity(exePath, cmdLine, expected)
}

func genericKillPID(pid int, force bool) bool {
	if pid <= 0 {
		return false
	}
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	return syscall.Kill(pid, sig) == nil
}
