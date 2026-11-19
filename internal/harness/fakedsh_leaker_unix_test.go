//go:build !windows

package harness

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// startStdoutLeaker leaves behind what a real Host leaves behind: a child of its
// own that inherits the stdout handle and outlives the Host's process group, so
// the shell's copy of that pipe never reaches EOF. The pid goes where the test
// can end it.
func startStdoutLeaker() {
	pidFile := os.Getenv("FAKE_DSH_LEAKER_PID")
	if pidFile == "" {
		return
	}
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "FAKE_DSH=leaker")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// A new session leaves the Host's process group, which is what the Host's
	// own daemons do: stopping the group no longer reaches them.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		fmt.Println("leaker:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600); err != nil {
		fmt.Println("leaker pid:", err)
		os.Exit(1)
	}
	_ = cmd.Process.Release()
}
