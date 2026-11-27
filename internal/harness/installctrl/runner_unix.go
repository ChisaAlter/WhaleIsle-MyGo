//go:build !windows

package installctrl

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"whaleisle/internal/processutil"
)

func runPluginCommand(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(); err != nil {
		_ = r.Close()
		_ = w.Close()
		return nil, err
	}
	_ = w.Close()

	var buf bytes.Buffer
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		_, _ = io.Copy(&buf, io.LimitReader(r, maxInstallLogBytes+4096))
		_, _ = io.Copy(io.Discard, r)
	}()

	waitErr := cmd.Wait()
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	select {
	case <-drainDone:
	case <-time.After(processutil.PipeGrace):
	}
	_ = r.Close()
	<-drainDone

	return buf.Bytes(), waitErr
}
