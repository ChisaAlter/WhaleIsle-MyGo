//go:build windows

package components

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSameWindowsExePathAndTerminateContext(t *testing.T) {
	sysExe := filepath.Join(t.TempDir(), "dummy.exe")
	if !sameWindowsExePath(`\\?\`+sysExe, sysExe) {
		t.Fatalf("expected extended-length prefix path to match normal path")
	}

	cmd := exec.Command("cmd", "/c", "ping", "-n", "10", "127.0.0.1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start ping: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	proc, err := openBridgeProcess(cmd.Process.Pid, cmd.Path)
	if err != nil {
		t.Fatalf("openBridgeProcess failed for matching exe: %v", err)
	}
	defer proc.Close()

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := proc.Terminate(cancelled); err == nil {
		t.Fatal("expected Terminate with canceled context to return error immediately")
	}
}
