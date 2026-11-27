//go:build windows

package installctrl

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"whaleisle/internal/processutil"
)

func runPluginCommand(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	processutil.BackgroundTree(cmd)

	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = w
	cmd.Stderr = w

	job, jobErr := createPluginJob()
	var jobMu sync.Mutex
	terminateJob := func() {
		jobMu.Lock()
		h := job
		jobMu.Unlock()
		if h != 0 {
			_ = windows.TerminateJobObject(h, 1)
		}
	}
	closeJob := func() {
		jobMu.Lock()
		h := job
		job = 0
		jobMu.Unlock()
		if h != 0 {
			_ = windows.CloseHandle(h)
		}
	}
	defer closeJob()

	var assigned bool
	cmd.Cancel = func() error {
		jobMu.Lock()
		hasJob := assigned && job != 0
		h := job
		jobMu.Unlock()
		if hasJob {
			_ = windows.TerminateJobObject(h, 1)
			return nil
		}
		if cmd.Process != nil {
			killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			kill := processutil.CommandContext(killCtx, "taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
			processutil.Background(kill)
			_ = kill.Run()
			return cmd.Process.Kill()
		}
		return nil
	}

	if err := cmd.Start(); err != nil {
		_ = r.Close()
		_ = w.Close()
		return nil, err
	}
	_ = w.Close()

	if jobErr == nil && job != 0 && cmd.Process != nil {
		if proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid)); err == nil {
			if windows.AssignProcessToJobObject(job, proc) == nil {
				jobMu.Lock()
				assigned = true
				jobMu.Unlock()
			}
			_ = windows.CloseHandle(proc)
		}
	}
	jobMu.Lock()
	isAssigned := assigned
	jobMu.Unlock()
	if !isAssigned {
		closeJob()
	}

	var buf bytes.Buffer
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		_, _ = io.Copy(&buf, io.LimitReader(r, maxInstallLogBytes+4096))
		_, _ = io.Copy(io.Discard, r)
	}()

	waitErr := cmd.Wait()

	// Ensure no descendant (pnpm/postinstall worker) survives to hold locks in
	// profile/node_modules before rollbackInstall or subsequent profile operations.
	if isAssigned {
		terminateJob()
		waitPluginJobEmpty(job, 2*time.Second)
		closeJob()
	} else if ctx.Err() != nil && cmd.Process != nil {
		killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		kill := processutil.CommandContext(killCtx, "taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
		processutil.Background(kill)
		_ = kill.Run()
		cancel()
	}

	select {
	case <-drainDone:
	case <-time.After(processutil.PipeGrace):
	}
	_ = r.Close()
	<-drainDone

	return buf.Bytes(), waitErr
}

func createPluginJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)),
		uint32(unsafe.Sizeof(limits)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

func waitPluginJobEmpty(job windows.Handle, budget time.Duration) {
	if job == 0 {
		return
	}
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		var data struct {
			Assigned, Listed uint32
			PIDs             [64]uintptr
		}
		var returned uint32
		if err := windows.QueryInformationJobObject(
			job,
			windows.JobObjectBasicProcessIdList,
			uintptr(unsafe.Pointer(&data)),
			uint32(unsafe.Sizeof(data)),
			&returned,
		); err != nil {
			return
		}
		if data.Assigned == 0 {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
}
